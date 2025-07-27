package server_tests

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fuzumoe/linkTorch-api/internal/handler"
	"github.com/fuzumoe/linkTorch-api/internal/server"
	"github.com/fuzumoe/linkTorch-api/internal/service"
	"github.com/fuzumoe/linkTorch-api/tests/utils"
)

func TestRouterIntegration(t *testing.T) {

	gin.SetMode(gin.TestMode)

	db := utils.SetupTest(t)

	healthService := service.NewHealthService(db, "IntegrationTest")

	healthHandler := handler.NewHealthHandler(healthService)

	r := gin.New()

	server.RegisterRoutes(
		r,
		"test-secret",
		func(c *gin.Context) { c.Next() },
		[]server.RouteRegistrar{healthHandler},
		[]server.RouteRegistrar{},
	)

	ts := httptest.NewServer(r)
	defer ts.Close()

	t.Run("API Status Endpoint", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/api/v1/status")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err)
		assert.Equal(t, "Hello World!", result["message"])
		assert.Equal(t, "IntegrationTest", result["service"])
		assert.Equal(t, "running", result["status"])
	})
	t.Run("Handler Health Endpoint", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/api/v1/health")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err)
		assert.Equal(t, "ok", result["status"])
		assert.Equal(t, "healthy", result["database"])
		assert.Equal(t, "IntegrationTest", result["service"])
		assert.Contains(t, result, "checked")
	})

	t.Run("Route Not Found", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/non-existent")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})
}
