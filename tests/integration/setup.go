package integration

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var (
	rootDSN    string
	testDBName string
	testDB     *gorm.DB
)

// InitTestSuite initializes the test suite with database connection.
func InitTestSuite() error {
	// Load .env file from project root.
	if err := loadEnvFile(); err != nil {
		fmt.Printf("Warning: failed to load .env file: %v\n", err)
	}

	// Build root DSN from environment variables.
	rootDSN = buildRootDSN()

	// Get test database name
	testDBName = os.Getenv("TEST_MYSQL_DB_NAME")
	if testDBName == "" {
		testDBName = "urlinsight_test"
	}

	// Connect to MySQL without specifying database.
	config := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent), // Reduce noise in tests
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
	}

	rootDB, err := gorm.Open(mysql.Open(rootDSN), config)
	if err != nil {
		return fmt.Errorf("failed to connect to MySQL root: %w", err)
	}

	// Drop database if it exists.
	if err := rootDB.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS `%s`", testDBName)).Error; err != nil {
		return fmt.Errorf("failed to drop existing test database: %w", err)
	}

	// Create test database.
	if err := rootDB.Exec(fmt.Sprintf("CREATE DATABASE `%s`", testDBName)).Error; err != nil {
		return fmt.Errorf("failed to create test database: %w", err)
	}

	// Grant permissions to the regular user for the test database.
	regularUser := getEnvOrDefault("DB_USER", "urlinsight_user")
	if err := rootDB.Exec(fmt.Sprintf("GRANT ALL PRIVILEGES ON `%s`.* TO '%s'@'%%'", testDBName, regularUser)).Error; err != nil {
		fmt.Printf("Warning: failed to grant permissions to %s: %v\n", regularUser, err)
	}
	if err := rootDB.Exec("FLUSH PRIVILEGES").Error; err != nil {
		fmt.Printf("Warning: failed to flush privileges: %v\n", err)
	}

	// Close root connection.
	if sqlDB, err := rootDB.DB(); err == nil {
		sqlDB.Close()
	}

	// Connect to the test database using regular user.
	testDSN := buildTestDSN(testDBName)

	testDB, err = gorm.Open(mysql.Open(testDSN), config)
	if err != nil {
		return fmt.Errorf("failed to connect to test database: %w", err)
	}

	// Configure connection pool for testing.
	if sqlDB, err := testDB.DB(); err == nil {
		sqlDB.SetMaxOpenConns(10)
		sqlDB.SetMaxIdleConns(5)
		sqlDB.SetConnMaxLifetime(5 * time.Minute)

		// Test connection
		if err := sqlDB.Ping(); err != nil {
			return fmt.Errorf("failed to ping test database: %w", err)
		}
	}

	fmt.Printf("Test database '%s' created and connected successfully\n", testDBName)
	return nil
}

// loadEnvFile loads the .env file from the project root.
func loadEnvFile() error {
	// Try to find the .env file by looking up the directory tree.
	dir, err := os.Getwd()
	if err != nil {
		return err
	}

	// Look for .env file in current directory and parent directories.
	for {
		envPath := filepath.Join(dir, ".env")
		if _, err := os.Stat(envPath); err == nil {
			return godotenv.Load(envPath)
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return fmt.Errorf(".env file not found")
}

// buildRootDSN builds the root MySQL DSN from environment variables.
func buildRootDSN() string {
	// Get database configuration from environment.
	host := getEnvOrDefault("DB_HOST", "localhost")
	port := getEnvOrDefault("DB_PORT", "3309")

	// For root connection, always use root user and root password.
	user := getEnvOrDefault("TEST_MYSQL_ROOT_USER", "root")
	password := getEnvOrDefault("TEST_MYSQL_ROOT_PASSWORD", getEnvOrDefault("MYSQL_ROOT_PASSWORD", "root_secret"))

	// Build root DSN (without database name).
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/?parseTime=true", user, password, host, port)
	fmt.Printf("Root DSN: %s\n", dsn)
	return dsn
}

// buildTestDSN builds the test database DSN from environment variables.
func buildTestDSN(dbName string) string {
	// Get database configuration from environment
	host := getEnvOrDefault("DB_HOST", "localhost")
	port := getEnvOrDefault("DB_PORT", "3309")

	// For test connection, use the regular database user.
	user := getEnvOrDefault("TEST_MYSQL_USER", getEnvOrDefault("DB_USER", "urlinsight_user"))
	password := getEnvOrDefault("TEST_MYSQL_PASSWORD", getEnvOrDefault("DB_PASSWORD", "secret"))

	// Build test DSN (with database name).
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true", user, password, host, port, dbName)
	fmt.Printf("Test DSN: %s\n", dsn)
	return dsn
}

// getEnvOrDefault returns environment variable value or default.
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return strings.Trim(value, `"`)
	}
	return defaultValue
}

// CleanupTestSuite cleans up the test suite.
func CleanupTestSuite() {
	if testDB == nil {
		return
	}

	// Close test database connection first.
	if sqlDB, err := testDB.DB(); err == nil {
		sqlDB.Close()
	}

	// Connect to MySQL root to drop the test database.
	rootDB, err := gorm.Open(mysql.Open(rootDSN), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		fmt.Printf("Warning: failed to connect to MySQL root for cleanup: %v\n", err)
		return
	}
	defer func() {
		if sqlDB, err := rootDB.DB(); err == nil {
			sqlDB.Close()
		}
	}()

	// Drop the test database.
	if err := rootDB.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS `%s`", testDBName)).Error; err != nil {
		fmt.Printf("Warning: failed to drop test database: %v\n", err)
	} else {
		fmt.Printf("✓ Test database '%s' dropped successfully\n", testDBName)
	}
}

// SetupTest prepares a clean database state for each individual test.
func SetupTest(t *testing.T) *gorm.DB {
	require.NotNil(t, testDB, "test database should be initialized")

	// Clean any existing data.
	CleanTestData(t)

	return testDB
}

// CleanTestData removes all data from tables without dropping them.
func CleanTestData(t *testing.T) {
	if testDB == nil {
		return
	}

	// Disable foreign key checks temporarily.
	testDB.Exec("SET FOREIGN_KEY_CHECKS = 0")

	// Truncate all tables.
	tables := []string{
		"blacklisted_tokens",
		"links",
		"analysis_results",
		"urls",
		"users",
	}

	for _, table := range tables {
		if testDB.Migrator().HasTable(table) {
			result := testDB.Exec(fmt.Sprintf("TRUNCATE TABLE `%s`", table))
			if result.Error != nil {
				t.Logf("Warning: failed to truncate table %s: %v", table, result.Error)
			}
		}
	}

	// Re-enable foreign key checks.
	testDB.Exec("SET FOREIGN_KEY_CHECKS = 1")
}

// SkipIfDBUnavailable skips the test if database is not available.
func SkipIfDBUnavailable(t *testing.T) {
	if testDB == nil {
		t.Skip("Database not available for integration tests")
	}

	if sqlDB, err := testDB.DB(); err != nil || sqlDB.Ping() != nil {
		t.Skip("Database connection not available")
	}
}

// GetTestDB returns the test database instance.
func GetTestDB() *gorm.DB {
	return testDB
}
