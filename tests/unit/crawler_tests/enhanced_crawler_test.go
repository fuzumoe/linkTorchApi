package crawler_test

import (
	"context"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/fuzumoe/linkTorch-api/internal/crawler"
	"github.com/fuzumoe/linkTorch-api/internal/model"
	"github.com/fuzumoe/linkTorch-api/internal/repository"
)

type MockURLRepository struct {
	mock.Mock
}

func (m *MockURLRepository) Create(u *model.URL) error {
	args := m.Called(u)
	return args.Error(0)
}

func (m *MockURLRepository) FindByID(id uint) (*model.URL, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.URL), args.Error(1)
}

func (m *MockURLRepository) CountByUser(userID uint) (int, error) {
	args := m.Called(userID)
	return args.Int(0), args.Error(1)
}

func (m *MockURLRepository) ListByUser(userID uint, p repository.Pagination) ([]model.URL, error) {
	args := m.Called(userID, p)
	return args.Get(0).([]model.URL), args.Error(1)
}

func (m *MockURLRepository) Update(u *model.URL) error {
	args := m.Called(u)
	return args.Error(0)
}

func (m *MockURLRepository) Delete(id uint) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockURLRepository) UpdateStatus(id uint, status string) error {
	args := m.Called(id, status)
	return args.Error(0)
}

func (m *MockURLRepository) SaveResults(id uint, res *model.AnalysisResult, links []model.Link) error {
	args := m.Called(id, res, links)
	return args.Error(0)
}

func (m *MockURLRepository) Results(id uint) (*model.URL, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.URL), args.Error(1)
}

func (m *MockURLRepository) ResultsWithDetails(id uint) (*model.URL, []*model.AnalysisResult, []*model.Link, error) {
	args := m.Called(id)
	return args.Get(0).(*model.URL), args.Get(1).([]*model.AnalysisResult), args.Get(2).([]*model.Link), args.Error(3)
}

type MockAnalyzer struct {
	mock.Mock
}

func (m *MockAnalyzer) Analyze(ctx context.Context, u *url.URL) (*model.AnalysisResult, []model.Link, error) {
	args := m.Called(ctx, u)
	return args.Get(0).(*model.AnalysisResult), args.Get(1).([]model.Link), args.Error(2)
}

func TestEnhancedCrawler(t *testing.T) {

	mockRepo := new(MockURLRepository)
	mockAnalyzer := new(MockAnalyzer)

	pool := crawler.New(mockRepo, mockAnalyzer, 2, 10, 1*time.Second)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go pool.Start(ctx)

	time.Sleep(100 * time.Millisecond)

	testURL := &model.URL{
		ID:          1,
		OriginalURL: "http://example.com",
		Status:      model.StatusQueued,
	}

	mockRepo.On("UpdateStatus", uint(1), model.StatusRunning).Return(nil)
	mockRepo.On("FindByID", uint(1)).Return(testURL, nil)

	analysisResult := &model.AnalysisResult{
		URLID:       1,
		Title:       "Example Domain",
		HTMLVersion: "HTML5",
	}

	links := []model.Link{
		{
			URLID:      1,
			Href:       "http://example.com/page1",
			IsExternal: false,
			StatusCode: 200,
		},
	}

	mockAnalyzer.On("Analyze", mock.Anything, mock.Anything).Return(analysisResult, links, nil)

	mockRepo.On("SaveResults", uint(1), analysisResult, links).Return(nil)
	mockRepo.On("FindByID", uint(1)).Return(testURL, nil)
	mockRepo.On("UpdateStatus", uint(1), model.StatusDone).Return(nil)

	var wg sync.WaitGroup
	wg.Add(1)

	resultCh := pool.GetResults()
	var receivedResult crawler.CrawlResult

	go func() {
		defer wg.Done()
		select {
		case result := <-resultCh:
			receivedResult = result
		case <-time.After(2 * time.Second):
			t.Error("Timed out waiting for result")
		}
	}()

	pool.EnqueueWithPriority(1, 8)

	wg.Wait()

	assert.Equal(t, uint(1), receivedResult.URLID)
	assert.Equal(t, model.StatusDone, receivedResult.Status)
	assert.Equal(t, 1, receivedResult.LinkCount)
	assert.Nil(t, receivedResult.Error)

	pool.AdjustWorkers(crawler.ControlCommand{
		Action: "add",
		Count:  1,
	})

	time.Sleep(100 * time.Millisecond)

	pool.AdjustWorkers(crawler.ControlCommand{
		Action: "remove",
		Count:  1,
	})

	cancel()
	time.Sleep(100 * time.Millisecond)
}
