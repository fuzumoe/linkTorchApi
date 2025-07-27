package e2e_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fuzumoe/linkTorch-api/internal/model"
	"github.com/fuzumoe/linkTorch-api/tests/utils"
)

// Helper to get authorization token
func getAuthToken(t *testing.T) string {
	email := os.Getenv("TEST_USER_EMAIL")
	require.NotEmpty(t, email, "TEST_USER_EMAIL must be set")
	pass := os.Getenv("TEST_USER_PASSWORD")
	require.NotEmpty(t, pass, "TEST_USER_PASSWORD must be set")

	payload := map[string]string{"email": email, "password": pass}
	bodyJSON, _ := json.Marshal(payload)

	c := utils.NewClient()
	resp, err := c.Post(apiPath("/login/jwt"), "application/json", bytes.NewReader(bodyJSON))
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode, "Login should succeed")

	var body map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	token, ok := body["token"].(string)
	require.True(t, ok, "Response should contain token")
	require.NotEmpty(t, token, "Token should not be empty")

	return token
}

// TestURLEndpoints_E2E tests all URL endpoints in a single test function
func TestURLEndpoints_E2E(t *testing.T) {
	// Get auth token for all requests
	token := getAuthToken(t)
	client := utils.NewClient()

	var createdURLID uint

	// Generate a unique URL to avoid the duplicate entry error
	uniqueURL := fmt.Sprintf("https://example-%d.com", time.Now().UnixNano())

	// Create URL Test
	t.Run("Create URL", func(t *testing.T) {
		reqBody := map[string]string{
			"original_url": uniqueURL,
		}
		bodyJSON, _ := json.Marshal(reqBody)

		req, err := http.NewRequest("POST", apiPath("/urls"), bytes.NewReader(bodyJSON))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		// Print response body if status code is not as expected
		if resp.StatusCode != http.StatusCreated {
			var errResp map[string]interface{}
			_ = json.NewDecoder(resp.Body).Decode(&errResp)
			t.Logf("Create URL Error response: %v", errResp)
			t.FailNow()
		}

		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var respBody map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&respBody)
		require.NoError(t, err)
		idFloat, ok := respBody["id"].(float64)
		require.True(t, ok, "Response should include an ID")
		createdURLID = uint(idFloat)
		t.Logf("Created URL with ID: %d", createdURLID)
	})

	// List URLs Test
	t.Run("List URLs", func(t *testing.T) {
		req, err := http.NewRequest("GET", apiPath("/urls?page=1&page_size=10"), nil)
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var response struct {
			Data       []model.URLDTO          `json:"data"`
			Pagination model.PaginationMetaDTO `json:"pagination"`
		}
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)

		// Verify pagination metadata
		assert.Equal(t, 1, response.Pagination.Page)
		assert.Equal(t, 10, response.Pagination.PageSize)
		assert.GreaterOrEqual(t, response.Pagination.TotalItems, 1)
		assert.GreaterOrEqual(t, response.Pagination.TotalPages, 1)

		// Verify data
		assert.GreaterOrEqual(t, len(response.Data), 1, "Should have at least one URL")
	})

	// Get URL Test
	t.Run("Get URL", func(t *testing.T) {
		req, err := http.NewRequest("GET", apiPath("/urls/"+strconv.Itoa(int(createdURLID))), nil)
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var dto model.URLDTO
		err = json.NewDecoder(resp.Body).Decode(&dto)
		require.NoError(t, err)
		assert.Equal(t, createdURLID, dto.ID)
		assert.Equal(t, uniqueURL, dto.OriginalURL)
	})

	// Update URL Test
	t.Run("Update URL", func(t *testing.T) {
		// Generate another unique URL for the update
		updatedURL := fmt.Sprintf("https://updated-%d.com", time.Now().UnixNano())

		updateBody := map[string]interface{}{
			"original_url": updatedURL,
			"status":       model.StatusRunning,
		}
		bodyJSON, _ := json.Marshal(updateBody)

		req, err := http.NewRequest("PUT", apiPath("/urls/"+strconv.Itoa(int(createdURLID))), bytes.NewReader(bodyJSON))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var respBody map[string]string
		err = json.NewDecoder(resp.Body).Decode(&respBody)
		require.NoError(t, err)
		assert.Equal(t, "updated", respBody["message"])
	})

	// Start Crawl Test
	t.Run("Start Crawl", func(t *testing.T) {
		req, err := http.NewRequest("PATCH", apiPath("/urls/"+strconv.Itoa(int(createdURLID))+"/start"), nil)
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusAccepted, resp.StatusCode)

		var respBody map[string]string
		err = json.NewDecoder(resp.Body).Decode(&respBody)
		require.NoError(t, err)
		assert.Equal(t, model.StatusQueued, respBody["status"])
	})

	// Stop Crawl Test
	t.Run("Stop Crawl", func(t *testing.T) {
		req, err := http.NewRequest("PATCH", apiPath("/urls/"+strconv.Itoa(int(createdURLID))+"/stop"), nil)
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusAccepted, resp.StatusCode)

		var respBody map[string]string
		err = json.NewDecoder(resp.Body).Decode(&respBody)
		require.NoError(t, err)
		assert.Equal(t, model.StatusStopped, respBody["status"])
	})

	// Get Results Test
	t.Run("Get Results", func(t *testing.T) {
		// Allow some time for potential asynchronous analysis
		time.Sleep(100 * time.Millisecond)

		req, err := http.NewRequest("GET", apiPath("/urls/"+strconv.Itoa(int(createdURLID))+"/results"), nil)
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			var dto model.URLResultsDTO
			err = json.NewDecoder(resp.Body).Decode(&dto)
			require.NoError(t, err)
			assert.Equal(t, createdURLID, dto.URL.ID)
		} else {
			assert.Equal(t, http.StatusNotFound, resp.StatusCode)
		}
	})

	// Delete URL Test
	t.Run("Delete URL", func(t *testing.T) {
		req, err := http.NewRequest("DELETE", apiPath("/urls/"+strconv.Itoa(int(createdURLID))), nil)
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var respBody map[string]string
		err = json.NewDecoder(resp.Body).Decode(&respBody)
		require.NoError(t, err)
		assert.Equal(t, "deleted", respBody["message"])
	})
}
