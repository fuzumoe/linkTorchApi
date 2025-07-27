package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/fuzumoe/linkTorch-api/internal/crawler"
	"github.com/fuzumoe/linkTorch-api/internal/handler"
	"github.com/fuzumoe/linkTorch-api/internal/model"
	"github.com/fuzumoe/linkTorch-api/internal/repository"
)

// MockURLService implements the service.URLService interface for testing
type MockURLService struct {
	mock.Mock
}

func (m *MockURLService) Create(input *model.CreateURLInputDTO) (uint, error) {
	args := m.Called(input)
	return args.Get(0).(uint), args.Error(1)
}

func (m *MockURLService) Get(id uint) (*model.URLDTO, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.URLDTO), args.Error(1)
}

func (m *MockURLService) List(userID uint, p repository.Pagination) (*model.PaginatedResponse[model.URLDTO], error) {
	args := m.Called(userID, p)
	return args.Get(0).(*model.PaginatedResponse[model.URLDTO]), args.Error(1)
}

func (m *MockURLService) Update(id uint, input *model.UpdateURLInput) error {
	args := m.Called(id, input)
	return args.Error(0)
}

func (m *MockURLService) Delete(id uint) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockURLService) Start(id uint) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockURLService) Stop(id uint) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockURLService) StartWithPriority(id uint, priority int) error {
	args := m.Called(id, priority)
	return args.Error(0)
}

func (m *MockURLService) Results(id uint) (*model.URLDTO, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.URLDTO), args.Error(1)
}

func (m *MockURLService) ResultsWithDetails(id uint) (*model.URL, []*model.AnalysisResult, []*model.Link, error) {
	args := m.Called(id)
	var url *model.URL
	var analysisResults []*model.AnalysisResult
	var links []*model.Link

	if args.Get(0) != nil {
		url = args.Get(0).(*model.URL)
	}
	if args.Get(1) != nil {
		analysisResults = args.Get(1).([]*model.AnalysisResult)
	}
	if args.Get(2) != nil {
		links = args.Get(2).([]*model.Link)
	}

	return url, analysisResults, links, args.Error(3)
}

func (m *MockURLService) GetCrawlResults() <-chan crawler.CrawlResult {
	args := m.Called()
	return args.Get(0).(<-chan crawler.CrawlResult)
}

func (m *MockURLService) AdjustCrawlerWorkers(action string, count int) error {
	args := m.Called(action, count)
	return args.Error(0)
}

// setupHandler sets up a test handler with a gin engine and mocked URLService.
func setupHandler(t *testing.T) (*gin.Engine, *MockURLService) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	// Mock authentication
	r.Use(func(c *gin.Context) {
		c.Set("user_id", uint(1))
		c.Next()
	})

	// Create handler with mock service
	urlService := &MockURLService{}
	urlHandler := handler.NewURLHandler(urlService)

	// Register routes
	apiGroup := r.Group("/api")
	urlHandler.RegisterProtectedRoutes(apiGroup)

	return r, urlService
}

func TestCreate(t *testing.T) {
	r, urlService := setupHandler(t)

	// Mock service behavior
	urlService.On("Create", &model.CreateURLInputDTO{
		OriginalURL: "http://example.com",
		UserID:      uint(1),
	}).Return(uint(1), nil)

	// Create test request
	reqBody := []byte(`{"original_url":"http://example.com"}`)
	req, _ := http.NewRequest(http.MethodPost, "/api/urls", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Perform request
	r.ServeHTTP(w, req)

	// Assert response
	assert.Equal(t, http.StatusCreated, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, float64(1), response["id"])

	// Verify mock was called
	urlService.AssertExpectations(t)
}

func TestGet(t *testing.T) {
	r, urlService := setupHandler(t)

	// Mock service behavior
	urlService.On("Get", uint(1)).Return(&model.URLDTO{
		ID:          1,
		OriginalURL: "http://example.com",
		Status:      model.StatusQueued,
		UserID:      1,
	}, nil)

	// Create test request
	req, _ := http.NewRequest(http.MethodGet, "/api/urls/1", nil)
	w := httptest.NewRecorder()

	// Perform request
	r.ServeHTTP(w, req)

	// Assert response
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, float64(1), response["id"])
	assert.Equal(t, "http://example.com", response["original_url"])
	assert.Equal(t, model.StatusQueued, response["status"])

	// Verify mock was called
	urlService.AssertExpectations(t)
}

func TestList(t *testing.T) {
	r, urlService := setupHandler(t)

	// Mock service behavior
	urlService.On("List", uint(1), repository.Pagination{
		Page:     1,
		PageSize: 10,
	}).Return(&model.PaginatedResponse[model.URLDTO]{
		Data: []model.URLDTO{{
			ID:          1,
			OriginalURL: "http://example.com",
			Status:      model.StatusQueued,
			UserID:      1,
		}},
		Pagination: model.PaginationMetaDTO{
			Page:       1,
			PageSize:   10,
			TotalItems: 1,
			TotalPages: 1,
		},
	}, nil)

	// Create test request
	req, _ := http.NewRequest(http.MethodGet, "/api/urls?page=1&page_size=10", nil)
	w := httptest.NewRecorder()

	// Perform request
	r.ServeHTTP(w, req)

	// Assert response
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.NotNil(t, response["data"])
	assert.NotNil(t, response["pagination"])

	// Verify mock was called
	urlService.AssertExpectations(t)
}

func TestUpdate(t *testing.T) {
	r, urlService := setupHandler(t)

	// Mock service behavior
	urlService.On("Update", uint(1), &model.UpdateURLInput{
		OriginalURL: "http://updated-example.com",
	}).Return(nil)

	// Create test request
	reqBody := []byte(`{"original_url":"http://updated-example.com"}`)
	req, _ := http.NewRequest(http.MethodPut, "/api/urls/1", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Perform request
	r.ServeHTTP(w, req)

	// Assert response
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "updated", response["message"])

	// Verify mock was called
	urlService.AssertExpectations(t)
}

func TestDelete(t *testing.T) {
	r, urlService := setupHandler(t)

	// Mock service behavior
	urlService.On("Delete", uint(1)).Return(nil)

	// Create test request
	req, _ := http.NewRequest(http.MethodDelete, "/api/urls/1", nil)
	w := httptest.NewRecorder()

	// Perform request
	r.ServeHTTP(w, req)

	// Assert response
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "deleted", response["message"])

	// Verify mock was called
	urlService.AssertExpectations(t)
}

func TestStart(t *testing.T) {
	r, urlService := setupHandler(t)

	// Mock service behavior
	urlService.On("Start", uint(1)).Return(nil)

	// Create test request
	req, _ := http.NewRequest(http.MethodPatch, "/api/urls/1/start", nil)
	w := httptest.NewRecorder()

	// Perform request
	r.ServeHTTP(w, req)

	// Assert response
	assert.Equal(t, http.StatusAccepted, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, model.StatusQueued, response["status"])

	// Verify mock was called
	urlService.AssertExpectations(t)
}

func TestStop(t *testing.T) {
	r, urlService := setupHandler(t)

	// Mock service behavior
	urlService.On("Stop", uint(1)).Return(nil)

	// Create test request
	req, _ := http.NewRequest(http.MethodPatch, "/api/urls/1/stop", nil)
	w := httptest.NewRecorder()

	// Perform request
	r.ServeHTTP(w, req)

	// Assert response
	assert.Equal(t, http.StatusAccepted, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, model.StatusStopped, response["status"])

	// Verify mock was called
	urlService.AssertExpectations(t)
}

func TestResults(t *testing.T) {
	r, urlService := setupHandler(t)

	// Mock service behavior - the handler calls ResultsWithDetails, not Results
	urlService.On("ResultsWithDetails", uint(1)).Return(
		&model.URL{
			ID:          1,
			OriginalURL: "http://example.com",
			Status:      model.StatusDone,
			UserID:      1,
		},
		[]*model.AnalysisResult{
			{
				ID:    1,
				URLID: 1,
				Title: "Example Site",
			},
		},
		[]*model.Link{
			{
				ID:         1,
				URLID:      1,
				Href:       "http://example.com/link1",
				IsExternal: false,
			},
		},
		nil,
	)

	// Create test request
	req, _ := http.NewRequest(http.MethodGet, "/api/urls/1/results", nil)
	w := httptest.NewRecorder()

	// Perform request
	r.ServeHTTP(w, req)

	// Assert response
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	// Check the structure matches URLResultsDTO
	assert.NotNil(t, response["url"])
	assert.NotNil(t, response["analysis_results"])
	assert.NotNil(t, response["links"])

	// Verify mock was called
	urlService.AssertExpectations(t)
}
