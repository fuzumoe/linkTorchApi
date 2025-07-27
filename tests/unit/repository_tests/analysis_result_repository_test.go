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

// setupAnaMockDB initializes a GORM DB backed by sqlmock.
func setupAnaMockDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)

	gormDB, err := gorm.Open(mysql.New(mysql.Config{
		Conn:                      db,
		SkipInitializeWithVersion: true,
	}), &gorm.Config{})
	require.NoError(t, err)

	return gormDB, mock
}

func TestAnalysisResultRepo(t *testing.T) {
	t.Run("Create", func(t *testing.T) {
		db, mock := setupAnaMockDB(t)
		repo := repository.NewAnalysisResultRepo(db)
		testResult := &model.AnalysisResult{
			URLID:        42,
			HTMLVersion:  "HTML5",
			Title:        "Test Page",
			H1Count:      2,
			H2Count:      5,
			H3Count:      3,
			H4Count:      0,
			H5Count:      0,
			H6Count:      0,
			HasLoginForm: true,
		}
		// For the Create method, we also pass an empty links slice.
		links := []model.Link{}

		mock.ExpectBegin()
		exec := mock.ExpectExec(regexp.QuoteMeta(
			"INSERT INTO `analysis_results` (`url_id`,`html_version`,`title`,`h1_count`,`h2_count`,`h3_count`,`h4_count`,`h5_count`,`h6_count`,`has_login_form`,`internal_link_count`,`external_link_count`,`broken_link_count`,`created_at`,`updated_at`,`deleted_at`) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)",
		))
		exec.WithArgs(
			testResult.URLID,
			testResult.HTMLVersion,
			testResult.Title,
			testResult.H1Count,
			testResult.H2Count,
			testResult.H3Count,
			testResult.H4Count,
			testResult.H5Count,
			testResult.H6Count,
			testResult.HasLoginForm,
			0,                // internal_link_count default
			0,                // external_link_count default
			0,                // broken_link_count default
			sqlmock.AnyArg(), // created_at
			sqlmock.AnyArg(), // updated_at
			sqlmock.AnyArg(), // deleted_at
		).WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()

		err := repo.Create(testResult, links)
		assert.NoError(t, err)
		// The Create method would update the model's ID on success.
		assert.Equal(t, uint(1), testResult.ID)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("ListByURL", func(t *testing.T) {
		db, mock := setupAnaMockDB(t)
		repo := repository.NewAnalysisResultRepo(db)
		urlID := uint(5)
		pagination := repository.Pagination{Page: 1, PageSize: 10}

		rows := sqlmock.NewRows([]string{
			"id", "url_id", "html_version", "title",
			"h1_count", "h2_count", "h3_count", "h4_count", "h5_count", "h6_count",
			"has_login_form", "created_at", "updated_at", "deleted_at",
		}).AddRow(
			1, urlID, "HTML5", "First Analysis", 2, 5, 3, 0, 0, 0, true,
			time.Date(2025, 7, 10, 0, 0, 0, 0, time.UTC),
			time.Date(2025, 7, 10, 0, 0, 0, 0, time.UTC),
			nil,
		).AddRow(
			2, urlID, "HTML4", "Second Analysis", 1, 3, 2, 1, 0, 0, false,
			time.Date(2025, 7, 11, 0, 0, 0, 0, time.UTC),
			time.Date(2025, 7, 11, 0, 0, 0, 0, time.UTC),
			nil,
		)

		// Query for first page should include an ORDER BY clause.
		mock.ExpectQuery(regexp.QuoteMeta(
			"SELECT * FROM `analysis_results` WHERE url_id = ? AND `analysis_results`.`deleted_at` IS NULL ORDER BY created_at DESC LIMIT ?",
		)).WithArgs(urlID, pagination.Limit()).WillReturnRows(rows)

		results, err := repo.ListByURL(urlID, pagination)
		assert.NoError(t, err)
		assert.Len(t, results, 2)
		assert.Equal(t, "HTML5", results[0].HTMLVersion)
		assert.Equal(t, "First Analysis", results[0].Title)
		assert.Equal(t, "HTML4", results[1].HTMLVersion)
		assert.Equal(t, "Second Analysis", results[1].Title)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("ListByURL_EmptyResult", func(t *testing.T) {
		db, mock := setupAnaMockDB(t)
		repo := repository.NewAnalysisResultRepo(db)
		urlID := uint(999)
		pagination := repository.Pagination{Page: 1, PageSize: 10}

		mock.ExpectQuery(regexp.QuoteMeta(
			"SELECT * FROM `analysis_results` WHERE url_id = ? AND `analysis_results`.`deleted_at` IS NULL ORDER BY created_at DESC LIMIT ?",
		)).WithArgs(urlID, pagination.Limit()).WillReturnRows(sqlmock.NewRows([]string{}))

		results, err := repo.ListByURL(urlID, pagination)
		assert.NoError(t, err)
		assert.Empty(t, results)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("ListByURL_WithPagination", func(t *testing.T) {
		db, mock := setupAnaMockDB(t)
		repo := repository.NewAnalysisResultRepo(db)
		urlID := uint(5)
		// For page 2 with PageSize = 2: Limit() = 2, Offset() = 2
		pagination := repository.Pagination{Page: 2, PageSize: 2}

		rows := sqlmock.NewRows([]string{
			"id", "url_id", "html_version", "title",
			"h1_count", "h2_count", "h3_count", "h4_count", "h5_count", "h6_count",
			"has_login_form", "created_at", "updated_at", "deleted_at",
		}).AddRow(
			3, urlID, "HTML5", "Third Analysis", 3, 6, 4, 0, 0, 0, true,
			time.Date(2025, 7, 12, 0, 0, 0, 0, time.UTC),
			time.Date(2025, 7, 12, 0, 0, 0, 0, time.UTC),
			nil,
		).AddRow(
			4, urlID, "HTML4", "Fourth Analysis", 1, 2, 1, 1, 0, 0, false,
			time.Date(2025, 7, 13, 0, 0, 0, 0, time.UTC),
			time.Date(2025, 7, 13, 0, 0, 0, 0, time.UTC),
			nil,
		)

		mock.ExpectQuery(regexp.QuoteMeta(
			"SELECT * FROM `analysis_results` WHERE url_id = ? AND `analysis_results`.`deleted_at` IS NULL ORDER BY created_at DESC LIMIT ? OFFSET ?",
		)).WithArgs(urlID, pagination.Limit(), pagination.Offset()).WillReturnRows(rows)

		results, err := repo.ListByURL(urlID, pagination)
		assert.NoError(t, err)
		assert.Len(t, results, 2)
		assert.Equal(t, "Third Analysis", results[0].Title)
		assert.Equal(t, "Fourth Analysis", results[1].Title)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}
