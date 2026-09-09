package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/MonuChaudhary14/Archon/pkg/telemetry"
	"github.com/gin-gonic/gin"
)

func TestTracingMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	ctx := context.Background()
	shutdown, _ := telemetry.InitTracer(ctx, "test-api", "localhost:4317")
	defer func() {
		if shutdown != nil {
			_ = shutdown(ctx)
		}
	}()

	r := gin.New()
	r.Use(TracingMiddleware())

	r.GET("/test-trace", func(c *gin.Context) {
		c.Set("userID", 42)
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req, _ := http.NewRequest(http.MethodGet, "/test-trace", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	traceID := w.Header().Get("X-Trace-ID")
	if traceID == "" {
		t.Fatalf("expected X-Trace-ID header to be present")
	}
}
