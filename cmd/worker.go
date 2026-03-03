package cmd

import (
	"context"
	"fmt"
	"log"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/asynq-test/internal/config"
	"github.com/asynq-test/internal/scheduler"
	"github.com/asynq-test/internal/worker"
)

var workerCmd = &cobra.Command{
	Use:   "worker",
	Short: "Start worker server with embedded scheduler",
	Long:  "Runs the asynq worker server (task consumer) and the embedded periodic task scheduler in the same process.",
	RunE:  runWorker,
}

func init() {
	rootCmd.AddCommand(workerCmd)
	workerCmd.Flags().IntP("concurrency", "c", 0, "Worker concurrency (overrides config)")
	workerCmd.Flags().String("pod-id", "", "Pod identifier (overrides config/hostname)")
	_ = viper.BindPFlag("worker.concurrency", workerCmd.Flags().Lookup("concurrency"))
	_ = viper.BindPFlag("pod.id", workerCmd.Flags().Lookup("pod-id"))
}

func runWorker(cmd *cobra.Command, args []string) error {
	cfg := config.C
	log.Printf("[%s] Starting asynq worker (concurrency=%d)", cfg.Pod.ID, cfg.Worker.Concurrency)

	// ── Build server + mux ──────────────────────────────────────
	srv := worker.NewServer(cfg)
	mux := worker.NewServeMux(cfg)

	// ── Start server (non-blocking) ─────────────────────────────
	if err := srv.Start(mux); err != nil {
		return fmt.Errorf("failed to start asynq server: %w", err)
	}
	log.Printf("[%s] Worker server started", cfg.Pod.ID)

	// ── Start scheduler leader-election loop ─────────────────────
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	schedErr := make(chan error, 1)
	go func() {
		if err := scheduler.RunWithLeaderElection(ctx, cfg); err != nil {
			schedErr <- err
		}
	}()
	log.Printf("[%s] Scheduler leader-election loop started", cfg.Pod.ID)

	// ── Wait for shutdown signal or scheduler failure ────────────

	select {
	case <-ctx.Done():
		log.Printf("[%s] Received shutdown signal — draining workers", cfg.Pod.ID)
	case err := <-schedErr:
		log.Printf("[%s] Scheduler error: %v — shutting down", cfg.Pod.ID, err)
		stop()
	}

	// Graceful shutdown.
	log.Printf("[%s] Stopping worker from pulling new tasks…", cfg.Pod.ID)
	srv.Stop()

	log.Printf("[%s] Waiting for scheduler loop to exit…", cfg.Pod.ID)

	log.Printf("[%s] Shutting down worker server…", cfg.Pod.ID)
	srv.Shutdown()

	log.Printf("[%s] Shutdown complete", cfg.Pod.ID)
	return nil
}
