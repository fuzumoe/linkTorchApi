package app_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fuzumoe/urlinsight-backend/internal/app"
	"github.com/fuzumoe/urlinsight-backend/tests/integration"
)

// TestMain sets up and tears down the test environment
func TestMain(m *testing.M) {
	// Initialize the test database
	if err := integration.InitTestSuite(); err != nil {
		println("Failed to setup test suite:", err.Error())
		os.Exit(1)
	}

	// Run tests
	code := m.Run()

	// Clean up
	integration.CleanupTestSuite()
	os.Exit(code)
}

// TestAppRun_Integration verifies that the app can connect to a real database and run migrations
func TestAppRun_Integration(t *testing.T) {
	// Ensure database is available
	integration.CheckDBAvailability()

	// Set the DATABASE_URL environment variable to point to our test database
	db := integration.GetTestDB()
	sqlDB, err := db.DB()
	require.NoError(t, err)

	// Get the DSN from the test database
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		// If not set, use a default based on test configuration
		dsn = os.Getenv("TEST_MYSQL_DSN")
		if dsn == "" {
			dsn = "urlinsight_user:secret@tcp(localhost:3309)/urlinsight_test?parseTime=true"
		}
		os.Setenv("DATABASE_URL", dsn)
	}

	// Close the existing connection to allow app.Run() to create its own
	sqlDB.Close()

	// Run the app which will load config, connect to DB, and run migrations
	err = app.Run()
	assert.NoError(t, err, "App should run successfully")

	// Additional assertion - try to run it again to ensure idempotency
	err = app.Run()
	assert.NoError(t, err, "App should be able to run again (idempotent)")

}
