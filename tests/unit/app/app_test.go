package app_test

import (
	"errors"
	"testing"

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

// setupHooks replaces the hooks for a successful run
func setupHooks(t *testing.T) {
	app.LoadConfig = func() (*configs.Config, error) {
		return &configs.Config{DatabaseURL: "dsn"}, nil
	}
	app.NewDB = func(dsn string) (*gorm.DB, error) {
		assert.Equal(t, "dsn", dsn)
		// Return a dummy *gorm.DB; MigrateDB stub won't use it
		return &gorm.DB{}, nil
	}
	app.MigrateDB = func(m repository.Migrator) error {
		return nil
	}
}

// teardownHooks restores original hook functions
func teardownHooks() {
	app.LoadConfig = origLoadConfig
	app.NewDB = origNewDB
	app.MigrateDB = origMigrateDB
}

func TestRun_Success(t *testing.T) {
	setupHooks(t)
	defer teardownHooks()

	err := app.Run()
	require.NoError(t, err)
}

func TestRun_ConfigError(t *testing.T) {
	setupHooks(t)
	app.LoadConfig = func() (*configs.Config, error) {
		return nil, errors.New("fail load")
	}
	defer teardownHooks()

	err := app.Run()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "config load error")
}

func TestRun_DBError(t *testing.T) {
	setupHooks(t)
	app.NewDB = func(dsn string) (*gorm.DB, error) {
		return nil, errors.New("fail db")
	}
	defer teardownHooks()

	err := app.Run()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "db init error")
}

func TestRun_MigrateError(t *testing.T) {
	setupHooks(t)
	app.MigrateDB = func(m repository.Migrator) error {
		return errors.New("fail migrate")
	}
	defer teardownHooks()

	err := app.Run()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "migration error")
}
