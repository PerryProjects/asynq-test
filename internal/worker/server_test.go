package worker

import (
	"testing"

	"github.com/asynq-test/internal/config"
)

func TestNewServerAndMux(t *testing.T) {
	t.Parallel()
	t.Cleanup(func() {})

	cfg := config.Config{
		Redis: config.RedisConfig{Addr: "127.0.0.1:0"},
		Pod:   config.PodConfig{ID: "pod-test"},
		Worker: config.WorkerConfig{
			Concurrency: 2,
			Queues: map[string]int{
				"default": 1,
			},
		},
	}

	t.Run("green creates server", func(t *testing.T) {
		t.Parallel()
		srv := NewServer(cfg)
		if srv == nil {
			t.Fatal("NewServer() returned nil")
		}
	})

	t.Run("green creates mux", func(t *testing.T) {
		t.Parallel()
		mux := NewServeMux(cfg)
		if mux == nil {
			t.Fatal("NewServeMux() returned nil")
		}
	})
}
