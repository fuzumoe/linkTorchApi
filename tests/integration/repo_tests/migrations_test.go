package repository_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/fuzumoe/urlinsight-backend/internal/model"
	"github.com/fuzumoe/urlinsight-backend/internal/repository"
	"github.com/fuzumoe/urlinsight-backend/tests/integration" // <- Add this import
)

// TestMain handles setup and teardown for all integration tests
func TestMain(m *testing.M) {
	// Use the setup functions
	if err := integration.InitTestSuite(); err != nil {
		println("Failed to setup test suite:", err.Error())
		os.Exit(1)
	}

	code := m.Run()

	integration.CleanupTestSuite()
	os.Exit(code)
}

func TestMigrate_MySQLIntegration(t *testing.T) {
	integration.SkipIfDBUnavailable(t)

	// Setup: Get clean database
	db := integration.SetupTest(t)

	// Test: Run migrations
	err := repository.Migrate(db)
	assert.NoError(t, err, "migrations should run without error")

	// Verify: Each model's table exists
	migrator := db.Migrator()
	for _, m := range model.AllModels {
		exists := migrator.HasTable(m)
		assert.Truef(t, exists, "table for model %T should exist after migration", m)
	}

	// Test: Migrations are idempotent
	err = repository.Migrate(db)
	assert.NoError(t, err, "migrations should be idempotent")
}
