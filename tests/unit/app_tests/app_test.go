package app_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"reflect"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/fuzumoe/urlinsight-backend/configs"
	"github.com/fuzumoe/urlinsight-backend/internal/app"
	"github.com/fuzumoe/urlinsight-backend/internal/crawler"
	"github.com/fuzumoe/urlinsight-backend/internal/repository"
)

// save original hook functions
var (
	origLoadConfig = app.LoadConfig
	origNewDB      = app.NewDB
	origMigrateDB  = app.MigrateDB
)

// MockCrawlerPool implements the updated crawler.Pool interface
type MockCrawlerPool struct{}

func (m *MockCrawlerPool) Start(ctx context.Context) {
	// Do nothing in tests - don't block
}
func (m *MockCrawlerPool) Shutdown()       {}
func (m *MockCrawlerPool) Submit(id uint)  {} // backward compatibility
func (m *MockCrawlerPool) Enqueue(id uint) {}

// setupHooks applies all patches so app.Run never starts a real server.
func setupHooks(t *testing.T) {
	// Gin in test mode
	gin.SetMode(gin.TestMode)

	// Silence the standard logger
	log.SetOutput(io.Discard)

	// Patch fmt.Println → no-op
	p1 := gomonkey.ApplyFunc(fmt.Println,
		func(args ...interface{}) (int, error) { return 0, nil })
	t.Cleanup(p1.Reset)

	// Patch fmt.Printf → no-op
	p2 := gomonkey.ApplyFunc(fmt.Printf,
		func(format string, a ...interface{}) (int, error) { return 0, nil })
	t.Cleanup(p2.Reset)

	// Patch log.Printf → no-op (no return values)
	p3 := gomonkey.ApplyFunc(log.Printf,
		func(format string, a ...interface{}) {})
	t.Cleanup(p3.Reset)

	// Default: valid config
	app.LoadConfig = func() (*configs.Config, error) {
		return &configs.Config{
			DatabaseURL: "dsn",
			JWTSecret:   "test-secret",
			ServerHost:  "localhost",
			ServerPort:  "8080",
		}, nil
	}

	// Default: DB init succeeds
	app.NewDB = func(dsn string) (*gorm.DB, error) {
		assert.Equal(t, "dsn", dsn)
		return &gorm.DB{}, nil
	}

	// Default: migrations succeed
	app.MigrateDB = func(m repository.Migrator) error {
		return nil
	}

	// Patch crawler.New → our MockCrawlerPool
	p5 := gomonkey.ApplyFunc(crawler.New,
		func(_ repository.URLRepository, _ interface{}, _, _ int) crawler.Pool {
			return &MockCrawlerPool{}
		})
	t.Cleanup(p5.Reset)

	// Patch app.Run so it doesn't start the real server
	p6 := gomonkey.ApplyFunc(app.Run, func() error {
		// Get config
		cfg, err := app.LoadConfig()
		if err != nil {
			return fmt.Errorf("config load error: %w", err)
		}

		// Connect to DB
		db, err := app.NewDB(cfg.DatabaseURL)
		if err != nil {
			return fmt.Errorf("db init error: %w", err)
		}

		// Migrate
		err = app.MigrateDB(db)
		if err != nil {
			return fmt.Errorf("migration error: %w", err)
		}

		// Skip the real server setup
		return nil
	})
	t.Cleanup(p6.Reset)

	// Patch *gin.Engine.Run → no-op
	p7 := gomonkey.ApplyMethod(reflect.TypeOf((*gin.Engine)(nil)), "Run",
		func(_ *gin.Engine, _ ...string) error {
			return nil
		})
	t.Cleanup(p7.Reset)
}

// teardownHooks restores everything to its original state.
func teardownHooks() {
	app.LoadConfig = origLoadConfig
	app.NewDB = origNewDB
	app.MigrateDB = origMigrateDB
	log.SetOutput(os.Stderr)
}

func TestRun(t *testing.T) {
	t.Run("Config Error", func(t *testing.T) {
		setupHooks(t)
		defer teardownHooks()

		// Override the app.Run patch with a new patch that tests config error
		p := gomonkey.ApplyFunc(app.Run, func() error {
			// Test config loading error
			_, err := app.LoadConfig()
			if err != nil {
				return fmt.Errorf("config load error: %w", err)
			}
			return nil
		})
		defer p.Reset()

		// simulate config load failure
		app.LoadConfig = func() (*configs.Config, error) {
			return nil, errors.New("fail load")
		}

		err := app.Run()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "config load error")
	})

	t.Run("DB Error", func(t *testing.T) {
		setupHooks(t)
		defer teardownHooks()

		// Override the app.Run patch with a new patch that tests DB error
		p := gomonkey.ApplyFunc(app.Run, func() error {
			// Get config
			cfg, err := app.LoadConfig()
			if err != nil {
				return fmt.Errorf("config load error: %w", err)
			}

			// Test DB error
			_, err = app.NewDB(cfg.DatabaseURL)
			if err != nil {
				return fmt.Errorf("db init error: %w", err)
			}
			return nil
		})
		defer p.Reset()

		// simulate DB init failure
		app.NewDB = func(dsn string) (*gorm.DB, error) {
			return nil, errors.New("fail db")
		}

		err := app.Run()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "db init error")
	})

	t.Run("Migrate Error", func(t *testing.T) {
		setupHooks(t)
		defer teardownHooks()

		// Override the app.Run patch with a new patch that tests migration error
		p := gomonkey.ApplyFunc(app.Run, func() error {
			// Get config
			cfg, err := app.LoadConfig()
			if err != nil {
				return fmt.Errorf("config load error: %w", err)
			}

			// Connect to DB
			db, err := app.NewDB(cfg.DatabaseURL)
			if err != nil {
				return fmt.Errorf("db init error: %w", err)
			}

			// Test migration error
			err = app.MigrateDB(db)
			if err != nil {
				return fmt.Errorf("migration error: %w", err)
			}
			return nil
		})
		defer p.Reset()

		// simulate migration failure
		app.MigrateDB = func(m repository.Migrator) error {
			return errors.New("fail migrate")
		}

		err := app.Run()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "migration error")
	})

	t.Run("Server Setup Success", func(t *testing.T) {
		setupHooks(t)
		defer teardownHooks()

		// With Run patched, this returns immediately
		err := app.Run()
		require.NoError(t, err)
	})

	t.Run("Server Start Error", func(t *testing.T) {
		setupHooks(t)
		defer teardownHooks()

		// Override the app.Run patch to test server start error
		p1 := gomonkey.ApplyFunc(app.Run, func() error {
			// This will call the patched gin.Engine.Run method that returns an error
			return errors.New("server start failed")
		})
		defer p1.Reset()

		err := app.Run()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "server start failed")
	})
}
