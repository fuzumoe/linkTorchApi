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

	"github.com/fuzumoe/urlinsight-backend/internal/model"
	"github.com/fuzumoe/urlinsight-backend/internal/repository"
)

// setupBlMockDB initializes a GORM DB backed by sqlmock.
func setupBlMockDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)

	gormDB, err := gorm.Open(mysql.New(mysql.Config{
		Conn:                      db,
		SkipInitializeWithVersion: true,
	}), &gorm.Config{})
	require.NoError(t, err)

	return gormDB, mock
}

func TestBlacklistedTokenRepo(t *testing.T) {
	t.Run("Create", func(t *testing.T) {
		db, mock := setupBlMockDB(t)
		repo := repository.NewBlacklistedTokenRepo(db)
		expiryTime := time.Now().Add(24 * time.Hour)
		testToken := &model.BlacklistedToken{
			JTI:       "test-jwt-id-123",
			ExpiresAt: expiryTime,
		}

		mock.ExpectBegin()
		// Fix: Match the actual fields GORM is using
		exec := mock.ExpectExec(regexp.QuoteMeta(
			"INSERT INTO `blacklisted_tokens` (`jti`,`expires_at`,`created_at`,`deleted_at`) VALUES (?,?,?,?)",
		))
		exec.WithArgs(
			testToken.JTI,
			testToken.ExpiresAt,
			sqlmock.AnyArg(),
			sqlmock.AnyArg(),
		)
		exec.WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()

		err := repo.Create(testToken)
		assert.NoError(t, err)
		assert.Equal(t, uint(1), testToken.ID)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Exists Found", func(t *testing.T) {
		db, mock := setupBlMockDB(t)
		repo := repository.NewBlacklistedTokenRepo(db)
		jti := "existing-jwt-id"

		mock.ExpectQuery(regexp.QuoteMeta(
			"SELECT count(*) FROM `blacklisted_tokens` WHERE jti = ? AND `blacklisted_tokens`.`deleted_at` IS NULL",
		)).WithArgs(jti).WillReturnRows(
			sqlmock.NewRows([]string{"count(*)"}).AddRow(1),
		)

		exists, err := repo.Exists(jti)
		assert.NoError(t, err)
		assert.True(t, exists)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Exists NotFound", func(t *testing.T) {
		db, mock := setupBlMockDB(t)
		repo := repository.NewBlacklistedTokenRepo(db)
		jti := "non-existing-jwt-id"

		mock.ExpectQuery(regexp.QuoteMeta(
			"SELECT count(*) FROM `blacklisted_tokens` WHERE jti = ? AND `blacklisted_tokens`.`deleted_at` IS NULL",
		)).WithArgs(jti).WillReturnRows(
			sqlmock.NewRows([]string{"count(*)"}).AddRow(0),
		)

		exists, err := repo.Exists(jti)
		assert.NoError(t, err)
		assert.False(t, exists)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Exists Error", func(t *testing.T) {
		db, mock := setupBlMockDB(t)
		repo := repository.NewBlacklistedTokenRepo(db)
		jti := "error-jwt-id"

		mock.ExpectQuery(regexp.QuoteMeta(
			"SELECT count(*) FROM `blacklisted_tokens` WHERE jti = ? AND `blacklisted_tokens`.`deleted_at` IS NULL",
		)).WithArgs(jti).WillReturnError(gorm.ErrInvalidDB)

		exists, err := repo.Exists(jti)
		assert.Error(t, err)
		assert.False(t, exists)
		assert.Equal(t, gorm.ErrInvalidDB, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Delete Expired Success", func(t *testing.T) {
		db, mock := setupBlMockDB(t)
		repo := repository.NewBlacklistedTokenRepo(db)

		mock.ExpectBegin()
		// Fix: Match the soft delete that GORM is using
		mock.ExpectExec(regexp.QuoteMeta(
			"UPDATE `blacklisted_tokens` SET `deleted_at`=? WHERE expires_at < NOW() AND `blacklisted_tokens`.`deleted_at` IS NULL",
		)).WithArgs(
			sqlmock.AnyArg(),
		).WillReturnResult(sqlmock.NewResult(0, 5)) // 5 rows deleted
		mock.ExpectCommit()

		err := repo.DeleteExpired()
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Delete Expired NoExpiredTokens", func(t *testing.T) {
		db, mock := setupBlMockDB(t)
		repo := repository.NewBlacklistedTokenRepo(db)

		mock.ExpectBegin()
		// Fix: Match the soft delete that GORM is using
		mock.ExpectExec(regexp.QuoteMeta(
			"UPDATE `blacklisted_tokens` SET `deleted_at`=? WHERE expires_at < NOW() AND `blacklisted_tokens`.`deleted_at` IS NULL",
		)).WithArgs(
			sqlmock.AnyArg(),
		).WillReturnResult(sqlmock.NewResult(0, 0)) // 0 rows deleted
		mock.ExpectCommit()

		err := repo.DeleteExpired()
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Delete Expired Error", func(t *testing.T) {
		db, mock := setupBlMockDB(t)
		repo := repository.NewBlacklistedTokenRepo(db)

		mock.ExpectBegin()
		// Fix: Match the soft delete that GORM is using
		mock.ExpectExec(regexp.QuoteMeta(
			"UPDATE `blacklisted_tokens` SET `deleted_at`=? WHERE expires_at < NOW() AND `blacklisted_tokens`.`deleted_at` IS NULL",
		)).WithArgs(
			sqlmock.AnyArg(),
		).WillReturnError(gorm.ErrInvalidTransaction)
		mock.ExpectRollback()

		err := repo.DeleteExpired()
		assert.Error(t, err)
		assert.Equal(t, gorm.ErrInvalidTransaction, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}
