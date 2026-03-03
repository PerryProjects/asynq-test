package config

import (
	"os"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config holds all application configuration.
type Config struct {
	Redis         RedisConfig         `mapstructure:"redis"`
	Worker        WorkerConfig        `mapstructure:"worker"`
	Scheduler     SchedulerConfig     `mapstructure:"scheduler"`
	Web           WebConfig           `mapstructure:"web"`
	Pod           PodConfig           `mapstructure:"pod"`
	Serialization SerializationConfig `mapstructure:"serialization"`
}

// RedisConfig holds Redis connection settings.
type RedisConfig struct {
	Addr     string `mapstructure:"addr"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

// WorkerConfig holds worker server settings.
type WorkerConfig struct {
	Concurrency     int            `mapstructure:"concurrency"`
	Queues          map[string]int `mapstructure:"queues"`
	StrictPriority  bool           `mapstructure:"strict_priority"`
	ShutdownTimeout time.Duration  `mapstructure:"shutdown_timeout"`
}

// SchedulerConfig holds periodic task scheduler settings.
type SchedulerConfig struct {
	Timezone              string             `mapstructure:"timezone"`
	Tasks                 []ScheduledTaskDef `mapstructure:"tasks"`
	LeaderLockKey         string             `mapstructure:"leader_lock_key"`
	LeaderLockTTL         time.Duration      `mapstructure:"leader_lock_ttl"`
	LeaderRefreshInterval time.Duration      `mapstructure:"leader_refresh_interval"`
	LeaderRetryInterval   time.Duration      `mapstructure:"leader_retry_interval"`
}

// ScheduledTaskDef defines a single periodic task.
type ScheduledTaskDef struct {
	Cronspec  string        `mapstructure:"cronspec"`
	Type      string        `mapstructure:"type"`
	Queue     string        `mapstructure:"queue"`
	UniqueTTL time.Duration `mapstructure:"unique_ttl"`
}

// WebConfig holds web UI server settings.
type WebConfig struct {
	Port int `mapstructure:"port"`
}

// PodConfig holds pod identity settings.
type PodConfig struct {
	ID string `mapstructure:"id"`
}

// SerializationConfig holds payload serialization format settings.
type SerializationConfig struct {
	Format string `mapstructure:"format"`
}

// C is the global configuration instance.
var C Config

// Load reads configuration from file + env vars into the global C.
func Load() error {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("/etc/asynqtest/")

	// Env var binding
	viper.SetEnvPrefix("ASYNQ")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	// Defaults
	viper.SetDefault("redis.addr", "localhost:6379")
	viper.SetDefault("redis.password", "")
	viper.SetDefault("redis.db", 0)
	viper.SetDefault("worker.concurrency", 10)
	viper.SetDefault("worker.shutdown_timeout", "10s")
	viper.SetDefault("web.port", 8888)
	viper.SetDefault("scheduler.timezone", "UTC")
	viper.SetDefault("scheduler.leader_lock_key", "asynq-scheduler-leader")
	viper.SetDefault("scheduler.leader_lock_ttl", "15s")
	viper.SetDefault("scheduler.leader_refresh_interval", "5s")
	viper.SetDefault("scheduler.leader_retry_interval", "2s")
	viper.SetDefault("serialization.format", "json")

	if err := viper.ReadInConfig(); err != nil {
		// Config file not found is not fatal — we can run on env vars alone.
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return err
		}
	}

	if err := viper.Unmarshal(&C); err != nil {
		return err
	}

	// Pod ID: prefer env POD_ID, then config, then hostname.
	if envPod := os.Getenv("POD_ID"); envPod != "" {
		C.Pod.ID = envPod
	}
	if C.Pod.ID == "" {
		hostname, _ := os.Hostname()
		C.Pod.ID = hostname
	}

	return nil
}
