package repository_test

import (
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"github.com/fuzumoe/linkTorch-api/internal/model"
	"github.com/fuzumoe/linkTorch-api/internal/repository"
)

// setupTokenMockDB initializes a GORM DB backed by sqlmock.
func setupTokenMockDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)

	gormDB, err := gorm.Open(mysql.New(mysql.Config{
		Conn:                      db,
		SkipInitializeWithVersion: true,
	}), &gorm.Config{})
	require.NoError(t, err)

	return gormDB, mock
}

func TestTokenRepo(t *testing.T) {
	t.Run("Add", func(t *testing.T) {
		db, mock := setupTokenMockDB(t)
		repo := repository.NewTokenRepo(db)
		expiryTime := time.Now().Add(24 * time.Hour)
		testToken := &model.BlacklistedToken{
			JTI:       "test-jwt-id-123",
			ExpiresAt: expiryTime,
		}

		mock.ExpectBegin()
		// Update to match the actual SQL query (without ON DUPLICATE KEY)
		exec := mock.ExpectExec(regexp.QuoteMeta(
			"INSERT INTO `blacklisted_tokens` (`jti`,`expires_at`,`created_at`,`deleted_at`) VALUES (?,?,?,?)",
		))
		exec.WithArgs(
			testToken.JTI,
			testToken.ExpiresAt,
			sqlmock.AnyArg(), // CreatedAt will be set by GORM
			sqlmock.AnyArg(), // DeletedAt will be NULL
		)
		exec.WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()

		err := repo.Add(testToken)
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("IsBlacklisted Found", func(t *testing.T) {
		db, mock := setupTokenMockDB(t)
		repo := repository.NewTokenRepo(db)
		jti := "existing-jwt-id"

		// Update to match the actual SQL query (without expires_at > ?)
		mock.ExpectQuery(regexp.QuoteMeta(
			"SELECT count(*) FROM `blacklisted_tokens` WHERE jti = ? AND `blacklisted_tokens`.`deleted_at` IS NULL",
		)).WithArgs(jti).WillReturnRows(
			sqlmock.NewRows([]string{"count(*)"}).AddRow(1),
		)

		isBlacklisted, err := repo.IsBlacklisted(jti)
		assert.NoError(t, err)
		assert.True(t, isBlacklisted)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("IsBlacklisted NotFound", func(t *testing.T) {
		db, mock := setupTokenMockDB(t)
		repo := repository.NewTokenRepo(db)
		jti := "non-existing-jwt-id"

		// Update to match the actual SQL query (without expires_at > ?)
		mock.ExpectQuery(regexp.QuoteMeta(
			"SELECT count(*) FROM `blacklisted_tokens` WHERE jti = ? AND `blacklisted_tokens`.`deleted_at` IS NULL",
		)).WithArgs(jti).WillReturnRows(
			sqlmock.NewRows([]string{"count(*)"}).AddRow(0),
		)

		isBlacklisted, err := repo.IsBlacklisted(jti)
		assert.NoError(t, err)
		assert.False(t, isBlacklisted)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("IsBlacklisted Error", func(t *testing.T) {
		db, mock := setupTokenMockDB(t)
		repo := repository.NewTokenRepo(db)
		jti := "error-jwt-id"

		// Update to match the actual SQL query (without expires_at > ?)
		mock.ExpectQuery(regexp.QuoteMeta(
			"SELECT count(*) FROM `blacklisted_tokens` WHERE jti = ? AND `blacklisted_tokens`.`deleted_at` IS NULL",
		)).WithArgs(jti).WillReturnError(gorm.ErrInvalidDB)

		isBlacklisted, err := repo.IsBlacklisted(jti)
		assert.Error(t, err)
		assert.False(t, isBlacklisted)
		assert.Equal(t, gorm.ErrInvalidDB, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("RemoveExpired Success", func(t *testing.T) {
		db, mock := setupTokenMockDB(t)
		repo := repository.NewTokenRepo(db)

		mock.ExpectBegin()
		mock.ExpectExec(regexp.QuoteMeta(
			"UPDATE `blacklisted_tokens` SET `deleted_at`=? WHERE expires_at < ? AND `blacklisted_tokens`.`deleted_at` IS NULL",
		)).WithArgs(
			sqlmock.AnyArg(),
			sqlmock.AnyArg(),
		).WillReturnResult(sqlmock.NewResult(0, 5))
		mock.ExpectCommit()

		err := repo.RemoveExpired()
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("RemoveExpired NoExpiredTokens", func(t *testing.T) {
		db, mock := setupTokenMockDB(t)
		repo := repository.NewTokenRepo(db)

		mock.ExpectBegin()
		mock.ExpectExec(regexp.QuoteMeta(
			"UPDATE `blacklisted_tokens` SET `deleted_at`=? WHERE expires_at < ? AND `blacklisted_tokens`.`deleted_at` IS NULL",
		)).WithArgs(
			sqlmock.AnyArg(),
			sqlmock.AnyArg(),
		).WillReturnResult(sqlmock.NewResult(0, 0)) // 0 rows deleted
		mock.ExpectCommit()

		err := repo.RemoveExpired()
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("RemoveExpired Error", func(t *testing.T) {
		db, mock := setupTokenMockDB(t)
		repo := repository.NewTokenRepo(db)

		mock.ExpectBegin()
		mock.ExpectExec(regexp.QuoteMeta(
			"UPDATE `blacklisted_tokens` SET `deleted_at`=? WHERE expires_at < ? AND `blacklisted_tokens`.`deleted_at` IS NULL",
		)).WithArgs(
			sqlmock.AnyArg(),
			sqlmock.AnyArg(),
		).WillReturnError(gorm.ErrInvalidTransaction)
		mock.ExpectRollback()

		err := repo.RemoveExpired()
		assert.Error(t, err)
		assert.Equal(t, gorm.ErrInvalidTransaction, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Add with zero CreatedAt", func(t *testing.T) {
		db, mock := setupTokenMockDB(t)
		repo := repository.NewTokenRepo(db)
		testToken := &model.BlacklistedToken{
			JTI:       "test-jwt-id-123",
			ExpiresAt: time.Now().Add(24 * time.Hour),
			// CreatedAt intentionally omitted
		}

		// Set up the mock with the actual SQL query
		mock.ExpectBegin()
		mock.ExpectExec(regexp.QuoteMeta(
			"INSERT INTO `blacklisted_tokens` (`jti`,`expires_at`,`created_at`,`deleted_at`) VALUES (?,?,?,?)",
		)).WithArgs(
			testToken.JTI,
			testToken.ExpiresAt,
			sqlmock.AnyArg(), // CreatedAt will be set by GORM
			sqlmock.AnyArg(), // DeletedAt will be NULL
		).WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()

		err := repo.Add(testToken)
		assert.NoError(t, err)
		assert.False(t, testToken.CreatedAt.IsZero(), "CreatedAt should be set")
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}
