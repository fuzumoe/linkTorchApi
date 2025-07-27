package service_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/fuzumoe/linkTorch-api/internal/crawler"
	"github.com/fuzumoe/linkTorch-api/internal/model"
	"github.com/fuzumoe/linkTorch-api/internal/repository"
	"github.com/fuzumoe/linkTorch-api/internal/service"
)

// DummyCrawlerPool updated to match new interface
type DummyCrawlerPool struct{}

func (d *DummyCrawlerPool) Start(ctx context.Context)                 {}
func (d *DummyCrawlerPool) Enqueue(id uint)                           {}
func (d *DummyCrawlerPool) EnqueueWithPriority(id uint, priority int) {}
func (d *DummyCrawlerPool) Shutdown()                                 {}
func (d *DummyCrawlerPool) GetResults() <-chan crawler.CrawlResult {
	return make(chan crawler.CrawlResult)
}
func (d *DummyCrawlerPool) AdjustWorkers(cmd crawler.ControlCommand) {}

// MockCrawlerPool updated to match new interface
type MockCrawlerPool struct {
	mock.Mock
}

func (m *MockCrawlerPool) Start(ctx context.Context) {
	m.Called(ctx)
}
func (m *MockCrawlerPool) Enqueue(id uint) {
	m.Called(id)
}
func (m *MockCrawlerPool) EnqueueWithPriority(id uint, priority int) {
	m.Called(id, priority)
}
func (m *MockCrawlerPool) Shutdown() {
	m.Called()
}
func (m *MockCrawlerPool) GetResults() <-chan crawler.CrawlResult {
	args := m.Called()
	return args.Get(0).(<-chan crawler.CrawlResult)
}
func (m *MockCrawlerPool) AdjustWorkers(cmd crawler.ControlCommand) {
	m.Called(cmd)
}

// MockURLRepo mocks implementation of repository.URLRepository.
type MockURLRepo struct {
	mock.Mock
}

func (m *MockURLRepo) Create(url *model.URL) error {
	args := m.Called(url)
	return args.Error(0)
}

func (m *MockURLRepo) FindByID(id uint) (*model.URL, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.URL), args.Error(1)
}

func (m *MockURLRepo) ListByUser(userID uint, p repository.Pagination) ([]model.URL, error) {
	args := m.Called(userID, p)
	return args.Get(0).([]model.URL), args.Error(1)
}

func (m *MockURLRepo) CountByUser(userID uint) (int, error) {
	args := m.Called(userID)
	return args.Int(0), args.Error(1)
}

func (m *MockURLRepo) Update(url *model.URL) error {
	args := m.Called(url)
	return args.Error(0)
}

func (m *MockURLRepo) Delete(id uint) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockURLRepo) UpdateStatus(id uint, status string) error {
	args := m.Called(id, status)
	return args.Error(0)
}

func (m *MockURLRepo) Results(id uint) (*model.URL, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.URL), args.Error(1)
}

func (m *MockURLRepo) SaveResults(urlID uint, analysisRes *model.AnalysisResult, links []model.Link) error {
	args := m.Called(urlID, analysisRes, links)
	return args.Error(0)
}

// New method added to fully implement repository.URLRepository.
func (m *MockURLRepo) ResultsWithDetails(id uint) (*model.URL, []*model.AnalysisResult, []*model.Link, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, nil, nil, args.Error(3)
	}
	return args.Get(0).(*model.URL), args.Get(1).([]*model.AnalysisResult), args.Get(2).([]*model.Link), args.Error(3)
}

func TestURLService_Create(t *testing.T) {
	mockRepo := new(MockURLRepo)
	dummyPool := &DummyCrawlerPool{}
	svc := service.NewURLService(mockRepo, dummyPool)

	input := &model.CreateURLInputDTO{
		UserID:      1,
		OriginalURL: "https://example.com",
	}

	t.Run("Success", func(t *testing.T) {
		mockRepo.
			On("Create", mock.MatchedBy(func(u *model.URL) bool {
				return u.UserID == input.UserID && u.OriginalURL == input.OriginalURL
			})).
			Run(func(args mock.Arguments) {
				url := args.Get(0).(*model.URL)
				url.ID = 42
			}).
			Return(nil).
			Once()

		id, err := svc.Create(input)
		assert.NoError(t, err)
		assert.Equal(t, uint(42), id)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Repository Error", func(t *testing.T) {
		expectedErr := errors.New("database error")
		mockRepo.
			On("Create", mock.MatchedBy(func(u *model.URL) bool {
				return u.UserID == input.UserID && u.OriginalURL == input.OriginalURL
			})).
			Return(expectedErr).
			Once()

		id, err := svc.Create(input)
		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.Equal(t, uint(0), id)
		mockRepo.AssertExpectations(t)
	})
}

func TestURLService_Get(t *testing.T) {
	mockRepo := new(MockURLRepo)
	dummyPool := &DummyCrawlerPool{}
	svc := service.NewURLService(mockRepo, dummyPool)

	urlID := uint(42)
	testURL := &model.URL{
		ID:          urlID,
		UserID:      1,
		OriginalURL: "https://example.com",
		Status:      "queued",
	}

	t.Run("Success", func(t *testing.T) {
		mockRepo.On("FindByID", urlID).Return(testURL, nil).Once()

		dto, err := svc.Get(urlID)
		require.NoError(t, err)
		assert.NotNil(t, dto)
		assert.Equal(t, urlID, dto.ID)
		assert.Equal(t, testURL.UserID, dto.UserID)
		assert.Equal(t, testURL.OriginalURL, dto.OriginalURL)
		assert.Equal(t, testURL.Status, dto.Status)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Not Found", func(t *testing.T) {
		expectedErr := errors.New("record not found")
		mockRepo.On("FindByID", urlID).Return(nil, expectedErr).Once()

		dto, err := svc.Get(urlID)
		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.Nil(t, dto)
		mockRepo.AssertExpectations(t)
	})
}

func TestURLService_List(t *testing.T) {
	mockRepo := new(MockURLRepo)
	dummyPool := &DummyCrawlerPool{}
	svc := service.NewURLService(mockRepo, dummyPool)

	userID := uint(1)
	pagination := repository.Pagination{Page: 1, PageSize: 10}
	urls := []model.URL{
		{ID: 1, UserID: userID, OriginalURL: "https://example1.com", Status: "done"},
		{ID: 2, UserID: userID, OriginalURL: "https://example2.com", Status: "queued"},
	}

	t.Run("Success", func(t *testing.T) {
		mockRepo.On("ListByUser", userID, pagination).Return(urls, nil).Once()
		mockRepo.On("CountByUser", userID).Return(2, nil).Once()

		result, err := svc.List(userID, pagination)
		require.NoError(t, err)
		require.NotNil(t, result)

		// Check pagination metadata
		assert.Equal(t, 1, result.Pagination.Page)
		assert.Equal(t, 10, result.Pagination.PageSize)
		assert.Equal(t, 2, result.Pagination.TotalItems)
		assert.Equal(t, 1, result.Pagination.TotalPages)

		// Check data
		require.Len(t, result.Data, 2)
		assert.Equal(t, uint(1), result.Data[0].ID)
		assert.Equal(t, userID, result.Data[0].UserID)
		assert.Equal(t, "https://example1.com", result.Data[0].OriginalURL)
		assert.Equal(t, "done", result.Data[0].Status)

		assert.Equal(t, uint(2), result.Data[1].ID)
		assert.Equal(t, userID, result.Data[1].UserID)
		assert.Equal(t, "https://example2.com", result.Data[1].OriginalURL)
		assert.Equal(t, "queued", result.Data[1].Status)

		mockRepo.AssertExpectations(t)
	})

	t.Run("Empty Results", func(t *testing.T) {
		mockRepo.On("ListByUser", userID, pagination).Return([]model.URL{}, nil).Once()
		mockRepo.On("CountByUser", userID).Return(0, nil).Once()

		result, err := svc.List(userID, pagination)
		require.NoError(t, err)
		assert.Empty(t, result.Data)
		assert.Equal(t, 0, result.Pagination.TotalItems)
		assert.Equal(t, 0, result.Pagination.TotalPages)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Repository Error on ListByUser", func(t *testing.T) {
		expectedErr := errors.New("database error")
		mockRepo.On("ListByUser", userID, pagination).Return([]model.URL{}, expectedErr).Once()

		result, err := svc.List(userID, pagination)
		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.Nil(t, result)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Repository Error on CountByUser", func(t *testing.T) {
		mockRepo.On("ListByUser", userID, pagination).Return(urls, nil).Once()
		expectedErr := errors.New("count error")
		mockRepo.On("CountByUser", userID).Return(0, expectedErr).Once()

		result, err := svc.List(userID, pagination)
		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.Nil(t, result)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Multiple Pages", func(t *testing.T) {
		// Test with 21 total items, which should result in 3 pages with pageSize 10
		mockRepo.On("ListByUser", userID, pagination).Return(urls, nil).Once()
		mockRepo.On("CountByUser", userID).Return(21, nil).Once()

		result, err := svc.List(userID, pagination)
		require.NoError(t, err)
		assert.Equal(t, 21, result.Pagination.TotalItems)
		assert.Equal(t, 3, result.Pagination.TotalPages) // Ceil(21/10) = 3
		mockRepo.AssertExpectations(t)
	})
}

func TestURLService_Update(t *testing.T) {
	mockRepo := new(MockURLRepo)
	dummyPool := &DummyCrawlerPool{}
	svc := service.NewURLService(mockRepo, dummyPool)
	urlID := uint(42)

	t.Run("Update Original URL", func(t *testing.T) {
		existingURL := &model.URL{
			ID:          urlID,
			UserID:      1,
			OriginalURL: "https://old-example.com",
			Status:      "queued",
		}
		input := &model.UpdateURLInput{OriginalURL: "https://new-example.com"}

		mockRepo.On("FindByID", urlID).Return(existingURL, nil).Once()
		mockRepo.On("Update", mock.AnythingOfType("*model.URL")).Return(nil).Once()

		err := svc.Update(urlID, input)
		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Update Status", func(t *testing.T) {
		existingURL := &model.URL{
			ID:          urlID,
			UserID:      1,
			OriginalURL: "https://old-example.com",
			Status:      "queued",
		}
		input := &model.UpdateURLInput{Status: "done"}

		mockRepo.On("FindByID", urlID).Return(existingURL, nil).Once()
		mockRepo.On("Update", mock.AnythingOfType("*model.URL")).Return(nil).Once()

		err := svc.Update(urlID, input)
		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Invalid Status", func(t *testing.T) {
		existingURL := &model.URL{
			ID:          urlID,
			UserID:      1,
			OriginalURL: "https://old-example.com",
			Status:      "queued",
		}
		input := &model.UpdateURLInput{Status: "invalid_status"}

		mockRepo.On("FindByID", urlID).Return(existingURL, nil).Once()
		err := svc.Update(urlID, input)
		assert.Error(t, err)
		assert.Equal(t, "invalid status value", err.Error())
		mockRepo.AssertExpectations(t)
	})

	t.Run("URL Not Found", func(t *testing.T) {
		input := &model.UpdateURLInput{OriginalURL: "https://new-example.com"}
		expectedErr := errors.New("record not found")
		mockRepo.On("FindByID", urlID).Return(nil, expectedErr).Once()

		err := svc.Update(urlID, input)
		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Update Error", func(t *testing.T) {
		existingURL := &model.URL{
			ID:          urlID,
			UserID:      1,
			OriginalURL: "https://old-example.com",
			Status:      "queued",
		}
		input := &model.UpdateURLInput{OriginalURL: "https://new-example.com"}
		mockRepo.On("FindByID", urlID).Return(existingURL, nil).Once()
		expectedErr := errors.New("update error")
		mockRepo.On("Update", mock.AnythingOfType("*model.URL")).Return(expectedErr).Once()

		err := svc.Update(urlID, input)
		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		mockRepo.AssertExpectations(t)
	})
}

func TestURLService_Delete(t *testing.T) {
	mockRepo := new(MockURLRepo)
	dummyPool := &DummyCrawlerPool{}
	svc := service.NewURLService(mockRepo, dummyPool)
	urlID := uint(42)

	t.Run("Success", func(t *testing.T) {
		mockRepo.On("Delete", urlID).Return(nil).Once()
		err := svc.Delete(urlID)
		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Not Found", func(t *testing.T) {
		expectedErr := errors.New("url not found")
		mockRepo.On("Delete", urlID).Return(expectedErr).Once()
		err := svc.Delete(urlID)
		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		mockRepo.AssertExpectations(t)
	})
}

func TestURLService_Start(t *testing.T) {
	mockRepo := new(MockURLRepo)
	mockPool := new(MockCrawlerPool)
	svc := service.NewURLService(mockRepo, mockPool)
	urlID := uint(100)

	t.Run("Success", func(t *testing.T) {
		testURL := &model.URL{
			ID:          urlID,
			OriginalURL: "http://example.com",
			Status:      model.StatusQueued,
		}

		mockRepo.On("FindByID", urlID).Return(testURL, nil).Once()
		mockRepo.On("UpdateStatus", urlID, model.StatusQueued).Return(nil).Once()
		mockPool.On("Enqueue", urlID).Return().Once()

		err := svc.Start(urlID)
		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
		mockPool.AssertExpectations(t)
	})

	t.Run("URL Not Found", func(t *testing.T) {
		expectedErr := errors.New("record not found")
		mockRepo.On("FindByID", urlID).Return(nil, expectedErr).Once()

		err := svc.Start(urlID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot start crawling")
		assert.Contains(t, err.Error(), expectedErr.Error())
		mockRepo.AssertExpectations(t)
	})

	t.Run("UpdateStatus Error", func(t *testing.T) {
		testURL := &model.URL{
			ID:          urlID,
			OriginalURL: "http://example.com",
			Status:      model.StatusQueued,
		}
		expectedErr := errors.New("update status error")
		mockRepo.On("FindByID", urlID).Return(testURL, nil).Once()
		mockRepo.On("UpdateStatus", urlID, model.StatusQueued).Return(expectedErr).Once()

		err := svc.Start(urlID)
		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		mockRepo.AssertExpectations(t)
	})
}

func TestURLService_Stop(t *testing.T) {
	mockRepo := new(MockURLRepo)
	dummyPool := &DummyCrawlerPool{}
	svc := service.NewURLService(mockRepo, dummyPool)
	urlID := uint(100)

	t.Run("Success", func(t *testing.T) {
		testURL := &model.URL{
			ID:          urlID,
			OriginalURL: "http://example.com",
			Status:      model.StatusRunning,
		}

		mockRepo.On("FindByID", urlID).Return(testURL, nil).Once()
		mockRepo.On("UpdateStatus", urlID, model.StatusError).Return(nil).Once()

		err := svc.Stop(urlID)
		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("URL Not Found", func(t *testing.T) {
		expectedErr := errors.New("record not found")
		mockRepo.On("FindByID", urlID).Return(nil, expectedErr).Once()

		err := svc.Stop(urlID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot stop crawling")
		assert.Contains(t, err.Error(), expectedErr.Error())
		mockRepo.AssertExpectations(t)
	})

	t.Run("UpdateStatus Error", func(t *testing.T) {
		testURL := &model.URL{
			ID:          urlID,
			OriginalURL: "http://example.com",
			Status:      model.StatusRunning,
		}
		expectedErr := errors.New("update status error")
		mockRepo.On("FindByID", urlID).Return(testURL, nil).Once()
		mockRepo.On("UpdateStatus", urlID, model.StatusError).Return(expectedErr).Once()

		err := svc.Stop(urlID)
		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		mockRepo.AssertExpectations(t)
	})
}

func TestURLService_Results(t *testing.T) {
	mockRepo := new(MockURLRepo)
	dummyPool := &DummyCrawlerPool{}
	svc := service.NewURLService(mockRepo, dummyPool)
	urlID := uint(55)
	testURL := &model.URL{
		ID:          urlID,
		UserID:      99,
		OriginalURL: "https://results.test",
		Status:      "completed",
	}

	mockRepo.On("Results", urlID).Return(testURL, nil).Once()

	dto, err := svc.Results(urlID)
	require.NoError(t, err)
	assert.Equal(t, urlID, dto.ID)
	assert.Equal(t, uint(99), dto.UserID)
	assert.Equal(t, "https://results.test", dto.OriginalURL)
	mockRepo.AssertExpectations(t)
}

func TestURLService_ResultsWithDetails(t *testing.T) {
	mockRepo := new(MockURLRepo)
	dummyPool := &DummyCrawlerPool{}
	svc := service.NewURLService(mockRepo, dummyPool)
	urlID := uint(77)

	// Prepare dummy detailed data
	testURL := &model.URL{
		ID:          urlID,
		UserID:      101,
		OriginalURL: "https://detailed.test",
		Status:      "completed",
		CreatedAt:   mustParseTime("2025-07-11T00:00:00Z"),
		UpdatedAt:   mustParseTime("2025-07-11T01:00:00Z"),
	}
	analysisResults := []*model.AnalysisResult{} // empty slice for test
	links := []*model.Link{}                     // empty slice for test

	mockRepo.On("ResultsWithDetails", urlID).
		Return(testURL, analysisResults, links, nil).
		Once()

	urlOut, ars, ls, err := svc.ResultsWithDetails(urlID)
	require.NoError(t, err)
	assert.Equal(t, urlID, urlOut.ID)
	assert.Empty(t, ars)
	assert.Empty(t, ls)
	mockRepo.AssertExpectations(t)
}

// mustParseTime is a helper to parse RFC3339 time and panic on error.
func mustParseTime(s string) time.Time {
	parsed, err := time.Parse(time.RFC3339, s)
	if err != nil {
		panic(fmt.Sprintf("failed to parse time: %s", s))
	}
	return parsed
}
