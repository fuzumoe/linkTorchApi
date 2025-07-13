package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fuzumoe/urlinsight-backend/internal/analyzer"
	"github.com/fuzumoe/urlinsight-backend/internal/crawler"
	"github.com/fuzumoe/urlinsight-backend/internal/model"
	"github.com/fuzumoe/urlinsight-backend/internal/repository"
	"github.com/fuzumoe/urlinsight-backend/internal/service"
	"github.com/fuzumoe/urlinsight-backend/tests/utils"
)

func TestURLService_Integration(t *testing.T) {
	// Setup test database.
	db := utils.SetupTest(t)
	defer utils.CleanTestData(t)

	// Setup test environment
	var (
		testUser      *model.User
		urlService    service.URLService
		crawlerCtx    context.Context
		cancelCrawler context.CancelFunc
	)

	t.Run("Setup", func(t *testing.T) {
		// Create repositories.
		userRepo := repository.NewUserRepo(db)
		urlRepo := repository.NewURLRepo(db)

		// Create a HTML analyzer for testing
		htmlAnalyzer := analyzer.NewHTMLAnalyzer()

		// Create a crawler pool with context for proper shutdown
		crawlerCtx, cancelCrawler = context.WithCancel(context.Background())
		crawlerPool := crawler.New(urlRepo, htmlAnalyzer, 1, 5, 1*time.Second)

		// Start the crawler pool in a goroutine (it now blocks until context is cancelled)
		go crawlerPool.Start(crawlerCtx)

		// Create URLService with the crawler pool
		urlService = service.NewURLService(urlRepo, crawlerPool)

		// Create a test user.
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

	// Ensure crawler is shutdown when test completes
	defer func() {
		if cancelCrawler != nil {
			cancelCrawler()
		}
	}()

	t.Run("Create and Get", func(t *testing.T) {
		// Create a URL through URLService.
		createInput := &model.CreateURLInputDTO{
			UserID:      testUser.ID,
			OriginalURL: "https://example.com",
		}
		createdID, err := urlService.Create(createInput)
		require.NoError(t, err, "Should create URL without error.")
		require.NotZero(t, createdID, "Created URL ID should be set.")

		// Get the URL back.
		urlDTO, err := urlService.Get(createdID)
		require.NoError(t, err, "Should get URL without error.")
		assert.Equal(t, "https://example.com", urlDTO.OriginalURL, "OriginalURL should match the input.")
	})

	t.Run("List", func(t *testing.T) {
		// Create several URLs for the test user.
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

		// List URLs with pagination.
		pagination := repository.Pagination{
			Page:     1,
			PageSize: 10,
		}
		urlList, err := urlService.List(testUser.ID, pagination)
		require.NoError(t, err, "Should list URLs without error.")
		assert.GreaterOrEqual(t, len(urlList), 3, "Should return at least 3 URLs.")
	})

	t.Run("Update", func(t *testing.T) {
		// Create a URL to update.
		createInput := &model.CreateURLInputDTO{
			UserID:      testUser.ID,
			OriginalURL: "https://example.com/old",
		}
		createdID, err := urlService.Create(createInput)
		require.NoError(t, err, "Should create URL without error.")

		// Update the URL.
		updateInput := &model.UpdateURLInput{
			OriginalURL: "https://example.com/new",
			Status:      model.StatusRunning, // allowed status value.
		}
		err = urlService.Update(createdID, updateInput)
		require.NoError(t, err, "Should update URL without error.")

		// Verify the update.
		updatedDTO, err := urlService.Get(createdID)
		require.NoError(t, err, "Should get URL without error.")
		assert.Equal(t, "https://example.com/new", updatedDTO.OriginalURL, "OriginalURL should be updated.")
		assert.Equal(t, model.StatusRunning, updatedDTO.Status, "Status should be updated to running.")
	})

	t.Run("Delete", func(t *testing.T) {
		// Create a URL to delete.
		createInput := &model.CreateURLInputDTO{
			UserID:      testUser.ID,
			OriginalURL: "https://example.com/delete",
		}
		createdID, err := urlService.Create(createInput)
		require.NoError(t, err, "Should create URL without error.")
		require.NotZero(t, createdID, "Created URL ID should be set.")

		// Delete the URL.
		err = urlService.Delete(createdID)
		require.NoError(t, err, "Should delete URL without error.")

		// Attempt to get the deleted URL.
		_, err = urlService.Get(createdID)
		assert.Error(t, err, "Getting a deleted URL should return an error.")
	})

	t.Run("Start", func(t *testing.T) {
		// Create a URL to start crawling
		createInput := &model.CreateURLInputDTO{
			UserID:      testUser.ID,
			OriginalURL: "https://example.com/start",
		}
		createdID, err := urlService.Create(createInput)
		require.NoError(t, err, "Should create URL without error.")

		// Start crawling the URL
		err = urlService.Start(createdID)
		require.NoError(t, err, "Should start crawling without error.")

		// Verify the URL status is updated to queued
		urlDTO, err := urlService.Get(createdID)
		require.NoError(t, err, "Should get URL without error.")
		assert.Equal(t, model.StatusQueued, urlDTO.Status, "Status should be queued after starting.")
	})

	t.Run("Stop", func(t *testing.T) {
		// Create a URL to stop crawling
		createInput := &model.CreateURLInputDTO{
			UserID:      testUser.ID,
			OriginalURL: "https://example.com/stop",
		}
		createdID, err := urlService.Create(createInput)
		require.NoError(t, err, "Should create URL without error.")

		// First set status to running
		updateInput := &model.UpdateURLInput{
			Status: model.StatusRunning,
		}
		err = urlService.Update(createdID, updateInput)
		require.NoError(t, err, "Should update URL status to running without error.")

		// Now stop crawling the URL
		err = urlService.Stop(createdID)
		require.NoError(t, err, "Should stop crawling without error.")

		// Verify the URL status is updated to error
		urlDTO, err := urlService.Get(createdID)
		require.NoError(t, err, "Should get URL without error.")

		// URLService.Stop now sets status to 'error' as per the implementation
		assert.Equal(t, model.StatusError, urlDTO.Status,
			"Status should be set to 'error' when stopping a URL via the service.")
	})

	t.Run("Results", func(t *testing.T) {
		// Create a URL for testing results
		createInput := &model.CreateURLInputDTO{
			UserID:      testUser.ID,
			OriginalURL: "https://example.com/results",
		}
		createdID, err := urlService.Create(createInput)
		require.NoError(t, err, "Should create URL without error.")

		// Call Results method - should work even without actual analysis results
		resultsDTO, err := urlService.Results(createdID)
		require.NoError(t, err, "Should get results without error")
		assert.NotNil(t, resultsDTO, "Results DTO should not be nil")
		assert.Equal(t, "https://example.com/results", resultsDTO.OriginalURL,
			"Results should contain the original URL")
	})

	t.Run("ResultsWithDetails", func(t *testing.T) {
		// Create a URL for testing detailed results
		createInput := &model.CreateURLInputDTO{
			UserID:      testUser.ID,
			OriginalURL: "https://example.com/details",
		}
		createdID, err := urlService.Create(createInput)
		require.NoError(t, err, "Should create URL without error.")

		// Call ResultsWithDetails method
		url, analysisResults, links, err := urlService.ResultsWithDetails(createdID)
		require.NoError(t, err, "Should get detailed results without error")

		// Verify URL object
		assert.NotNil(t, url, "URL object should not be nil")
		assert.Equal(t, "https://example.com/details", url.OriginalURL,
			"URL should contain the original URL")
		assert.Equal(t, createdID, url.ID, "URL ID should match created ID")

		// In the current implementation, these collections might be nil if no analysis has been performed
		// That's okay - we just need to check that the call works and returns the URL properly
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
		// Create a URL first.
		createInput := &model.CreateURLInputDTO{
			UserID:      testUser.ID,
			OriginalURL: "https://example.com/error",
		}
		createdID, err := urlService.Create(createInput)
		require.NoError(t, err, "Should create URL without error.")

		t.Run("InvalidStatus", func(t *testing.T) {
			// Try to update with an invalid status value.
			updateInput := &model.UpdateURLInput{
				OriginalURL: "https://example.com/error-updated",
				Status:      "invalid_status",
			}
			err = urlService.Update(createdID, updateInput)
			assert.Error(t, err, "Updating with an invalid status should return an error.")
		})

		t.Run("NonExistentURL", func(t *testing.T) {
			// Try to start a URL that doesn't exist
			err = urlService.Start(9999)
			assert.Error(t, err, "Starting a non-existent URL should return an error.")
			assert.Contains(t, err.Error(), "cannot start crawling",
				"Error message should indicate the start operation failed")

			// Try to stop a URL that doesn't exist
			err = urlService.Stop(9999)
			assert.Error(t, err, "Stopping a non-existent URL should return an error.")
			assert.Contains(t, err.Error(), "cannot stop crawling",
				"Error message should indicate the stop operation failed")

			// Try to get results for a URL that doesn't exist
			_, err = urlService.Results(9999)
			assert.Error(t, err, "Getting results for a non-existent URL should return an error")

			// Try to get detailed results for a URL that doesn't exist
			_, _, _, err = urlService.ResultsWithDetails(9999)
			assert.Error(t, err, "Getting detailed results for a non-existent URL should return an error")
			assert.Contains(t, err.Error(), "failed to get detailed URL results",
				"Error message should indicate the operation failed")
		})
	})
}
