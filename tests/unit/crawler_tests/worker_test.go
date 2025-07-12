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

// mockRepo implements repository.URLRepository for testing.
type mockRepo struct {
	mu sync.Mutex
	// Track method calls and arguments.
	statusUpdates     map[uint][]string
	findByIDCalls     []uint
	saveResultsCalled bool
}

func newMockRepo() *mockRepo {
	return &mockRepo{
		statusUpdates: make(map[uint][]string),
	}
}

func (r *mockRepo) UpdateStatus(id uint, status string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.statusUpdates[id] = append(r.statusUpdates[id], status)
	return nil
}

// FindByID now returns a *model.URL.
func (r *mockRepo) FindByID(id uint) (*model.URL, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.findByIDCalls = append(r.findByIDCalls, id)
	return &model.URL{
		OriginalURL: "http://example.com",
	}, nil
}

func (r *mockRepo) SaveResults(id uint, res *model.AnalysisResult, links []model.Link) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.saveResultsCalled = true
	return nil
}

// Implement Create with the correct signature.
func (r *mockRepo) Create(u *model.URL) error {
	return nil
}

// Add Delete method to satisfy the URLRepository interface.
func (r *mockRepo) Delete(id uint) error {
	return nil
}

// Implement ListByUser with the correct signature.
func (r *mockRepo) ListByUser(userID uint, p repository.Pagination) ([]model.URL, error) {
	return []model.URL{}, nil
}

// Add Results method to satisfy the URLRepository interface.
func (r *mockRepo) Results(id uint) (*model.URL, error) {
	return &model.URL{
		OriginalURL: "http://example.com",
	}, nil
}

// Add Update method to satisfy the URLRepository interface.
func (r *mockRepo) Update(u *model.URL) error {
	return nil
}

// mockAnalyzer implements analyzer.Analyzer for testing.
type mockAnalyzer struct{}

func (a *mockAnalyzer) Analyze(ctx context.Context, u *url.URL) (*model.AnalysisResult, []model.Link, error) {
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

func TestWorker(t *testing.T) {
	// Create a context that can be canceled.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create our mocks.
	repo := newMockRepo()
	analyzer := &mockAnalyzer{}

	// Create worker.
	worker := crawler.NewWorker(1, ctx, repo, analyzer)

	t.Run("Process Task", func(t *testing.T) {
		// Create a channel.
		tasks := make(chan uint, 1)
		done := make(chan struct{})

		// Run worker in a goroutine.
		go func() {
			worker.Run(tasks)
			close(done)
		}()

		// Subtest: Send task.
		t.Run("Send Task", func(t *testing.T) {
			tasks <- 42
		})

		// Give some time for processing.
		time.Sleep(100 * time.Millisecond)

		// Clean shutdown.
		close(tasks)
		cancel()
		<-done
	})

	t.Run("Verify Calls", func(t *testing.T) {
		repo.mu.Lock()
		defer repo.mu.Unlock()

		t.Run("UpdateStatus calls", func(t *testing.T) {
			statusUpdates := repo.statusUpdates[42]
			require.GreaterOrEqual(t, len(statusUpdates), 2, "Expected at least two status updates")
			assert.Equal(t, model.StatusRunning, statusUpdates[0], "First status should be Running")
			assert.Equal(t, model.StatusDone, statusUpdates[len(statusUpdates)-1], "Last status should be Done")
		})

		t.Run("FindByID call", func(t *testing.T) {
			assert.Contains(t, repo.findByIDCalls, uint(42), "FindByID should be called with task ID 42")
		})

		t.Run("SaveResults call", func(t *testing.T) {
			assert.True(t, repo.saveResultsCalled, "SaveResults should be called")
		})
	})
}
