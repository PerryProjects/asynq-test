package config

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/spf13/viper"
)

var configTestMu sync.Mutex

func TestLoad(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		cfgYAML   string
		env       map[string]string
		assertion func(t *testing.T)
	}{
		{
			name:    "defaults without config file",
			cfgYAML: "",
			assertion: func(t *testing.T) {
				t.Helper()
				if C.Redis.Addr != "localhost:6379" {
					t.Fatalf("C.Redis.Addr = %q, want %q", C.Redis.Addr, "localhost:6379")
				}
				if C.Scheduler.K8sLeaderElection.LeaseName != "asynq-scheduler-leader" {
					t.Fatalf("lease name = %q, want asynq-scheduler-leader", C.Scheduler.K8sLeaderElection.LeaseName)
				}
				if C.Pod.ID == "" {
					t.Fatal("C.Pod.ID is empty")
				}
			},
		},
		{
			name: "env overrides config",
			env: map[string]string{
				"ASYNQ_WORKER_CONCURRENCY": "23",
				"POD_ID":                  "env-pod",
			},
			assertion: func(t *testing.T) {
				t.Helper()
				if C.Worker.Concurrency != 23 {
					t.Fatalf("C.Worker.Concurrency = %d, want 23", C.Worker.Concurrency)
				}
				if C.Pod.ID != "env-pod" {
					t.Fatalf("C.Pod.ID = %q, want env-pod", C.Pod.ID)
				}
			},
		},
		{
			name: "reads values from config file",
			cfgYAML: `redis:
  addr: "redis.test:6380"
worker:
  concurrency: 15
scheduler:
  timezone: "America/New_York"
  k8s_leader_election:
    enabled: true
    namespace: "jobs"
    lease_name: "lease-a"
    lease_duration: 21s
    renew_deadline: 13s
    retry_period: 3s
`,
			assertion: func(t *testing.T) {
				t.Helper()
				if C.Redis.Addr != "redis.test:6380" {
					t.Fatalf("C.Redis.Addr = %q, want redis.test:6380", C.Redis.Addr)
				}
				if C.Worker.Concurrency != 15 {
					t.Fatalf("C.Worker.Concurrency = %d, want 15", C.Worker.Concurrency)
				}
				if C.Scheduler.K8sLeaderElection.Namespace != "jobs" {
					t.Fatalf("namespace = %q, want jobs", C.Scheduler.K8sLeaderElection.Namespace)
				}
				if C.Scheduler.K8sLeaderElection.LeaseDuration != 21*time.Second {
					t.Fatalf("lease duration = %v, want 21s", C.Scheduler.K8sLeaderElection.LeaseDuration)
				}
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			helperLockConfigIsolation(t)
			helperResetConfigGlobals(t)
			helperSetEnvMap(t, tc.env)
			helperWithTempConfigDir(t, tc.cfgYAML)

			if err := Load(); err != nil {
				t.Fatalf("Load() error = %v", err)
			}
			tc.assertion(t)
		})
	}
}

func helperLockConfigIsolation(t *testing.T) {
	t.Helper()
	configTestMu.Lock()
	t.Cleanup(configTestMu.Unlock)
}

func helperResetConfigGlobals(t *testing.T) {
	t.Helper()
	old := C
	C = Config{}
	viper.Reset()
	t.Cleanup(func() {
		C = old
		viper.Reset()
	})
}

func helperSetEnvMap(t *testing.T, env map[string]string) {
	t.Helper()
	for k, v := range env {
		t.Setenv(k, v)
	}
}

func helperWithTempConfigDir(t *testing.T, cfgYAML string) {
	t.Helper()
	tempDir := t.TempDir()
	if cfgYAML != "" {
		cfgPath := filepath.Join(tempDir, "config.yaml")
		if err := os.WriteFile(cfgPath, []byte(cfgYAML), 0o600); err != nil {
			t.Fatalf("write config file: %v", err)
		}
	}
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error: %v", err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Chdir(%q) error: %v", tempDir, err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(oldWD); err != nil {
			t.Fatalf("restore wd error: %v", err)
		}
	})
}
