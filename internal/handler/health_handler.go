// internal/handler/health_handler.go
package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/fuzumoe/linkTorch-api/internal/service"
)

type HealthHandler struct {
	healthService service.HealthService
}

func NewHealthHandler(hs service.HealthService) *HealthHandler {
	return &HealthHandler{
		healthService: hs,
	}
}

// Home godoc
// @Summary      Root endpoint
// @Description  Returns a welcome message and service status
// @Tags         health
// @Produce      json
// @Success      200  {object}  map[string]interface{} "Returns message, service name, and status"
// @Router       /status [get]
func (h *HealthHandler) Home(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "Hello World!",
		"service": h.healthService.Check().Service,
		"status":  "running",
	})
}

// Health godoc
// @Summary      Check service health
// @Description  Get the status of server and database connection
// @Tags         health
// @Produce      json
// @Success      200  {object}  map[string]interface{} "Healthy service with database connection"
// @Failure      503  {object}  map[string]interface{} "Service available but database connection issues"
// @Router       /health [get]
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

func (h *HealthHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/status", h.Home)
	rg.GET("/health", h.Health)
}
