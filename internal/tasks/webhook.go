package tasks

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/hibiken/asynq"
)

// NewWebhookSendTask creates a new webhook:send task.
// Demonstrates: Unique, custom TaskID, SkipRetry on 4xx.
func NewWebhookSendTask(url, method string, simulateCode int) (*asynq.Task, error) {
	payload, err := marshalPayload(WebhookPayload{
		URL:          url,
		Method:       method,
		SimulateCode: simulateCode,
	})
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(
		TypeWebhookSend,
		payload,
		asynq.MaxRetry(5),
		asynq.Unique(1*time.Hour),
		asynq.Queue("critical"),
		asynq.Retention(2*time.Hour),
	), nil
}

// HandleWebhookSend processes webhook:send tasks.
// Demonstrates SkipRetry on 4xx errors, normal retry on 5xx.
func HandleWebhookSend(ctx context.Context, t *asynq.Task) error {
	var p WebhookPayload
	if err := unmarshalPayload(t.Payload(), &p); err != nil {
		return fmt.Errorf("failed to unmarshal webhook payload: %v: %w", err, asynq.SkipRetry)
	}

	taskID, _ := asynq.GetTaskID(ctx)
	retryCount, _ := asynq.GetRetryCount(ctx)

	log.Printf("[webhook:send] TaskID=%s Retry=%d → %s %s (simulating HTTP %d)",
		taskID, retryCount, p.Method, p.URL, p.SimulateCode)

	// Simulate HTTP call
	time.Sleep(300 * time.Millisecond)

	switch {
	case p.SimulateCode >= 200 && p.SimulateCode < 300:
		log.Printf("[webhook:send] TaskID=%s → Webhook delivered successfully (HTTP %d)", taskID, p.SimulateCode)
		return nil

	case p.SimulateCode >= 400 && p.SimulateCode < 500:
		// 4xx — client error, no point retrying → SkipRetry
		log.Printf("[webhook:send] TaskID=%s → Client error HTTP %d — skipping retry", taskID, p.SimulateCode)
		return fmt.Errorf("HTTP %d: client error: %w", p.SimulateCode, asynq.SkipRetry)

	case p.SimulateCode >= 500:
		// 5xx — server error, should retry
		log.Printf("[webhook:send] TaskID=%s → Server error HTTP %d — will retry", taskID, p.SimulateCode)
		return fmt.Errorf("HTTP %d: server error", p.SimulateCode)

	default:
		log.Printf("[webhook:send] TaskID=%s → Unexpected status %d", taskID, p.SimulateCode)
		return fmt.Errorf("unexpected HTTP status %d", p.SimulateCode)
	}
}
