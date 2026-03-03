package scheduler

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/bsm/redislock"
	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"

	"github.com/asynq-test/internal/config"
	"github.com/asynq-test/internal/tasks"
)

// NewScheduler creates an asynq.Scheduler with configured hooks and timezone.
func NewScheduler(cfg config.Config) (*asynq.Scheduler, error) {
	redisOpt := asynq.RedisClientOpt{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	}

	loc, err := time.LoadLocation(cfg.Scheduler.Timezone)
	if err != nil {
		log.Printf("[%s] Invalid timezone %q, falling back to UTC", cfg.Pod.ID, cfg.Scheduler.Timezone)
		loc = time.UTC
	}

	podID := cfg.Pod.ID

	scheduler := asynq.NewScheduler(redisOpt, &asynq.SchedulerOpts{
		Location: loc,
		LogLevel: asynq.InfoLevel,
		// PreEnqueueFunc logs each enqueue attempt with pod identity.
		PreEnqueueFunc: func(task *asynq.Task, opts []asynq.Option) {
			log.Printf("[%s] SCHEDULER attempting to enqueue %s", podID, task.Type())
		},

		// PostEnqueueFunc logs success or expected ErrDuplicateTask.
		PostEnqueueFunc: func(info *asynq.TaskInfo, err error) {
			if err != nil {
				log.Printf("[%s] SCHEDULER enqueue result: %v (expected for dedup)", podID, err)
				return
			}
			log.Printf("[%s] SCHEDULER enqueued %s id=%s queue=%s",
				podID, info.Type, info.ID, info.Queue)
		},
	})

	return scheduler, nil
}

// RegisterTasks registers configured periodic tasks on a scheduler.
func RegisterTasks(s *asynq.Scheduler, cfg config.Config) error {
	for _, td := range cfg.Scheduler.Tasks {
		task, opts, err := buildPeriodicTask(td)
		if err != nil {
			return err
		}

		entryID, err := s.Register(td.Cronspec, task, opts...)
		if err != nil {
			return err
		}
		log.Printf("[%s] SCHEDULER registered %s (cron=%q, queue=%s, unique_ttl=%v) entry=%s",
			cfg.Pod.ID, td.Type, td.Cronspec, td.Queue, td.UniqueTTL, entryID)
	}
	return nil
}

// RunWithLeaderElection runs scheduler only while this pod holds the Redis lock.
// On lock refresh failure, scheduler is stopped immediately to avoid split-brain.
func RunWithLeaderElection(ctx context.Context, cfg config.Config) error {
	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	defer redisClient.Close()
	locker := redislock.New(redisClient)
	inspector := asynq.NewInspector(asynq.RedisClientOpt{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	defer inspector.Close()

	lockKey := cfg.Scheduler.LeaderLockKey
	if lockKey == "" {
		lockKey = "asynq-scheduler-leader"
	}
	lockTTL := cfg.Scheduler.LeaderLockTTL
	if lockTTL <= 0 {
		lockTTL = 15 * time.Second
	}
	refreshEvery := cfg.Scheduler.LeaderRefreshInterval
	if refreshEvery <= 0 {
		refreshEvery = lockTTL / 3
	}
	if refreshEvery <= 0 {
		refreshEvery = 5 * time.Second
	}
	retryEvery := cfg.Scheduler.LeaderRetryInterval
	if retryEvery <= 0 {
		retryEvery = 2 * time.Second
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		lock, err := locker.Obtain(ctx, lockKey, lockTTL, nil)
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return nil
			}
			if errors.Is(err, redislock.ErrNotObtained) {
				log.Printf("[%s] Scheduler standby (leader lock held by another pod)", cfg.Pod.ID)
			} else {
				log.Printf("[%s] Scheduler standby (leader lock not acquired): %v", cfg.Pod.ID, err)
			}
			select {
			case <-ctx.Done():
				return nil
			case <-time.After(retryEvery):
			}
			continue
		}

		log.Printf("[%s] Scheduler leader lock acquired (key=%q ttl=%s)", cfg.Pod.ID, lockKey, lockTTL)

		if err := cleanupSchedulerState(ctx, redisClient, inspector, cfg.Pod.ID); err != nil {
			log.Printf("[%s] Scheduler cleanup warning: %v", cfg.Pod.ID, err)
		}

		sched, err := NewScheduler(cfg)
		if err != nil {
			_ = lock.Release(context.Background())
			return err
		}

		if err := RegisterTasks(sched, cfg); err != nil {
			sched.Shutdown()
			_ = lock.Release(context.Background())
			return err
		}

		if err := sched.Start(); err != nil {
			sched.Shutdown()
			_ = lock.Release(context.Background())
			return err
		}

		leaderCtx, cancelLeader := context.WithCancel(ctx)
		refreshErrCh := make(chan error, 1)
		go func() {
			ticker := time.NewTicker(refreshEvery)
			defer ticker.Stop()
			for {
				select {
				case <-leaderCtx.Done():
					return
				case <-ticker.C:
					if err := lock.Refresh(leaderCtx, lockTTL, nil); err != nil {
						refreshErrCh <- err
						return
					}
				}
			}
		}()

		select {
		case <-ctx.Done():
		case err := <-refreshErrCh:
			log.Printf("[%s] Scheduler leadership lost: %v", cfg.Pod.ID, err)
		}

		cancelLeader()
		sched.Shutdown()

		releaseCtx, releaseCancel := context.WithTimeout(context.Background(), 2*time.Second)
		err = lock.Release(releaseCtx)
		releaseCancel()
		if err != nil {
			if !errors.Is(err, redislock.ErrLockNotHeld) {
				log.Printf("[%s] Scheduler leader lock release error: %v", cfg.Pod.ID, err)
			}
		}

		if ctx.Err() != nil {
			return nil
		}

		select {
		case <-ctx.Done():
			return nil
		case <-time.After(retryEvery):
		}
	}
}

// cleanupSchedulerState clears existing scheduler metadata before registering.
// Asynq v0.26.0 does not expose Scheduler.History on Scheduler, so we use
// Inspector.SchedulerEntries + direct cleanup of scheduler metadata keys.
func cleanupSchedulerState(ctx context.Context, redisClient *redis.Client, inspector *asynq.Inspector, podID string) error {
	entries, err := inspector.SchedulerEntries()
	if err != nil {
		return err
	}
	if len(entries) > 0 {
		log.Printf("[%s] Scheduler cleanup: found %d existing cron entries", podID, len(entries))
	}

	keys, err := redisClient.ZRange(ctx, "asynq:schedulers", 0, -1).Result()
	if err != nil {
		return err
	}
	if len(keys) == 0 {
		return nil
	}

	pipe := redisClient.TxPipeline()
	for _, key := range keys {
		pipe.Del(ctx, key)
		pipe.ZRem(ctx, "asynq:schedulers", key)
	}
	if _, err := pipe.Exec(ctx); err != nil {
		return err
	}
	return nil
}

// buildPeriodicTask creates an asynq.Task + options for a scheduled task definition.
func buildPeriodicTask(td config.ScheduledTaskDef) (*asynq.Task, []asynq.Option, error) {
	var task *asynq.Task
	var err error

	switch td.Type {
	case tasks.TypeReportGenerate:
		task, err = tasks.NewReportGenerateTask("daily-summary",
			time.Now().AddDate(0, 0, -1).Format("2006-01-02"),
			time.Now().Format("2006-01-02"))
	case tasks.TypeEmailDeliver:
		task, err = tasks.NewEmailDeliverTask(
			"digest@example.com",
			"Periodic Digest",
			"This is an automated periodic email digest.")
	case tasks.TypeWebhookSend:
		task, err = tasks.NewWebhookSendTask(
			"https://httpbin.org/post",
			"POST",
			200)
	default:
		task = asynq.NewTask(td.Type, nil)
	}
	if err != nil {
		return nil, nil, err
	}

	opts := []asynq.Option{
		asynq.Queue(td.Queue),
	}
	if td.UniqueTTL > 0 {
		opts = append(opts, asynq.Unique(td.UniqueTTL))
	}

	return task, opts, nil
}
