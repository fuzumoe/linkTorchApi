package app

import (
	"fmt"

	"github.com/gin-gonic/gin"

	"github.com/fuzumoe/urlinsight-backend/configs"
	"github.com/fuzumoe/urlinsight-backend/internal/handler"
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

// Run initializes the application, connects to the database, and starts the HTTP server.
func Run() error {
	// Load configuration
	cfg, err := LoadConfig()
	if err != nil {
		return fmt.Errorf("config load error: %w", err)
	}

	// Connect & migrate DB
	db, err := NewDB(cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("db init error: %w", err)
	}
	if err := MigrateDB(db); err != nil {
		return fmt.Errorf("migration error: %w", err)
	}

	// Instantiate services
	healthSvc := service.NewHealthService(db, "URLInsight Backend")
	// TODO: userSvc := service.NewUserService(...)
	// TODO: urlSvc  := service.NewURLService(...)
	// TODO: authSvc, tokenSvc, linkSvc, analysisSvc...

	// Instantiate handlers
	healthH := handler.NewHealthHandler(healthSvc)
	// TODO: userH := handler.NewUserHandler(userSvc)
	// TODO: urlH  := handler.NewURLHandler(urlSvc)
	// TODO: authH, linkH, analysisH...

	// Build router and register routes
	router := gin.New()
	server.RegisterRoutes(
		router,
		cfg.JWTSecret,
		[]server.RouteRegistrar{
			healthH, // public health endpoints
			// userH, urlH, authH... for public GETs
		},
		[]server.RouteRegistrar{
			// userH, urlH, authH... for protected writes
		},
	)

	//  Run the HTTP server (blocks until error or shutdown)
	addr := fmt.Sprintf("%s:%s", cfg.ServerHost, cfg.ServerPort)
	return router.Run(addr)
}
