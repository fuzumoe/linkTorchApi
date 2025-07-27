package app

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/fuzumoe/linkTorch-api/configs"
	"github.com/fuzumoe/linkTorch-api/internal/analyzer"
	"github.com/fuzumoe/linkTorch-api/internal/crawler"
	"github.com/fuzumoe/linkTorch-api/internal/handler"
	"github.com/fuzumoe/linkTorch-api/internal/middleware"
	"github.com/fuzumoe/linkTorch-api/internal/model"
	"github.com/fuzumoe/linkTorch-api/internal/repository"
	"github.com/fuzumoe/linkTorch-api/internal/server"
	"github.com/fuzumoe/linkTorch-api/internal/service"
)

var (
	LoadConfig = configs.Load
	NewDB      = repository.NewDB
	MigrateDB  = repository.Migrate
)

type RouteRegistrarFunc func(rg *gin.RouterGroup)

func (f RouteRegistrarFunc) RegisterRoutes(rg *gin.RouterGroup) {
	f(rg)
}

func Run() error {
	cfg, err := LoadConfig()
	if err != nil {
		return fmt.Errorf("config load error: %w", err)
	}

	db, err := NewDB(cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("db init error: %w", err)
	}
	if err := MigrateDB(db); err != nil {
		return fmt.Errorf("migration error: %w", err)
	}

	userRepo := repository.NewUserRepo(db)
	authRepo := repository.NewTokenRepo(db)
	urlRepo := repository.NewURLRepo(db)

	healthSvc := service.NewHealthService(db, "LinkTorch API")
	userSvc := service.NewUserService(userRepo)
	authSVC := service.NewAuthService(
		userRepo,
		authRepo,
		cfg.JWTSecret,
		cfg.JWTLifetime,
	)

	htmlAnalyzer := analyzer.NewHTMLAnalyzer()
	crawlerPool := crawler.New(urlRepo, htmlAnalyzer, cfg.NumberOfCrawlers, cfg.MaxConcurrentCrawls, cfg.CrawlTimeout)

	urlSvc := service.NewURLService(urlRepo, crawlerPool)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go crawlerPool.Start(ctx)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		sig := <-sigCh
		log.Printf("Received signal: %v. Initiating graceful shutdown...", sig)
		cancel()
	}()

	if cfg.ServerMode == "debug" && cfg.DevUserEmail != "" && cfg.DevUserPassword != "" {
		createUserInput := &model.CreateUserInput{
			Email:    cfg.DevUserEmail,
			Password: cfg.DevUserPassword,
			Username: cfg.DevUserName,
		}
		user, userErr := userSvc.Register(createUserInput)
		if userErr != nil {
			fmt.Printf("Notice: Dev user already exists or could not be created: %v\n", userErr)

			existingUser, authErr := userSvc.Authenticate(cfg.DevUserEmail, cfg.DevUserPassword)
			if authErr == nil && existingUser != nil {
				token, tokenErr := authSVC.Generate(existingUser.ID)
				basicCred := base64.StdEncoding.EncodeToString([]byte(cfg.DevUserEmail + ":" + cfg.DevUserPassword))
				if tokenErr == nil {
					fmt.Printf("🔑 Development credentials (with token):\n")
					fmt.Printf("Bearer %s\n", token)
					fmt.Printf("Basic %s\n", basicCred)
				} else {
					fmt.Printf("🔑 Development credentials (token generation failed: %v):\n", tokenErr)
				}
			} else {
				fmt.Printf("🔑 Development credentials (couldn't authenticate: %v):\n", authErr)
			}
			fmt.Printf("🔑 Development credentials:\n")
			fmt.Printf("   Email: %s\n", cfg.DevUserEmail)
			fmt.Printf("   Username: %s\n", cfg.DevUserName)
			fmt.Printf("   Password: %s\n", cfg.DevUserPassword)
		} else {
			// User was created successfully, now generate token.
			token, tokenErr := authSVC.Generate(user.ID)
			basicCred := base64.StdEncoding.EncodeToString([]byte(cfg.DevUserEmail + ":" + cfg.DevUserPassword))
			if tokenErr != nil {
				fmt.Printf("🔑 Created development user (token generation failed: %v):\n", tokenErr)
			} else {
				fmt.Printf("🔑 Created development user with token:\n")
				fmt.Printf("Bearer %s\n", token)
				fmt.Printf("Basic %s\n", basicCred)
			}
			fmt.Printf("   Email: %s\n", user.Email)
			fmt.Printf("   Username: %s\n", user.Username)
			fmt.Printf("   Password: %s\n", cfg.DevUserPassword)
		}
	}

	dualAuthMiddleware := middleware.AuthMiddleware(authSVC)

	healthH := handler.NewHealthHandler(healthSvc)
	authH := handler.NewAuthHandler(authSVC, userSvc)
	urlH := handler.NewURLHandler(urlSvc)
	userH := handler.NewUserHandler(userSvc)

	router := gin.New()
	publicRegs := []server.RouteRegistrar{
		RouteRegistrarFunc(func(rg *gin.RouterGroup) {
			authH.RegisterPublicRoutes(rg)
		}),
		healthH,
	}
	protectedRegs := []server.RouteRegistrar{
		RouteRegistrarFunc(func(rg *gin.RouterGroup) {
			authH.RegisterProtectedRoutes(rg)
		}),
		RouteRegistrarFunc(func(rg *gin.RouterGroup) {
			urlH.RegisterProtectedRoutes(rg)
		}),
		RouteRegistrarFunc(func(rg *gin.RouterGroup) {
			userH.RegisterProtectedRoutes(rg)
		}),
	}
	server.RegisterRoutes(
		router,
		cfg.JWTSecret,
		dualAuthMiddleware,
		publicRegs,
		protectedRegs,
	)

	addr := fmt.Sprintf("%s:%s", cfg.ServerHost, cfg.ServerPort)
	srv := &http.Server{
		Addr:    addr,
		Handler: router,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server listen error: %v", err)
		}
	}()

	log.Printf("Server running on %s. Press Ctrl+C to exit.", addr)

	<-ctx.Done()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("server shutdown failed: %w", err)
	}

	log.Println("HTTP server shut down gracefully. Exiting application.")
	return nil
}
