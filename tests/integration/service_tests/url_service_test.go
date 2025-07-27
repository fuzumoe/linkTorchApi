package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fuzumoe/linkTorch-api/internal/analyzer"
	"github.com/fuzumoe/linkTorch-api/internal/crawler"
	"github.com/fuzumoe/linkTorch-api/internal/model"
	"github.com/fuzumoe/linkTorch-api/internal/repository"
	"github.com/fuzumoe/linkTorch-api/internal/service"
	"github.com/fuzumoe/linkTorch-api/tests/utils"
)

func TestURLService_Integration(t *testing.T) {

	db := utils.SetupTest(t)
	defer utils.CleanTestData(t)

	var (
		testUser      *model.User
		urlService    service.URLService
		crawlerCtx    context.Context
		cancelCrawler context.CancelFunc
	)

	t.Run("Setup", func(t *testing.T) {

		userRepo := repository.NewUserRepo(db)
		urlRepo := repository.NewURLRepo(db)

		htmlAnalyzer := analyzer.NewHTMLAnalyzer()

		crawlerCtx, cancelCrawler = context.WithCancel(context.Background())
		crawlerPool := crawler.New(urlRepo, htmlAnalyzer, 1, 5, 1*time.Second)

		go crawlerPool.Start(crawlerCtx)

		urlService = service.NewURLService(urlRepo, crawlerPool)

		testUser = &model.User{
			Username:  "testuser",
			Email:     "test@example.com",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		err := userRepo.Create(testUser)
		require.NoError(t, err, "Should create test user without error.")
		require.NotZero(t, testUser.ID, "User ID should be set after creation.")
	})

	defer func() {
		if cancelCrawler != nil {
			cancelCrawler()
		}
	}()

	t.Run("Create and Get", func(t *testing.T) {

		createInput := &model.CreateURLInputDTO{
			UserID:      testUser.ID,
			OriginalURL: "https://example.com",
		}
		createdID, err := urlService.Create(createInput)
		require.NoError(t, err, "Should create URL without error.")
		require.NotZero(t, createdID, "Created URL ID should be set.")

		urlDTO, err := urlService.Get(createdID)
		require.NoError(t, err, "Should get URL without error.")
		assert.Equal(t, "https://example.com", urlDTO.OriginalURL, "OriginalURL should match the input.")
	})

	t.Run("List", func(t *testing.T) {
		urlsToCreate := []string{
			"https://example.com/1",
			"https://example.com/2",
			"https://example.com/3",
		}
		for _, orig := range urlsToCreate {
			input := &model.CreateURLInputDTO{
				UserID:      testUser.ID,
				OriginalURL: orig,
			}
			_, err := urlService.Create(input)
			require.NoError(t, err, "Should create URL without error.")
		}

		pagination := repository.Pagination{
			Page:     1,
			PageSize: 10,
		}

		paginatedResult, err := urlService.List(testUser.ID, pagination)
		require.NoError(t, err, "Should list URLs without error.")

		assert.GreaterOrEqual(t, len(paginatedResult.Data), 3, "Should return at least 3 URLs.")

		assert.Equal(t, 1, paginatedResult.Pagination.Page, "Page should be 1")
		assert.Equal(t, 10, paginatedResult.Pagination.PageSize, "PageSize should be 10")
		assert.GreaterOrEqual(t, paginatedResult.Pagination.TotalItems, 3, "TotalItems should be at least 3")
		assert.GreaterOrEqual(t, paginatedResult.Pagination.TotalPages, 1, "Should have at least 1 page")

		foundURLs := 0
		for _, dto := range paginatedResult.Data {
			for _, orig := range urlsToCreate {
				if dto.OriginalURL == orig {
					foundURLs++
					break
				}
			}
		}
		assert.GreaterOrEqual(t, foundURLs, 3, "Should find at least 3 of the URLs we created")
	})

	t.Run("Update", func(t *testing.T) {

		createInput := &model.CreateURLInputDTO{
			UserID:      testUser.ID,
			OriginalURL: "https://example.com/old",
		}
		createdID, err := urlService.Create(createInput)
		require.NoError(t, err, "Should create URL without error.")

		updateInput := &model.UpdateURLInput{
			OriginalURL: "https://example.com/new",
			Status:      model.StatusRunning,
		}
		err = urlService.Update(createdID, updateInput)
		require.NoError(t, err, "Should update URL without error.")

		updatedDTO, err := urlService.Get(createdID)
		require.NoError(t, err, "Should get URL without error.")
		assert.Equal(t, "https://example.com/new", updatedDTO.OriginalURL, "OriginalURL should be updated.")
		assert.Equal(t, model.StatusRunning, updatedDTO.Status, "Status should be updated to running.")
	})

	t.Run("Delete", func(t *testing.T) {
		createInput := &model.CreateURLInputDTO{
			UserID:      testUser.ID,
			OriginalURL: "https://example.com/delete",
		}
		createdID, err := urlService.Create(createInput)
		require.NoError(t, err, "Should create URL without error.")
		require.NotZero(t, createdID, "Created URL ID should be set.")

		err = urlService.Delete(createdID)
		require.NoError(t, err, "Should delete URL without error.")

		_, err = urlService.Get(createdID)
		assert.Error(t, err, "Getting a deleted URL should return an error.")
	})

	t.Run("Start", func(t *testing.T) {
		createInput := &model.CreateURLInputDTO{
			UserID:      testUser.ID,
			OriginalURL: "https://example.com/start",
		}
		createdID, err := urlService.Create(createInput)
		require.NoError(t, err, "Should create URL without error.")

		err = urlService.Start(createdID)
		require.NoError(t, err, "Should start crawling without error.")

		urlDTO, err := urlService.Get(createdID)
		require.NoError(t, err, "Should get URL without error.")
		assert.Equal(t, model.StatusQueued, urlDTO.Status, "Status should be queued after starting.")
	})

	t.Run("Stop", func(t *testing.T) {

		createInput := &model.CreateURLInputDTO{
			UserID:      testUser.ID,
			OriginalURL: "https://example.com/stop",
		}
		createdID, err := urlService.Create(createInput)
		require.NoError(t, err, "Should create URL without error.")

		updateInput := &model.UpdateURLInput{
			Status: model.StatusRunning,
		}
		err = urlService.Update(createdID, updateInput)
		require.NoError(t, err, "Should update URL status to running without error.")

		err = urlService.Stop(createdID)
		require.NoError(t, err, "Should stop crawling without error.")

		urlDTO, err := urlService.Get(createdID)
		require.NoError(t, err, "Should get URL without error.")

		assert.Equal(t, model.StatusError, urlDTO.Status,
			"Status should be set to 'error' when stopping a URL via the service.")
	})

	t.Run("Results", func(t *testing.T) {

		createInput := &model.CreateURLInputDTO{
			UserID:      testUser.ID,
			OriginalURL: "https://example.com/results",
		}
		createdID, err := urlService.Create(createInput)
		require.NoError(t, err, "Should create URL without error.")

		resultsDTO, err := urlService.Results(createdID)
		require.NoError(t, err, "Should get results without error")
		assert.NotNil(t, resultsDTO, "Results DTO should not be nil")
		assert.Equal(t, "https://example.com/results", resultsDTO.OriginalURL,
			"Results should contain the original URL")
	})

	t.Run("ResultsWithDetails", func(t *testing.T) {

		createInput := &model.CreateURLInputDTO{
			UserID:      testUser.ID,
			OriginalURL: "https://example.com/details",
		}
		createdID, err := urlService.Create(createInput)
		require.NoError(t, err, "Should create URL without error.")

		url, analysisResults, links, err := urlService.ResultsWithDetails(createdID)
		require.NoError(t, err, "Should get detailed results without error")

		assert.NotNil(t, url, "URL object should not be nil")
		assert.Equal(t, "https://example.com/details", url.OriginalURL,
			"URL should contain the original URL")
		assert.Equal(t, createdID, url.ID, "URL ID should match created ID")

		if analysisResults != nil {
			assert.IsType(t, []*model.AnalysisResult{}, analysisResults,
				"Analysis results should be of the correct type when not nil")
		}

		if links != nil {
			assert.IsType(t, []*model.Link{}, links,
				"Links should be of the correct type when not nil")
		}
	})

	t.Run("ErrorCases", func(t *testing.T) {

		createInput := &model.CreateURLInputDTO{
			UserID:      testUser.ID,
			OriginalURL: "https://example.com/error",
		}
		createdID, err := urlService.Create(createInput)
		require.NoError(t, err, "Should create URL without error.")

		t.Run("InvalidStatus", func(t *testing.T) {

			updateInput := &model.UpdateURLInput{
				OriginalURL: "https://example.com/error-updated",
				Status:      "invalid_status",
			}
			err = urlService.Update(createdID, updateInput)
			assert.Error(t, err, "Updating with an invalid status should return an error.")
		})

		t.Run("NonExistentURL", func(t *testing.T) {
			err = urlService.Start(9999)
			assert.Error(t, err, "Starting a non-existent URL should return an error.")
			assert.Contains(t, err.Error(), "cannot start crawling",
				"Error message should indicate the start operation failed")

			err = urlService.Stop(9999)
			assert.Error(t, err, "Stopping a non-existent URL should return an error.")
			assert.Contains(t, err.Error(), "cannot stop crawling",
				"Error message should indicate the stop operation failed")

			_, err = urlService.Results(9999)
			assert.Error(t, err, "Getting results for a non-existent URL should return an error")

			_, _, _, err = urlService.ResultsWithDetails(9999)
			assert.Error(t, err, "Getting detailed results for a non-existent URL should return an error")
			assert.Contains(t, err.Error(), "failed to get detailed URL results",
				"Error message should indicate the operation failed")
		})
	})
}
