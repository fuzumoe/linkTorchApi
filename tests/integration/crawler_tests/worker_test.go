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

type dummyAnalyzer struct{}

func (a *dummyAnalyzer) Analyze(ctx context.Context, u *url.URL) (*model.AnalysisResult, []model.Link, error) {
	result := &model.AnalysisResult{
		HTMLVersion: "HTML5",
		Title:       "Integration Test Title",
	}
	links := []model.Link{
		{Href: "http://example.com/integration", StatusCode: 200},
	}
	return result, links, nil
}

// TestWorkerIntegration tests the worker's ability to process tasks with a real database.
func TestWorkerIntegration(t *testing.T) {
	var (
		db    = utils.SetupTest(t)
		user  model.User
		dummy model.URL
	)
	require.NotNil(t, db)

	// Setup and record creation.
	t.Run("Setup and Create Records", func(t *testing.T) {
		err := db.AutoMigrate(&model.User{}, &model.URL{}, &model.AnalysisResult{}, &model.Link{})
		require.NoError(t, err)

		// Create a real user record.
		user = model.User{
			Username: "Integration Tester",
			Email:    fmt.Sprintf("tester_%d@example.com", time.Now().UnixNano()),
		}
		err = db.Create(&user).Error
		require.NoError(t, err)

		// Insert a dummy URL record with a valid UserID.
		dummy = model.URL{
			OriginalURL: "http://example.com/integration",
			Status:      model.StatusQueued,
			UserID:      user.ID,
		}
		err = db.Create(&dummy).Error
		require.NoError(t, err)
	})

	// Execute worker.
	t.Run("Run Worker", func(t *testing.T) {
		// Create repository and dummy analyzer.
		urlRepo := repository.NewURLRepo(db)
		analyzer := &dummyAnalyzer{}

		// Create a cancelable context.
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Create a worker with the real repository and dummy analyzer.
		worker := crawler.NewWorker(int(dummy.ID), ctx, urlRepo, analyzer)

		// Run the worker in a separate goroutine using a tasks channel.
		tasks := make(chan uint, 1)
		go worker.Run(tasks)

		// Enqueue the task.
		tasks <- dummy.ID

		// Allow time for the worker to process the task.
		time.Sleep(500 * time.Millisecond)

		// Shutdown the worker.
		cancel()
		close(tasks)
	})

	// Verify the results.
	t.Run("Verify Task Processing", func(t *testing.T) {
		var updated model.URL
		err := db.First(&updated, dummy.ID).Error
		require.NoError(t, err)

		// Assert that the record status is updated to "done".
		assert.Equal(t, model.StatusDone, updated.Status, "Expected URL record status to be %s", model.StatusDone)
	})
}
