package handler_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/fuzumoe/linkTorch-api/internal/handler"
	"github.com/fuzumoe/linkTorch-api/internal/service"
	"github.com/fuzumoe/linkTorch-api/tests/utils"
)

func TestHealthHandlerIntegration(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db := utils.SetupTest(t)
	defer utils.CleanTestData(t)

	healthService := service.NewHealthService(db, "IntegrationHealthTest")

	h := handler.NewHealthHandler(healthService)

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

		assert.True(t, time.Since(checkedTime) < 5*time.Second)
	})
}
