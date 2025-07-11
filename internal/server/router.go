package server

import (
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	// Import the docs when they're generated
	_ "github.com/fuzumoe/urlinsight-backend/docs"
)

// RouteRegistrar defines anything that can wire its routes into a Gin group.
type RouteRegistrar interface {
	// RegisterRoutes should add one or more routes on the provided router group.
	RegisterRoutes(rg *gin.RouterGroup)
}

// RegisterRoutes mounts the public and protected routes on the given Gin engine.
func RegisterRoutes(
	r *gin.Engine,
	jwtSecret string,
	publicRegs []RouteRegistrar,
	protectedRegs []RouteRegistrar,
) {
	// Global middleware
	r.Use(gin.Logger(), gin.Recovery())

	// Public API v1
	public := r.Group("/api/v1")
	for _, reg := range publicRegs {
		reg.RegisterRoutes(public)
	}

	// Add Swagger endpoint
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
}
