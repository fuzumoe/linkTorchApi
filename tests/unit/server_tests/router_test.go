package server_tests

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/fuzumoe/linkTorch-api/internal/server"
)

type MockRegistrar struct {
	RegisterRoutesCalled bool
	RoutePattern         string
	RouteHandler         gin.HandlerFunc
}

func (m *MockRegistrar) RegisterRoutes(rg *gin.RouterGroup) {
	m.RegisterRoutesCalled = true
	rg.GET(m.RoutePattern, m.RouteHandler)
}

func TestRegisterRoutes(t *testing.T) {

	gin.SetMode(gin.TestMode)
	r := gin.New()

	mockPublicRegistrar := &MockRegistrar{
		RoutePattern: "/test-public",
		RouteHandler: func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"result": "public-route"})
		},
	}

	server.RegisterRoutes(
		r,
		"test-secret",
		func(c *gin.Context) { c.Next() },
		[]server.RouteRegistrar{mockPublicRegistrar},
		[]server.RouteRegistrar{},
	)

	ts := httptest.NewServer(r)
	defer ts.Close()

	t.Run("Swagger Endpoint", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/swagger/index.html")
		assert.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("Public Route", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/api/v1/test-public")
		assert.NoError(t, err)

		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&result)
		assert.NoError(t, err)
		assert.Equal(t, "public-route", result["result"])

		assert.True(t, mockPublicRegistrar.RegisterRoutesCalled)
	})

	t.Run("Root Endpoint", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/")
		assert.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("Health Endpoint", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/health")
		assert.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("Route Not Found", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/non-existent")
		assert.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})
}
