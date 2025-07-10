package repository_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/fuzumoe/urlinsight-backend/internal/model"
	"github.com/fuzumoe/urlinsight-backend/internal/repository"
	"github.com/fuzumoe/urlinsight-backend/tests/integration"
)

// TestMigrate_MySQLIntegration tests the migration process against a real MySQL database.
func TestMigrate_MySQLIntegration(t *testing.T) {
	dsn := os.Getenv("TEST_MYSQL_DSN")
	if dsn == "" {
		dsn = "urlinsight_user:secret@tcp(localhost:3309)/urlinsight_test?parseTime=true"
		t.Log("TEST_MYSQL_DSN not set, using fallback DSN:", dsn)
	}

	if err := integration.InitTestSuite(); err != nil {
		println("Failed to setup test suite:", err.Error())
		os.Exit(1)
	}

	// Ensure the test database is available; panic if not.
	integration.CheckDBAvailability()

	// Setup: Get clean database
	db := integration.SetupTest(t)

	// Get the Migrator from *gorm.DB (this implements the repository.Migrator interface)
	migrator := db.Migrator()

	// Test: Run migrations using the migrator
	err := repository.Migrate(migrator)
	assert.NoError(t, err, "migrations should run without error")

	// Verify: Each model's table exists
	for _, mdl := range model.AllModels {
		exists := migrator.HasTable(mdl)
		assert.Truef(t, exists, "table for model %T should exist after migration", mdl)
	}

	// Test: Migrations are idempotent
	err = repository.Migrate(migrator)
	assert.NoError(t, err, "migrations should be idempotent")
}
