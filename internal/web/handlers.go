package web

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/hibiken/asynq"
)

// DashboardHandler serves the main dashboard page.
func DashboardHandler(inspector *asynq.Inspector) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Gather queue info.
		queues, err := inspector.Queues()
		if err != nil {
			c.HTML(http.StatusInternalServerError, "index.html", gin.H{
				"Error": err.Error(),
			})
			return
		}

		type QueueStat struct {
			Name      string
			Size      int
			Pending   int
			Active    int
			Scheduled int
			Retry     int
			Archived  int
			Completed int
			Paused    bool
		}

		var queueStats []QueueStat
		for _, q := range queues {
			info, err := inspector.GetQueueInfo(q)
			if err != nil {
				continue
			}
			queueStats = append(queueStats, QueueStat{
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

		// Gather server info.
		servers, err := inspector.Servers()
		if err != nil {
			servers = nil
		}

		type ServerInfo struct {
			ID          string
			Host        string
			Concurrency int
			Queues      map[string]int
			ActiveCount int
			Status      string
		}

		var serverInfos []ServerInfo
		for _, s := range servers {
			serverInfos = append(serverInfos, ServerInfo{
				ID:          s.ID,
				Host:        s.Host,
				Concurrency: s.Concurrency,
				Queues:      s.Queues,
				ActiveCount: len(s.ActiveWorkers),
				Status:      s.Status,
			})
		}

		c.HTML(http.StatusOK, "index.html", gin.H{
			"Queues":  queueStats,
			"Servers": serverInfos,
		})
	}
}
