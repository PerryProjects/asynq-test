package tasks

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/hibiken/asynq"
)

// NewEmailDeliverTask creates a new email:deliver task.
func NewEmailDeliverTask(to, subject, body string) (*asynq.Task, error) {
	payload, err := json.Marshal(EmailPayload{
		To:      to,
		Subject: subject,
		Body:    body,
	})
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(
		TypeEmailDeliver,
		payload,
		asynq.MaxRetry(3),
		asynq.Retention(2*time.Hour),
	), nil
}

// HandleEmailDeliver processes email:deliver tasks.
func HandleEmailDeliver(ctx context.Context, t *asynq.Task) error {
	var p EmailPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return fmt.Errorf("failed to unmarshal email payload: %v: %w", err, asynq.SkipRetry)
	}

	taskID, _ := asynq.GetTaskID(ctx)
	retryCount, _ := asynq.GetRetryCount(ctx)

	log.Printf("[email:deliver] TaskID=%s Retry=%d → Sending to=%s subject=%q",
		taskID, retryCount, p.To, p.Subject)

	// Simulate email sending
	time.Sleep(500 * time.Millisecond)

	log.Printf("[email:deliver] TaskID=%s → Email sent successfully to %s", taskID, p.To)
	return nil
}
