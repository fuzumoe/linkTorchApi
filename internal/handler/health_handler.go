// internal/handler/health_handler.go
package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/fuzumoe/urlinsight-backend/internal/service"
)

// HealthHandler provides health and status endpoints.
// HealthHandler handles HTTP requests related to application health.
type HealthHandler struct {
	healthService service.HealthService
}

// NewHealthHandler creates a new HealthHandler.
// NewHealthHandler creates a new HealthHandler.
func NewHealthHandler(hs service.HealthService) *HealthHandler {
	return &HealthHandler{
		healthService: hs,
	}
}

// Home returns a simple “running” status for the root endpoint.
func (h *HealthHandler) Home(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "Hello World!",
		"service": h.healthService.Check().Service,
		"status":  "running",
	})
}

// Health returns application and database health.
func (h *HealthHandler) Health(c *gin.Context) {
	stat := h.healthService.Check()
	code := http.StatusOK
	if !stat.Healthy {
		code = http.StatusServiceUnavailable
	}
	c.JSON(code, gin.H{
		"service":  stat.Service,
		"status":   "ok",
		"database": stat.Database,
		"checked":  stat.Checked.Format(time.RFC3339),
	})
}

// RegisterRoutes mounts the health endpoints on the given router group.
func (h *HealthHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/", h.Home)
	rg.GET("/health", h.Health)
}
