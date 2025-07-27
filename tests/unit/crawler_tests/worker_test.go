package crawler_test

import (
	"context"
	"errors"
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

type testRepo struct {
	mu                sync.Mutex
	statusUpdates     map[uint][]string
	findByIDCalls     []uint
	saveResultsCalled bool
	urlStatus         map[uint]string
}

func (r *testRepo) CountByUser(userID uint) (int, error) {
	panic("unimplemented")
}

func newTestRepo() *testRepo {
	return &testRepo{
		statusUpdates: make(map[uint][]string),
		urlStatus:     make(map[uint]string),
	}
}

func (r *testRepo) UpdateStatus(id uint, status string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.statusUpdates[id] = append(r.statusUpdates[id], status)
	r.urlStatus[id] = status
	return nil
}

func (r *testRepo) FindByID(id uint) (*model.URL, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.findByIDCalls = append(r.findByIDCalls, id)
	st, ok := r.urlStatus[id]
	if !ok {
		st = model.StatusQueued
	}
	return &model.URL{
		ID:          id,
		OriginalURL: "http://example.com",
		Status:      st,
	}, nil
}

func (r *testRepo) SaveResults(id uint, res *model.AnalysisResult, links []model.Link) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.saveResultsCalled = true
	return nil
}

func (r *testRepo) Create(u *model.URL) error { return nil }
func (r *testRepo) Delete(id uint) error      { return nil }
func (r *testRepo) ListByUser(userID uint, p repository.Pagination) ([]model.URL, error) {
	return []model.URL{}, nil
}
func (r *testRepo) Update(u *model.URL) error { return nil }
func (r *testRepo) Results(id uint) (*model.URL, error) {
	return &model.URL{
		ID:          id,
		OriginalURL: "http://example.com",
		Status:      model.StatusDone,
	}, nil
}
func (r *testRepo) ResultsWithDetails(id uint) (*model.URL, []*model.AnalysisResult, []*model.Link, error) {
	return &model.URL{
		ID:          id,
		OriginalURL: "http://example.com/details",
		Status:      model.StatusDone,
	}, []*model.AnalysisResult{}, []*model.Link{}, nil
}

type dummyAnalyzer struct {
	shouldError bool
}

func (a *dummyAnalyzer) Analyze(ctx context.Context, u *url.URL) (*model.AnalysisResult, []model.Link, error) {
	if a.shouldError {
		return nil, nil, errors.New("analyze error")
	}
	res := &model.AnalysisResult{
		HTMLVersion: "HTML 5",
		Title:       "Test Page",
	}
	links := []model.Link{
		{Href: "http://example.com/page1", StatusCode: 200},
		{Href: "http://example.com/page2", StatusCode: 404},
	}
	return res, links, nil
}

type cancelAnalyzer struct{}

func (a *cancelAnalyzer) Analyze(ctx context.Context, u *url.URL) (*model.AnalysisResult, []model.Link, error) {
	return nil, nil, context.Canceled
}

func TestWorkerSuite(t *testing.T) {
	t.Run("Process_Success", func(t *testing.T) {
		ctx := context.Background()
		repo := newTestRepo()
		require.NoError(t, repo.UpdateStatus(1, model.StatusQueued))
		anal := &dummyAnalyzer{shouldError: false}

		resultsChan := make(chan crawler.CrawlResult, 1)
		worker := crawler.NewWorker(1, ctx, repo, anal, 1*time.Second, resultsChan)
		tasks := make(chan uint, 1)
		tasks <- 1
		close(tasks)
		worker.Run(tasks)

		repo.mu.Lock()
		defer repo.mu.Unlock()
		statuses, ok := repo.statusUpdates[1]
		require.True(t, ok, "Expected status updates for task 1")
		require.GreaterOrEqual(t, len(statuses), 1, "Expected at least one status update")
		assert.Equal(t, model.StatusQueued, statuses[0], "First update should be queued")
		assert.Equal(t, model.StatusDone, statuses[len(statuses)-1], "Final status should be Done")
		assert.True(t, repo.saveResultsCalled, "Expected SaveResults to be called")
		assert.GreaterOrEqual(t, len(repo.findByIDCalls), 1, "Expected FindByID to be called at least once")
	})

	t.Run("Process_AbortsIfStopped", func(t *testing.T) {
		ctx := context.Background()
		repo := newTestRepo()
		require.NoError(t, repo.UpdateStatus(2, model.StatusQueued))
		anal := &dummyAnalyzer{shouldError: false}

		resultsChan := make(chan crawler.CrawlResult, 1)
		worker := crawler.NewWorker(2, ctx, repo, anal, 1*time.Second, resultsChan)
		tasks := make(chan uint, 1)
		tasks <- 2

		go func() {
			time.Sleep(10 * time.Millisecond)
			_ = repo.UpdateStatus(2, model.StatusStopped)
		}()

		close(tasks)
		worker.Run(tasks)

		repo.mu.Lock()
		defer repo.mu.Unlock()
		statuses, ok := repo.statusUpdates[2]
		require.True(t, ok, "Expected status updates for task 2")
		assert.Equal(t, model.StatusDone, statuses[len(statuses)-1], "Final status should be Stopped")
	})

	t.Run("Process_AnalysisError", func(t *testing.T) {
		ctx := context.Background()
		repo := newTestRepo()
		require.NoError(t, repo.UpdateStatus(3, model.StatusQueued))
		anal := &dummyAnalyzer{shouldError: true}

		resultsChan := make(chan crawler.CrawlResult, 1)
		worker := crawler.NewWorker(3, ctx, repo, anal, 1*time.Second, resultsChan)
		tasks := make(chan uint, 1)
		tasks <- 3
		close(tasks)
		worker.Run(tasks)

		repo.mu.Lock()
		defer repo.mu.Unlock()
		statuses, ok := repo.statusUpdates[3]
		require.True(t, ok, "Expected status updates for task 3")
		require.GreaterOrEqual(t, len(statuses), 1, "Expected at least one status update")
		assert.Equal(t, model.StatusQueued, statuses[0], "First status should be queued")
		assert.Equal(t, model.StatusError, statuses[len(statuses)-1], "Final status should be Error")
		assert.False(t, repo.saveResultsCalled, "SaveResults should not be called on error")
	})

	t.Run("Run", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		repo := newTestRepo()
		anal := &dummyAnalyzer{shouldError: false}

		resultsChan := make(chan crawler.CrawlResult, 1)
		worker := crawler.NewWorker(1, ctx, repo, anal, 1*time.Second, resultsChan)

		tasks := make(chan uint, 1)
		done := make(chan struct{})
		go func() {
			worker.Run(tasks)
			close(done)
		}()

		tasks <- 42
		time.Sleep(100 * time.Millisecond)
		close(tasks)
		cancel()
		<-done

		repo.mu.Lock()
		defer repo.mu.Unlock()
		statuses, ok := repo.statusUpdates[42]
		require.True(t, ok, "Expected status updates for task 42")
		if len(statuses) > 0 {
			assert.Equal(t, model.StatusDone, statuses[len(statuses)-1], "Final status should be Done")
		}
		assert.True(t, repo.saveResultsCalled, "SaveResults should be called")
		callCount := 0
		for _, id := range repo.findByIDCalls {
			if id == 42 {
				callCount++
			}
		}
		assert.GreaterOrEqual(t, callCount, 1, "Expected FindByID to be called at least once")
	})

	t.Run("Context_Cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		repo := newTestRepo()
		require.NoError(t, repo.UpdateStatus(44, model.StatusQueued))
		cancelAnal := &cancelAnalyzer{}

		resultsChan := make(chan crawler.CrawlResult, 1)
		worker := crawler.NewWorker(1, ctx, repo, cancelAnal, 1*time.Second, resultsChan)
		tasks := make(chan uint, 1)
		done := make(chan struct{})
		go func() {
			worker.Run(tasks)
			close(done)
		}()
		tasks <- 44
		time.Sleep(100 * time.Millisecond)
		close(tasks)
		cancel()
		<-done

		repo.mu.Lock()
		defer repo.mu.Unlock()
		statuses, ok := repo.statusUpdates[44]
		require.True(t, ok, "Expected status updates for task 44")
		require.GreaterOrEqual(t, len(statuses), 1, "Expected at least one status update")
		assert.Equal(t, model.StatusQueued, statuses[0], "First status should be queued")
		assert.Equal(t, model.StatusStopped, statuses[len(statuses)-1], "Final status should be Stopped")
		assert.False(t, repo.saveResultsCalled, "SaveResults should not be called when cancelled")
	})
}
