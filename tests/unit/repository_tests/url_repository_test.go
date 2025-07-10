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

// setupMockDB initializes a GORM DB backed by sqlmock.
func setupMockDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)

	gormDB, err := gorm.Open(mysql.New(mysql.Config{
		Conn:                      db,
		SkipInitializeWithVersion: true,
	}), &gorm.Config{})
	require.NoError(t, err)

	return gormDB, mock
}

func TestURLRepo(t *testing.T) {
	t.Run("Create", func(t *testing.T) {
		db, mock := setupMockDB(t)
		repo := repository.NewURLRepo(db)
		testURL := &model.URL{
			UserID:      42,
			OriginalURL: "https://example.com",
			// Default status is applied by GORM/default value in the model
		}

		mock.ExpectBegin()
		// INSERT SQL must match what GORM generates.
		exec := mock.ExpectExec(regexp.QuoteMeta(
			"INSERT INTO `urls` (`user_id`,`original_url`,`status`,`created_at`,`updated_at`,`deleted_at`) VALUES (?,?,?,?,?,?)",
		))
		exec.WithArgs(
			testURL.UserID,
			testURL.OriginalURL,
			"queued",         // assuming default status is "queued"
			sqlmock.AnyArg(), // created_at
			sqlmock.AnyArg(), // updated_at
			sqlmock.AnyArg(), // deleted_at (usually nil)
		).WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()

		err := repo.Create(testURL)
		assert.NoError(t, err)
		// Assuming that GORM assigns the returned id.
		assert.Equal(t, uint(1), testURL.ID)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("FindByID_Success", func(t *testing.T) {
		db, mock := setupMockDB(t)
		repo := repository.NewURLRepo(db)
		id := uint(7)

		// Main SELECT query
		exprMain := mock.ExpectQuery(regexp.QuoteMeta(
			"SELECT * FROM `urls` WHERE `urls`.`id` = ? AND `urls`.`deleted_at` IS NULL ORDER BY `urls`.`id` LIMIT ?",
		))
		exprMain.WithArgs(id, 1).WillReturnRows(
			sqlmock.NewRows([]string{"id", "user_id", "original_url", "status", "created_at", "updated_at", "deleted_at"}).
				AddRow(id, 42, "https://u.test", "queued", time.Date(2025, 7, 10, 0, 0, 0, 0, time.UTC),
					time.Date(2025, 7, 10, 0, 0, 0, 0, time.UTC), nil),
		)

		// Preload AnalysisResults (if any)
		mock.ExpectQuery(regexp.QuoteMeta(
			"SELECT * FROM `analysis_results` WHERE `analysis_results`.`url_id` = ? AND `analysis_results`.`deleted_at` IS NULL",
		)).WithArgs(id).WillReturnRows(sqlmock.NewRows([]string{"id"}))

		// Preload Links (if any)
		mock.ExpectQuery(regexp.QuoteMeta(
			"SELECT * FROM `links` WHERE `links`.`url_id` = ? AND `links`.`deleted_at` IS NULL",
		)).WithArgs(id).WillReturnRows(sqlmock.NewRows([]string{"id"}))

		u, err := repo.FindByID(id)
		assert.NoError(t, err)
		assert.Equal(t, id, u.ID)
		assert.Equal(t, uint(42), u.UserID)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("FindByID_NotFound", func(t *testing.T) {
		db, mock := setupMockDB(t)
		repo := repository.NewURLRepo(db)
		id := uint(999)

		mock.ExpectQuery(regexp.QuoteMeta(
			"SELECT * FROM `urls` WHERE `urls`.`id` = ? AND `urls`.`deleted_at` IS NULL ORDER BY `urls`.`id` LIMIT ?",
		)).WithArgs(id, 1).WillReturnError(gorm.ErrRecordNotFound)

		_, err := repo.FindByID(id)
		assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("ListByUser", func(t *testing.T) {
		db, mock := setupMockDB(t)
		repo := repository.NewURLRepo(db)
		userID := uint(5)

		// Adjusted expected SQL to include the soft-delete clause and limit.
		mock.ExpectQuery(regexp.QuoteMeta(
			"SELECT * FROM `urls` WHERE user_id = ? AND `urls`.`deleted_at` IS NULL LIMIT ?",
		)).WithArgs(userID, 10).WillReturnRows(
			sqlmock.NewRows([]string{"id", "user_id", "original_url", "status", "created_at", "updated_at", "deleted_at"}).
				AddRow(1, userID, "url1", "queued", time.Date(2025, 7, 10, 0, 0, 0, 0, time.UTC),
					time.Date(2025, 7, 10, 0, 0, 0, 0, time.UTC), nil).
				AddRow(2, userID, "url2", "queued", time.Date(2025, 7, 10, 1, 0, 0, 0, time.UTC),
					time.Date(2025, 7, 10, 1, 0, 0, 0, time.UTC), nil),
		)

		urls, err := repo.ListByUser(userID, repository.Pagination{Page: 1, PageSize: 10})
		assert.NoError(t, err)
		assert.Len(t, urls, 2)
		assert.Equal(t, "url1", urls[0].OriginalURL)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Update", func(t *testing.T) {
		db, mock := setupMockDB(t)
		repo := repository.NewURLRepo(db)
		testURL := &model.URL{
			ID:          3,
			UserID:      1,
			OriginalURL: "old",
			Status:      "queued",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
		// Modify URL for update.
		testURL.OriginalURL = "new"

		mock.ExpectBegin()
		// GORM generates an UPDATE statement; adjust pattern as needed.
		mock.ExpectExec(regexp.QuoteMeta(
			"UPDATE `urls` SET `user_id`=?,`original_url`=?,`status`=?,`created_at`=?,`updated_at`=?,`deleted_at`=? WHERE `urls`.`deleted_at` IS NULL AND `id` = ?",
		)).WithArgs(
			testURL.UserID, testURL.OriginalURL, testURL.Status,
			testURL.CreatedAt, sqlmock.AnyArg(), nil, testURL.ID,
		).WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()

		err := repo.Update(testURL)
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Delete_Success", func(t *testing.T) {
		db, mock := setupMockDB(t)
		repo := repository.NewURLRepo(db)

		// For soft delete, GORM updates the deleted_at column.
		mock.ExpectBegin()
		mock.ExpectExec(regexp.QuoteMeta(
			"UPDATE `urls` SET `deleted_at`=? WHERE `urls`.`id` = ? AND `urls`.`deleted_at` IS NULL",
		)).WithArgs(sqlmock.AnyArg(), 4).WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()

		err := repo.Delete(4)
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Delete_NotFound", func(t *testing.T) {
		db, mock := setupMockDB(t)
		repo := repository.NewURLRepo(db)

		// For record not found, rows affected is 0.
		mock.ExpectBegin()
		mock.ExpectExec(regexp.QuoteMeta(
			"UPDATE `urls` SET `deleted_at`=? WHERE `urls`.`id` = ? AND `urls`.`deleted_at` IS NULL",
		)).WithArgs(sqlmock.AnyArg(), 999).WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectCommit()

		err := repo.Delete(999)
		assert.EqualError(t, err, "url not found")
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}
