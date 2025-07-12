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

func TestLoginBasic_Success(t *testing.T) {
	email := os.Getenv("TEST_USER_EMAIL")
	require.NotEmpty(t, email, "TEST_USER_EMAIL must be set")
	pass := os.Getenv("TEST_USER_PASSWORD")
	require.NotEmpty(t, pass, "TEST_USER_PASSWORD must be set")

	c := utils.NewClient()
	basic := base64.StdEncoding.EncodeToString([]byte(email + ":" + pass))
	req, err := http.NewRequest("POST", apiPath("/login/basic"), nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Basic "+basic)

	resp, err := c.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var body map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	token, ok := body["token"].(string)
	require.True(t, ok)
	assert.NotEmpty(t, token)
}

func TestLoginBasic_MissingOrInvalidHeader(t *testing.T) {
	c := utils.NewClient()
	resp, err := c.Post(apiPath("/login/basic"), "application/json", nil)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestLoginJWT_Success(t *testing.T) {
	email := os.Getenv("TEST_USER_EMAIL")
	require.NotEmpty(t, email)
	pass := os.Getenv("TEST_USER_PASSWORD")
	require.NotEmpty(t, pass)

	payload := map[string]string{"email": email, "password": pass}
	bodyJSON, _ := json.Marshal(payload)

	c := utils.NewClient()
	resp, err := c.Post(apiPath("/login/jwt"), "application/json", bytes.NewReader(bodyJSON))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var body map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	token, ok := body["token"].(string)
	require.True(t, ok)
	assert.NotEmpty(t, token)
}

func TestLoginJWT_InvalidCreds(t *testing.T) {
	payload := map[string]string{"email": "no@one.com", "password": "wrong"}
	bodyJSON, _ := json.Marshal(payload)

	c := utils.NewClient()
	resp, err := c.Post(apiPath("/login/jwt"), "application/json", bytes.NewReader(bodyJSON))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}
