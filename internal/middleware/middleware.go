package middleware

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/hibiken/asynq"
	"golang.org/x/time/rate"
)

// RateLimitError signals that the task should be requeued after a delay.
type RateLimitError struct {
	RetryIn time.Duration
}

func (e *RateLimitError) Error() string {
	return fmt.Sprintf("rate limited (retry in %v)", e.RetryIn)
}

// IsRateLimitError checks if the error is a rate limit error.
func IsRateLimitError(err error) bool {
	_, ok := err.(*RateLimitError)
	return ok
}

// RetryDelayFunc returns a custom retry delay — respects rate-limit delays,
// otherwise uses exponential backoff.
func RetryDelayFunc(n int, err error, task *asynq.Task) time.Duration {
	var rlErr *RateLimitError
	if ok := isRateLimitErr(err, &rlErr); ok {
		return rlErr.RetryIn
	}
	return asynq.DefaultRetryDelayFunc(n, err, task)
}

func isRateLimitErr(err error, target **RateLimitError) bool {
	if rl, ok := err.(*RateLimitError); ok {
		*target = rl
		return true
	}
	return false
}

// LoggingMiddleware logs task processing with pod identity.
func LoggingMiddleware(podID string) func(asynq.Handler) asynq.Handler {
	return func(next asynq.Handler) asynq.Handler {
		return asynq.HandlerFunc(func(ctx context.Context, t *asynq.Task) error {
			start := time.Now()
			taskID, _ := asynq.GetTaskID(ctx)
			retryCount, _ := asynq.GetRetryCount(ctx)
			queueName, _ := asynq.GetQueueName(ctx)

			log.Printf("[%s] START task=%s id=%s queue=%s retry=%d",
				podID, t.Type(), taskID, queueName, retryCount)

			err := next.ProcessTask(ctx, t)
			elapsed := time.Since(start)

			if err != nil {
				log.Printf("[%s] FAIL  task=%s id=%s queue=%s elapsed=%v err=%v",
					podID, t.Type(), taskID, queueName, elapsed, err)
			} else {
				log.Printf("[%s] DONE  task=%s id=%s queue=%s elapsed=%v",
					podID, t.Type(), taskID, queueName, elapsed)
			}
			return err
		})
	}
}

// RecoveryMiddleware catches panics and logs them with pod identity.
func RecoveryMiddleware(podID string) func(asynq.Handler) asynq.Handler {
	return func(next asynq.Handler) asynq.Handler {
		return asynq.HandlerFunc(func(ctx context.Context, t *asynq.Task) (err error) {
			defer func() {
				if r := recover(); r != nil {
					taskID, _ := asynq.GetTaskID(ctx)
					log.Printf("[%s] PANIC task=%s id=%s recover=%v",
						podID, t.Type(), taskID, r)
					err = fmt.Errorf("panic recovered: %v", r)
				}
			}()
			return next.ProcessTask(ctx, t)
		})
	}
}

// RateLimitMiddleware limits task processing rate.
// Rate: 10 events/sec, burst of 30.
func RateLimitMiddleware() func(asynq.Handler) asynq.Handler {
	limiter := rate.NewLimiter(10, 30)
	return func(next asynq.Handler) asynq.Handler {
		return asynq.HandlerFunc(func(ctx context.Context, t *asynq.Task) error {
			if !limiter.Allow() {
				retryIn := time.Duration(1+rand.Intn(5)) * time.Second
				return &RateLimitError{RetryIn: retryIn}
			}
			return next.ProcessTask(ctx, t)
		})
	}
}

// ErrorHandler logs errors with pod identity.
func ErrorHandler(podID string) asynq.ErrorHandler {
	return asynq.ErrorHandlerFunc(func(ctx context.Context, task *asynq.Task, err error) {
		retried, _ := asynq.GetRetryCount(ctx)
		maxRetry, _ := asynq.GetMaxRetry(ctx)
		taskID, _ := asynq.GetTaskID(ctx)
		log.Printf("[%s] ERROR task=%s id=%s retry=%d/%d err=%v",
			podID, task.Type(), taskID, retried, maxRetry, err)
	})
}
