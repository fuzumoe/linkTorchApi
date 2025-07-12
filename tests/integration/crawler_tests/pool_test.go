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
		db   = utils.SetupTest(t)
		user model.User
		urls = make(map[string]model.URL)
	)
	require.NotNil(t, db)

	// Setup & Migration subtest.
	t.Run("Setup and Create Records", func(t *testing.T) {
		err := db.AutoMigrate(&model.User{}, &model.URL{}, &model.AnalysisResult{}, &model.Link{})

		require.NoError(t, err)

		user = model.User{
			Username: "Pool Tester",
			Email:    fmt.Sprintf("pool_%d@example.com", time.Now().UnixNano()),
		}
		err = db.Create(&user).Error
		require.NoError(t, err)

		t.Run("Create Basic URL", func(t *testing.T) {
			basicURL := model.URL{
				OriginalURL: "http://example.com/basic",
				Status:      model.StatusQueued,
				UserID:      user.ID,
			}
			err = db.Create(&basicURL).Error
			require.NoError(t, err)
			urls["basic"] = basicURL
		})

		t.Run("Create Priority URL", func(t *testing.T) {
			priorityURL := model.URL{
				OriginalURL: "http://example.com/priority",
				Status:      model.StatusQueued,
				UserID:      user.ID,
			}
			err = db.Create(&priorityURL).Error
			require.NoError(t, err)
			urls["priority"] = priorityURL
		})

		t.Run("Create Already Stopped URL", func(t *testing.T) {
			stoppedURL := model.URL{
				OriginalURL: "http://example.com/stopped",
				Status:      model.StatusStopped,
				UserID:      user.ID,
			}
			err = db.Create(&stoppedURL).Error
			require.NoError(t, err)
			urls["stopped"] = stoppedURL
		})
	})

	// Test basic processing
	t.Run("Basic Processing", func(t *testing.T) {
		urlRepo := repository.NewURLRepo(db)
		analyzer := &dummyPAnalyzer{}
		pool := crawler.New(urlRepo, analyzer, 2, 10)

		// Create a context that can be cancelled
		ctx, cancel := context.WithCancel(context.Background())

		defer cancel()
		go pool.Start(ctx)

		t.Run("Enqueue Single URL", func(t *testing.T) {
			pool.Enqueue(urls["basic"].ID)
			time.Sleep(500 * time.Millisecond)
		})

		t.Run("Verify URL Status", func(t *testing.T) {
			var updated model.URL
			err := db.First(&updated, urls["basic"].ID).Error

			require.NoError(t, err)
			assert.Equal(t, model.StatusDone, updated.Status,
				"Expected basic URL status to be %s", model.StatusDone)
		})

		cancel()
		time.Sleep(100 * time.Millisecond)
	})

	t.Run("Multiple URLs", func(t *testing.T) {
		urlRepo := repository.NewURLRepo(db)
		analyzer := &dummyPAnalyzer{}
		pool := crawler.New(urlRepo, analyzer, 1, 5)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		go pool.Start(ctx)

		t.Run("Enqueue Multiple URLs", func(t *testing.T) {
			// Reset URL statuses first
			err := db.Model(&model.URL{}).Where("id IN ?", []uint{urls["basic"].ID, urls["priority"].ID}).
				Update("status", model.StatusQueued).Error

			require.NoError(t, err)

			pool.Enqueue(urls["basic"].ID)
			pool.Enqueue(urls["priority"].ID)

			time.Sleep(1 * time.Second)
		})

		t.Run("Verify Both URLs Processed", func(t *testing.T) {
			var basicURL, priorityURL model.URL

			err := db.First(&basicURL, urls["basic"].ID).Error
			require.NoError(t, err)
			assert.Equal(t, model.StatusDone, basicURL.Status, "Basic URL should be done")

			err = db.First(&priorityURL, urls["priority"].ID).Error
			require.NoError(t, err)
			assert.Equal(t, model.StatusDone, priorityURL.Status, "Priority URL should be done")
		})

		cancel()
		time.Sleep(100 * time.Millisecond)
	})

	t.Run("Context Cancellation", func(t *testing.T) {
		urlRepo := repository.NewURLRepo(db)
		analyzer := &dummyPAnalyzer{}
		pool := crawler.New(urlRepo, analyzer, 1, 5)
		ctx, cancel := context.WithCancel(context.Background())

		cancelURL := model.URL{
			OriginalURL: "http://example.com/cancel",
			Status:      model.StatusQueued,
			UserID:      user.ID,
		}
		err := db.Create(&cancelURL).Error
		require.NoError(t, err)

		t.Run("Start and Cancel Immediately", func(t *testing.T) {

			go pool.Start(ctx)

			pool.Enqueue(cancelURL.ID)

			cancel()

			time.Sleep(100 * time.Millisecond)
		})

		t.Run("Verify URL Status After Cancellation", func(t *testing.T) {
			var updated model.URL

			err := db.First(&updated, cancelURL.ID).Error

			require.NoError(t, err)

			t.Logf("URL status after cancellation: %s", updated.Status)
		})
	})

	t.Run("Already Stopped URL", func(t *testing.T) {
		urlRepo := repository.NewURLRepo(db)
		analyzer := &dummyPAnalyzer{}
		pool := crawler.New(urlRepo, analyzer, 1, 5)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		go pool.Start(ctx)

		t.Run("Enqueue Stopped URL", func(t *testing.T) {
			pool.Enqueue(urls["stopped"].ID)

			time.Sleep(500 * time.Millisecond)
		})

		t.Run("Verify Stopped URL Handling", func(t *testing.T) {
			var updated model.URL
			err := db.First(&updated, urls["stopped"].ID).Error
			require.NoError(t, err)

			t.Logf("Previously stopped URL now has status: %s", updated.Status)
		})

		cancel()
		time.Sleep(100 * time.Millisecond)
	})
}
