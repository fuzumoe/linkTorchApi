package server

import (
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	_ "github.com/fuzumoe/linkTorch-api/docs"
)

type RouteRegistrar interface {
	RegisterRoutes(rg *gin.RouterGroup)
}

func RegisterRoutes(
	r *gin.Engine,
	jwtSecret string,
	authMiddleware gin.HandlerFunc,
	publicRegs []RouteRegistrar,
	protectedRegs []RouteRegistrar,
) {

	r.Use(gin.Logger(), gin.Recovery())

	public := r.Group("/api/v1")
	for _, reg := range publicRegs {
		reg.RegisterRoutes(public)
	}

	protected := r.Group("/api/v1")
	protected.Use(authMiddleware)
	for _, reg := range protectedRegs {
		reg.RegisterRoutes(protected)
	}

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
}
