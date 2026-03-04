package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/hibiken/asynq"

	"github.com/asynq-test/internal/tasks"
)

func TestBuildTaskFromRequest(t *testing.T) {
	t.Parallel()
	t.Cleanup(func() {})

	tests := []struct {
		name        string
		req         EnqueueRequest
		wantErr     bool
		wantType    string
		wantOptsLen int
	}{
		{
			name: "green email with options",
			req: EnqueueRequest{
				Type:         tasks.TypeEmailDeliver,
				Payload:      mustJSON(t, tasks.EmailPayload{To: "u@test.com", Subject: "s", Body: "b"}),
				Queue:        "critical",
				DelaySeconds: 5,
				MaxRetry:     3,
				UniqueTTL:    7,
			},
			wantType:    tasks.TypeEmailDeliver,
			wantOptsLen: 4,
		},
		{
			name: "red unknown type",
			req: EnqueueRequest{
				Type:    "x:unknown",
				Payload: json.RawMessage(`{}`),
			},
			wantErr: true,
		},
		{
			name: "red invalid payload",
			req: EnqueueRequest{
				Type:    tasks.TypeEmailDeliver,
				Payload: json.RawMessage(`{"to":`),
			},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			task, opts, err := buildTaskFromRequest(tc.req)
			if tc.wantErr {
				if err == nil {
					t.Fatal("buildTaskFromRequest() expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("buildTaskFromRequest() error: %v", err)
			}
			if task.Type() != tc.wantType {
				t.Fatalf("task.Type()=%q want %q", task.Type(), tc.wantType)
			}
			if len(opts) != tc.wantOptsLen {
				t.Fatalf("len(opts)=%d want %d", len(opts), tc.wantOptsLen)
			}
		})
	}
}

func TestIntParam(t *testing.T) {
	t.Parallel()
	t.Cleanup(func() {})

	tests := []struct {
		name      string
		query     string
		defaultV  int
		want      int
	}{
		{name: "green parses integer", query: "?n=12", defaultV: 5, want: 12},
		{name: "red invalid integer uses default", query: "?n=x", defaultV: 5, want: 5},
		{name: "red missing integer uses default", query: "", defaultV: 8, want: 8},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
			ctx.Request = httptest.NewRequest(http.MethodGet, "/"+tc.query, nil)
			if got := intParam(ctx, "n", tc.defaultV); got != tc.want {
				t.Fatalf("intParam()=%d want %d", got, tc.want)
			}
		})
	}
}

func TestEnqueueHandler_InvalidRequests(t *testing.T) {
	t.Parallel()
	t.Cleanup(func() {})

	h := EnqueueHandler(asynq.RedisClientOpt{Addr: "127.0.0.1:0"})

	t.Run("red invalid json body", func(t *testing.T) {
		t.Parallel()
		w := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(w)
		ctx.Request = httptest.NewRequest(http.MethodPost, "/api/tasks/enqueue", strings.NewReader("{"))
		ctx.Request.Header.Set("Content-Type", "application/json")

		h(ctx)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status=%d want %d", w.Code, http.StatusBadRequest)
		}
	})

	t.Run("red unknown task type", func(t *testing.T) {
		t.Parallel()
		w := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(w)
		ctx.Request = httptest.NewRequest(http.MethodPost, "/api/tasks/enqueue", strings.NewReader(`{"type":"x","payload":{}}`))
		ctx.Request.Header.Set("Content-Type", "application/json")

		h(ctx)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status=%d want %d", w.Code, http.StatusBadRequest)
		}
	})
}

func mustJSON(t *testing.T, v any) json.RawMessage {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("json.Marshal error: %v", err)
	}
	return b
}

func init() {
	gin.SetMode(gin.TestMode)
}
