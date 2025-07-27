package repository_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/fuzumoe/linkTorch-api/internal/model"
	"github.com/fuzumoe/linkTorch-api/internal/repository"
	"github.com/fuzumoe/linkTorch-api/tests/utils"
)

func TestURLRepo_Integration(t *testing.T) {
	// Get a clean database state.
	db := utils.SetupTest(t)

	// Create repositories.
	urlRepo := repository.NewURLRepo(db)
	userRepo := repository.NewUserRepo(db)

	// Create a test user.
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

	// Create variable for another user that will be used in multiple tests
	var anotherUser *model.User

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
		anotherUser = &model.User{
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

		// Test listing URLs for our test user.
		urls, err := urlRepo.ListByUser(testUser.ID, defaultPage)
		require.NoError(t, err, "Should list URLs by user")
		assert.Len(t, urls, 2, "Should have 2 URLs for test user")

		for _, u := range urls {
			assert.Equal(t, testUser.ID, u.UserID, "URL should belong to test user")
		}

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
		updatedURL, err := urlRepo.FindByID(testURL.ID)
		require.NoError(t, err, "Should find updated URL")
		assert.Equal(t, "done", updatedURL.Status, "Status should be updated to 'done'")
		assert.Equal(t, "https://updated-example.com", updatedURL.OriginalURL, "OriginalURL should be updated")
	})

	t.Run("UpdateStatus", func(t *testing.T) {
		// Use a valid status value that fits in the DB column.
		newStatus := "done"
		err := urlRepo.UpdateStatus(testURL.ID, newStatus)
		require.NoError(t, err, "Should update status without error")
		statusURL, err := urlRepo.FindByID(testURL.ID)
		require.NoError(t, err, "Should find URL after status update")
		assert.Equal(t, newStatus, statusURL.Status, "Status should be updated")
	})

	t.Run("SaveResults", func(t *testing.T) {
		// Create a new URL so existing analysis results don't interfere.
		newURL := &model.URL{
			UserID:      testUser.ID,
			OriginalURL: "https://save-results.com",
			Status:      "queued",
		}
		err := urlRepo.Create(newURL)
		require.NoError(t, err, "Should create URL for SaveResults test")

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

		err = urlRepo.SaveResults(newURL.ID, analysisRes, links)
		require.NoError(t, err, "Should save analysis results without error")

		// Retrieve the URL with preloaded relations.
		resultURL, err := urlRepo.FindByID(newURL.ID)
		require.NoError(t, err, "Should retrieve URL with results")
		require.NotEmpty(t, resultURL.AnalysisResults, "AnalysisResults should not be empty")
		require.NotEmpty(t, resultURL.Links, "Links should not be empty")
		// Verify analysis result details.
		assert.Equal(t, "Analysis Title", resultURL.AnalysisResults[0].Title)
		assert.Equal(t, "HTML5", resultURL.AnalysisResults[0].HTMLVersion)
		// Verify links order and Href.
		assert.Len(t, resultURL.Links, 2, "Should have 2 links")
		assert.Contains(t, []string{"https://example.com/link1", "https://example.com/link2"},
			resultURL.Links[0].Href, "Should contain link1 or link2")
		assert.Contains(t, []string{"https://example.com/link1", "https://example.com/link2"},
			resultURL.Links[1].Href, "Should contain link1 or link2")
	})

	// New test for Results method
	t.Run("Results", func(t *testing.T) {
		// Create a new URL for testing Results
		resultsURL := &model.URL{
			UserID:      testUser.ID,
			OriginalURL: "https://results-test.com",
			Status:      "done",
		}
		err := urlRepo.Create(resultsURL)
		require.NoError(t, err, "Should create URL for Results test")

		// Add analysis result and links
		analysisRes := &model.AnalysisResult{
			URLID:        resultsURL.ID,
			HTMLVersion:  "HTML5",
			Title:        "Results Test Page",
			H1Count:      3,
			H2Count:      6,
			HasLoginForm: false,
		}
		err = db.Create(analysisRes).Error
		require.NoError(t, err, "Should create analysis result for Results test")

		links := []model.Link{
			{URLID: resultsURL.ID, Href: "https://results-link1.com", StatusCode: 200},
			{URLID: resultsURL.ID, Href: "https://results-link2.com", StatusCode: 404},
		}
		err = db.CreateInBatches(links, 10).Error
		require.NoError(t, err, "Should create links for Results test")

		// Test the Results method
		url, err := urlRepo.Results(resultsURL.ID)
		require.NoError(t, err, "Should get results without error")
		assert.Equal(t, resultsURL.ID, url.ID, "URL ID should match")
		assert.Equal(t, "https://results-test.com", url.OriginalURL, "URL should have correct original URL")

		// Verify analysis results
		require.NotEmpty(t, url.AnalysisResults, "Analysis results should be preloaded")
		assert.Equal(t, "Results Test Page", url.AnalysisResults[0].Title, "Analysis result should have correct title")
		assert.Equal(t, 3, url.AnalysisResults[0].H1Count, "Analysis result should have correct H1 count")

		// Verify links
		require.NotEmpty(t, url.Links, "Links should be preloaded")
		assert.Len(t, url.Links, 2, "Should have 2 links")
		// Links might be returned in any order, so we check both possibilities
		linkHrefs := []string{url.Links[0].Href, url.Links[1].Href}
		assert.Contains(t, linkHrefs, "https://results-link1.com", "Links should include link1")
		assert.Contains(t, linkHrefs, "https://results-link2.com", "Links should include link2")

		// Test not found case
		_, err = urlRepo.Results(9999)
		assert.Error(t, err, "Should return error for non-existent URL")
	})

	// New test for ResultsWithDetails method
	t.Run("ResultsWithDetails", func(t *testing.T) {
		// Create a new URL for testing ResultsWithDetails
		detailsURL := &model.URL{
			UserID:      testUser.ID,
			OriginalURL: "https://details-test.com",
			Status:      "done",
		}
		err := urlRepo.Create(detailsURL)
		require.NoError(t, err, "Should create URL for ResultsWithDetails test")

		// Add analysis result and links
		analysisRes := &model.AnalysisResult{
			URLID:             detailsURL.ID,
			HTMLVersion:       "HTML5",
			Title:             "Details Test Page",
			H1Count:           4,
			H2Count:           8,
			HasLoginForm:      true,
			InternalLinkCount: 5,
			ExternalLinkCount: 3,
			BrokenLinkCount:   1,
		}
		err = db.Create(analysisRes).Error
		require.NoError(t, err, "Should create analysis result for ResultsWithDetails test")

		links := []model.Link{
			{URLID: detailsURL.ID, Href: "https://details-internal.com", IsExternal: false, StatusCode: 200},
			{URLID: detailsURL.ID, Href: "https://details-external.com", IsExternal: true, StatusCode: 200},
			{URLID: detailsURL.ID, Href: "https://details-broken.com", IsExternal: true, StatusCode: 404},
		}
		err = db.CreateInBatches(links, 10).Error
		require.NoError(t, err, "Should create links for ResultsWithDetails test")

		// Test the ResultsWithDetails method - note return type of links is []*model.Link
		url, analysisResults, linkPointers, err := urlRepo.ResultsWithDetails(detailsURL.ID)
		require.NoError(t, err, "Should get detailed results without error")

		// Verify URL
		assert.Equal(t, detailsURL.ID, url.ID, "URL ID should match")
		assert.Equal(t, "https://details-test.com", url.OriginalURL, "URL should have correct original URL")

		// Verify analysis results
		require.NotNil(t, analysisResults, "Analysis results should not be nil")
		assert.Len(t, analysisResults, 1, "Should have 1 analysis result")
		assert.Equal(t, "Details Test Page", analysisResults[0].Title, "Analysis result should have correct title")
		assert.Equal(t, 4, analysisResults[0].H1Count, "Analysis result should have correct H1 count")
		assert.Equal(t, 5, analysisResults[0].InternalLinkCount, "Analysis result should have correct internal link count")
		assert.Equal(t, 3, analysisResults[0].ExternalLinkCount, "Analysis result should have correct external link count")
		assert.Equal(t, 1, analysisResults[0].BrokenLinkCount, "Analysis result should have correct broken link count")
		assert.True(t, analysisResults[0].HasLoginForm, "Analysis result should have correct login form flag")

		// Verify links - linkPointers is now []*model.Link
		require.NotNil(t, linkPointers, "Links should not be nil")
		assert.Len(t, linkPointers, 3, "Should have 3 links")

		// Create maps to easily verify links exist regardless of order
		linkMap := make(map[string]*model.Link)
		for _, link := range linkPointers {
			linkMap[link.Href] = link // link is already a *model.Link
		}

		// Verify each expected link exists
		internalLink, found := linkMap["https://details-internal.com"]
		assert.True(t, found, "Should have internal link")
		assert.False(t, internalLink.IsExternal, "Internal link should be marked as not external")
		assert.Equal(t, 200, internalLink.StatusCode, "Internal link should have correct status code")

		externalLink, found := linkMap["https://details-external.com"]
		assert.True(t, found, "Should have external link")
		assert.True(t, externalLink.IsExternal, "External link should be marked as external")
		assert.Equal(t, 200, externalLink.StatusCode, "External link should have correct status code")

		brokenLink, found := linkMap["https://details-broken.com"]
		assert.True(t, found, "Should have broken link")
		assert.True(t, brokenLink.IsExternal, "Broken link should be marked as external")
		assert.Equal(t, 404, brokenLink.StatusCode, "Broken link should have correct status code")

		// Test not found case
		_, _, _, err = urlRepo.ResultsWithDetails(9999)
		assert.Error(t, err, "Should return error for non-existent URL")
	})

	t.Run("Delete", func(t *testing.T) {
		err := urlRepo.Delete(testURL.ID)
		require.NoError(t, err, "Should delete URL without error")
		_, err = urlRepo.FindByID(testURL.ID)
		assert.ErrorIs(t, err, gorm.ErrRecordNotFound, "Deleted URL should not be found")
		// Test deleting non-existent URL.
		err = urlRepo.Delete(9999)
		assert.EqualError(t, err, "url not found", "Should return error when deleting non-existent URL")
	})

	// Add the CountByUser test case
	t.Run("CountByUser", func(t *testing.T) {
		// We have created several URLs for testUser in previous tests:
		// - testURL (now deleted)
		// - secondURL
		// - newURL (from SaveResults test)
		// - resultsURL (from Results test)
		// - detailsURL (from ResultsWithDetails test)
		// And one URL for anotherUser:
		// - otherUserURL

		// Test count for testUser (should be 4 URLs still active after testURL was deleted)
		count, err := urlRepo.CountByUser(testUser.ID)
		require.NoError(t, err, "Should count URLs without error")
		assert.Equal(t, 4, count, "Should have 4 active URLs for testUser")

		// Test count for anotherUser (should be 1)
		count, err = urlRepo.CountByUser(anotherUser.ID)
		require.NoError(t, err, "Should count URLs without error")
		assert.Equal(t, 1, count, "Should have 1 URL for anotherUser")

		// Test count for a non-existent user (should be 0)
		count, err = urlRepo.CountByUser(9999)
		require.NoError(t, err, "Should not error for non-existent user")
		assert.Equal(t, 0, count, "Should have 0 URLs for non-existent user")

		// Create one more URL for testUser to ensure the count increases
		additionalURL := &model.URL{
			UserID:      testUser.ID,
			OriginalURL: "https://count-test.com",
			Status:      "queued",
		}
		err = urlRepo.Create(additionalURL)
		require.NoError(t, err, "Should create additional URL")

		// Verify updated count
		newCount, err := urlRepo.CountByUser(testUser.ID)
		require.NoError(t, err, "Should count URLs without error")
		assert.Equal(t, 5, newCount, "Should have 5 active URLs after adding one more")
	})

	utils.CleanTestData(t)
}
