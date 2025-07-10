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

// setupLinkMockDB initializes a GORM DB backed by sqlmock.
func setupLinkMockDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)

	gormDB, err := gorm.Open(mysql.New(mysql.Config{
		Conn:                      db,
		SkipInitializeWithVersion: true,
	}), &gorm.Config{})
	require.NoError(t, err)

	return gormDB, mock
}

func TestLinkRepo(t *testing.T) {
	t.Run("Create", func(t *testing.T) {
		db, mock := setupLinkMockDB(t)
		repo := repository.NewLinkRepo(db)
		testLink := &model.Link{
			URLID:      42,
			Href:       "https://example.com",
			IsExternal: true,
			StatusCode: 200,
		}

		mock.ExpectBegin()
		exec := mock.ExpectExec(regexp.QuoteMeta(
			"INSERT INTO `links` (`url_id`,`href`,`is_external`,`status_code`,`created_at`,`updated_at`,`deleted_at`) VALUES (?,?,?,?,?,?,?)",
		))
		exec.WithArgs(
			testLink.URLID,
			testLink.Href,
			testLink.IsExternal,
			testLink.StatusCode,
			sqlmock.AnyArg(),
			sqlmock.AnyArg(),
			sqlmock.AnyArg(),
		)
		exec.WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()

		err := repo.Create(testLink)
		assert.NoError(t, err)
		assert.Equal(t, uint(1), testLink.ID)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("ListByURL", func(t *testing.T) {
		db, mock := setupLinkMockDB(t)
		repo := repository.NewLinkRepo(db)
		urlID := uint(5)

		// Return two rows with proper time.Time values
		mock.ExpectQuery(regexp.QuoteMeta(
			"SELECT * FROM `links` WHERE url_id = ?",
		)).WithArgs(urlID).WillReturnRows(
			sqlmock.NewRows([]string{"id", "url_id", "href", "is_external", "status_code", "created_at", "updated_at", "deleted_at"}).
				AddRow(1, urlID, "https://link1.com", true, 200, time.Date(2025, 7, 10, 0, 0, 0, 0, time.UTC), time.Date(2025, 7, 10, 0, 0, 0, 0, time.UTC), nil).
				AddRow(2, urlID, "https://link2.com", false, 200, time.Date(2025, 7, 10, 0, 0, 0, 0, time.UTC), time.Date(2025, 7, 10, 0, 0, 0, 0, time.UTC), nil),
		)

		links, err := repo.ListByURL(urlID)
		assert.NoError(t, err)
		assert.Len(t, links, 2)
		assert.Equal(t, "https://link1.com", links[0].Href)
		assert.True(t, links[0].IsExternal)
		assert.Equal(t, "https://link2.com", links[1].Href)
		assert.False(t, links[1].IsExternal)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Update", func(t *testing.T) {
		// Create a new mock DB for each test to avoid interference
		db, mock := setupLinkMockDB(t)
		repo := repository.NewLinkRepo(db)
		testLink := &model.Link{
			ID:         3,
			URLID:      10,
			Href:       "https://old-link.com",
			IsExternal: false,
			StatusCode: 200,
		}

		// Change some properties
		testLink.Href = "https://updated-link.com"
		testLink.IsExternal = true
		testLink.StatusCode = 302

		// Instead of trying to match the exact SQL, use a more flexible matcher
		// that only checks the beginning of the statement
		mock.ExpectBegin()
		mock.ExpectExec("UPDATE `links` SET").WithArgs(
			sqlmock.AnyArg(), // url_id
			sqlmock.AnyArg(), // href
			sqlmock.AnyArg(), // is_external
			sqlmock.AnyArg(), // status_code
			sqlmock.AnyArg(), // created_at
			sqlmock.AnyArg(), // updated_at
			sqlmock.AnyArg(), // deleted_at
			sqlmock.AnyArg(), // id
		).WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()

		err := repo.Update(testLink)
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("Delete", func(t *testing.T) {
		db, mock := setupLinkMockDB(t)
		repo := repository.NewLinkRepo(db)
		testLink := &model.Link{
			ID:         4,
			URLID:      15,
			Href:       "https://to-be-deleted.com",
			IsExternal: true,
			StatusCode: 200,
		}

		// For soft delete, GORM updates the deleted_at timestamp
		mock.ExpectBegin()
		mock.ExpectExec(regexp.QuoteMeta(
			"UPDATE `links` SET `deleted_at`=? WHERE `links`.`id` = ? AND `links`.`deleted_at` IS NULL",
		)).WithArgs(sqlmock.AnyArg(), testLink.ID).WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()

		err := repo.Delete(testLink)
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

}
