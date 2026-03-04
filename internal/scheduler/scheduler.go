package scheduler

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/bsm/redislock"
	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"

	"github.com/asynq-test/internal/config"
	"github.com/asynq-test/internal/tasks"
)

var errK8sLeaderElectionUnavailable = errors.New("kubernetes leader election unavailable")

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

// RunWithLeaderElection prefers Kubernetes lease election and falls back to Redis lock election.
// Redis lock election is used when Kubernetes election is not available in the runtime environment.
func RunWithLeaderElection(ctx context.Context, cfg config.Config) error {
	err := runWithK8sLeaderElection(ctx, cfg)
	if err == nil {
		return nil
	}
	if !errors.Is(err, errK8sLeaderElectionUnavailable) {
		return err
	}

	log.Printf("[%s] Kubernetes leader election unavailable: %v; falling back to Redis lock", cfg.Pod.ID, err)
	return runWithRedisLeaderElection(ctx, cfg)
}

func runWithK8sLeaderElection(ctx context.Context, cfg config.Config) error {
	k8sCfg := cfg.Scheduler.K8sLeaderElection
	if !k8sCfg.Enabled {
		return fmt.Errorf("%w: disabled by configuration", errK8sLeaderElectionUnavailable)
	}

	restCfg, err := rest.InClusterConfig()
	if err != nil {
		return fmt.Errorf("%w: %v", errK8sLeaderElectionUnavailable, err)
	}

	clientset, err := kubernetes.NewForConfig(restCfg)
	if err != nil {
		return fmt.Errorf("%w: %v", errK8sLeaderElectionUnavailable, err)
	}

	namespace := resolveK8sNamespace(k8sCfg.Namespace)
	if namespace == "" {
		return fmt.Errorf("%w: namespace not detected", errK8sLeaderElectionUnavailable)
	}

	leaseName := k8sCfg.LeaseName
	if leaseName == "" {
		leaseName = cfg.Scheduler.LeaderLockKey
	}
	if leaseName == "" {
		leaseName = "asynq-scheduler-leader"
	}

	leaseDuration := k8sCfg.LeaseDuration
	if leaseDuration <= 0 {
		leaseDuration = 15 * time.Second
	}
	renewDeadline := k8sCfg.RenewDeadline
	if renewDeadline <= 0 || renewDeadline >= leaseDuration {
		renewDeadline = (leaseDuration * 2) / 3
	}
	retryPeriod := k8sCfg.RetryPeriod
	if retryPeriod <= 0 || retryPeriod >= renewDeadline {
		retryPeriod = renewDeadline / 3
	}
	if retryPeriod <= 0 {
		retryPeriod = 2 * time.Second
	}

	lock := &resourcelock.LeaseLock{
		LeaseMeta: metav1.ObjectMeta{
			Name:      leaseName,
			Namespace: namespace,
		},
		Client: clientset.CoordinationV1(),
		LockConfig: resourcelock.ResourceLockConfig{
			Identity: cfg.Pod.ID,
		},
	}

	electionCtx, cancelElection := context.WithCancel(ctx)
	defer cancelElection()
	leaderErrCh := make(chan error, 1)

	elector, err := leaderelection.NewLeaderElector(leaderelection.LeaderElectionConfig{
		Lock:            lock,
		LeaseDuration:   leaseDuration,
		RenewDeadline:   renewDeadline,
		RetryPeriod:     retryPeriod,
		ReleaseOnCancel: true,
		Name:            leaseName,
		Callbacks: leaderelection.LeaderCallbacks{
			OnStartedLeading: func(leaderCtx context.Context) {
				log.Printf("[%s] Scheduler leadership acquired via Kubernetes lease %s/%s", cfg.Pod.ID, namespace, leaseName)
				if runErr := runSchedulerAsLeader(leaderCtx, cfg); runErr != nil {
					select {
					case leaderErrCh <- runErr:
					default:
					}
					cancelElection()
				}
			},
			OnStoppedLeading: func() {
				log.Printf("[%s] Scheduler leadership released via Kubernetes lease", cfg.Pod.ID)
			},
			OnNewLeader: func(identity string) {
				if identity != cfg.Pod.ID {
					log.Printf("[%s] Scheduler leader is %s (Kubernetes lease)", cfg.Pod.ID, identity)
				}
			},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to initialize Kubernetes leader election: %w", err)
	}

	log.Printf("[%s] Scheduler leader election started (mode=k8s lease=%s/%s)", cfg.Pod.ID, namespace, leaseName)
	elector.Run(electionCtx)

	select {
	case runErr := <-leaderErrCh:
		return runErr
	default:
		if ctx.Err() != nil {
			return nil
		}
		return nil
	}
}

func runWithRedisLeaderElection(ctx context.Context, cfg config.Config) error {
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

		leaderCtx, cancelLeader := context.WithCancel(ctx)
		schedulerErrCh := make(chan error, 1)
		go func() {
			schedulerErrCh <- runSchedulerAsLeader(leaderCtx, cfg)
		}()

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
		case err := <-schedulerErrCh:
			cancelLeader()
			_ = lock.Release(context.Background())
			return err
		}

		cancelLeader()
		select {
		case <-schedulerErrCh:
		case <-time.After(2 * time.Second):
		}

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

func runSchedulerAsLeader(ctx context.Context, cfg config.Config) error {
	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	defer redisClient.Close()

	inspector := asynq.NewInspector(asynq.RedisClientOpt{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	defer inspector.Close()

	if err := cleanupSchedulerState(ctx, redisClient, inspector, cfg.Pod.ID); err != nil {
		log.Printf("[%s] Scheduler cleanup warning: %v", cfg.Pod.ID, err)
	}

	sched, err := NewScheduler(cfg)
	if err != nil {
		return err
	}

	if err := RegisterTasks(sched, cfg); err != nil {
		sched.Shutdown()
		return err
	}

	if err := sched.Start(); err != nil {
		sched.Shutdown()
		return err
	}

	<-ctx.Done()
	sched.Shutdown()
	return nil
}

func resolveK8sNamespace(configNamespace string) string {
	return resolveK8sNamespaceWithReaders(configNamespace, os.Getenv, os.ReadFile)
}

func resolveK8sNamespaceWithReaders(
	configNamespace string,
	getenv func(string) string,
	readFile func(string) ([]byte, error),
) string {
	if ns := strings.TrimSpace(configNamespace); ns != "" {
		return ns
	}
	if ns := strings.TrimSpace(getenv("POD_NAMESPACE")); ns != "" {
		return ns
	}
	data, err := readFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
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
