package repository_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/fuzumoe/urlinsight-backend/internal/model"
	"github.com/fuzumoe/urlinsight-backend/internal/repository"
	"github.com/fuzumoe/urlinsight-backend/tests/utils"
)

func TestURLRepo_Integration(t *testing.T) {
	// Get a clean database state.
	db := utils.SetupTest(t)

	// Create repositories
	urlRepo := repository.NewURLRepo(db)
	userRepo := repository.NewUserRepo(db)

	// First create a test user.
	testUser := &model.User{
		Username: "urlowner",
		Email:    "urlowner@example.com",
		Password: "password123",
	}
	err := userRepo.Create(testUser)
	require.NoError(t, err, "Should create user without error")
	require.NotZero(t, testUser.ID, "User ID should be set")

	// Test URL data.
	testURL := &model.URL{
		UserID:      testUser.ID,
		OriginalURL: "https://example.com",
		Status:      "queued",
	}

	defaultPage := repository.Pagination{Page: 1, PageSize: 10}

	t.Run("Create", func(t *testing.T) {
		err := urlRepo.Create(testURL)
		require.NoError(t, err, "Should create URL without error")
		assert.NotZero(t, testURL.ID, "URL ID should be set after creation")
		assert.False(t, testURL.CreatedAt.IsZero(), "CreatedAt should be set")
		assert.False(t, testURL.UpdatedAt.IsZero(), "UpdatedAt should be set")
	})

	t.Run("FindByID", func(t *testing.T) {
		// Create related entities for preloading.
		analysisResult := &model.AnalysisResult{
			URLID:        testURL.ID,
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
		err := db.Create(analysisResult).Error
		require.NoError(t, err, "Should create analysis result")

		link := &model.Link{
			URLID:      testURL.ID,
			Href:       "https://linked-site.com",
			IsExternal: true,
			StatusCode: 200,
		}
		err = db.Create(link).Error
		require.NoError(t, err, "Should create link")

		// Now test FindByID with preloading.
		foundURL, err := urlRepo.FindByID(testURL.ID)
		require.NoError(t, err, "Should find URL by ID")

		// Verify basic URL fields.
		assert.Equal(t, testURL.ID, foundURL.ID)
		assert.Equal(t, testURL.UserID, foundURL.UserID)
		assert.Equal(t, testURL.OriginalURL, foundURL.OriginalURL)
		assert.Equal(t, testURL.Status, foundURL.Status)

		// Verify preloaded relations.
		require.NotEmpty(t, foundURL.AnalysisResults, "AnalysisResults should be preloaded")
		assert.Equal(t, "Test Page", foundURL.AnalysisResults[0].Title)
		assert.Equal(t, "HTML5", foundURL.AnalysisResults[0].HTMLVersion)
		assert.Equal(t, 2, foundURL.AnalysisResults[0].H1Count)
		assert.True(t, foundURL.AnalysisResults[0].HasLoginForm)

		require.NotEmpty(t, foundURL.Links, "Links should be preloaded")
		assert.Equal(t, "https://linked-site.com", foundURL.Links[0].Href)
		assert.True(t, foundURL.Links[0].IsExternal)
		assert.Equal(t, 200, foundURL.Links[0].StatusCode)

		// Test not found case.
		_, err = urlRepo.FindByID(9999)
		assert.ErrorIs(t, err, gorm.ErrRecordNotFound, "Should return record not found for non-existent ID")
	})

	t.Run("ListByUser", func(t *testing.T) {
		// Create another URL for the same user.
		secondURL := &model.URL{
			UserID:      testUser.ID,
			OriginalURL: "https://another-example.com",
			Status:      "queued",
		}
		err := urlRepo.Create(secondURL)
		require.NoError(t, err, "Should create second URL")

		// Create a URL for a different user.
		anotherUser := &model.User{
			Username: "anotheruser",
			Email:    "another@example.com",
			Password: "anotherpassword",
		}
		err = userRepo.Create(anotherUser)
		require.NoError(t, err, "Should create another user")

		otherUserURL := &model.URL{
			UserID:      anotherUser.ID,
			OriginalURL: "https://other-user-site.com",
			Status:      "queued",
		}
		err = urlRepo.Create(otherUserURL)
		require.NoError(t, err, "Should create URL for other user")

		// Test listing URLs for our test user with default pagination.
		urls, err := urlRepo.ListByUser(testUser.ID, defaultPage)
		require.NoError(t, err, "Should list URLs by user")
		assert.Len(t, urls, 2, "Should have 2 URLs for test user")

		// Verify the returned URLs belong to our test user.
		for _, u := range urls {
			assert.Equal(t, testUser.ID, u.UserID, "URL should belong to test user")
		}

		// Test listing for the other user with default pagination.
		otherUserURLs, err := urlRepo.ListByUser(anotherUser.ID, defaultPage)
		require.NoError(t, err, "Should list URLs for other user")
		assert.Len(t, otherUserURLs, 1, "Should have 1 URL for other user")
		assert.Equal(t, anotherUser.ID, otherUserURLs[0].UserID, "URL should belong to other user")
	})

	t.Run("Update", func(t *testing.T) {
		// Change URL properties.
		testURL.Status = "done"
		testURL.OriginalURL = "https://updated-example.com"

		err := urlRepo.Update(testURL)
		require.NoError(t, err, "Should update URL without error")

		// Verify the changes were saved.
		updatedURL, err := urlRepo.FindByID(testURL.ID)
		require.NoError(t, err, "Should find updated URL")
		assert.Equal(t, "done", updatedURL.Status, "Status should be updated to 'done'")
		assert.Equal(t, "https://updated-example.com", updatedURL.OriginalURL, "OriginalURL should be updated")
	})

	t.Run("Delete", func(t *testing.T) {
		err := urlRepo.Delete(testURL.ID)
		require.NoError(t, err, "Should delete URL without error")

		// Verify URL was deleted.
		_, err = urlRepo.FindByID(testURL.ID)
		assert.ErrorIs(t, err, gorm.ErrRecordNotFound, "Deleted URL should not be found")

		// Test deleting non-existent URL.
		err = urlRepo.Delete(9999)
		assert.EqualError(t, err, "url not found", "Should return error when deleting non-existent URL")
	})

	utils.CleanTestData(t)
}
