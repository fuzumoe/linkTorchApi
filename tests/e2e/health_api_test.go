package e2e_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	utils "github.com/fuzumoe/urlinsight-backend/tests/utils"
)

// TestStatusEndpoint_E2E tests the status endpoint of the API.
func TestStatusEndpoint_E2E(t *testing.T) {
	c := utils.NewClient()

	resp, err := c.Get(apiPath("/status"))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var body map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&body)
	require.NoError(t, err)

	assert.Equal(t, "Hello World!", body["message"])
	assert.Equal(t, "running", body["status"])
	assert.NotEmpty(t, body["service"])
}

// TestHealthEndpoint_E2E tests the health endpoint of the API.
func TestHealthEndpoint_E2E(t *testing.T) {
	c := utils.NewClient()

	resp, err := c.Get(apiPath("/health"))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Contains(t, []int{http.StatusOK, http.StatusServiceUnavailable}, resp.StatusCode)

	var body map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&body)
	require.NoError(t, err)

	assert.NotEmpty(t, body["service"])
	assert.Equal(t, "ok", body["status"])
	assert.NotEmpty(t, body["checked"])
}
