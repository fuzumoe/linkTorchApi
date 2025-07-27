package repository

import (
	"fmt"

	"github.com/fuzumoe/linkTorch-api/internal/model"
)

type Migrator interface {
	AutoMigrate(dst ...any) error
}

func Migrate(m Migrator) error {
	for _, mdl := range model.AllModels {
		if err := m.AutoMigrate(mdl); err != nil {
			return fmt.Errorf("auto-migrate %T: %w", mdl, err)
		}
	}
	return nil
}
