package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// RouteRegistrar defines anything that can wire its routes into a Gin group.
type RouteRegistrar interface {
	// RegisterRoutes should add one or more routes on the provided router group.
	RegisterRoutes(rg *gin.RouterGroup)
}

// RegisterRoutes wires up health, public, and protected routes.
func RegisterRoutes(
	r *gin.Engine,
	jwtSecret string,
	publicRegs []RouteRegistrar,
	protectedRegs []RouteRegistrar,
) {
	// Global middleware
	r.Use(gin.Logger(), gin.Recovery())

	r.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "Welcome to URL Insight Backend!"})
	})
	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Public API v1
	public := r.Group("/api/v1")
	for _, reg := range publicRegs {
		reg.RegisterRoutes(public)
	}

}
