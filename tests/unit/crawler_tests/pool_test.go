package crawler_test

import (
	"context"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fuzumoe/urlinsight-backend/internal/crawler"
	"github.com/fuzumoe/urlinsight-backend/internal/model"
	"github.com/fuzumoe/urlinsight-backend/internal/repository"
)

// mockPRepo implements repository.URLRepository for testing.
type mockPRepo struct {
	mu                sync.Mutex
	statusUpdates     map[uint][]string
	findByIDCalls     []uint
	saveResultsCalled bool
}

func newMockPRepo() *mockPRepo {
	return &mockPRepo{
		statusUpdates: make(map[uint][]string),
	}
}

func (r *mockPRepo) UpdateStatus(id uint, status string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.statusUpdates[id] = append(r.statusUpdates[id], status)
	return nil
}

func (r *mockPRepo) FindByID(id uint) (*model.URL, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.findByIDCalls = append(r.findByIDCalls, id)
	return &model.URL{
		OriginalURL: "http://example.com",
		// The repository in this test does not store status, so we let the worker manage it.
	}, nil
}

func (r *mockPRepo) SaveResults(id uint, res *model.AnalysisResult, links []model.Link) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.saveResultsCalled = true
	return nil
}

// Stub implementations for the rest of URLRepository.
func (r *mockPRepo) Create(u *model.URL) error { return nil }
func (r *mockPRepo) Delete(id uint) error      { return nil }
func (r *mockPRepo) ListByUser(userID uint, p repository.Pagination) ([]model.URL, error) {
	return []model.URL{}, nil
}
func (r *mockPRepo) Update(u *model.URL) error { return nil }
func (r *mockPRepo) Results(id uint) (*model.URL, error) {
	return &model.URL{OriginalURL: "http://example.com"}, nil
}
func (r *mockPRepo) ResultsWithDetails(id uint) (*model.URL, []*model.AnalysisResult, []*model.Link, error) {
	return &model.URL{OriginalURL: "http://example.com/details"}, []*model.AnalysisResult{}, []*model.Link{}, nil
}

// mockPAnalyzer implements analyzer.Analyzer for testing.
type mockPAnalyzer struct{}

func (a *mockPAnalyzer) Analyze(ctx context.Context, u *url.URL) (*model.AnalysisResult, []model.Link, error) {
	result := &model.AnalysisResult{
		HTMLVersion: "HTML 5",
		Title:       "Test Page",
	}
	links := []model.Link{
		{Href: "http://example.com/page1", StatusCode: 200},
		{Href: "http://example.com/page2", StatusCode: 404},
	}
	return result, links, nil
}

func TestPool_ProcessTasks(t *testing.T) {
	// Create a pool with the mock repository and analyzer.
	mockRepo := newMockPRepo()
	mockAnal := &mockPAnalyzer{}

	// Create a pool with 2 workers, buffer size of 10 and a crawl timeout of 1 second.
	pool := crawler.New(mockRepo, mockAnal, 2, 10, 1*time.Second)

	// Create a context that can be cancelled.
	ctx, cancel := context.WithCancel(context.Background())

	t.Run("Start and Enqueue Tasks", func(t *testing.T) {
		// Start the pool in a goroutine since Start() blocks until the context is cancelled.
		go pool.Start(ctx)

		// Enqueue several tasks.
		taskIDs := []uint{1, 2, 3}
		for _, id := range taskIDs {
			pool.Enqueue(id)
		}

		// Allow time for tasks to be processed.
		time.Sleep(150 * time.Millisecond)
	})

	t.Run("Shutdown Pool", func(t *testing.T) {
		// Shutdown the pool by cancelling the context.
		cancel()
		// Give some time for the pool to clean up.
		time.Sleep(150 * time.Millisecond)
	})

	t.Run("Verify Task Processing", func(t *testing.T) {
		// Check that for each task enqueued, UpdateStatus was called.
		mockRepo.mu.Lock()
		defer mockRepo.mu.Unlock()
		for _, id := range []uint{1, 2, 3} {
			statuses, ok := mockRepo.statusUpdates[id]
			require.True(t, ok, "Expected task id %d to have status updates", id)
			// Expect at least two status updates: one for running and one for done (worker calls UpdateStatus twice).
			require.GreaterOrEqual(t, len(statuses), 2, "Expected at least two status updates for task id %d", id)
		}

		// Check that FindByID was called for each task.
		for _, id := range []uint{1, 2, 3} {
			assert.Contains(t, mockRepo.findByIDCalls, id, "Expected FindByID to be called for task id %d", id)
		}

		// Verify SaveResults was called at least once.
		assert.True(t, mockRepo.saveResultsCalled, "Expected SaveResults to be called")
	})
}
