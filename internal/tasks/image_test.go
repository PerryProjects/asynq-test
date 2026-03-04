package tasks

import (
	"context"
	"errors"
	"testing"

	"github.com/hibiken/asynq"
)

func TestImageTask(t *testing.T) {
	t.Parallel()
	helperWithPayloadFormat(t, PayloadFormatJSON)

	t.Run("green constructor creates task", func(t *testing.T) {
		t.Parallel()
		task, err := NewImageResizeTask("https://x/y.jpg", 10, 20)
		if err != nil {
			t.Fatalf("NewImageResizeTask() error: %v", err)
		}
		if task.Type() != TypeImageResize {
			t.Fatalf("task.Type()=%q want %q", task.Type(), TypeImageResize)
		}
	})

	t.Run("red handler rejects invalid payload", func(t *testing.T) {
		t.Parallel()
		err := (&ImageProcessor{}).ProcessTask(context.Background(), asynq.NewTask(TypeImageResize, []byte("bad")))
		if err == nil || !errors.Is(err, asynq.SkipRetry) {
			t.Fatalf("ProcessTask() error=%v want wrapped SkipRetry", err)
		}
	})

	t.Run("red handler exits on canceled context", func(t *testing.T) {
		t.Parallel()
		task, err := NewImageResizeTask("https://x/y.jpg", 10, 20)
		if err != nil {
			t.Fatalf("NewImageResizeTask() error: %v", err)
		}
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		t.Cleanup(cancel)

		err = (&ImageProcessor{}).ProcessTask(ctx, task)
		if err == nil {
			t.Fatal("ProcessTask() expected cancel error, got nil")
		}
	})
}
