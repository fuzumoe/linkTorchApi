package e2e_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	utils "github.com/fuzumoe/linkTorch-api/tests/utils"
)

// TestHealthAPI_E2E tests both health-related endpoints of the API.
func TestHealthAPI_E2E(t *testing.T) {
	client := utils.NewClient()

	// Test status endpoint
	t.Run("Status Endpoint", func(t *testing.T) {
		resp, err := client.Get(apiPath("/status"))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var body map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&body)
		require.NoError(t, err)

		assert.Equal(t, "Hello World!", body["message"])
		assert.Equal(t, "running", body["status"])
		assert.NotEmpty(t, body["service"])
	})

	// Test health endpoint
	t.Run("Health Endpoint", func(t *testing.T) {
		resp, err := client.Get(apiPath("/health"))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Contains(t, []int{http.StatusOK, http.StatusServiceUnavailable}, resp.StatusCode)

		var body map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&body)
		require.NoError(t, err)

		assert.NotEmpty(t, body["service"])
		assert.Equal(t, "ok", body["status"])
		assert.NotEmpty(t, body["checked"])
	})

}
