package cmd

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/hibiken/asynq"
	"github.com/spf13/cobra"

	"github.com/asynq-test/internal/config"
	"github.com/asynq-test/internal/tasks"
)

// CLI flags
var (
	enqTaskType  string
	enqPayload   string
	enqQueue     string
	enqDelay     int
	enqMaxRetry  int
	enqUniqueTTL int
)

func NewEnqueueCmd() *cobra.Command {
	enqueueCmd := &cobra.Command{
		Use:   "enqueue",
		Short: "Enqueue a task from the CLI",
		Long:  "One-shot CLI command to enqueue a task into Redis for processing by worker pods.",
		RunE:  runEnqueue,
	}

	enqueueCmd.Flags().StringVarP(&enqTaskType, "type", "t", "", "Task type (required)")
	enqueueCmd.Flags().StringVarP(&enqPayload, "payload", "P", "{}", "JSON payload")
	enqueueCmd.Flags().StringVarP(&enqQueue, "queue", "q", "", "Queue name")
	enqueueCmd.Flags().IntVarP(&enqDelay, "delay", "d", 0, "Delay in seconds before processing")
	enqueueCmd.Flags().IntVarP(&enqMaxRetry, "max-retry", "r", 0, "Max retry count")
	enqueueCmd.Flags().IntVarP(&enqUniqueTTL, "unique", "u", 0, "Unique TTL in seconds")
	_ = enqueueCmd.MarkFlagRequired("type")

	return enqueueCmd
}

func runEnqueue(cmd *cobra.Command, args []string) error {
	cfg := config.C

	client := asynq.NewClient(asynq.RedisClientOpt{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	defer client.Close()

	task, err := buildCLITask(enqTaskType, enqPayload)
	if err != nil {
		return fmt.Errorf("failed to build task: %w", err)
	}

	// Build options.
	var opts []asynq.Option
	if enqQueue != "" {
		opts = append(opts, asynq.Queue(enqQueue))
	}
	if enqDelay > 0 {
		opts = append(opts, asynq.ProcessIn(time.Duration(enqDelay)*time.Second))
	}
	if enqMaxRetry > 0 {
		opts = append(opts, asynq.MaxRetry(enqMaxRetry))
	}
	if enqUniqueTTL > 0 {
		opts = append(opts, asynq.Unique(time.Duration(enqUniqueTTL)*time.Second))
	}

	info, err := client.Enqueue(task, opts...)
	if err != nil {
		return fmt.Errorf("failed to enqueue task: %w", err)
	}

	log.Printf("Enqueued task: id=%s type=%s queue=%s max_retry=%d",
		info.ID, info.Type, info.Queue, info.MaxRetry)
	return nil
}

func buildCLITask(taskType, payloadJSON string) (*asynq.Task, error) {
	switch taskType {
	case tasks.TypeEmailDeliver:
		var p tasks.EmailPayload
		if err := json.Unmarshal([]byte(payloadJSON), &p); err != nil {
			return nil, err
		}
		return tasks.NewEmailDeliverTask(p.To, p.Subject, p.Body)

	case tasks.TypeImageResize:
		var p tasks.ImagePayload
		if err := json.Unmarshal([]byte(payloadJSON), &p); err != nil {
			return nil, err
		}
		return tasks.NewImageResizeTask(p.URL, p.Width, p.Height)

	case tasks.TypeReportGenerate:
		var p tasks.ReportPayload
		if err := json.Unmarshal([]byte(payloadJSON), &p); err != nil {
			return nil, err
		}
		return tasks.NewReportGenerateTask(p.ReportType, p.StartDate, p.EndDate)

	case tasks.TypeWebhookSend:
		var p tasks.WebhookPayload
		if err := json.Unmarshal([]byte(payloadJSON), &p); err != nil {
			return nil, err
		}
		return tasks.NewWebhookSendTask(p.URL, p.Method, p.SimulateCode)

	case tasks.TypeNotificationSend:
		var p tasks.NotificationPayload
		if err := json.Unmarshal([]byte(payloadJSON), &p); err != nil {
			return nil, err
		}
		return tasks.NewNotificationSendTask(p.UserID, p.Message, p.Channel)

	default:
		// Generic: pass raw JSON as payload.
		return asynq.NewTask(taskType, []byte(payloadJSON)), nil
	}
}
