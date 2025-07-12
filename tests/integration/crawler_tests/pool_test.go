package crawler_test

import (
	"context"
	"fmt"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fuzumoe/urlinsight-backend/internal/crawler"
	"github.com/fuzumoe/urlinsight-backend/internal/model"
	"github.com/fuzumoe/urlinsight-backend/internal/repository"
	"github.com/fuzumoe/urlinsight-backend/tests/utils"
)

type dummyPAnalyzer struct{}

func (a *dummyPAnalyzer) Analyze(ctx context.Context, u *url.URL) (*model.AnalysisResult, []model.Link, error) {
	result := &model.AnalysisResult{
		HTMLVersion: "HTML5",
		Title:       "Pool Integration Test",
	}
	links := []model.Link{
		{Href: "http://example.com/pool", StatusCode: 200},
	}
	return result, links, nil
}

// TestPoolIntegration tests the integration of the crawler pool with a real database.
func TestPoolIntegration(t *testing.T) {
	var (
		db    = utils.SetupTest(t)
		user  model.User
		dummy model.URL
	)
	require.NotNil(t, db)

	// Setup & Migration subtest
	t.Run("Setup and Create Records", func(t *testing.T) {
		err := db.AutoMigrate(&model.User{}, &model.URL{}, &model.AnalysisResult{}, &model.Link{})
		require.NoError(t, err)

		// Create a real user record.
		user = model.User{
			Username: "Pool Tester",
			Email:    fmt.Sprintf("pool_%d@example.com", time.Now().UnixNano()),
		}
		err = db.Create(&user).Error
		require.NoError(t, err)

		// Insert a dummy URL record with a valid UserID.
		dummy = model.URL{
			OriginalURL: "http://example.com/pool",
			Status:      model.StatusQueued,
			UserID:      user.ID,
		}
		err = db.Create(&dummy).Error
		require.NoError(t, err)
	})

	// Execute pool workers in a subtest.
	t.Run("Run Pool and Process Tasks", func(t *testing.T) {
		// Create a real repository using the URL repository implementation.
		urlRepo := repository.NewURLRepo(db)

		// Use the dummy analyzer.
		analyzer := &dummyPAnalyzer{}

		// Create a pool with 2 workers and a buffer size of 10.
		pool := crawler.New(urlRepo, analyzer, 2, 10)
		pool.Start()

		// Enqueue the dummy task (using the dummy record's ID).
		pool.Enqueue(dummy.ID)

		// Optionally, enqueue more tasks:
		// pool.Enqueue(dummy.ID)

		// Allow time for the pool workers to process the task.
		time.Sleep(1 * time.Second)

		// Shutdown the pool.
		pool.Shutdown()
	})

	// Verification subtest.
	t.Run("Verify Task Processing", func(t *testing.T) {
		var updated model.URL
		err := db.First(&updated, dummy.ID).Error
		require.NoError(t, err)

		// Assert that the URL record's status is updated to "done".
		assert.Equal(t, model.StatusDone, updated.Status, "Expected URL record status to be %s", model.StatusDone)
	})
}
