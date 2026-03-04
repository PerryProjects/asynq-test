package middleware

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/hibiken/asynq"
)

func TestRateLimitErrorAndClassifier(t *testing.T) {
	t.Parallel()
	t.Cleanup(func() {})

	t.Run("green rate-limit error is recognized", func(t *testing.T) {
		t.Parallel()
		err := &RateLimitError{RetryIn: 3 * time.Second}
		if !IsRateLimitError(err) {
			t.Fatal("IsRateLimitError() = false, want true")
		}
		if got := err.Error(); !strings.Contains(got, "retry in 3s") {
			t.Fatalf("RateLimitError.Error() = %q, expected retry duration", got)
		}
	})

	t.Run("red generic error is not recognized", func(t *testing.T) {
		t.Parallel()
		if IsRateLimitError(errors.New("x")) {
			t.Fatal("IsRateLimitError() = true, want false")
		}
	})
}

func TestRetryDelayFunc(t *testing.T) {
	t.Parallel()
	t.Cleanup(func() {})

	task := asynq.NewTask("t", nil)

	t.Run("green uses explicit rate-limit delay", func(t *testing.T) {
		t.Parallel()
		err := &RateLimitError{RetryIn: 4 * time.Second}
		if got := RetryDelayFunc(2, err, task); got != 4*time.Second {
			t.Fatalf("RetryDelayFunc() = %v, want 4s", got)
		}
	})

	t.Run("red-path falls back to default retry delay", func(t *testing.T) {
		t.Parallel()
		err := errors.New("regular")
		got := RetryDelayFunc(2, err, task)
		if got <= 0 {
			t.Fatalf("RetryDelayFunc() = %v, want positive duration", got)
		}
		if got == (&RateLimitError{RetryIn: 4 * time.Second}).RetryIn {
			t.Fatalf("RetryDelayFunc() unexpectedly matched rate-limit delay: %v", got)
		}
	})
}

func TestRecoveryMiddleware(t *testing.T) {
	t.Parallel()
	t.Cleanup(func() {})

	mw := RecoveryMiddleware("pod-x")
	h := mw(asynq.HandlerFunc(func(context.Context, *asynq.Task) error {
		panic("boom")
	}))
	if err := h.ProcessTask(context.Background(), asynq.NewTask("x", nil)); err == nil || !strings.Contains(err.Error(), "panic recovered") {
		t.Fatalf("ProcessTask() error = %v, want recovered panic error", err)
	}
}

func TestLoggingMiddlewarePassThrough(t *testing.T) {
	t.Parallel()
	t.Cleanup(func() {})

	called := false
	sentinel := errors.New("sentinel")
	h := LoggingMiddleware("pod-y")(asynq.HandlerFunc(func(context.Context, *asynq.Task) error {
		called = true
		return sentinel
	}))
	err := h.ProcessTask(context.Background(), asynq.NewTask("x", nil))
	if !called {
		t.Fatal("wrapped handler was not called")
	}
	if !errors.Is(err, sentinel) {
		t.Fatalf("ProcessTask() error = %v, want sentinel", err)
	}
}

func TestRateLimitMiddlewareAllowsAtLeastFirstCall(t *testing.T) {
	t.Parallel()
	t.Cleanup(func() {})

	called := false
	h := RateLimitMiddleware()(asynq.HandlerFunc(func(context.Context, *asynq.Task) error {
		called = true
		return nil
	}))
	if err := h.ProcessTask(context.Background(), asynq.NewTask("x", nil)); err != nil {
		t.Fatalf("first ProcessTask() error = %v, want nil", err)
	}
	if !called {
		t.Fatal("wrapped handler was not called")
	}
}

func TestErrorHandlerDoesNotPanic(t *testing.T) {
	t.Parallel()
	t.Cleanup(func() {})

	h := ErrorHandler("pod-z")
	h.HandleError(context.Background(), asynq.NewTask("x", nil), errors.New("err"))
}
