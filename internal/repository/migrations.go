package repository

import (
	"fmt"

	"github.com/fuzumoe/urlinsight-backend/internal/model"
)

// Migrator is an interface for types that can run AutoMigrate (e.g., *gorm.DB).
type Migrator interface {
	AutoMigrate(dst ...interface{}) error
}

// Migrate performs auto-migration for all registered GORM models.
// Pass a Migrator (like *gorm.DB) to run migrations.
func Migrate(m Migrator) error {
	for _, modelInstance := range model.AllModels {
		if err := m.AutoMigrate(modelInstance); err != nil {
			return fmt.Errorf("auto-migrate %T: %w", modelInstance, err)
		}
	}
	return nil
}
