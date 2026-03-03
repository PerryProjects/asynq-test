package web

import (
	"html/template"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/hibiken/asynq"
	"github.com/hibiken/asynqmon"

	"github.com/asynq-test/internal/config"
)

// NewRouter sets up Gin router with dashboard, API, and embedded Asynqmon.
func NewRouter(cfg config.Config) *gin.Engine {
	router := gin.Default()

	// Parse embedded template.
	tmpl := template.Must(template.New("index.html").Parse(indexHTML))
	router.SetHTMLTemplate(tmpl)

	// Redis connection for Inspector + Asynqmon.
	redisOpt := asynq.RedisClientOpt{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	}

	inspector := asynq.NewInspector(redisOpt)

	// ── Dashboard page ──────────────────────────────────────────────
	router.GET("/", DashboardHandler(inspector))

	// ── REST API ────────────────────────────────────────────────────
	api := router.Group("/api")
	{
		api.POST("/tasks/enqueue", EnqueueHandler(redisOpt))
		api.GET("/queues", QueuesHandler(inspector))
		api.GET("/servers", ServersHandler(inspector))
		api.POST("/queues/:name/pause", PauseQueueHandler(inspector))
		api.POST("/queues/:name/unpause", UnpauseQueueHandler(inspector))
		api.DELETE("/tasks/:queue/:id", DeleteTaskHandler(inspector))
	}

	// ── Embedded Asynqmon ───────────────────────────────────────────
	h := asynqmon.New(asynqmon.Options{
		RootPath:     "/monitoring",
		RedisConnOpt: redisOpt,
	})
	router.Any("/monitoring/*any", gin.WrapH(h))

	// ── Health check ────────────────────────────────────────────────
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	return router
}
