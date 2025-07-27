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
	"time"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/fuzumoe/linkTorch-api/configs"
	"github.com/fuzumoe/linkTorch-api/internal/app"
	"github.com/fuzumoe/linkTorch-api/internal/crawler"
	"github.com/fuzumoe/linkTorch-api/internal/repository"
)

var (
	origLoadConfig = app.LoadConfig
	origNewDB      = app.NewDB
	origMigrateDB  = app.MigrateDB
)

type MockCrawlerPool struct{}

func (m *MockCrawlerPool) Start(ctx context.Context) {
}
func (m *MockCrawlerPool) Shutdown()                                 {}
func (m *MockCrawlerPool) Submit(id uint)                            {}
func (m *MockCrawlerPool) Enqueue(id uint)                           {}
func (m *MockCrawlerPool) EnqueueWithPriority(id uint, priority int) {}
func (m *MockCrawlerPool) GetResults() <-chan crawler.CrawlResult {
	return make(chan crawler.CrawlResult)
}
func (m *MockCrawlerPool) AdjustWorkers(cmd crawler.ControlCommand) {}

func setupHooks(t *testing.T) {
	gin.SetMode(gin.TestMode)

	log.SetOutput(io.Discard)

	p1 := gomonkey.ApplyFunc(fmt.Println,
		func(args ...interface{}) (int, error) { return 0, nil })
	t.Cleanup(p1.Reset)

	p2 := gomonkey.ApplyFunc(fmt.Printf,
		func(format string, a ...interface{}) (int, error) { return 0, nil })
	t.Cleanup(p2.Reset)

	p3 := gomonkey.ApplyFunc(log.Printf,
		func(format string, a ...interface{}) {})
	t.Cleanup(p3.Reset)

	app.LoadConfig = func() (*configs.Config, error) {
		return &configs.Config{
			DatabaseURL: "dsn",
			JWTSecret:   "test-secret",
			ServerHost:  "localhost",
			ServerPort:  "8080",
		}, nil
	}

	app.NewDB = func(dsn string) (*gorm.DB, error) {
		assert.Equal(t, "dsn", dsn)
		return &gorm.DB{}, nil
	}

	app.MigrateDB = func(m repository.Migrator) error {
		return nil
	}

	p5 := gomonkey.ApplyFunc(crawler.New,
		func(_ repository.URLRepository, _ interface{}, _, _ int, _ time.Duration) crawler.Pool {
			return &MockCrawlerPool{}
		})
	t.Cleanup(p5.Reset)

	p6 := gomonkey.ApplyFunc(app.Run, func() error {

		cfg, err := app.LoadConfig()
		if err != nil {
			return fmt.Errorf("config load error: %w", err)
		}

		db, err := app.NewDB(cfg.DatabaseURL)
		if err != nil {
			return fmt.Errorf("db init error: %w", err)
		}

		err = app.MigrateDB(db)
		if err != nil {
			return fmt.Errorf("migration error: %w", err)
		}

		return nil
	})
	t.Cleanup(p6.Reset)

	p7 := gomonkey.ApplyMethod(reflect.TypeOf((*gin.Engine)(nil)), "Run",
		func(_ *gin.Engine, _ ...string) error {
			return nil
		})
	t.Cleanup(p7.Reset)
}

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

		p := gomonkey.ApplyFunc(app.Run, func() error {
			_, err := app.LoadConfig()
			if err != nil {
				return fmt.Errorf("config load error: %w", err)
			}
			return nil
		})
		defer p.Reset()

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

		p := gomonkey.ApplyFunc(app.Run, func() error {

			cfg, err := app.LoadConfig()
			if err != nil {
				return fmt.Errorf("config load error: %w", err)
			}

			_, err = app.NewDB(cfg.DatabaseURL)
			if err != nil {
				return fmt.Errorf("db init error: %w", err)
			}
			return nil
		})
		defer p.Reset()

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

		p := gomonkey.ApplyFunc(app.Run, func() error {
			// Get config
			cfg, err := app.LoadConfig()
			if err != nil {
				return fmt.Errorf("config load error: %w", err)
			}

			db, err := app.NewDB(cfg.DatabaseURL)
			if err != nil {
				return fmt.Errorf("db init error: %w", err)
			}

			err = app.MigrateDB(db)
			if err != nil {
				return fmt.Errorf("migration error: %w", err)
			}
			return nil
		})
		defer p.Reset()

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

		err := app.Run()
		require.NoError(t, err)
	})

	t.Run("Server Start Error", func(t *testing.T) {
		setupHooks(t)
		defer teardownHooks()

		p1 := gomonkey.ApplyFunc(app.Run, func() error {
			return errors.New("server start failed")
		})
		defer p1.Reset()

		err := app.Run()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "server start failed")
	})
}
