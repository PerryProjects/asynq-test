# Asynq Multi-Pod Prototype

A comprehensive prototype demonstrating [hibiken/asynq](https://github.com/hibiken/asynq) in a multi-pod environment with full feature coverage.

## Architecture

```
┌─────────────┐   ┌──────────────────┐
│   Redis      │◄──┤  Web UI + API    │ :8888
│   :6379      │   │  + Asynqmon      │ :8888/monitoring
└──────┬───────┘   └──────────────────┘
       │
       ├──────────┬──────────┬──────────┐
       │          │          │          │
  ┌────┴───┐ ┌───┴────┐ ┌───┴────┐ ┌───┴────┐
  │worker-1│ │worker-2│ │worker-3│ │worker-4│
  │+sched  │ │+sched  │ │+sched  │ │+sched  │
  └────────┘ └────────┘ └────────┘ └────────┘
```

Each worker pod runs both the asynq server (consumer) and an embedded scheduler (producer/cron). Deduplication via `asynq.Unique`/`asynq.TaskID` ensures only one instance of each periodic task gets enqueued across all pods.

## Quick Start

```bash
docker compose up --build
```

Then open:

- **Dashboard**: http://localhost:8888
- **Asynqmon**: http://localhost:8888/monitoring
- **Redis**: localhost:6379

## Task Types

| Task                | Queue    | Features                                         |
|---------------------|----------|--------------------------------------------------|
| `email:deliver`     | default  | HandlerFunc, MaxRetry(3), Retention(2h)          |
| `image:resize`      | default  | Struct handler, Timeout(30s), ResultWriter       |
| `report:generate`   | low      | Deadline, long-running simulation, periodic cron |
| `webhook:send`      | critical | Unique(1h), SkipRetry on 4xx, strict priority    |
| `notification:send` | default  | Group aggregation — batches per user             |

## CLI Usage

```bash
# Start worker
./asynqtest worker --concurrency 20

# Start web UI
./asynqtest web --port 9090

# Enqueue tasks from CLI
./asynqtest enqueue -t email:deliver -P '{"to":"user@test.com","subject":"Hi","body":"Hello"}'
./asynqtest enqueue -t image:resize -P '{"url":"https://img.com/a.jpg","width":800,"height":600}'
./asynqtest enqueue -t webhook:send -P '{"url":"https://httpbin.org/post","method":"POST","simulate_code":500}' -q critical
./asynqtest enqueue -t notification:send -P '{"user_id":42,"message":"Hello","channel":"push"}'

# With options
./asynqtest enqueue -t email:deliver -P '{"to":"a@b.com","subject":"x","body":"y"}' -d 30 -r 5 -u 60
```

## Features Demonstrated

- Immediate & delayed enqueue (`ProcessIn`)
- Priority queues (weighted: critical:6, default:3, low:1)
- Max retries & custom retry delay (exponential backoff)
- Timeout & Deadline
- Unique tasks & custom TaskID
- Task retention
- Group aggregation (notification batching)
- Result writing via `ResultWriter`
- HandlerFunc & struct Handler patterns
- ServeMux prefix routing & nested mux
- Middleware chain (logging, recovery, rate-limiting)
- Context helpers (`GetTaskID`, `GetRetryCount`, `GetQueueName`)
- ErrorHandler with pod identity
- IsFailure (rate-limited tasks don't count as failures)
- SkipRetry (4xx webhook errors)
- Periodic tasks via embedded scheduler
- Scheduler hooks (PreEnqueue, PostEnqueue, EnqueueErrorHandler)
- Inspector API (queues, servers, task management)
- Queue pause/unpause
- HealthCheckFunc
- Graceful shutdown (SIGTERM/SIGINT)
- Multi-pod load distribution
- Dedup across pods (Unique on periodic tasks)
- Embedded Asynqmon monitoring UI

## Configuration

Configuration is loaded from `config.yaml` and can be overridden with environment variables prefixed with `ASYNQ_`:

```
ASYNQ_REDIS_ADDR=redis:6379
ASYNQ_WORKER_CONCURRENCY=20
ASYNQ_WEB_PORT=9090
POD_ID=my-worker
```

## Tech Stack

- Go 1.26.0
- hibiken/asynq v0.26.0
- hibiken/asynqmon v0.7.2 (embedded)
- spf13/cobra v1.9.1
- spf13/viper v1.20.1
- gin-gonic/gin v1.12.0
- htmx 1.9.12
- Redis latest
- Docker + Docker Compose
