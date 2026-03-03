package web

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hibiken/asynq"

	"github.com/asynq-test/internal/tasks"
)

// EnqueueRequest is the JSON body for POST /api/tasks/enqueue.
type EnqueueRequest struct {
	Type         string          `json:"type"`
	Payload      json.RawMessage `json:"payload"`
	Queue        string          `json:"queue"`
	DelaySeconds int             `json:"delay_seconds"`
	MaxRetry     int             `json:"max_retry"`
	UniqueTTL    int             `json:"unique_ttl_seconds"`
}

// EnqueueHandler creates tasks from the web UI / API.
func EnqueueHandler(redisOpt asynq.RedisClientOpt) gin.HandlerFunc {
	client := asynq.NewClient(redisOpt)

	return func(c *gin.Context) {
		var req EnqueueRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		task, opts, err := buildTaskFromRequest(req)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		info, err := client.Enqueue(task, opts...)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message":   "Task enqueued successfully",
			"task_id":   info.ID,
			"queue":     info.Queue,
			"type":      info.Type,
			"max_retry": info.MaxRetry,
		})
	}
}

func buildTaskFromRequest(req EnqueueRequest) (*asynq.Task, []asynq.Option, error) {
	var task *asynq.Task
	var err error

	switch req.Type {
	case tasks.TypeEmailDeliver:
		var p tasks.EmailPayload
		if err := json.Unmarshal(req.Payload, &p); err != nil {
			return nil, nil, fmt.Errorf("invalid email payload: %w", err)
		}
		task, err = tasks.NewEmailDeliverTask(p.To, p.Subject, p.Body)

	case tasks.TypeImageResize:
		var p tasks.ImagePayload
		if err := json.Unmarshal(req.Payload, &p); err != nil {
			return nil, nil, fmt.Errorf("invalid image payload: %w", err)
		}
		task, err = tasks.NewImageResizeTask(p.URL, p.Width, p.Height)

	case tasks.TypeReportGenerate:
		var p tasks.ReportPayload
		if err := json.Unmarshal(req.Payload, &p); err != nil {
			return nil, nil, fmt.Errorf("invalid report payload: %w", err)
		}
		task, err = tasks.NewReportGenerateTask(p.ReportType, p.StartDate, p.EndDate)

	case tasks.TypeWebhookSend:
		var p tasks.WebhookPayload
		if err := json.Unmarshal(req.Payload, &p); err != nil {
			return nil, nil, fmt.Errorf("invalid webhook payload: %w", err)
		}
		task, err = tasks.NewWebhookSendTask(p.URL, p.Method, p.SimulateCode)

	case tasks.TypeNotificationSend:
		var p tasks.NotificationPayload
		if err := json.Unmarshal(req.Payload, &p); err != nil {
			return nil, nil, fmt.Errorf("invalid notification payload: %w", err)
		}
		task, err = tasks.NewNotificationSendTask(p.UserID, p.Message, p.Channel)

	default:
		return nil, nil, fmt.Errorf("unknown task type: %s", req.Type)
	}

	if err != nil {
		return nil, nil, err
	}

	// Build additional options from request.
	var opts []asynq.Option
	if req.Queue != "" {
		opts = append(opts, asynq.Queue(req.Queue))
	}
	if req.DelaySeconds > 0 {
		opts = append(opts, asynq.ProcessIn(time.Duration(req.DelaySeconds)*time.Second))
	}
	if req.MaxRetry > 0 {
		opts = append(opts, asynq.MaxRetry(req.MaxRetry))
	}
	if req.UniqueTTL > 0 {
		opts = append(opts, asynq.Unique(time.Duration(req.UniqueTTL)*time.Second))
	}

	return task, opts, nil
}

// QueuesHandler returns queue stats as JSON.
func QueuesHandler(inspector *asynq.Inspector) gin.HandlerFunc {
	return func(c *gin.Context) {
		queues, err := inspector.Queues()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		type QueueInfo struct {
			Name      string `json:"name"`
			Size      int    `json:"size"`
			Pending   int    `json:"pending"`
			Active    int    `json:"active"`
			Scheduled int    `json:"scheduled"`
			Retry     int    `json:"retry"`
			Archived  int    `json:"archived"`
			Completed int    `json:"completed"`
			Paused    bool   `json:"paused"`
		}

		var result []QueueInfo
		for _, q := range queues {
			info, err := inspector.GetQueueInfo(q)
			if err != nil {
				continue
			}
			result = append(result, QueueInfo{
				Name:      info.Queue,
				Size:      info.Size,
				Pending:   info.Pending,
				Active:    info.Active,
				Scheduled: info.Scheduled,
				Retry:     info.Retry,
				Archived:  info.Archived,
				Completed: info.Completed,
				Paused:    info.Paused,
			})
		}
		c.JSON(http.StatusOK, result)
	}
}

// ServersHandler returns connected worker server info.
func ServersHandler(inspector *asynq.Inspector) gin.HandlerFunc {
	return func(c *gin.Context) {
		servers, err := inspector.Servers()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		type ServerOut struct {
			ID          string         `json:"id"`
			Host        string         `json:"host"`
			PID         int            `json:"pid"`
			Concurrency int            `json:"concurrency"`
			Queues      map[string]int `json:"queues"`
			ActiveCount int            `json:"active_count"`
			Status      string         `json:"status"`
			Started     string         `json:"started"`
		}

		var result []ServerOut
		for _, s := range servers {
			result = append(result, ServerOut{
				ID:          s.ID,
				Host:        s.Host,
				PID:         s.PID,
				Concurrency: s.Concurrency,
				Queues:      s.Queues,
				ActiveCount: len(s.ActiveWorkers),
				Status:      s.Status,
				Started:     s.Started.Format(time.RFC3339),
			})
		}
		c.JSON(http.StatusOK, result)
	}
}

// PauseQueueHandler pauses a queue.
func PauseQueueHandler(inspector *asynq.Inspector) gin.HandlerFunc {
	return func(c *gin.Context) {
		name := c.Param("name")
		if err := inspector.PauseQueue(name); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "Queue " + name + " paused"})
	}
}

// UnpauseQueueHandler unpauses a queue.
func UnpauseQueueHandler(inspector *asynq.Inspector) gin.HandlerFunc {
	return func(c *gin.Context) {
		name := c.Param("name")
		if err := inspector.UnpauseQueue(name); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "Queue " + name + " unpaused"})
	}
}

// DeleteTaskHandler deletes a task from a queue.
func DeleteTaskHandler(inspector *asynq.Inspector) gin.HandlerFunc {
	return func(c *gin.Context) {
		queueName := c.Param("queue")
		taskID := c.Param("id")

		// Try deleting from different states.
		var deleted bool
		for _, state := range []string{"pending", "scheduled", "retry", "archived", "completed"} {
			var err error
			switch state {
			case "pending":
				err = inspector.DeleteTask(queueName, taskID)
			case "scheduled":
				err = inspector.DeleteTask(queueName, taskID)
			case "retry":
				err = inspector.DeleteTask(queueName, taskID)
			case "archived":
				err = inspector.DeleteTask(queueName, taskID)
			case "completed":
				err = inspector.DeleteTask(queueName, taskID)
			}
			if err == nil {
				deleted = true
				break
			}
		}

		if !deleted {
			c.JSON(http.StatusNotFound, gin.H{"error": "task not found in queue " + queueName})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "Task " + taskID + " deleted from " + queueName})
	}
}

// intParam parses an integer from query/form params with a default.
func intParam(c *gin.Context, name string, defaultVal int) int {
	if v := c.Query(name); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return defaultVal
}
