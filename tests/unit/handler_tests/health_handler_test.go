package handler_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/fuzumoe/urlinsight-backend/internal/handler"
	"github.com/fuzumoe/urlinsight-backend/internal/service"
)

// dummyHealthService implements service.HealthService for unit testing.
type dummyHealthService struct {
	response *service.HealthStatus
}

func (d *dummyHealthService) Check() *service.HealthStatus {
	return d.response
}

func TestHealthHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("Home Endpoint", func(t *testing.T) {
		// Create a dummy health service returning a healthy status.
		dummy := &dummyHealthService{
			response: &service.HealthStatus{
				Service:  "TestService",
				Database: "healthy",
				Healthy:  true,
				Checked:  time.Now().UTC(),
			},
		}

		h := handler.NewHealthHandler(dummy)
		router := gin.New()
		router.GET("/", h.Home)

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)

		var resp map[string]interface{}
		err := json.Unmarshal(rec.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.Equal(t, "Hello World!", resp["message"])
		assert.Equal(t, "TestService", resp["service"])
		assert.Equal(t, "running", resp["status"])
	})

	testHealthEndpoint := func(t *testing.T, dbStatus string, healthy bool, expectedCode int) {
		dummy := &dummyHealthService{
			response: &service.HealthStatus{
				Service:  "TestService",
				Database: dbStatus,
				Healthy:  healthy,
				Checked:  time.Now().UTC(),
			},
		}
		h := handler.NewHealthHandler(dummy)
		router := gin.New()
		router.GET("/health", h.Health)

		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, expectedCode, rec.Code)

		var resp map[string]interface{}
		err := json.Unmarshal(rec.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.Equal(t, "TestService", resp["service"])
		assert.Equal(t, "ok", resp["status"])
		assert.Equal(t, dbStatus, resp["database"])
		assert.NotEmpty(t, resp["checked"])
	}

	t.Run("Health Endpoint Healthy", func(t *testing.T) {
		testHealthEndpoint(t, "healthy", true, http.StatusOK)
	})

	t.Run("Health Endpoint Unhealthy", func(t *testing.T) {
		testHealthEndpoint(t, "unhealthy", false, http.StatusServiceUnavailable)
	})
}
