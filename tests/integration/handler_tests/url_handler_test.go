package handler_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/fuzumoe/urlinsight-backend/internal/handler"
	"github.com/fuzumoe/urlinsight-backend/internal/model"
	"github.com/fuzumoe/urlinsight-backend/internal/repository"
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

func (m *MockURLService) List(userID uint, p repository.Pagination) ([]*model.URLDTO, error) {
	args := m.Called(userID, p)
	return args.Get(0).([]*model.URLDTO), args.Error(1)
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

func (m *MockURLService) Results(id uint) (*model.URLDTO, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.URLDTO), args.Error(1)
}

// Implementation of the new ResultsWithDetails method that returns pointers to models
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

// Setup creates a test router with auth middleware mocked
func setupTestRouter(urlService *MockURLService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	// Add middleware to simulate authentication
	r.Use(func(c *gin.Context) {
		// Simulate authenticated user with ID 1
		c.Set("user_id", uint(1))
		c.Next()
	})

	// Create handler with mock service
	urlHandler := handler.NewURLHandler(urlService)

	// Register routes
	apiGroup := r.Group("/api")
	urlHandler.RegisterProtectedRoutes(apiGroup)

	return r
}

func TestURLHandler_Integration(t *testing.T) {
	mockService := new(MockURLService)
	router := setupTestRouter(mockService)

	t.Run("Create", func(t *testing.T) {
		// Setup service mock
		mockService.On("Create", mock.MatchedBy(func(input *model.CreateURLInputDTO) bool {
			return input.UserID == 1 && input.OriginalURL == "https://example.com"
		})).Return(uint(42), nil).Once()

		// Prepare request
		reqBody := `{"original_url": "https://example.com"}`
		req, _ := http.NewRequest("POST", "/api/urls", bytes.NewBufferString(reqBody))
		req.Header.Set("Content-Type", "application/json")

		// Execute request
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Assert response
		assert.Equal(t, http.StatusCreated, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, float64(42), response["id"])

		// Verify mock expectations
		mockService.AssertExpectations(t)
	})

	t.Run("Get", func(t *testing.T) {
		// Setup service mock
		mockService.On("Get", uint(42)).Return(&model.URLDTO{
			ID:          42,
			OriginalURL: "https://example.com",
			Status:      "done",
		}, nil).Once()

		// Prepare and execute request
		req, _ := http.NewRequest("GET", "/api/urls/42", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Assert response
		assert.Equal(t, http.StatusOK, w.Code)

		var response model.URLDTO
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, uint(42), response.ID)
		assert.Equal(t, "https://example.com", response.OriginalURL)
		assert.Equal(t, "done", response.Status)

		// Verify mock expectations
		mockService.AssertExpectations(t)
	})

	t.Run("List", func(t *testing.T) {
		// Setup service mock
		expectedPagination := repository.Pagination{Page: 1, PageSize: 10}
		mockService.On("List", uint(1), expectedPagination).Return([]*model.URLDTO{
			{ID: 1, OriginalURL: "https://example1.com", Status: "done"},
			{ID: 2, OriginalURL: "https://example2.com", Status: "queued"},
		}, nil).Once()

		// Prepare and execute request
		req, _ := http.NewRequest("GET", "/api/urls", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Assert response
		assert.Equal(t, http.StatusOK, w.Code)

		var response []*model.URLDTO
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Len(t, response, 2)
		assert.Equal(t, "https://example1.com", response[0].OriginalURL)
		assert.Equal(t, "https://example2.com", response[1].OriginalURL)

		// Verify mock expectations
		mockService.AssertExpectations(t)
	})

	t.Run("Update", func(t *testing.T) {
		// Setup service mock
		mockService.On("Update", uint(42), mock.MatchedBy(func(input *model.UpdateURLInput) bool {
			return input.OriginalURL == "https://updated.com" && input.Status == "done"
		})).Return(nil).Once()

		// Prepare request
		reqBody := `{"original_url": "https://updated.com", "status": "done"}`
		req, _ := http.NewRequest("PUT", "/api/urls/42", bytes.NewBufferString(reqBody))
		req.Header.Set("Content-Type", "application/json")

		// Execute request
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Assert response
		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "updated", response["message"])

		// Verify mock expectations
		mockService.AssertExpectations(t)
	})

	t.Run("Delete", func(t *testing.T) {
		// Setup service mock
		mockService.On("Delete", uint(42)).Return(nil).Once()

		// Prepare and execute request
		req, _ := http.NewRequest("DELETE", "/api/urls/42", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Assert response
		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "deleted", response["message"])

		// Verify mock expectations
		mockService.AssertExpectations(t)
	})

	t.Run("Start", func(t *testing.T) {
		// Setup service mock
		mockService.On("Start", uint(42)).Return(nil).Once()

		// Prepare and execute request
		req, _ := http.NewRequest("PATCH", "/api/urls/42/start", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Assert response
		assert.Equal(t, http.StatusAccepted, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, model.StatusQueued, response["status"])

		// Verify mock expectations
		mockService.AssertExpectations(t)
	})

	t.Run("Stop", func(t *testing.T) {
		// Setup service mock
		mockService.On("Stop", uint(42)).Return(nil).Once()

		// Prepare and execute request
		req, _ := http.NewRequest("PATCH", "/api/urls/42/stop", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Assert response
		assert.Equal(t, http.StatusAccepted, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, model.StatusStopped, response["status"])

		// Verify mock expectations
		mockService.AssertExpectations(t)
	})

	t.Run("Results", func(t *testing.T) {
		// Setup mock data with pointers to match ResultsWithDetails return types
		url := &model.URL{
			ID:          42,
			UserID:      1,
			OriginalURL: "https://example.com",
			Status:      "done",
		}

		analysisResults := []*model.AnalysisResult{
			{
				ID:                1,
				URLID:             42,
				HTMLVersion:       "HTML5",
				Title:             "Example Page",
				H1Count:           2,
				InternalLinkCount: 5,
				ExternalLinkCount: 3,
				BrokenLinkCount:   1,
				HasLoginForm:      true,
			},
		}

		links := []*model.Link{
			{ID: 1, URLID: 42, Href: "https://internal.com", IsExternal: false, StatusCode: 200},
			{ID: 2, URLID: 42, Href: "https://external.com", IsExternal: true, StatusCode: 200},
			{ID: 3, URLID: 42, Href: "https://broken.com", IsExternal: true, StatusCode: 404},
		}

		// Setup service mock to return these pointers
		mockService.On("ResultsWithDetails", uint(42)).Return(url, analysisResults, links, nil).Once()

		// Prepare and execute request
		req, _ := http.NewRequest("GET", "/api/urls/42/results", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Assert response
		assert.Equal(t, http.StatusOK, w.Code)

		var response model.URLResultsDTO
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		// Verify URL data
		assert.Equal(t, uint(42), response.URL.ID)
		assert.Equal(t, "https://example.com", response.URL.OriginalURL)
		assert.Equal(t, "done", response.URL.Status)

		// Verify analysis results
		require.Len(t, response.AnalysisResults, 1)
		assert.Equal(t, "HTML5", response.AnalysisResults[0].HTMLVersion)
		assert.Equal(t, "Example Page", response.AnalysisResults[0].Title)
		assert.Equal(t, 2, response.AnalysisResults[0].H1Count)
		assert.Equal(t, 5, response.AnalysisResults[0].InternalLinkCount)
		assert.Equal(t, 3, response.AnalysisResults[0].ExternalLinkCount)
		assert.Equal(t, 1, response.AnalysisResults[0].BrokenLinkCount)
		assert.True(t, response.AnalysisResults[0].HasLoginForm)

		// Verify links
		require.Len(t, response.Links, 3)

		// Create a map to verify links by href since order may vary
		linkMap := make(map[string]*model.Link)
		for _, link := range response.Links {
			linkMap[link.Href] = link
		}

		// Verify each link
		internalLink, found := linkMap["https://internal.com"]
		assert.True(t, found)
		assert.False(t, internalLink.IsExternal)
		assert.Equal(t, 200, internalLink.StatusCode)

		externalLink, found := linkMap["https://external.com"]
		assert.True(t, found)
		assert.True(t, externalLink.IsExternal)
		assert.Equal(t, 200, externalLink.StatusCode)

		brokenLink, found := linkMap["https://broken.com"]
		assert.True(t, found)
		assert.True(t, brokenLink.IsExternal)
		assert.Equal(t, 404, brokenLink.StatusCode)

		// Verify mock expectations
		mockService.AssertExpectations(t)
	})

	t.Run("Results_NotFound", func(t *testing.T) {
		// Setup service mock to return a not found error
		mockService.On("ResultsWithDetails", uint(999)).Return(nil, nil, nil, fmt.Errorf("record not found")).Once()

		// Prepare and execute request
		req, _ := http.NewRequest("GET", "/api/urls/999/results", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Assert response
		assert.Equal(t, http.StatusNotFound, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "URL not found", response["error"])

		// Verify mock expectations
		mockService.AssertExpectations(t)
	})

	t.Run("InvalidID", func(t *testing.T) {
		// Test with an invalid ID format
		req, _ := http.NewRequest("GET", "/api/urls/invalid", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Assert response
		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "invalid id", response["error"])
	})
}

// TestURLHandler_ResultsWithNilValues tests handling of nil values in ResultsWithDetails
func TestURLHandler_ResultsWithNilValues(t *testing.T) {
	mockService := new(MockURLService)
	router := setupTestRouter(mockService)

	// Setup a case where URL is returned but analysis results and links are nil
	// This can happen when a URL exists but hasn't been analyzed yet
	url := &model.URL{
		ID:          42,
		UserID:      1,
		OriginalURL: "https://example.com",
		Status:      "queued", // Not analyzed yet
	}

	mockService.On("ResultsWithDetails", uint(42)).Return(url, nil, nil, nil).Once()

	// Prepare and execute request
	req, _ := http.NewRequest("GET", "/api/urls/42/results", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert response
	assert.Equal(t, http.StatusOK, w.Code)

	var response model.URLResultsDTO
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	// Verify URL data
	assert.Equal(t, uint(42), response.URL.ID)
	assert.Equal(t, "https://example.com", response.URL.OriginalURL)
	assert.Equal(t, "queued", response.URL.Status)

	// Verify nil collections don't cause issues
	assert.Nil(t, response.AnalysisResults)
	assert.Nil(t, response.Links)

	// Verify mock expectations
	mockService.AssertExpectations(t)
}
