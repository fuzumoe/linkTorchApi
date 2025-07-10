package app

import (
	"fmt"

	"github.com/fuzumoe/urlinsight-backend/configs"
	"github.com/fuzumoe/urlinsight-backend/internal/repository"
)

// hookable functions for dependency injection
var (
	LoadConfig = configs.Load
	NewDB      = repository.NewDB
	MigrateDB  = repository.Migrate
)

// Run loads config, opens DB, runs migrations.
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

	return nil
}
