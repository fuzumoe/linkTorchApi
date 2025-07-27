package repository_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fuzumoe/linkTorch-api/internal/model"
	"github.com/fuzumoe/linkTorch-api/internal/repository"
	"github.com/fuzumoe/linkTorch-api/tests/utils"
)

func TestAnalysisResultRepo_Integration(t *testing.T) {
	// Get a clean database state.
	db := utils.SetupTest(t)

	// Create repositories.
	analysisRepo := repository.NewAnalysisResultRepo(db)
	urlRepo := repository.NewURLRepo(db)
	userRepo := repository.NewUserRepo(db)

	// First create a test user and URL.
	testUser := &model.User{
		Username: "analysisowner",
		Email:    "analysisowner@example.com",
		Password: "password123",
	}
	err := userRepo.Create(testUser)
	require.NoError(t, err, "Should create user without error")
	require.NotZero(t, testUser.ID, "User ID should be set")

	testURL := &model.URL{
		UserID:      testUser.ID,
		OriginalURL: "https://example.com",
		Status:      "queued",
	}
	err = urlRepo.Create(testURL)
	require.NoError(t, err, "Should create URL without error")
	require.NotZero(t, testURL.ID, "URL ID should be set")

	// Test analysis data.
	testAnalysis := &model.AnalysisResult{
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

	defaultPage := repository.Pagination{Page: 1, PageSize: 10}

	t.Run("Create", func(t *testing.T) {
		// Pass nil for links if none exist.
		err := analysisRepo.Create(testAnalysis, nil)
		require.NoError(t, err, "Should create analysis result without error")
		assert.NotZero(t, testAnalysis.ID, "Analysis ID should be set after creation")
		assert.False(t, testAnalysis.CreatedAt.IsZero(), "CreatedAt should be set")
		assert.False(t, testAnalysis.UpdatedAt.IsZero(), "UpdatedAt should be set")
	})

	t.Run("ListByURL", func(t *testing.T) {
		// Create another analysis for the same URL.
		secondAnalysis := &model.AnalysisResult{
			URLID:        testURL.ID,
			HTMLVersion:  "HTML4",
			Title:        "Updated Page",
			H1Count:      1,
			H2Count:      3,
			H3Count:      2,
			H4Count:      1,
			H5Count:      0,
			H6Count:      0,
			HasLoginForm: false,
		}
		err := analysisRepo.Create(secondAnalysis, nil)
		require.NoError(t, err, "Should create second analysis")

		// Create another URL and analysis for it.
		anotherURL := &model.URL{
			UserID:      testUser.ID,
			OriginalURL: "https://another-example.com",
			Status:      "queued",
		}
		err = urlRepo.Create(anotherURL)
		require.NoError(t, err, "Should create another URL")

		otherURLAnalysis := &model.AnalysisResult{
			URLID:        anotherURL.ID,
			HTMLVersion:  "XHTML",
			Title:        "Another Site",
			H1Count:      1,
			H2Count:      2,
			H3Count:      1,
			H4Count:      0,
			H5Count:      0,
			H6Count:      0,
			HasLoginForm: true,
		}
		err = analysisRepo.Create(otherURLAnalysis, nil)
		require.NoError(t, err, "Should create analysis for other URL")

		// Test listing analyses for our test URL.
		analyses, err := analysisRepo.ListByURL(testURL.ID, defaultPage)
		require.NoError(t, err, "Should list analyses by URL")
		assert.Len(t, analyses, 2, "Should have 2 analyses for test URL")

		// Verify the returned analyses belong to our test URL.
		for _, a := range analyses {
			assert.Equal(t, testURL.ID, a.URLID, "Analysis should belong to test URL")
		}

		// Test listing for the other URL.
		otherURLAnalyses, err := analysisRepo.ListByURL(anotherURL.ID, defaultPage)
		require.NoError(t, err, "Should list analyses for other URL")
		assert.Len(t, otherURLAnalyses, 1, "Should have 1 analysis for other URL")
		assert.Equal(t, anotherURL.ID, otherURLAnalyses[0].URLID, "Analysis should belong to other URL")
		assert.Equal(t, "XHTML", otherURLAnalyses[0].HTMLVersion)
		assert.Equal(t, "Another Site", otherURLAnalyses[0].Title)
	})

	t.Run("ListByURL_EmptyResult", func(t *testing.T) {
		// Create a URL without analyses.
		emptyURL := &model.URL{
			UserID:      testUser.ID,
			OriginalURL: "https://empty-analysis.com",
			Status:      "queued",
		}
		err := urlRepo.Create(emptyURL)
		require.NoError(t, err, "Should create empty URL")

		// Should return empty slice, not error.
		analyses, err := analysisRepo.ListByURL(emptyURL.ID, defaultPage)
		require.NoError(t, err, "Should not error for URL without analyses")
		assert.Empty(t, analyses, "Should return empty slice for URL without analyses")
	})

	t.Run("Verify_Through_URL_Preload", func(t *testing.T) {
		// Verify that analyses are correctly associated with URLs by checking preloading.
		foundURL, err := urlRepo.FindByID(testURL.ID)
		require.NoError(t, err, "Should find URL with preloaded analyses")

		// Verify we have two analysis results.
		assert.Len(t, foundURL.AnalysisResults, 2, "URL should have 2 preloaded analyses")

		// Check that the preloaded analyses have the correct data.
		var foundOriginal, foundSecond bool
		for _, ar := range foundURL.AnalysisResults {
			if ar.HTMLVersion == "HTML5" && ar.Title == "Test Page" {
				foundOriginal = true
				assert.Equal(t, 2, ar.H1Count)
				assert.Equal(t, 5, ar.H2Count)
				assert.True(t, ar.HasLoginForm)
			}
			if ar.HTMLVersion == "HTML4" && ar.Title == "Updated Page" {
				foundSecond = true
				assert.Equal(t, 1, ar.H1Count)
				assert.Equal(t, 3, ar.H2Count)
				assert.False(t, ar.HasLoginForm)
			}
		}

		assert.True(t, foundOriginal, "Should find original analysis in preloaded data")
		assert.True(t, foundSecond, "Should find second analysis in preloaded data")
	})

	utils.CleanTestData(t)
}
