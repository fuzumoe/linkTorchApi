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

	"github.com/fuzumoe/linkTorch-api/internal/repository"
)

// NewMockDB creates a new mock GORM DB instance.
func NewMockDB() (*MockDB, error) {
	db, mock, err := sqlmock.New()
	if err != nil {
		return nil, err
	}
	// Mock the version query.
	mock.ExpectQuery("SELECT VERSION()").WillReturnRows(
		sqlmock.NewRows([]string{"version"}).AddRow("8.0.0"))

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

// MockDB aggregates the SQL and GORM DB objects.
type MockDB struct {
	DB     *sql.DB
	Mock   sqlmock.Sqlmock
	GormDB *gorm.DB
}

func TestNewDB(t *testing.T) {
	t.Run("Valid DSN", func(t *testing.T) {
		// Create SQL mock
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		// Set expectation for the version query.
		mock.ExpectQuery("SELECT VERSION()").
			WillReturnRows(sqlmock.NewRows([]string{"version"}).AddRow("8.0.0"))

		// Create a GORM DB using the mock connection.
		dialector := mysql.New(mysql.Config{Conn: db})
		gormDB, err := gorm.Open(dialector, &gorm.Config{})
		require.NoError(t, err)
		assert.NotNil(t, gormDB)

		// Verify all expectations were met.
		require.NoError(t, mock.ExpectationsWereMet(), "there were unfulfilled expectations")
	})

	t.Run("Malformed DSN", func(t *testing.T) {
		// An invalid DSN formats.
		malformedDSN := "user@password:host:port/dbname?"
		_, err := repository.NewDB(malformedDSN)
		assert.Error(t, err, "expected an error with malformed DSN")
	})

	t.Run("Connection Refused", func(t *testing.T) {
		refusedDSN := "root:password@tcp(localhost:65535)/nonexistent?parseTime=true"
		_, err := repository.NewDB(refusedDSN)
		assert.Error(t, err, "expected connection refused error")
		// The error message should indicate connection issues (may vary by environment).
		assert.Contains(t, strings.ToLower(err.Error()), "connect", "error should indicate connection issue")
	})

	t.Run("Connection Pool Settings", func(t *testing.T) {
		mockDB, err := NewMockDB()
		if err != nil {
			t.Skip("Could not create mock DB: " + err.Error())
		}
		defer mockDB.DB.Close()

		// Running a simple query to ensure connection works.
		rows, err := mockDB.DB.Query("SELECT VERSION()")
		if err == nil {
			rows.Close()
		}

		sqlDB, err := mockDB.GormDB.DB()
		require.NoError(t, err, "Failed to get sql.DB from GORM DB")
		assert.NotNil(t, sqlDB, "sql.DB should not be nil")
	})

	t.Run("Close Connection", func(t *testing.T) {
		// Create a new mock without any expectations.
		db, mock, err := sqlmock.New()
		require.NoError(t, err, "Failed to create SQL mock")

		// Explicitly expect Close.
		mock.ExpectClose()

		// Close the connection.
		err = db.Close()
		assert.NoError(t, err, "closing the connection should succeed")

		// Operations after closing should fail.
		err = db.Ping()
		assert.Error(t, err, "ping after close should fail")
	})
}
