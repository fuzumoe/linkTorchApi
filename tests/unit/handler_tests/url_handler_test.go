package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fuzumoe/urlinsight-backend/internal/handler"
	"github.com/fuzumoe/urlinsight-backend/internal/model"
	"github.com/fuzumoe/urlinsight-backend/internal/repository"
)

// dummyURLService is a dummy implementation of service.URLService for testing.
type dummyURLService struct{}

func (s *dummyURLService) Create(in *model.CreateURLInputDTO) (uint, error) {
	return 1, nil
}

func (s *dummyURLService) Get(id uint) (*model.URLDTO, error) {
	return &model.URLDTO{
		ID:          id,
		OriginalURL: "http://example.com",
		Status:      model.StatusQueued,
		UserID:      1,
	}, nil
}

func (s *dummyURLService) List(userID uint, p repository.Pagination) ([]*model.URLDTO, error) {
	return []*model.URLDTO{{
		ID:          1,
		OriginalURL: "http://example.com",
		Status:      model.StatusQueued,
		UserID:      userID,
	}}, nil
}

func (s *dummyURLService) Update(id uint, in *model.UpdateURLInput) error {
	return nil
}

func (s *dummyURLService) Delete(id uint) error {
	return nil
}

func (s *dummyURLService) Start(id uint) error {
	return nil
}

func (s *dummyURLService) Stop(id uint) error {
	return nil
}

func (s *dummyURLService) Results(id uint) (*model.URLDTO, error) {
	return &model.URLDTO{
		ID:          id,
		OriginalURL: "http://example.com",
		Status:      model.StatusDone,
		UserID:      1,
	}, nil
}

// ResultsWithDetails returns the raw URL with details needed by the Results handler.
func (s *dummyURLService) ResultsWithDetails(id uint) (*model.URL, []*model.AnalysisResult, []*model.Link, error) {
	return &model.URL{
		ID:          id,
		UserID:      1,
		OriginalURL: "http://example.com/results",
		Status:      model.StatusDone,
	}, []*model.AnalysisResult{}, []*model.Link{}, nil
}

// setupRouter returns a new Gin engine in test mode.
func setupRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	return gin.New()
}

func TestURLHandler(t *testing.T) {
	svc := &dummyURLService{}
	h := handler.NewURLHandler(svc)
	router := setupRouter()

	// Register testing endpoints.
	// For endpoints that require user auth, we simulate it by setting "user_id" in the context.
	router.POST("/api/urls", func(c *gin.Context) {
		c.Set("user_id", uint(1))
		h.Create(c)
	})
	router.GET("/api/urls", func(c *gin.Context) {
		c.Set("user_id", uint(1))
		h.List(c)
	})
	router.GET("/api/urls/:id", h.Get)
	router.PUT("/api/urls/:id", h.Update)
	router.DELETE("/api/urls/:id", h.Delete)
	router.PATCH("/api/urls/:id/start", h.Start)
	router.PATCH("/api/urls/:id/stop", h.Stop)
	router.GET("/api/urls/:id/results", h.Results)

	t.Run("Create", func(t *testing.T) {
		input := model.URLCreateRequestDTO{
			OriginalURL: "http://example.com",
		}
		jsonInput, err := json.Marshal(input)
		require.NoError(t, err)

		req, err := http.NewRequest("POST", "/api/urls", bytes.NewBuffer(jsonInput))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Expect HTTP 201 Created.
		assert.Equal(t, http.StatusCreated, w.Code)

		// Decode and check response.
		var resp map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		id, ok := resp["id"].(float64)
		require.True(t, ok, "response id not a number")
		assert.Equal(t, float64(1), id)
	})

	t.Run("List", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/api/urls?page=1&page_size=10", nil)
		require.NoError(t, err)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var dtos []*model.URLDTO
		err = json.Unmarshal(w.Body.Bytes(), &dtos)
		require.NoError(t, err)
		require.Len(t, dtos, 1)
		assert.Equal(t, "http://example.com", dtos[0].OriginalURL)
	})

	t.Run("Get", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/api/urls/1", nil)
		require.NoError(t, err)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var dto model.URLDTO
		err = json.Unmarshal(w.Body.Bytes(), &dto)
		require.NoError(t, err)
		assert.Equal(t, uint(1), dto.ID)
	})

	t.Run("Update", func(t *testing.T) {
		input := model.UpdateURLInput{
			Status: model.StatusDone,
		}
		jsonInput, err := json.Marshal(input)
		require.NoError(t, err)

		req, err := http.NewRequest("PUT", "/api/urls/1", bytes.NewBuffer(jsonInput))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp map[string]string
		err = json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "updated", resp["message"])
	})

	t.Run("Delete", func(t *testing.T) {
		req, err := http.NewRequest("DELETE", "/api/urls/1", nil)
		require.NoError(t, err)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp map[string]string
		err = json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "deleted", resp["message"])
	})

	t.Run("Start", func(t *testing.T) {
		req, err := http.NewRequest("PATCH", "/api/urls/1/start", nil)
		require.NoError(t, err)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusAccepted, w.Code)
		var resp map[string]string
		err = json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		// Expect the status to be returned as "queued"
		assert.Equal(t, model.StatusQueued, resp["status"])
	})

	t.Run("Stop", func(t *testing.T) {
		req, err := http.NewRequest("PATCH", "/api/urls/1/stop", nil)
		require.NoError(t, err)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusAccepted, w.Code)
		var resp map[string]string
		err = json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		// Expect the status to be returned as "stopped"
		assert.Equal(t, model.StatusStopped, resp["status"])
	})

	t.Run("Results", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/api/urls/1/results", nil)
		require.NoError(t, err)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var dto model.URLResultsDTO
		err = json.Unmarshal(w.Body.Bytes(), &dto)
		require.NoError(t, err)
		// Check that the URL inside the response has status "done" as returned by ResultsWithDetails.
		assert.Equal(t, model.StatusDone, dto.URL.Status)
	})
}
