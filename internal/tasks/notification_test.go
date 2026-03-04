package tasks

import (
	"context"
	"errors"
	"testing"

	"github.com/hibiken/asynq"
)

func TestNotificationTasks(t *testing.T) {
	t.Parallel()
	helperWithPayloadFormat(t, PayloadFormatJSON)

	t.Run("green constructor creates grouped notification task", func(t *testing.T) {
		t.Parallel()
		task, err := NewNotificationSendTask(42, "hello", "push")
		if err != nil {
			t.Fatalf("NewNotificationSendTask() error: %v", err)
		}
		if task.Type() != TypeNotificationSend {
			t.Fatalf("task.Type()=%q want %q", task.Type(), TypeNotificationSend)
		}
	})

	t.Run("green aggregator builds batch payload", func(t *testing.T) {
		t.Parallel()
		t1, _ := NewNotificationSendTask(1, "m1", "push")
		t2, _ := NewNotificationSendTask(2, "m2", "email")
		batchTask := AggregateNotifications("grp", []*asynq.Task{t1, t2})
		if batchTask.Type() != TypeNotificationBatch {
			t.Fatalf("batch task type=%q want %q", batchTask.Type(), TypeNotificationBatch)
		}
		var p NotificationBatchPayload
		if err := unmarshalPayload(batchTask.Payload(), &p); err != nil {
			t.Fatalf("unmarshal batch payload error: %v", err)
		}
		if p.Count != 2 {
			t.Fatalf("batch count=%d want 2", p.Count)
		}
	})

	t.Run("red batch handler rejects invalid payload", func(t *testing.T) {
		t.Parallel()
		err := HandleNotificationBatch(context.Background(), asynq.NewTask(TypeNotificationBatch, []byte("bad")))
		if err == nil || !errors.Is(err, asynq.SkipRetry) {
			t.Fatalf("HandleNotificationBatch() error=%v want wrapped SkipRetry", err)
		}
	})
}
