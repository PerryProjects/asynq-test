package tasks

import (
	"context"
	"errors"
	"testing"

	"github.com/hibiken/asynq"
)

func TestReportTask(t *testing.T) {
	t.Parallel()
	helperWithPayloadFormat(t, PayloadFormatJSON)

	t.Run("green constructor creates task", func(t *testing.T) {
		t.Parallel()
		task, err := NewReportGenerateTask("daily", "2025-01-01", "2025-01-02")
		if err != nil {
			t.Fatalf("NewReportGenerateTask() error: %v", err)
		}
		if task.Type() != TypeReportGenerate {
			t.Fatalf("task.Type()=%q want %q", task.Type(), TypeReportGenerate)
		}
	})

	t.Run("red handler rejects invalid payload", func(t *testing.T) {
		t.Parallel()
		err := HandleReportGenerate(context.Background(), asynq.NewTask(TypeReportGenerate, []byte("bad")))
		if err == nil || !errors.Is(err, asynq.SkipRetry) {
			t.Fatalf("HandleReportGenerate() error=%v want wrapped SkipRetry", err)
		}
	})

	t.Run("green handler stops quickly when context canceled", func(t *testing.T) {
		t.Parallel()
		task, err := NewReportGenerateTask("daily", "2025-01-01", "2025-01-02")
		if err != nil {
			t.Fatalf("NewReportGenerateTask() error: %v", err)
		}
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		t.Cleanup(cancel)

		err = HandleReportGenerate(ctx, task)
		if err == nil {
			t.Fatal("HandleReportGenerate() expected cancel error")
		}
	})
}
