package repository

import (
	"fmt"

	"github.com/fuzumoe/linkTorch-api/internal/model"
)

// Migrator defines the subset of methods needed for migrations.
type Migrator interface {
	AutoMigrate(dst ...any) error
}

// Migrate performs auto-migration for all registered GORM models using the provided Migrator.
func Migrate(m Migrator) error {
	for _, mdl := range model.AllModels {
		if err := m.AutoMigrate(mdl); err != nil {
			return fmt.Errorf("auto-migrate %T: %w", mdl, err)
		}
	}
	return nil
}
