package app_test

import (
	"errors"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/fuzumoe/urlinsight-backend/configs"
	"github.com/fuzumoe/urlinsight-backend/internal/app"
	"github.com/fuzumoe/urlinsight-backend/internal/repository"
)

// Save original hook functions
var (
	origLoadConfig = app.LoadConfig
	origNewDB      = app.NewDB
	origMigrateDB  = app.MigrateDB
)

// setupHooks replaces the hooks for a successful run.
func setupHooks(t *testing.T) {
	// For unit tests, disable actual server startup
	gin.SetMode(gin.TestMode)

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
}

// teardownHooks restores original hook functions.
func teardownHooks() {
	app.LoadConfig = origLoadConfig
	app.NewDB = origNewDB
	app.MigrateDB = origMigrateDB
}

func TestRun(t *testing.T) {
	t.Run("Config Error", func(t *testing.T) {
		setupHooks(t)
		// simulate configuration load error
		app.LoadConfig = func() (*configs.Config, error) {
			return nil, errors.New("fail load")
		}
		defer teardownHooks()

		err := app.Run()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "config load error")
	})

	t.Run("DB Error", func(t *testing.T) {
		setupHooks(t)
		// simulate database connection error
		app.NewDB = func(dsn string) (*gorm.DB, error) {
			return nil, errors.New("fail db")
		}
		defer teardownHooks()

		err := app.Run()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "db init error")
	})

	t.Run("Migrate Error", func(t *testing.T) {
		setupHooks(t)
		// simulate migration error
		app.MigrateDB = func(m repository.Migrator) error {
			return errors.New("fail migrate")
		}
		defer teardownHooks()

		err := app.Run()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "migration error")
	})

	t.Run("Server Setup Success", func(t *testing.T) {
		setupHooks(t)
		defer teardownHooks()

		// Mock the gin.Run method to prevent actual server startup
		patches := gomonkey.ApplyMethod((*gin.Engine)(nil), "Run", func(_ *gin.Engine, _ ...string) error {
			return nil // Successfully "started" the server without actually binding
		})
		defer patches.Reset()

		err := app.Run()
		require.NoError(t, err)
	})

	t.Run("Server Start Error", func(t *testing.T) {
		setupHooks(t)
		defer teardownHooks()

		// Mock the gin.Run method to simulate a server startup error
		patches := gomonkey.ApplyMethod((*gin.Engine)(nil), "Run", func(_ *gin.Engine, _ ...string) error {
			return errors.New("server start failed")
		})
		defer patches.Reset()

		err := app.Run()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "server start failed")
	})
}
