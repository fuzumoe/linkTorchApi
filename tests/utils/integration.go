package utils

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

	"github.com/fuzumoe/linkTorch-api/internal/model"
)

var (
	rootDSN    string
	testDBName string
	testDB     *gorm.DB
)

func InitTestSuite() error {

	if err := loadEnvFile(); err != nil {
		fmt.Printf("Warning: failed to load .env file: %v\n", err)
	}

	rootDSN = buildRootDSN()

	testDBName = os.Getenv("TEST_DATABASE")
	if testDBName == "" {
		testDBName = "linkTorch_test"
	}

	config := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
	}
	rootDB, err := gorm.Open(mysql.Open(rootDSN), config)
	if err != nil {
		return fmt.Errorf("failed to connect to MySQL root: %w", err)
	}

	if err := rootDB.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS `%s`", testDBName)).Error; err != nil {
		return fmt.Errorf("failed to drop existing test database: %w", err)
	}
	fmt.Printf("✓ Test database '%s' dropped successfully\n", testDBName)

	if err := rootDB.Exec(fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s`", testDBName)).Error; err != nil {
		return fmt.Errorf("failed to create test database: %w", err)
	}

	regularUser := getEnvOrDefault("DB_USER", "linkTorch_user")
	if err := rootDB.Exec(fmt.Sprintf("GRANT ALL PRIVILEGES ON `%s`.* TO '%s'@'%%'", testDBName, regularUser)).Error; err != nil {
		fmt.Printf("Warning: failed to grant permissions to %s: %v\n", regularUser, err)
	}
	if err := rootDB.Exec("FLUSH PRIVILEGES").Error; err != nil {
		fmt.Printf("Warning: failed to flush privileges: %v\n", err)
	}

	if sqlDB, err := rootDB.DB(); err == nil {
		sqlDB.Close()
	}

	testDSN := buildTestDSN(testDBName)
	testDB, err = gorm.Open(mysql.Open(testDSN), config)
	if err != nil {
		return fmt.Errorf("failed to connect to test database: %w", err)
	}

	fmt.Printf("Test database '%s' created and connected successfully\n", testDBName)
	return nil
}

func loadEnvFile() error {
	dir, err := os.Getwd()
	if err != nil {
		return err
	}

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

func buildRootDSN() string {
	host := getEnvOrDefault("DB_HOST", "localhost")
	port := getEnvOrDefault("DB_PORT", "3309")
	user := getEnvOrDefault("TEST_MYSQL_ROOT_USER", "root")
	password := getEnvOrDefault("TEST_MYSQL_ROOT_PASSWORD", getEnvOrDefault("MYSQL_ROOT_PASSWORD", "root_secret"))
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/?parseTime=true", user, password, host, port)
	fmt.Printf("Root DSN: %s\n", dsn)
	return dsn
}

func buildTestDSN(dbName string) string {
	host := getEnvOrDefault("DB_HOST", "localhost")
	port := getEnvOrDefault("DB_PORT", "3309")
	user := getEnvOrDefault("TEST_MYSQL_USER", getEnvOrDefault("DB_USER", "linkTorch_user"))
	password := getEnvOrDefault("TEST_MYSQL_PASSWORD", getEnvOrDefault("DB_PASSWORD", "secret"))
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true", user, password, host, port, dbName)
	fmt.Printf("Test DSN: %s\n", dsn)
	return dsn
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return strings.Trim(value, `"`)
	}
	return defaultValue
}

func SetupTest(t *testing.T) *gorm.DB {

	if err := InitTestSuite(); err != nil {
		println("Failed to setup test suite:", err.Error())
		os.Exit(1)
	}
	require.NotNil(t, testDB, "test database should be initialized")
	CleanTestData(t)
	return testDB
}

func SetupWithoutMigrations(t *testing.T) *gorm.DB {

	if err := InitTestSuite(); err != nil {
		println("Failed to setup test suite:", err.Error())
		os.Exit(1)
	}
	require.NotNil(t, testDB, "test database should be initialized")

	return testDB
}

func CleanTestData(t *testing.T) {

	err := testDB.Exec("SET FOREIGN_KEY_CHECKS = 0").Error
	require.NoError(t, err, "Failed to disable foreign key checks")

	models := []interface{}{
		&model.User{},
		&model.Link{},
		&model.AnalysisResult{},
		&model.URL{},
		&model.BlacklistedToken{},
	}

	for _, m := range models {
		if testDB.Migrator().HasTable(m) {
			err := testDB.Migrator().DropTable(m)
			require.NoError(t, err, "Failed to drop table for model %T", m)
		}
	}

	err = testDB.Exec("SET FOREIGN_KEY_CHECKS = 1").Error
	require.NoError(t, err, "Failed to re-enable foreign key checks")

	err = testDB.AutoMigrate(models...)
	require.NoError(t, err, "Failed to auto-migrate models")

	fmt.Println("✓ Tables recreated successfully")
}
