package web

import (
	"html/template"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/hibiken/asynq"
)

func TestDashboardHandler_RedisFailure(t *testing.T) {
	t.Parallel()
	t.Cleanup(func() {})

	inspector := asynq.NewInspector(asynq.RedisClientOpt{Addr: "127.0.0.1:0"})
	r := gin.New()
	r.SetHTMLTemplate(template.Must(template.New("index.html").Parse("{{.Error}}")))
	r.GET("/", DashboardHandler(inspector))

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d want %d", w.Code, http.StatusInternalServerError)
	}
}
