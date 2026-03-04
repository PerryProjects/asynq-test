package scheduler

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/asynq-test/internal/config"
	"github.com/asynq-test/internal/tasks"
)

func TestResolveK8sNamespaceWithReaders(t *testing.T) {
	t.Parallel()
	t.Cleanup(func() {})

	tests := []struct {
		name      string
		configNS  string
		envNS     string
		fileNS    string
		fileErr   error
		expected  string
	}{
		{
			name:     "config namespace has highest precedence",
			configNS: "  cfg-ns  ",
			envNS:    "env-ns",
			fileNS:   "file-ns",
			expected: "cfg-ns",
		},
		{
			name:     "env namespace is used when config is empty",
			envNS:    "  env-ns  ",
			fileNS:   "file-ns",
			expected: "env-ns",
		},
		{
			name:     "serviceaccount namespace is used when config and env are empty",
			fileNS:   "  file-ns  ",
			expected: "file-ns",
		},
		{
			name:     "empty is returned when all namespace sources are unavailable",
			fileErr:  errors.New("file not found"),
			expected: "",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			getenv := func(key string) string {
				if key == "POD_NAMESPACE" {
					return tc.envNS
				}
				return ""
			}
			readFile := func(_ string) ([]byte, error) {
				if tc.fileErr != nil {
					return nil, tc.fileErr
				}
				return []byte(tc.fileNS), nil
			}

			actual := resolveK8sNamespaceWithReaders(tc.configNS, getenv, readFile)
			if actual != tc.expected {
				t.Fatalf("resolveK8sNamespaceWithReaders() = %q, want %q", actual, tc.expected)
			}
		})
	}
}

func TestRunWithK8sLeaderElection_Disabled(t *testing.T) {
	t.Parallel()
	t.Cleanup(func() {})

	cfg := helperTestConfig()
	cfg.Scheduler.K8sLeaderElection.Enabled = false
	ctx := helperCanceledContext(t)

	err := runWithK8sLeaderElection(ctx, cfg)
	if !errors.Is(err, errK8sLeaderElectionUnavailable) {
		t.Fatalf("runWithK8sLeaderElection() error = %v, want wrapped %v", err, errK8sLeaderElectionUnavailable)
	}
}

func TestRunWithLeaderElection_FallsBackToRedisOnK8sUnavailable(t *testing.T) {
	t.Parallel()
	t.Cleanup(func() {})

	cfg := helperTestConfig()
	cfg.Scheduler.K8sLeaderElection.Enabled = false
	ctx := helperCanceledContext(t)

	if err := RunWithLeaderElection(ctx, cfg); err != nil {
		t.Fatalf("RunWithLeaderElection() error = %v, want nil", err)
	}
}

func TestBuildPeriodicTask(t *testing.T) {
	t.Parallel()
	t.Cleanup(func() {})

	tests := []struct {
		name            string
		typeName        string
		uniqueTTL       time.Duration
		wantTaskType    string
		wantPayloadSize int
		wantOptsLen     int
	}{
		{
			name:            "report task uses generated report payload",
			typeName:        tasks.TypeReportGenerate,
			uniqueTTL:       30 * time.Second,
			wantTaskType:    tasks.TypeReportGenerate,
			wantPayloadSize: 1,
			wantOptsLen:     2,
		},
		{
			name:            "email task uses generated email payload",
			typeName:        tasks.TypeEmailDeliver,
			uniqueTTL:       0,
			wantTaskType:    tasks.TypeEmailDeliver,
			wantPayloadSize: 1,
			wantOptsLen:     1,
		},
		{
			name:            "webhook task uses generated webhook payload",
			typeName:        tasks.TypeWebhookSend,
			uniqueTTL:       time.Minute,
			wantTaskType:    tasks.TypeWebhookSend,
			wantPayloadSize: 1,
			wantOptsLen:     2,
		},
		{
			name:            "unknown task type creates empty payload task",
			typeName:        "custom:task",
			uniqueTTL:       0,
			wantTaskType:    "custom:task",
			wantPayloadSize: 0,
			wantOptsLen:     1,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			td := config.ScheduledTaskDef{
				Type:      tc.typeName,
				Queue:     "default",
				UniqueTTL: tc.uniqueTTL,
			}

			task, opts, err := buildPeriodicTask(td)
			if err != nil {
				t.Fatalf("buildPeriodicTask() unexpected error: %v", err)
			}
			if task.Type() != tc.wantTaskType {
				t.Fatalf("task.Type() = %q, want %q", task.Type(), tc.wantTaskType)
			}

			payloadLen := len(task.Payload())
			if tc.wantPayloadSize == 0 && payloadLen != 0 {
				t.Fatalf("len(task.Payload()) = %d, want 0", payloadLen)
			}
			if tc.wantPayloadSize > 0 && payloadLen < tc.wantPayloadSize {
				t.Fatalf("len(task.Payload()) = %d, want >= %d", payloadLen, tc.wantPayloadSize)
			}
			if len(opts) != tc.wantOptsLen {
				t.Fatalf("len(opts) = %d, want %d", len(opts), tc.wantOptsLen)
			}
		})
	}
}

func helperCanceledContext(t *testing.T) context.Context {
	t.Helper()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	t.Cleanup(cancel)
	return ctx
}

func helperTestConfig() config.Config {
	return config.Config{
		Redis: config.RedisConfig{
			Addr: "127.0.0.1:6379",
			DB:   0,
		},
		Pod: config.PodConfig{ID: "test-pod"},
		Scheduler: config.SchedulerConfig{
			Timezone: "UTC",
			K8sLeaderElection: config.K8sLeaderConfig{
				Enabled: true,
			},
		},
	}
}
