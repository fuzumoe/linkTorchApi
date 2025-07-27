package server

import (
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	_ "github.com/fuzumoe/linkTorch-api/docs" // swagger docs
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
	authMiddleware gin.HandlerFunc,
	publicRegs []RouteRegistrar,
	protectedRegs []RouteRegistrar,
) {
	// Global middleware.
	r.Use(gin.Logger(), gin.Recovery())

	// Public API v1 group.
	public := r.Group("/api/v1")
	for _, reg := range publicRegs {
		reg.RegisterRoutes(public)
	}

	// Protected API v1 group.
	protected := r.Group("/api/v1")
	protected.Use(authMiddleware)
	for _, reg := range protectedRegs {
		reg.RegisterRoutes(protected)
	}

	// Add Swagger endpoint.
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
}
