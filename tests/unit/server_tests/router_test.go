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

// RegisterRoutes implements the RouteRegistrar interface.
func (m *MockRegistrar) RegisterRoutes(rg *gin.RouterGroup) {
	m.RegisterRoutesCalled = true
	rg.GET(m.RoutePattern, m.RouteHandler)
}

func TestRegisterRoutes(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)
	r := gin.New() // using New() avoids default middleware

	// Create a mock public registrar.
	mockPublicRegistrar := &MockRegistrar{
		RoutePattern: "/test-public",
		RouteHandler: func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"result": "public-route"})
		},
	}

	// Register routes using the updated router.RegisterRoutes which provides:
	// - Public API endpoints under "/api/v1"
	// - Swagger endpoint at "/swagger/*any"
	// (No root "/" or "/health" endpoints are registered.)
	server.RegisterRoutes(
		r,
		"test-secret",
		func(c *gin.Context) { c.Next() }, // Dummy auth middleware.
		[]server.RouteRegistrar{mockPublicRegistrar},
		[]server.RouteRegistrar{}, // No protected routes for now.
	)

	// Create a test HTTP server.
	ts := httptest.NewServer(r)
	defer ts.Close()

	t.Run("Swagger Endpoint", func(t *testing.T) {
		// Query the swagger endpoint.
		resp, err := http.Get(ts.URL + "/swagger/index.html")
		assert.NoError(t, err)
		defer resp.Body.Close()
		// Expect Swagger docs to load (HTTP 200).
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("Public Route", func(t *testing.T) {
		// Request the public route under /api/v1.
		resp, err := http.Get(ts.URL + "/api/v1/test-public")
		assert.NoError(t, err)
		defer resp.Body.Close()

		// Check status code.
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Decode and verify the response.
		var result map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&result)
		assert.NoError(t, err)
		assert.Equal(t, "public-route", result["result"])

		// Verify the registrar was called.
		assert.True(t, mockPublicRegistrar.RegisterRoutesCalled)
	})

	t.Run("Root Endpoint", func(t *testing.T) {
		// Since no root endpoint is registered, expect 404.
		resp, err := http.Get(ts.URL + "/")
		assert.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("Health Endpoint", func(t *testing.T) {
		// Since no health endpoint is registered, expect 404.
		resp, err := http.Get(ts.URL + "/health")
		assert.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("Route Not Found", func(t *testing.T) {
		// Request a non-existent route.
		resp, err := http.Get(ts.URL + "/non-existent")
		assert.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})
}
