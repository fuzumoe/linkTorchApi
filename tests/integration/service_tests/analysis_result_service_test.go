package service_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fuzumoe/urlinsight-backend/internal/model"
	"github.com/fuzumoe/urlinsight-backend/internal/repository"
	"github.com/fuzumoe/urlinsight-backend/internal/service"
	"github.com/fuzumoe/urlinsight-backend/tests/utils"
)

func TestAnalysisService_Integration(t *testing.T) {
	// Setup test database.
	db := utils.SetupTest(t)

	// Create repositories.
	userRepo := repository.NewUserRepo(db)
	urlRepo := repository.NewURLRepo(db)
	analysisRepo := repository.NewAnalysisResultRepo(db)

	// Create AnalysisService.
	analysisService := service.NewAnalysisService(analysisRepo)

	// Create a test user.
	testUser := &model.User{
		Username:  "analysisUser",
		Email:     "analysis@example.com",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	err := userRepo.Create(testUser)
	require.NoError(t, err, "Should create test user without error.")
	require.NotZero(t, testUser.ID, "User ID should be set after creation.")

	// Create a test URL for analysis results.
	testURL := &model.URL{
		UserID:      testUser.ID,
		OriginalURL: "https://test-analysis.com",
		Status:      "queued",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	err = urlRepo.Create(testURL)
	require.NoError(t, err, "Should create test URL without error.")
	require.NotZero(t, testURL.ID, "URL ID should be set after creation.")

	urlID := testURL.ID

	t.Run("Record", func(t *testing.T) {
		analysis := &model.AnalysisResult{
			URLID:        urlID,
			HTMLVersion:  "HTML5",
			Title:        "Analysis Test",
			H1Count:      3,
			H2Count:      2,
			H3Count:      0,
			H4Count:      0,
			H5Count:      0,
			H6Count:      0,
			HasLoginForm: true,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}

		err := analysisService.Record(analysis)
		require.NoError(t, err, "Should record analysis result without error")
		assert.NotZero(t, analysis.ID, "Analysis result ID should be set after recording")
	})

	t.Run("List", func(t *testing.T) {
		pagination := repository.Pagination{
			Page:     1,
			PageSize: 10,
		}

		// Record two analysis results for the same URL.
		analysis1 := &model.AnalysisResult{
			URLID:        urlID,
			HTMLVersion:  "HTML5",
			Title:        "First Analysis",
			H1Count:      1,
			H2Count:      2,
			H3Count:      3,
			H4Count:      0,
			H5Count:      0,
			H6Count:      0,
			HasLoginForm: true,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}
		analysis2 := &model.AnalysisResult{
			URLID:        urlID,
			HTMLVersion:  "HTML4",
			Title:        "Second Analysis",
			H1Count:      0,
			H2Count:      1,
			H3Count:      0,
			H4Count:      0,
			H5Count:      0,
			H6Count:      0,
			HasLoginForm: false,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}

		err := analysisService.Record(analysis1)
		require.NoError(t, err, "Should record first analysis result.")
		err = analysisService.Record(analysis2)
		require.NoError(t, err, "Should record second analysis result.")

		// List analyses through the service.
		results, err := analysisService.List(urlID, pagination)
		require.NoError(t, err, "Should list analysis results without error.")
		assert.GreaterOrEqual(t, len(results), 2, "Should return at least two analysis results for the URL.")

		var foundFirst, foundSecond bool
		for _, dto := range results {
			if dto.Title == "First Analysis" {
				foundFirst = true
				assert.Equal(t, "HTML5", dto.HTMLVersion, "HTML version should match for first analysis")
			}
			if dto.Title == "Second Analysis" {
				foundSecond = true
				assert.Equal(t, "HTML4", dto.HTMLVersion, "HTML version should match for second analysis")
			}
		}
		assert.True(t, foundFirst, "First analysis result should be in the list.")
		assert.True(t, foundSecond, "Second analysis result should be in the list.")
	})
}
