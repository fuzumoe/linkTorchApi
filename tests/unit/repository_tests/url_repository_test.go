package repository_test

import (
	"errors"
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
		}

		mock.ExpectBegin()
		exec := mock.ExpectExec(regexp.QuoteMeta(
			"INSERT INTO `urls` (`user_id`,`original_url`,`status`,`created_at`,`updated_at`,`deleted_at`) VALUES (?,?,?,?,?,?)",
		))
		exec.WithArgs(
			testURL.UserID,
			testURL.OriginalURL,
			"queued",
			sqlmock.AnyArg(),
			sqlmock.AnyArg(),
			sqlmock.AnyArg(),
		).WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()

		err := repo.Create(testURL)
		assert.NoError(t, err)
		assert.Equal(t, uint(1), testURL.ID)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("FindByID_Success", func(t *testing.T) {
		db, mock := setupMockDB(t)
		repo := repository.NewURLRepo(db)
		id := uint(7)

		exprMain := mock.ExpectQuery(regexp.QuoteMeta(
			"SELECT * FROM `urls` WHERE `urls`.`id` = ? AND `urls`.`deleted_at` IS NULL ORDER BY `urls`.`id` LIMIT ?",
		))
		exprMain.WithArgs(id, 1).WillReturnRows(
			sqlmock.NewRows([]string{"id", "user_id", "original_url", "status", "created_at", "updated_at", "deleted_at"}).
				AddRow(id, 42, "https://u.test", "queued",
					time.Date(2025, 7, 10, 0, 0, 0, 0, time.UTC),
					time.Date(2025, 7, 10, 0, 0, 0, 0, time.UTC),
					nil),
		)

		mock.ExpectQuery(regexp.QuoteMeta(
			"SELECT * FROM `analysis_results` WHERE `analysis_results`.`url_id` = ? AND `analysis_results`.`deleted_at` IS NULL",
		)).WithArgs(id).WillReturnRows(sqlmock.NewRows([]string{"id"}))
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
		pagination := repository.Pagination{Page: 1, PageSize: 10}

		mock.ExpectQuery(regexp.QuoteMeta(
			"SELECT * FROM `urls` WHERE user_id = ? AND `urls`.`deleted_at` IS NULL LIMIT ?",
		)).WithArgs(userID, pagination.Limit()).WillReturnRows(
			sqlmock.NewRows([]string{"id", "user_id", "original_url", "status", "created_at", "updated_at", "deleted_at"}).
				AddRow(1, userID, "url1", "queued",
					time.Date(2025, 7, 10, 0, 0, 0, 0, time.UTC),
					time.Date(2025, 7, 10, 0, 0, 0, 0, time.UTC), nil).
				AddRow(2, userID, "url2", "queued",
					time.Date(2025, 7, 10, 1, 0, 0, 0, time.UTC),
					time.Date(2025, 7, 10, 1, 0, 0, 0, time.UTC), nil),
		)

		urls, err := repo.ListByUser(userID, pagination)
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
		testURL.OriginalURL = "new"

		mock.ExpectBegin()
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

		mock.ExpectBegin()
		mock.ExpectExec(regexp.QuoteMeta(
			"UPDATE `urls` SET `deleted_at`=? WHERE `urls`.`id` = ? AND `urls`.`deleted_at` IS NULL",
		)).WithArgs(sqlmock.AnyArg(), 999).WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectCommit()

		err := repo.Delete(999)
		assert.EqualError(t, err, "url not found")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("UpdateStatus", func(t *testing.T) {
		db, mock := setupMockDB(t)
		repo := repository.NewURLRepo(db)
		id := uint(10)
		newStatus := "completed"

		mock.ExpectBegin()
		mock.ExpectExec(regexp.QuoteMeta(
			"UPDATE `urls` SET `status`=?,`updated_at`=? WHERE id = ? AND `urls`.`deleted_at` IS NULL",
		)).WithArgs(newStatus, sqlmock.AnyArg(), id).WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()

		err := repo.UpdateStatus(id, newStatus)
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("SaveResults", func(t *testing.T) {
		db, mock := setupMockDB(t)
		repo := repository.NewURLRepo(db)
		urlID := uint(20)
		analysisRes := &model.AnalysisResult{
			HTMLVersion:  "HTML5",
			Title:        "Analysis Title",
			H1Count:      2,
			H2Count:      3,
			H3Count:      0,
			H4Count:      0,
			H5Count:      0,
			H6Count:      0,
			HasLoginForm: true,
		}
		links := []model.Link{
			{Href: "https://example.com/link1"},
			{Href: "https://example.com/link2"},
		}

		mock.ExpectBegin()
		exec := mock.ExpectExec(regexp.QuoteMeta(
			"INSERT INTO `analysis_results` (`url_id`,`html_version`,`title`,`h1_count`,`h2_count`,`h3_count`,`h4_count`,`h5_count`,`h6_count`,`has_login_form`,`internal_link_count`,`external_link_count`,`broken_link_count`,`created_at`,`updated_at`,`deleted_at`) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)",
		))
		exec.WithArgs(
			urlID,
			analysisRes.HTMLVersion,
			analysisRes.Title,
			analysisRes.H1Count,
			analysisRes.H2Count,
			analysisRes.H3Count,
			analysisRes.H4Count,
			analysisRes.H5Count,
			analysisRes.H6Count,
			analysisRes.HasLoginForm,
			0,
			0,
			0,
			sqlmock.AnyArg(),
			sqlmock.AnyArg(),
			sqlmock.AnyArg(),
		).WillReturnResult(sqlmock.NewResult(30, 1))

		mock.ExpectExec(regexp.QuoteMeta(
			"INSERT INTO `links` (`url_id`,`href`,`is_external`,`status_code`,`created_at`,`updated_at`,`deleted_at`) VALUES (?,?,?,?,?,?,?),(?,?,?,?,?,?,?)",
		)).WithArgs(
			urlID, links[0].Href, false, 0, sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(),
			urlID, links[1].Href, false, 0, sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(),
		).WillReturnResult(sqlmock.NewResult(100, 2))
		mock.ExpectCommit()

		err := repo.SaveResults(urlID, analysisRes, links)
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Results", func(t *testing.T) {
		db, mock := setupMockDB(t)
		repo := repository.NewURLRepo(db)
		id := uint(15)
		mock.ExpectQuery(regexp.QuoteMeta(
			"SELECT * FROM `urls` WHERE `urls`.`id` = ? AND `urls`.`deleted_at` IS NULL ORDER BY `urls`.`id` LIMIT ?",
		)).WithArgs(id, 1).WillReturnRows(
			sqlmock.NewRows([]string{"id", "user_id", "original_url", "status", "created_at", "updated_at", "deleted_at"}).
				AddRow(id, 99, "https://results.test", "completed",
					time.Date(2025, 7, 11, 0, 0, 0, 0, time.UTC),
					time.Date(2025, 7, 11, 0, 0, 0, 0, time.UTC), nil),
		)
		mock.ExpectQuery(regexp.QuoteMeta(
			"SELECT * FROM `analysis_results` WHERE `analysis_results`.`url_id` = ? AND `analysis_results`.`deleted_at` IS NULL",
		)).WithArgs(id).WillReturnRows(sqlmock.NewRows([]string{"id"}))
		mock.ExpectQuery(regexp.QuoteMeta(
			"SELECT * FROM `links` WHERE `links`.`url_id` = ? AND `links`.`deleted_at` IS NULL",
		)).WithArgs(id).WillReturnRows(sqlmock.NewRows([]string{"id"}))

		u, err := repo.Results(id)
		assert.NoError(t, err)
		assert.Equal(t, id, u.ID)
		assert.Equal(t, uint(99), u.UserID)
		assert.Equal(t, "https://results.test", u.OriginalURL)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("ResultsWithDetails", func(t *testing.T) {
		db, mock := setupMockDB(t)
		repo := repository.NewURLRepo(db)
		id := uint(15)

		expectedJSON := `{"url":{"id":15,"user_id":99,"original_url":"https://results.test","status":"completed","created_at":"2025-07-11T00:00:00.000000Z","updated_at":"2025-07-11T00:00:00.000000Z"},"analysis_results":[],"links":[]}`

		mock.ExpectQuery(regexp.QuoteMeta(
			`SELECT
  JSON_OBJECT(
    'url',
      JSON_OBJECT(
        'id',           u.id,
        'user_id',      u.user_id,
        'original_url', u.original_url,
        'status',       u.status,
        'created_at',   DATE_FORMAT(u.created_at, '%Y-%m-%dT%H:%i:%s.%fZ'),
        'updated_at',   DATE_FORMAT(u.updated_at, '%Y-%m-%dT%H:%i:%s.%fZ')
      ),
    'analysis_results',
      (
        SELECT JSON_ARRAYAGG(
                 JSON_OBJECT(
                   'id',                  ar.id,
                   'url_id',              ar.url_id,
                   'html_version',        ar.html_version,
                   'title',               ar.title,
                   'h1_count',            ar.h1_count,
                   'h2_count',            ar.h2_count,
                   'h3_count',            ar.h3_count,
                   'h4_count',            ar.h4_count,
                   'h5_count',            ar.h5_count,
                   'h6_count',            ar.h6_count,
                   'has_login_form',      IF(ar.has_login_form = 1, CAST('true' AS JSON), CAST('false' AS JSON)),
                   'internal_link_count', ar.internal_link_count,
                   'external_link_count', ar.external_link_count,
                   'broken_link_count',   ar.broken_link_count,
                   'created_at',          DATE_FORMAT(ar.created_at, '%Y-%m-%dT%H:%i:%s.%fZ'),
                   'updated_at',          DATE_FORMAT(ar.updated_at, '%Y-%m-%dT%H:%i:%s.%fZ')
                 )
               )
        FROM   analysis_results ar
        WHERE  ar.url_id = u.id
        ORDER BY ar.created_at DESC
      ),
    'links',
      (
        SELECT JSON_ARRAYAGG(
                 JSON_OBJECT(
                   'id',          l.id,
                   'url_id',      l.url_id,
                   'href',        l.href,
                   'is_external', IF(l.is_external = 1, CAST('true' AS JSON), CAST('false' AS JSON)),
                   'status_code', l.status_code
                 )
               )
        FROM   links l
        WHERE  l.url_id = u.id
      )
  ) AS result_document
FROM urls u
WHERE u.id = ?`)).
			WithArgs(id).
			WillReturnRows(sqlmock.NewRows([]string{"result_document"}).AddRow(expectedJSON))

		resultURL, resultAR, resultLinks, err := repo.ResultsWithDetails(id)
		assert.NoError(t, err)

		assert.Equal(t, uint(15), resultURL.ID)
		assert.Equal(t, uint(99), resultURL.UserID)
		assert.Equal(t, "https://results.test", resultURL.OriginalURL)
		assert.Equal(t, "completed", resultURL.Status)

		assert.Len(t, resultAR, 0)
		assert.Len(t, resultLinks, 0)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("CountByUser", func(t *testing.T) {
		db, mock := setupMockDB(t)
		repo := repository.NewURLRepo(db)
		userID := uint(5)

		mock.ExpectQuery(regexp.QuoteMeta(
			"SELECT count(*) FROM `urls` WHERE user_id = ? AND `urls`.`deleted_at` IS NULL",
		)).WithArgs(userID).WillReturnRows(
			sqlmock.NewRows([]string{"count(*)"}).AddRow(10),
		)

		count, err := repo.CountByUser(userID)

		assert.NoError(t, err)
		assert.Equal(t, 10, count)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("CountByUser_Error", func(t *testing.T) {
		db, mock := setupMockDB(t)
		repo := repository.NewURLRepo(db)
		userID := uint(5)

		expectedErr := errors.New("database error")
		mock.ExpectQuery(regexp.QuoteMeta(
			"SELECT count(*) FROM `urls` WHERE user_id = ? AND `urls`.`deleted_at` IS NULL",
		)).WithArgs(userID).WillReturnError(expectedErr)

		count, err := repo.CountByUser(userID)

		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.Equal(t, 0, count)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}
