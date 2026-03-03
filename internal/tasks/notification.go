package tasks

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/hibiken/asynq"
)

// NewNotificationSendTask creates a notification:send task.
// Demonstrates: Group aggregation — tasks with the same group key are batched.
func NewNotificationSendTask(userID int, message, channel string) (*asynq.Task, error) {
	payload, err := marshalPayload(NotificationPayload{
		UserID:  userID,
		Message: message,
		Channel: channel,
	})
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(
		TypeNotificationSend,
		payload,
		asynq.MaxRetry(3),
		asynq.Retention(1*time.Hour),
		asynq.Group(fmt.Sprintf("user:%d", userID)),
	), nil
}

// AggregateNotifications is the GroupAggregator function.
// It merges multiple notification:send tasks into a single notification:batch task.
func AggregateNotifications(group string, tasks []*asynq.Task) *asynq.Task {
	log.Printf("[aggregator] Aggregating %d tasks from group %q", len(tasks), group)

	var userIDs []int
	var messages []string

	for _, t := range tasks {
		var p NotificationPayload
		if err := unmarshalPayload(t.Payload(), &p); err != nil {
			log.Printf("[aggregator] Failed to unmarshal task payload: %v", err)
			continue
		}
		userIDs = append(userIDs, p.UserID)
		messages = append(messages, p.Message)
	}

	batch := NotificationBatchPayload{
		UserIDs:  userIDs,
		Messages: messages,
		Count:    len(tasks),
		Group:    group,
	}
	payload, err := marshalPayload(batch)
	if err != nil {
		log.Printf("[aggregator] Failed to marshal batch payload: %v", err)
		return asynq.NewTask(TypeNotificationBatch, nil)
	}

	return asynq.NewTask(TypeNotificationBatch, payload)
}

// HandleNotificationSend processes individual notification:send tasks
// (only reached if aggregation is not triggered, e.g., single notification).
func HandleNotificationSend(ctx context.Context, t *asynq.Task) error {
	var p NotificationPayload
	if err := unmarshalPayload(t.Payload(), &p); err != nil {
		return fmt.Errorf("failed to unmarshal notification payload: %v: %w", err, asynq.SkipRetry)
	}

	taskID, _ := asynq.GetTaskID(ctx)
	log.Printf("[notification:send] TaskID=%s → Sending %s notification to user %d: %s",
		taskID, p.Channel, p.UserID, p.Message)

	time.Sleep(200 * time.Millisecond)

	log.Printf("[notification:send] TaskID=%s → Notification sent", taskID)
	return nil
}

// HandleNotificationBatch processes aggregated notification:batch tasks.
func HandleNotificationBatch(ctx context.Context, t *asynq.Task) error {
	var p NotificationBatchPayload
	if err := unmarshalPayload(t.Payload(), &p); err != nil {
		return fmt.Errorf("failed to unmarshal batch payload: %v: %w", err, asynq.SkipRetry)
	}

	taskID, _ := asynq.GetTaskID(ctx)
	log.Printf("[notification:batch] TaskID=%s → Processing batch of %d notifications for group %q",
		taskID, p.Count, p.Group)

	for i, msg := range p.Messages {
		log.Printf("[notification:batch] TaskID=%s → [%d/%d] user=%d msg=%q",
			taskID, i+1, p.Count, p.UserIDs[i], msg)
	}

	time.Sleep(500 * time.Millisecond)

	log.Printf("[notification:batch] TaskID=%s → Batch complete (%d notifications)", taskID, p.Count)
	return nil
}
