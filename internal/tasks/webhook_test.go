package tasks

import (
	"context"
	"errors"
	"testing"

	"github.com/hibiken/asynq"
)

func TestWebhookTask(t *testing.T) {
	t.Parallel()
	helperWithPayloadFormat(t, PayloadFormatJSON)

	t.Run("green constructor creates task", func(t *testing.T) {
		t.Parallel()
		task, err := NewWebhookSendTask("https://x", "POST", 200)
		if err != nil {
			t.Fatalf("NewWebhookSendTask() error: %v", err)
		}
		if task.Type() != TypeWebhookSend {
			t.Fatalf("task.Type()=%q want %q", task.Type(), TypeWebhookSend)
		}
	})

	tests := []struct {
		name      string
		status    int
		wantErr   bool
		wantSkip  bool
	}{
		{name: "green 2xx success", status: 200, wantErr: false},
		{name: "red 4xx skip retry", status: 404, wantErr: true, wantSkip: true},
		{name: "red 5xx retryable", status: 500, wantErr: true, wantSkip: false},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			task, err := NewWebhookSendTask("https://x", "POST", tc.status)
			if err != nil {
				t.Fatalf("NewWebhookSendTask() error: %v", err)
			}
			err = HandleWebhookSend(context.Background(), task)
			if tc.wantErr && err == nil {
				t.Fatal("HandleWebhookSend() expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("HandleWebhookSend() unexpected error: %v", err)
			}
			if tc.wantSkip && !errors.Is(err, asynq.SkipRetry) {
				t.Fatalf("HandleWebhookSend() error=%v want wrapped SkipRetry", err)
			}
		})
	}
}
