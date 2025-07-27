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
			Href:       "https://linked-site.com",
			IsExternal: true,
			StatusCode: 200,
		}

		mock.ExpectBegin()
		mock.ExpectExec(regexp.QuoteMeta(
			"INSERT INTO `links` (`url_id`,`href`,`is_external`,`status_code`,`created_at`,`updated_at`,`deleted_at`) VALUES (?,?,?,?,?,?,?)",
		)).WithArgs(
			testLink.URLID,
			testLink.Href,
			testLink.IsExternal,
			testLink.StatusCode,
			sqlmock.AnyArg(), // created_at
			sqlmock.AnyArg(), // updated_at
			sqlmock.AnyArg(), // deleted_at (nil)
		).WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()

		err := repo.Create(testLink)
		assert.NoError(t, err)
		assert.Equal(t, uint(1), testLink.ID)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("ListByURL", func(t *testing.T) {
		db, mock := setupLinkMockDB(t)
		repo := repository.NewLinkRepo(db)
		urlID := uint(42)
		pagination := repository.Pagination{Page: 1, PageSize: 10}

		rows := sqlmock.NewRows([]string{"id", "url_id", "href", "is_external", "status_code", "created_at", "updated_at", "deleted_at"}).
			AddRow(1, urlID, "https://example1.com", true, 200, time.Now(), time.Now(), nil).
			AddRow(2, urlID, "https://example2.com", false, 301, time.Now(), time.Now(), nil)

		mock.ExpectQuery(regexp.QuoteMeta(
			"SELECT * FROM `links` WHERE url_id = ? AND `links`.`deleted_at` IS NULL LIMIT ?",
		)).WithArgs(urlID, pagination.Limit()).WillReturnRows(rows)

		links, err := repo.ListByURL(urlID, pagination)
		assert.NoError(t, err)
		assert.Len(t, links, 2)
		assert.Equal(t, "https://example1.com", links[0].Href)
		assert.Equal(t, "https://example2.com", links[1].Href)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("ListByURL_WithPagination", func(t *testing.T) {
		db, mock := setupLinkMockDB(t)
		repo := repository.NewLinkRepo(db)
		urlID := uint(42)
		pagination := repository.Pagination{Page: 2, PageSize: 3}

		rows := sqlmock.NewRows([]string{"id", "url_id", "href", "is_external", "status_code", "created_at", "updated_at", "deleted_at"}).
			AddRow(4, urlID, "https://example4.com", true, 200, time.Now(), time.Now(), nil).
			AddRow(5, urlID, "https://example5.com", false, 301, time.Now(), time.Now(), nil).
			AddRow(6, urlID, "https://example6.com", true, 200, time.Now(), time.Now(), nil)

		mock.ExpectQuery(regexp.QuoteMeta(
			"SELECT * FROM `links` WHERE url_id = ? AND `links`.`deleted_at` IS NULL LIMIT ? OFFSET ?",
		)).WithArgs(urlID, pagination.Limit(), pagination.Offset()).WillReturnRows(rows)

		links, err := repo.ListByURL(urlID, pagination)
		assert.NoError(t, err)
		assert.Len(t, links, 3)
		assert.Equal(t, "https://example4.com", links[0].Href)
		assert.Equal(t, "https://example5.com", links[1].Href)
		assert.Equal(t, "https://example6.com", links[2].Href)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Update", func(t *testing.T) {
		db, mock := setupLinkMockDB(t)
		repo := repository.NewLinkRepo(db)
		testLink := &model.Link{
			ID:         1,
			URLID:      42,
			Href:       "https://updated-link.com",
			IsExternal: false,
			StatusCode: 302,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}

		mock.ExpectBegin()
		mock.ExpectExec(regexp.QuoteMeta(
			"UPDATE `links` SET `url_id`=?,`href`=?,`is_external`=?,`status_code`=?,`created_at`=?,`updated_at`=?,`deleted_at`=? WHERE `links`.`deleted_at` IS NULL AND `id` = ?",
		)).WithArgs(
			testLink.URLID,
			testLink.Href,
			testLink.IsExternal,
			testLink.StatusCode,
			testLink.CreatedAt,
			sqlmock.AnyArg(), // updated_at will be updated
			nil,              // deleted_at is nil
			testLink.ID,
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
			ID:    3,
			URLID: 42,
		}

		mock.ExpectBegin()
		mock.ExpectExec(regexp.QuoteMeta(
			"UPDATE `links` SET `deleted_at`=? WHERE `links`.`id` = ? AND `links`.`deleted_at` IS NULL",
		)).WithArgs(
			sqlmock.AnyArg(), // deleted_at timestamp
			testLink.ID,
		).WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()

		err := repo.Delete(testLink)
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("CountByURL", func(t *testing.T) {
		db, mock := setupLinkMockDB(t)
		repo := repository.NewLinkRepo(db)
		urlID := uint(42)

		rows := sqlmock.NewRows([]string{"count(*)"}).
			AddRow(5)

		mock.ExpectQuery(regexp.QuoteMeta(
			"SELECT count(*) FROM `links` WHERE url_id = ? AND `links`.`deleted_at` IS NULL",
		)).WithArgs(urlID).WillReturnRows(rows)

		count, err := repo.CountByURL(urlID)
		assert.NoError(t, err)
		assert.Equal(t, 5, count)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("CountByURL_Error", func(t *testing.T) {
		db, mock := setupLinkMockDB(t)
		repo := repository.NewLinkRepo(db)
		urlID := uint(42)

		mock.ExpectQuery(regexp.QuoteMeta(
			"SELECT count(*) FROM `links` WHERE url_id = ? AND `links`.`deleted_at` IS NULL",
		)).WithArgs(urlID).WillReturnError(gorm.ErrInvalidDB)

		count, err := repo.CountByURL(urlID)
		assert.Error(t, err)
		assert.Equal(t, 0, count)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}
