package web

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/asynq-test/internal/config"
)

func TestNewRouter_HealthEndpoint(t *testing.T) {
	t.Parallel()
	t.Cleanup(func() {})

	r := NewRouter(config.Config{
		Redis: config.RedisConfig{Addr: "127.0.0.1:0"},
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status=%d want %d", w.Code, http.StatusOK)
	}
}
