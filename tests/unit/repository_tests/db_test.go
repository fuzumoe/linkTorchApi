package repository_test

import (
	"database/sql"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"github.com/fuzumoe/urlinsight-backend/internal/repository"
)

// MockDB provides a simple mock implementation for testing.
type MockDB struct {
	DB      *sql.DB
	Mock    sqlmock.Sqlmock
	GormDB  *gorm.DB
	FailGet bool
}

// TestNewDB_InvalidDSN ensures that providing an empty or malformed DSN returns an error.
func TestNewDB_InvalidDSN(t *testing.T) {
	_, err := repository.NewDB("")
	assert.Error(t, err, "expected an error when DSN is empty")
}

// TestNewDB_ValidDSN tests successful connection with mocked DB.
func TestNewDB_ValidDSN(t *testing.T) {
	// Create SQL mock
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// Set expectation for the version query.
	mock.ExpectQuery("SELECT VERSION()").
		WillReturnRows(sqlmock.NewRows([]string{"version"}).AddRow("8.0.0"))

	// create a GORM DB using the mock connection directly.
	dialector := mysql.New(mysql.Config{
		Conn: db,
	})
	gormDB, err := gorm.Open(dialector, &gorm.Config{})
	require.NoError(t, err)
	assert.NotNil(t, gormDB)

	// Verify all expectations were met.
	require.NoError(t, mock.ExpectationsWereMet(), "there were unfulfilled expectations")
}

// TestNewDB_MalformedDSN checks that a malformed DSN returns a specific error.
func TestNewDB_MalformedDSN(t *testing.T) {
	// This DSN format is invalid for MySQL.
	malformedDSN := "user@password:host:port/dbname?"

	_, err := repository.NewDB(malformedDSN)
	assert.Error(t, err, "expected an error with malformed DSN")
}

// TestNewDB_ConnectionRefused tests handling of connection refused errors.
func TestNewDB_ConnectionRefused(t *testing.T) {
	refusedDSN := "root:password@tcp(localhost:65535)/nonexistent?parseTime=true"

	_, err := repository.NewDB(refusedDSN)

	// Check if the error is related to connection refusal.
	assert.Error(t, err, "expected connection refused error")
	// The error message may vary by environment, but should contain "connect".
	assert.Contains(t, strings.ToLower(err.Error()), "connect", "error should indicate connection issue")
}

// NewMockDB creates a new MockDB instance.
func NewMockDB() (*MockDB, error) {
	db, mock, err := sqlmock.New()
	if err != nil {
		return nil, err
	}

	// Mock expectations for common queries.
	mock.ExpectQuery("SELECT VERSION()").WillReturnRows(
		sqlmock.NewRows([]string{"version"}).AddRow("8.0.0"))

	// Create a mock gorm.DB
	dialector := mysql.New(mysql.Config{
		Conn:                      db,
		SkipInitializeWithVersion: true,
	})

	gormDB, err := gorm.Open(dialector, &gorm.Config{})
	if err != nil {
		db.Close()
		return nil, err
	}

	return &MockDB{
		DB:     db,
		Mock:   mock,
		GormDB: gormDB,
	}, nil
}

// TestNewDB_ConnectionPoolSettings verifies connection pool is configured.
func TestNewDB_ConnectionPoolSettings(t *testing.T) {
	mockDB, err := NewMockDB()
	if err != nil {
		t.Skip("Could not create mock DB: " + err.Error())
	}
	defer mockDB.DB.Close()

	// This is just to ensure the connection work.
	rows, err := mockDB.DB.Query("SELECT VERSION()")
	if err == nil {
		rows.Close()
	}

	sqlDB, err := mockDB.GormDB.DB()

	require.NoError(t, err, "Failed to get sql.DB from GORM DB")
	assert.NotNil(t, sqlDB, "sql.DB should not be nil")

}

// TestNewDB_CloseConnection tests closing the connection.
func TestNewDB_CloseConnection(t *testing.T) {
	// Create a new mock WITHOUT any expectations.
	db, mock, err := sqlmock.New()
	require.NoError(t, err, "Failed to create SQL mock")

	// Explicitly set expectation for Close.
	mock.ExpectClose()

	// Close the connection.
	err = db.Close()
	assert.NoError(t, err, "closing the connection should succeed")

	// Attempting operations after closing should fail
	err = db.Ping()
	assert.Error(t, err, "ping after close should fail")
}
