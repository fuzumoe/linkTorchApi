package crawler_test

import (
	"context"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fuzumoe/linkTorch-api/internal/crawler"
	"github.com/fuzumoe/linkTorch-api/internal/model"
	"github.com/fuzumoe/linkTorch-api/internal/repository"
)

type mockPRepo struct {
	mu                sync.Mutex
	statusUpdates     map[uint][]string
	findByIDCalls     []uint
	saveResultsCalled bool
}

func (r *mockPRepo) CountByUser(userID uint) (int, error) {
	panic("unimplemented")
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
	}, nil
}

func (r *mockPRepo) SaveResults(id uint, res *model.AnalysisResult, links []model.Link) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.saveResultsCalled = true
	return nil
}

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

	mockRepo := newMockPRepo()
	mockAnal := &mockPAnalyzer{}

	pool := crawler.New(mockRepo, mockAnal, 2, 10, 1*time.Second)

	ctx, cancel := context.WithCancel(context.Background())

	t.Run("Start and Enqueue Tasks", func(t *testing.T) {
		go pool.Start(ctx)

		taskIDs := []uint{1, 2, 3}
		for _, id := range taskIDs {
			pool.Enqueue(id)
		}

		time.Sleep(150 * time.Millisecond)
	})

	t.Run("Shutdown Pool", func(t *testing.T) {
		cancel()
		time.Sleep(150 * time.Millisecond)
	})

	t.Run("Verify Task Processing", func(t *testing.T) {
		mockRepo.mu.Lock()
		defer mockRepo.mu.Unlock()
		for _, id := range []uint{1, 2, 3} {
			statuses, ok := mockRepo.statusUpdates[id]
			require.True(t, ok, "Expected task id %d to have status updates", id)
			require.GreaterOrEqual(t, len(statuses), 2, "Expected at least two status updates for task id %d", id)
		}

		for _, id := range []uint{1, 2, 3} {
			assert.Contains(t, mockRepo.findByIDCalls, id, "Expected FindByID to be called for task id %d", id)
		}

		assert.True(t, mockRepo.saveResultsCalled, "Expected SaveResults to be called")
	})
}
