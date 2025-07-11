package app

import (
	"fmt"

	"github.com/gin-gonic/gin"

	"github.com/fuzumoe/urlinsight-backend/configs"
	"github.com/fuzumoe/urlinsight-backend/internal/handler"
	"github.com/fuzumoe/urlinsight-backend/internal/middleware"
	"github.com/fuzumoe/urlinsight-backend/internal/repository"
	"github.com/fuzumoe/urlinsight-backend/internal/server"
	"github.com/fuzumoe/urlinsight-backend/internal/service"
)

// hookable for tests
var (
	LoadConfig = configs.Load
	NewDB      = repository.NewDB
	MigrateDB  = repository.Migrate
)

// A helper function type so we can use functions as RouteRegistrar.
type RouteRegistrarFunc func(rg *gin.RouterGroup)

// RegisterRoutes implements the RouteRegistrar interface.
func (f RouteRegistrarFunc) RegisterRoutes(rg *gin.RouterGroup) {
	f(rg)
}

func Run() error {
	// Load configuration.
	cfg, err := LoadConfig()
	if err != nil {
		return fmt.Errorf("config load error: %w", err)
	}

	// Connect & migrate DB.
	db, err := NewDB(cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("db init error: %w", err)
	}
	if err := MigrateDB(db); err != nil {
		return fmt.Errorf("migration error: %w", err)
	}

	// Initialize repositories.
	userRepo := repository.NewUserRepo(db)
	authRepo := repository.NewTokenRepo(db)

	// Instantiate services.
	healthSvc := service.NewHealthService(db, "URLInsight Backend")
	userSvc := service.NewUserService(userRepo)
	authSVC := service.NewAuthService(
		userRepo,
		authRepo,
		cfg.JWTSecret,
		cfg.JWTLifetime,
	)
	// Initialize DualAuthMiddleware with the auth service and user service.
	dualAuthMiddleware := middleware.AuthMiddleware(authSVC)

	// Instantiate handlers.
	healthH := handler.NewHealthHandler(healthSvc)
	authH := handler.NewAuthHandler(authSVC, userSvc)

	// Build router and register routes.
	router := gin.New()

	// Create route registrars that wrap the handler methods.
	publicRegs := []server.RouteRegistrar{
		RouteRegistrarFunc(func(rg *gin.RouterGroup) {
			// Register public endpoints for auth (login endpoints).
			authH.RegisterPublicRoutes(rg)
		}),
		healthH, // assuming healthH implements RouteRegistrar.
	}

	protectedRegs := []server.RouteRegistrar{
		RouteRegistrarFunc(func(rg *gin.RouterGroup) {
			// Register protected endpoints for auth (register & logout endpoints).
			authH.RegisterProtectedRoutes(rg)
		}),
		// add additional protected route registrars as needed.
	}

	// And then pass it in:
	server.RegisterRoutes(
		router,
		cfg.JWTSecret,
		dualAuthMiddleware, // now using the dual authentication middleware
		publicRegs,
		protectedRegs,
	)

	// Run the HTTP server.
	addr := fmt.Sprintf("%s:%s", cfg.ServerHost, cfg.ServerPort)
	return router.Run(addr)
}
