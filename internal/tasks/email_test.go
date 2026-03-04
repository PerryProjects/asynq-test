package tasks

import (
	"context"
	"errors"
	"testing"

	"github.com/hibiken/asynq"
)

func TestEmailTask(t *testing.T) {
	t.Parallel()
	helperWithPayloadFormat(t, PayloadFormatJSON)

	t.Run("green constructor creates task", func(t *testing.T) {
		t.Parallel()
		task, err := NewEmailDeliverTask("u@test.com", "hello", "body")
		if err != nil {
			t.Fatalf("NewEmailDeliverTask() error: %v", err)
		}
		if task.Type() != TypeEmailDeliver {
			t.Fatalf("task.Type()=%q want %q", task.Type(), TypeEmailDeliver)
		}
	})

	t.Run("green handler accepts valid payload", func(t *testing.T) {
		t.Parallel()
		task, err := NewEmailDeliverTask("u@test.com", "hello", "body")
		if err != nil {
			t.Fatalf("NewEmailDeliverTask() error: %v", err)
		}
		if err := HandleEmailDeliver(context.Background(), task); err != nil {
			t.Fatalf("HandleEmailDeliver() error: %v", err)
		}
	})

	t.Run("red handler rejects invalid payload", func(t *testing.T) {
		t.Parallel()
		err := HandleEmailDeliver(context.Background(), asynq.NewTask(TypeEmailDeliver, []byte("not-json")))
		if err == nil || !errors.Is(err, asynq.SkipRetry) {
			t.Fatalf("HandleEmailDeliver() error=%v want wrapped SkipRetry", err)
		}
	})
}
