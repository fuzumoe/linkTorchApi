package repository_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/fuzumoe/linkTorch-api/internal/model"
	"github.com/fuzumoe/linkTorch-api/internal/repository"
	"github.com/fuzumoe/linkTorch-api/tests/utils"
)

func TestMigrate_MySQLIntegration(t *testing.T) {

	// Get a clean database state.
	db := utils.SetupTest(t)

	t.Run("Migrate", func(t *testing.T) {
		// Test: Run migrations.
		err := repository.Migrate(db)
		assert.NoError(t, err, "migrations should run without error")

		// Verify: Each model's table exists.
		migrator := db.Migrator()
		for _, m := range model.AllModels {
			exists := migrator.HasTable(m)
			assert.Truef(t, exists, "table for model %T should exist after migration", m)
		}

		// Test: Migrations are idempotent.
		err = repository.Migrate(db)
		assert.NoError(t, err, "migrations should be idempotent")
	})

	utils.CleanTestData(t)

}
