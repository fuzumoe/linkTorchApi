package server_tests

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fuzumoe/urlinsight-backend/internal/handler"
	"github.com/fuzumoe/urlinsight-backend/internal/server"
	"github.com/fuzumoe/urlinsight-backend/internal/service"
	"github.com/fuzumoe/urlinsight-backend/tests/integration"
)

func TestRouterIntegration(t *testing.T) {
	// Set up test mode
	gin.SetMode(gin.TestMode)

	// Set up test database
	db := integration.SetupTest(t)
	defer integration.CleanTestData(t)

	// Create real services
	healthService := service.NewHealthService(db, "IntegrationTest")

	// Create real handlers
	healthHandler := handler.NewHealthHandler(healthService)

	// Create a new router
	r := gin.New()

	// Register routes with real handlers
	server.RegisterRoutes(
		r,
		"test-secret",
		[]server.RouteRegistrar{healthHandler}, // Use real health handler
		[]server.RouteRegistrar{},              // No protected routes for this test
	)

	// Create a test HTTP server
	ts := httptest.NewServer(r)
	defer ts.Close()

	// Test the root endpoint
	t.Run("Root Endpoint", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err)
		assert.Equal(t, "Welcome to URL Insight Backend!", result["message"])
	})

	// Test the health endpoint provided by the router itself
	t.Run("Router Health Endpoint", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/health")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err)
		assert.Equal(t, "ok", result["status"])
	})

	// Test the health endpoint provided by the health handler
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

	// Test the home endpoint provided by the health handler
	t.Run("Handler Home Endpoint", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/api/v1/")
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

	// Test non-existent route
	t.Run("Route Not Found", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/non-existent")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})
}
