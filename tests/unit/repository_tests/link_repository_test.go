package repository_test

import (
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/fuzumoe/urlinsight-backend/internal/model"
	"github.com/fuzumoe/urlinsight-backend/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// setupLinkMockDB initializes a new GORM DB instance backed by sqlmock.
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

func TestLinkRepo_Create(t *testing.T) {
	db, mock := setupLinkMockDB(t)
	repo := repository.NewLinkRepo(db)

	// Prepare a test Link with all fields.
	testLink := &model.Link{
		URLID:      10,
		Href:       "https://example.com",
		IsExternal: false,
		StatusCode: 0,
	}

	mock.ExpectBegin()
	// Expect INSERT statement with all columns.
	exec := mock.ExpectExec(regexp.QuoteMeta(
		"INSERT INTO `links` (`url_id`,`href`,`is_external`,`status_code`,`created_at`,`updated_at`,`deleted_at`) VALUES (?,?,?,?,?,?,?)",
	))
	exec.WithArgs(
		testLink.URLID,
		testLink.Href,
		testLink.IsExternal,
		testLink.StatusCode,
		sqlmock.AnyArg(), // created_at
		sqlmock.AnyArg(), // updated_at
		sqlmock.AnyArg(), // deleted_at, likely nil
	).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	err := repo.Create(testLink)
	assert.NoError(t, err)
	// Check that GORM assigned ID 1.
	assert.Equal(t, uint(1), testLink.ID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestLinkRepo_ListByURL(t *testing.T) {
	db, mock := setupLinkMockDB(t)
	repo := repository.NewLinkRepo(db)
	urlID := uint(10)

	mock.ExpectQuery(regexp.QuoteMeta(
		"SELECT * FROM `links` WHERE url_id = ? AND `links`.`deleted_at` IS NULL LIMIT ?",
	)).WithArgs(urlID, 10).WillReturnRows(
		sqlmock.NewRows([]string{"id", "url_id", "href", "is_external", "status_code", "created_at", "updated_at", "deleted_at"}).
			AddRow(1, urlID, "https://link1.com", false, 200,
				time.Date(2025, 7, 10, 12, 0, 0, 0, time.UTC),
				time.Date(2025, 7, 10, 12, 0, 0, 0, time.UTC), nil).
			AddRow(2, urlID, "https://link2.com", true, 404,
				time.Date(2025, 7, 10, 13, 0, 0, 0, time.UTC),
				time.Date(2025, 7, 10, 13, 0, 0, 0, time.UTC), nil),
	)

	links, err := repo.ListByURL(urlID, repository.Pagination{Page: 1, PageSize: 10})
	assert.NoError(t, err)
	assert.Len(t, links, 2)
	// Verify returned record.
	assert.Equal(t, "https://link1.com", links[0].Href)
	assert.Equal(t, false, links[0].IsExternal)
	assert.Equal(t, 200, links[0].StatusCode)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestLinkRepo_Update(t *testing.T) {
	db, mock := setupLinkMockDB(t)
	repo := repository.NewLinkRepo(db)
	testLink := &model.Link{
		ID:         3,
		URLID:      10,
		Href:       "https://old.com",
		IsExternal: false,
		StatusCode: 0,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	// Update the Href and mark as external.
	testLink.Href = "https://new.com"
	testLink.IsExternal = true

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(
		"UPDATE `links` SET `url_id`=?,`href`=?,`is_external`=?,`status_code`=?,`created_at`=?,`updated_at`=?,`deleted_at`=? WHERE `links`.`deleted_at` IS NULL AND `id` = ?",
	)).WithArgs(
		testLink.URLID, testLink.Href, testLink.IsExternal, testLink.StatusCode,
		testLink.CreatedAt, sqlmock.AnyArg(), nil, testLink.ID,
	).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	err := repo.Update(testLink)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestLinkRepo_Delete_Success(t *testing.T) {
	db, mock := setupLinkMockDB(t)
	repo := repository.NewLinkRepo(db)

	// Create a dummy link to delete.
	testLink := &model.Link{ID: 4}
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(
		"UPDATE `links` SET `deleted_at`=? WHERE `links`.`id` = ? AND `links`.`deleted_at` IS NULL",
	)).WithArgs(sqlmock.AnyArg(), testLink.ID).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	err := repo.Delete(testLink)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestLinkRepo_Delete_NotFound(t *testing.T) {
	db, mock := setupLinkMockDB(t)
	repo := repository.NewLinkRepo(db)

	// Dummy link for non-existent record.
	testLink := &model.Link{ID: 999}
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(
		"UPDATE `links` SET `deleted_at`=? WHERE `links`.`id` = ? AND `links`.`deleted_at` IS NULL",
	)).WithArgs(sqlmock.AnyArg(), testLink.ID).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectCommit()

	err := repo.Delete(testLink)
	// Expect error "link not found" when no rows are affected.
	assert.EqualError(t, err, "link not found")
	assert.NoError(t, mock.ExpectationsWereMet())
}
