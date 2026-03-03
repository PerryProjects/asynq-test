package worker

import (
	"log"
	"time"

	"github.com/hibiken/asynq"

	"github.com/asynq-test/internal/config"
	"github.com/asynq-test/internal/middleware"
	"github.com/asynq-test/internal/tasks"
)

// NewServer creates and configures an asynq.Server with full feature set.
func NewServer(cfg config.Config) *asynq.Server {
	redisOpt := asynq.RedisClientOpt{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	}

	srv := asynq.NewServer(redisOpt, asynq.Config{
		Concurrency:     cfg.Worker.Concurrency,
		Queues:          cfg.Worker.Queues,
		StrictPriority:  cfg.Worker.StrictPriority,
		ShutdownTimeout: cfg.Worker.ShutdownTimeout,

		// Custom retry delay — supports rate-limit requeue delays.
		RetryDelayFunc: middleware.RetryDelayFunc,

		// IsFailure: rate-limited tasks are not counted as failures.
		IsFailure: func(err error) bool {
			return !middleware.IsRateLimitError(err)
		},

		// ErrorHandler logs errors with pod identity.
		ErrorHandler: middleware.ErrorHandler(cfg.Pod.ID),

		// HealthCheckFunc runs periodically to verify Redis connectivity.
		HealthCheckFunc: func(err error) {
			if err != nil {
				log.Printf("[%s] HEALTHCHECK Redis unhealthy: %v", cfg.Pod.ID, err)
			}
		},
		HealthCheckInterval: 15 * time.Second,

		// Group aggregation for notification batching.
		GroupAggregator:  asynq.GroupAggregatorFunc(tasks.AggregateNotifications),
		GroupGracePeriod: 30 * time.Second,
		GroupMaxDelay:    2 * time.Minute,
		GroupMaxSize:     10,
	})

	return srv
}

// NewServeMux creates a ServeMux with all handlers and middleware registered.
// Uses nested ServeMux to demonstrate prefix routing.
func NewServeMux(cfg config.Config) *asynq.ServeMux {
	mux := asynq.NewServeMux()

	// Global middleware
	mux.Use(middleware.LoggingMiddleware(cfg.Pod.ID))
	mux.Use(middleware.RecoveryMiddleware(cfg.Pod.ID))
	mux.Use(middleware.RateLimitMiddleware())

	// --- email: handlers ---
	emailMux := asynq.NewServeMux()
	emailMux.HandleFunc(tasks.TypeEmailDeliver, tasks.HandleEmailDeliver)
	mux.Handle("email:", emailMux)

	// --- image: handler (struct-based) ---
	mux.Handle("image:", &tasks.ImageProcessor{})

	// --- report: handlers ---
	reportMux := asynq.NewServeMux()
	reportMux.HandleFunc(tasks.TypeReportGenerate, tasks.HandleReportGenerate)
	mux.Handle("report:", reportMux)

	// --- webhook: handlers ---
	webhookMux := asynq.NewServeMux()
	webhookMux.HandleFunc(tasks.TypeWebhookSend, tasks.HandleWebhookSend)
	mux.Handle("webhook:", webhookMux)

	// --- notification: handlers (send + batch) ---
	notifMux := asynq.NewServeMux()
	notifMux.HandleFunc(tasks.TypeNotificationSend, tasks.HandleNotificationSend)
	notifMux.HandleFunc(tasks.TypeNotificationBatch, tasks.HandleNotificationBatch)
	mux.Handle("notification:", notifMux)

	return mux
}
