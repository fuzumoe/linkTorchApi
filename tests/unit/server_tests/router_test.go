package server_tests

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/fuzumoe/urlinsight-backend/internal/server"
)

// MockRegistrar is a mock implementation of RouteRegistrar
type MockRegistrar struct {
	RegisterRoutesCalled bool
	RoutePattern         string
	RouteHandler         gin.HandlerFunc
}

// RegisterRoutes implements the RouteRegistrar interface
func (m *MockRegistrar) RegisterRoutes(rg *gin.RouterGroup) {
	m.RegisterRoutesCalled = true
	rg.GET(m.RoutePattern, m.RouteHandler)
}

func TestRegisterRoutes(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)
	r := gin.New() // Use New() instead of Default() to avoid middleware

	// Create mock registrars
	mockPublicRegistrar := &MockRegistrar{
		RoutePattern: "/test-public",
		RouteHandler: func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"result": "public-route"})
		},
	}

	// Register routes
	server.RegisterRoutes(
		r,
		"test-secret",
		[]server.RouteRegistrar{mockPublicRegistrar},
		[]server.RouteRegistrar{}, // No protected routes for now
	)

	// Create a test HTTP server
	ts := httptest.NewServer(r)
	defer ts.Close()

	// Test cases
	t.Run("Root Endpoint", func(t *testing.T) {
		// Make a request to the root endpoint
		resp, err := http.Get(ts.URL + "/")
		assert.NoError(t, err)
		defer resp.Body.Close()

		// Check status code
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Check response body
		var result map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&result)
		assert.NoError(t, err)
		assert.Equal(t, "Welcome to URL Insight Backend!", result["message"])
	})

	t.Run("Health Endpoint", func(t *testing.T) {
		// Make a request to the health endpoint
		resp, err := http.Get(ts.URL + "/health")
		assert.NoError(t, err)
		defer resp.Body.Close()

		// Check status code
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Check response body
		var result map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&result)
		assert.NoError(t, err)
		assert.Equal(t, "ok", result["status"])
	})

	t.Run("Public Route", func(t *testing.T) {
		// Make a request to the public route
		resp, err := http.Get(ts.URL + "/api/v1/test-public")
		assert.NoError(t, err)
		defer resp.Body.Close()

		// Check status code
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Check response body
		var result map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&result)
		assert.NoError(t, err)
		assert.Equal(t, "public-route", result["result"])

		// Verify the registrar was called
		assert.True(t, mockPublicRegistrar.RegisterRoutesCalled)
	})

	t.Run("Route Not Found", func(t *testing.T) {
		// Make a request to a non-existent route
		resp, err := http.Get(ts.URL + "/non-existent")
		assert.NoError(t, err)
		defer resp.Body.Close()

		// Check status code - should be 404 Not Found
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})
}
