package handler_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	// Import from the actual folder "handler" but alias it as "handler"
	handler "github.com/fuzumoe/urlinsight-backend/internal/handler"
	"github.com/fuzumoe/urlinsight-backend/internal/service"
	"github.com/fuzumoe/urlinsight-backend/tests/integration"
)

func TestHealthHandlerIntegration(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Use the project test setup.
	db := integration.SetupTest(t)
	defer integration.CleanTestData(t)

	// Create a real HealthService using the live database.
	healthService := service.NewHealthService(db, "IntegrationHealthTest")

	// Create the HealthHandler with the real service.
	h := handler.NewHealthHandler(healthService)

	// Setup a Gin router with the handlers.
	router := gin.New()
	router.GET("/", h.Home)
	router.GET("/health", h.Health)

	t.Run("Home Endpoint", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)

		var resp map[string]interface{}
		err := json.Unmarshal(rec.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.Equal(t, "Hello World!", resp["message"])
		assert.Equal(t, "IntegrationHealthTest", resp["service"])
		assert.Equal(t, "running", resp["status"])
	})

	t.Run("Health Endpoint", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		// Retrieve the current health status from the service.
		status := healthService.Check()
		expectedCode := http.StatusOK
		if !status.Healthy {
			expectedCode = http.StatusServiceUnavailable
		}
		assert.Equal(t, expectedCode, rec.Code)

		var resp map[string]interface{}
		err := json.Unmarshal(rec.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.Equal(t, "IntegrationHealthTest", resp["service"])
		assert.Equal(t, "ok", resp["status"])
		assert.Equal(t, status.Database, resp["database"])

		checkedStr, ok := resp["checked"].(string)
		assert.True(t, ok, "checked timestamp should be a string")
		checkedTime, err := time.Parse(time.RFC3339, checkedStr)
		assert.NoError(t, err)
		// Ensure the checked timestamp is recent.
		assert.True(t, time.Since(checkedTime) < 5*time.Second)
	})
}
