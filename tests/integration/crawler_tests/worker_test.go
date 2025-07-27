package crawler_test

import (
	"context"
	"fmt"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fuzumoe/linkTorch-api/internal/crawler"
	"github.com/fuzumoe/linkTorch-api/internal/model"
	"github.com/fuzumoe/linkTorch-api/internal/repository"
	"github.com/fuzumoe/linkTorch-api/tests/utils"
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

type slowDummyAnalyzer struct{}

func (a *slowDummyAnalyzer) Analyze(ctx context.Context, u *url.URL) (*model.AnalysisResult, []model.Link, error) {
	select {
	case <-ctx.Done():
		return nil, nil, ctx.Err()
	case <-time.After(5 * time.Second):
		result := &model.AnalysisResult{
			HTMLVersion: "HTML5",
			Title:       "Slow Analyzer Result",
		}
		return result, nil, nil
	}
}

func TestWorkerIntegration(t *testing.T) {
	db := utils.SetupTest(t)
	require.NotNil(t, db, "Database should be initialized")

	t.Run("Database Setup", func(t *testing.T) {
		err := db.AutoMigrate(&model.User{}, &model.URL{}, &model.AnalysisResult{}, &model.Link{})
		require.NoError(t, err, "Database migration should succeed")
	})

	t.Run("Normal Processing Flow", func(t *testing.T) {
		var user model.User
		var url model.URL

		t.Run("Create Test Data", func(t *testing.T) {
			user = model.User{
				Username: "Worker Tester",
				Email:    fmt.Sprintf("worker_%d@example.com", time.Now().UnixNano()),
			}
			err := db.Create(&user).Error
			require.NoError(t, err, "User creation should succeed")

			url = model.URL{
				OriginalURL: "http://example.com/worker-test",
				Status:      model.StatusQueued,
				UserID:      user.ID,
			}
			err = db.Create(&url).Error
			require.NoError(t, err, "URL creation should succeed")
		})

		t.Run("Execute Worker", func(t *testing.T) {
			urlRepo := repository.NewURLRepo(db)
			analyzer := &dummyAnalyzer{}

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			resultsChan := make(chan crawler.CrawlResult, 1)
			worker := crawler.NewWorker(1, ctx, urlRepo, analyzer, 1*time.Second, resultsChan)

			tasks := make(chan uint, 1)
			workerDone := make(chan struct{})

			go func() {
				worker.Run(tasks)
				close(workerDone)
			}()

			t.Run("Send Task", func(t *testing.T) {
				tasks <- url.ID
				t.Logf("Task with ID %d sent to worker", url.ID)
			})

			time.Sleep(500 * time.Millisecond)

			t.Run("Shutdown Worker", func(t *testing.T) {
				close(tasks)
				cancel()
				<-workerDone
				t.Log("Worker shut down successfully")
			})
		})

		t.Run("Verify Results", func(t *testing.T) {
			t.Run("URL Status", func(t *testing.T) {
				var updated model.URL
				err := db.First(&updated, url.ID).Error
				require.NoError(t, err, "Should find URL in database")
				assert.Equal(t, model.StatusDone, updated.Status,
					"URL status should be updated to %s", model.StatusDone)
			})

			t.Run("Analysis Results", func(t *testing.T) {
				var results model.AnalysisResult
				err := db.Where("url_id = ?", url.ID).First(&results).Error
				require.NoError(t, err, "Analysis results should be saved")
				assert.Equal(t, "Integration Test Title", results.Title,
					"Analysis title should match expected value")
			})

			t.Run("Saved Links", func(t *testing.T) {
				var count int64
				err := db.Model(&model.Link{}).Where("url_id = ?", url.ID).Count(&count).Error
				require.NoError(t, err, "Should be able to count links")
				assert.Greater(t, count, int64(0), "At least one link should be saved")
			})
		})
	})

	t.Run("Context Cancellation Flow", func(t *testing.T) {
		var user model.User
		var cancelURL model.URL

		t.Run("Create Test Data", func(t *testing.T) {
			err := db.First(&user).Error
			if err != nil {
				user = model.User{
					Username: "Cancel Tester",
					Email:    fmt.Sprintf("cancel_%d@example.com", time.Now().UnixNano()),
				}
				err = db.Create(&user).Error
				require.NoError(t, err, "User creation should succeed")
			}

			cancelURL = model.URL{
				OriginalURL: "http://example.com/cancel-test",
				Status:      model.StatusQueued,
				UserID:      user.ID,
			}
			err = db.Create(&cancelURL).Error
			require.NoError(t, err, "URL creation should succeed")
		})

		t.Run("Execute and Cancel Worker", func(t *testing.T) {
			urlRepo := repository.NewURLRepo(db)
			slowAnalyzer := &slowDummyAnalyzer{}

			ctx, cancel := context.WithCancel(context.Background())

			resultsChan := make(chan crawler.CrawlResult, 1)
			worker := crawler.NewWorker(2, ctx, urlRepo, slowAnalyzer, 1*time.Second, resultsChan)
			tasks := make(chan uint, 1)
			workerDone := make(chan struct{})

			go func() {
				worker.Run(tasks)
				close(workerDone)
			}()

			t.Run("Send Task and Cancel", func(t *testing.T) {
				tasks <- cancelURL.ID
				t.Logf("Task with ID %d sent to worker", cancelURL.ID)

				cancel()
				t.Log("Context cancelled immediately")

				close(tasks)

				<-workerDone
				t.Log("Worker shut down after cancellation")
			})

			time.Sleep(100 * time.Millisecond)
		})

		t.Run("Verify Cancellation Behavior", func(t *testing.T) {
			var updated model.URL
			err := db.First(&updated, cancelURL.ID).Error
			require.NoError(t, err, "Should find URL in database")

			t.Logf("URL status after cancellation: %s", updated.Status)

			if updated.Status == model.StatusQueued {
				t.Log("URL remained in queued state after cancellation")
			} else if updated.Status == model.StatusStopped {
				t.Log("URL was marked as stopped after cancellation")
			} else {
				t.Errorf("Unexpected URL status after cancellation: %s", updated.Status)
			}
			var resultsCount int64
			err = db.Model(&model.AnalysisResult{}).Where("url_id = ?", cancelURL.ID).Count(&resultsCount).Error
			require.NoError(t, err, "Should be able to count results")
			assert.Equal(t, int64(0), resultsCount, "No analysis results should be saved after cancellation")
		})
	})
}
