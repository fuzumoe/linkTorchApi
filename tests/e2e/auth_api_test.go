package e2e_test

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fuzumoe/urlinsight-backend/tests/utils"
)

// TestBasicAuth_E2E tests basic authentication endpoints
func TestBasicAuth_E2E(t *testing.T) {
	client := utils.NewClient()
	email := os.Getenv("TEST_USER_EMAIL")
	require.NotEmpty(t, email, "TEST_USER_EMAIL must be set")
	pass := os.Getenv("TEST_USER_PASSWORD")
	require.NotEmpty(t, pass, "TEST_USER_PASSWORD must be set")

	// Test successful login with Basic auth
	t.Run("Successful Login", func(t *testing.T) {
		basic := base64.StdEncoding.EncodeToString([]byte(email + ":" + pass))
		req, err := http.NewRequest("POST", apiPath("/login/basic"), nil)
		require.NoError(t, err)
		req.Header.Set("Authorization", "Basic "+basic)

		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var body map[string]interface{}
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
		token, ok := body["token"].(string)
		require.True(t, ok)
		assert.NotEmpty(t, token)
	})

	// Test missing or invalid header
	t.Run("Missing or Invalid Header", func(t *testing.T) {
		resp, err := client.Post(apiPath("/login/basic"), "application/json", nil)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}

// TestJWTAuth_E2E tests JWT authentication endpoints
func TestJWTAuth_E2E(t *testing.T) {
	client := utils.NewClient()
	email := os.Getenv("TEST_USER_EMAIL")
	require.NotEmpty(t, email, "TEST_USER_EMAIL must be set")
	pass := os.Getenv("TEST_USER_PASSWORD")
	require.NotEmpty(t, pass, "TEST_USER_PASSWORD must be set")

	// Test successful login with JWT
	t.Run("Successful Login", func(t *testing.T) {
		payload := map[string]string{"email": email, "password": pass}
		bodyJSON, _ := json.Marshal(payload)

		resp, err := client.Post(apiPath("/login/jwt"), "application/json", bytes.NewReader(bodyJSON))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var body map[string]interface{}
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
		token, ok := body["token"].(string)
		require.True(t, ok)
		assert.NotEmpty(t, token)
	})

	// Test invalid credentials
	t.Run("Invalid Credentials", func(t *testing.T) {
		payload := map[string]string{"email": "no@one.com", "password": "wrong"}
		bodyJSON, _ := json.Marshal(payload)

		resp, err := client.Post(apiPath("/login/jwt"), "application/json", bytes.NewReader(bodyJSON))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}
