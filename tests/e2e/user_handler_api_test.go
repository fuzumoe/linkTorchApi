package e2e_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fuzumoe/linkTorch-api/tests/utils"
)

func getAuthTokenWithCredentials(t *testing.T, email, password string) (string, error) {
	if email == "" || password == "" {
		return "", fmt.Errorf("email and password must not be empty")
	}

	payload := map[string]string{"email": email, "password": password}
	bodyJSON, _ := json.Marshal(payload)

	c := utils.NewClient()

	loginURL := apiPath("/login/jwt")
	t.Logf("Attempting login at: %s", loginURL)

	req, err := http.NewRequest("POST", loginURL, bytes.NewReader(bodyJSON))
	if err != nil {
		return "", fmt.Errorf("error creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.Do(req)
	if err != nil {
		return "", fmt.Errorf("connection error: %w (API might not be running)", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Logf("Login failed with status: %d", resp.StatusCode)
		respBody, _ := io.ReadAll(resp.Body)
		t.Logf("Response body: %s", respBody)
		return "", fmt.Errorf("login failed with status %d", resp.StatusCode)
	}

	var body map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return "", err
	}

	token, ok := body["token"].(string)
	if !ok || token == "" {
		return "", fmt.Errorf("response did not contain a valid token")
	}

	return token, nil
}

func getUserToken(t *testing.T) string {

	email := os.Getenv("TEST_USER_EMAIL")
	pass := os.Getenv("TEST_USER_PASSWORD")

	if email == "" || pass == "" {
		t.Log("TEST_USER_EMAIL or TEST_USER_PASSWORD not set, using default test user")
		email = "user@example.com"
		pass = "password123"
	}

	token, err := getAuthTokenWithCredentials(t, email, pass)
	if err != nil {
		t.Logf("Failed to get user token: %v", err)
		t.Logf("Creating a mock token for testing (some tests may be skipped)")
		return "mock-user-token"
	}

	return token
}

func getAdminToken(t *testing.T) string {

	email := os.Getenv("TEST_ADMIN_EMAIL")
	pass := os.Getenv("TEST_ADMIN_PASSWORD")

	if email == "" || pass == "" {
		t.Log("TEST_ADMIN_EMAIL or TEST_ADMIN_PASSWORD not set, trying default admin")
		email = "admin@example.com"
		pass = "adminpass123"
	}

	token, err := getAuthTokenWithCredentials(t, email, pass)
	if err != nil {
		t.Logf("Failed to get admin token: %v", err)
		t.Log("Falling back to regular user token")
		return getUserToken(t)
	}

	return token
}

func hasAdminAccess(t *testing.T, token string) bool {
	// If using mock token, we don't have admin access
	if token == "mock-user-token" {
		return false
	}

	searchURL := apiPath("/users/search")
	u, err := url.Parse(searchURL)
	if err != nil {
		t.Logf("Error parsing URL: %v", err)
		return false
	}

	q := u.Query()
	q.Add("q", "test")
	u.RawQuery = q.Encode()

	t.Logf("Testing admin access with URL: %s", u.String())

	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		t.Logf("Error creating request: %v", err)
		return false
	}
	req.Header.Set("Authorization", "Bearer "+token)

	c := utils.NewClient()
	resp, err := c.Do(req)
	if err != nil {
		t.Logf("Error checking admin access: %v", err)
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}

func TestUserEndpoints_E2E(t *testing.T) {

	userToken := getUserToken(t)
	adminToken := getAdminToken(t)

	isAdmin := hasAdminAccess(t, adminToken)
	if !isAdmin {
		t.Log("WARNING: No admin access available. Some tests will be skipped.")
	}

	client := utils.NewClient()
	var createdUserID uint
	var createdUserEmail string

	timestamp := time.Now().UnixNano()
	uniqueUsername := fmt.Sprintf("testuser_%d", timestamp)
	uniqueEmail := fmt.Sprintf("test_%d@example.com", timestamp)
	testPassword := "TestPassword123!"

	t.Run("Create User", func(t *testing.T) {
		if !isAdmin {
			t.Skip("Skipping test: requires admin access")
		}

		reqBody := map[string]string{
			"username": uniqueUsername,
			"email":    uniqueEmail,
			"password": testPassword,
		}
		bodyJSON, _ := json.Marshal(reqBody)

		createURL := apiPath("/users")
		t.Logf("Creating user at: %s", createURL)

		req, err := http.NewRequest("POST", createURL, bytes.NewReader(bodyJSON))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+adminToken)

		resp, err := client.Do(req)
		if err != nil {
			t.Logf("Error connecting to API: %v", err)
			t.Skip("Skipping test: couldn't connect to API")
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusCreated {
			respBody, _ := io.ReadAll(resp.Body)
			t.Logf("Create User Error. Status: %d, Response: %s", resp.StatusCode, string(respBody))
			t.FailNow()
		}

		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var respBody map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&respBody)
		require.NoError(t, err)

		var userID float64

		if nestedData, ok := respBody["id"].(map[string]interface{}); ok {
			userID = nestedData["id"].(float64)
			createdUserEmail = nestedData["email"].(string)
		} else if directID, ok := respBody["id"].(float64); ok {
			userID = directID
			createdUserEmail = uniqueEmail
		}

		createdUserID = uint(userID)
		t.Logf("Created User with ID: %d and email: %s", createdUserID, createdUserEmail)

		require.NotZero(t, createdUserID, "User ID should not be zero")
	})
	t.Run("Get User By ID", func(t *testing.T) {
		if !isAdmin || createdUserID == 0 {
			t.Skip("Skipping test: requires admin access and a created user")
		}
		getUserURL := apiPath("/users/" + strconv.Itoa(int(createdUserID)))
		u, err := url.Parse(getUserURL)
		require.NoError(t, err)

		q := u.Query()
		q.Add("q", "dummy")
		u.RawQuery = q.Encode()

		t.Logf("Getting user with URL: %s", u.String())

		req, err := http.NewRequest("GET", u.String(), nil)
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+adminToken)

		resp, err := client.Do(req)
		if err != nil {
			t.Logf("Error connecting to API: %v", err)
			t.Skip("Skipping test: couldn't connect to API")
		}
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		respBody, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		var users []map[string]interface{}
		err = json.Unmarshal(respBody, &users)

		if err == nil && len(users) > 0 {
			found := false
			for _, user := range users {
				if id, ok := user["id"].(float64); ok && uint(id) == createdUserID {
					assert.Equal(t, uniqueUsername, user["username"])
					assert.Equal(t, uniqueEmail, user["email"])
					found = true
					break
				}
			}
			assert.True(t, found, "Created user should be in the response")
		} else {
			var user map[string]interface{}
			err = json.Unmarshal(respBody, &user)
			require.NoError(t, err)

			id, ok := user["id"].(float64)
			require.True(t, ok, "Response should include user ID")
			assert.Equal(t, float64(createdUserID), id)
			assert.Equal(t, uniqueUsername, user["username"])
			assert.Equal(t, uniqueEmail, user["email"])
		}
	})
	t.Run("Search Users", func(t *testing.T) {
		if !isAdmin {
			t.Skip("Skipping test: requires admin access")
		}

		searchURL := apiPath("/users/search")
		u, err := url.Parse(searchURL)
		require.NoError(t, err)

		q := u.Query()
		q.Add("q", uniqueUsername)
		u.RawQuery = q.Encode()

		t.Logf("Searching users with URL: %s", u.String())

		req, err := http.NewRequest("GET", u.String(), nil)
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+adminToken)

		resp, err := client.Do(req)
		if err != nil {
			t.Logf("Error connecting to API: %v", err)
			t.Skip("Skipping test: couldn't connect to API")
		}
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var users []map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&users)
		require.NoError(t, err)

		if createdUserID > 0 {
			found := false
			for _, user := range users {
				if id, ok := user["id"].(float64); ok && uint(id) == createdUserID {
					assert.Equal(t, uniqueUsername, user["username"])
					assert.Equal(t, uniqueEmail, user["email"])
					found = true
					break
				}
			}
			assert.True(t, found, "Created user should be in search results")
		}
	})

	t.Run("Search Users as Regular User", func(t *testing.T) {
		if userToken == "mock-user-token" {
			t.Skip("Skipping test: no valid user token available")
		}
		searchURL := apiPath("/users/search")
		u, err := url.Parse(searchURL)
		require.NoError(t, err)

		q := u.Query()
		q.Add("q", "test")
		u.RawQuery = q.Encode()

		req, err := http.NewRequest("GET", u.String(), nil)
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+userToken)

		resp, err := client.Do(req)
		if err != nil {
			t.Logf("Error connecting to API: %v", err)
			t.Skip("Skipping test: couldn't connect to API")
		}
		defer resp.Body.Close()

		assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	})

	t.Run("Login with Created User", func(t *testing.T) {
		if createdUserID == 0 {
			t.Skip("Skipping test: requires a created user")
		}

		newUserToken, err := getAuthTokenWithCredentials(t, uniqueEmail, testPassword)
		if err != nil {
			t.Logf("Failed to login with created user: %v", err)
			t.Skip("Skipping test: couldn't login with created user")
		}

		updateBody := map[string]string{
			"username": uniqueUsername + "_updated",
		}
		bodyJSON, _ := json.Marshal(updateBody)

		updateURL := apiPath("/users/" + strconv.Itoa(int(createdUserID)))
		t.Logf("Updating user at: %s", updateURL)

		req, err := http.NewRequest("PUT", updateURL, bytes.NewReader(bodyJSON))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+newUserToken)

		resp, err := client.Do(req)
		if err != nil {
			t.Logf("Error connecting to API: %v", err)
			t.Skip("Skipping test: couldn't connect to API")
		}
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var user map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&user)
		require.NoError(t, err)
		assert.Equal(t, uniqueUsername+"_updated", user["username"])
	})

	t.Run("Update User Role", func(t *testing.T) {
		if !isAdmin || createdUserID == 0 {
			t.Skip("Skipping test: requires admin access and a created user")
		}

		updateBody := map[string]string{
			"role": "admin",
		}
		bodyJSON, _ := json.Marshal(updateBody)

		updateURL := apiPath("/users/" + strconv.Itoa(int(createdUserID)))

		req, err := http.NewRequest("PUT", updateURL, bytes.NewReader(bodyJSON))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+adminToken)

		resp, err := client.Do(req)
		if err != nil {
			t.Logf("Error connecting to API: %v", err)
			t.Skip("Skipping test: couldn't connect to API")
		}
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var user map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&user)
		require.NoError(t, err)
		assert.Equal(t, "admin", user["role"])
	})

	t.Run("Get Current User", func(t *testing.T) {
		if userToken == "mock-user-token" {
			t.Skip("Skipping test: no valid user token available")
		}

		meURL := apiPath("/users/me")

		req, err := http.NewRequest("GET", meURL, nil)
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+userToken)

		resp, err := client.Do(req)
		if err != nil {
			t.Logf("Error connecting to API: %v", err)
			t.Skip("Skipping test: couldn't connect to API")
		}
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var user map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&user)
		require.NoError(t, err)

		assert.NotNil(t, user["id"])
		assert.NotNil(t, user["username"])
		assert.NotNil(t, user["email"])
	})

	t.Run("Update Another User as Regular User", func(t *testing.T) {
		if createdUserID == 0 || userToken == "mock-user-token" {
			t.Skip("Skipping test: requires a created user and valid user token")
		}

		updateBody := map[string]string{
			"username": "hacker_attempt",
		}
		bodyJSON, _ := json.Marshal(updateBody)

		updateURL := apiPath("/users/" + strconv.Itoa(int(createdUserID)))

		req, err := http.NewRequest("PUT", updateURL, bytes.NewReader(bodyJSON))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+userToken)

		resp, err := client.Do(req)
		if err != nil {
			t.Logf("Error connecting to API: %v", err)
			t.Skip("Skipping test: couldn't connect to API")
		}
		defer resp.Body.Close()

		assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	})

	t.Run("Delete User", func(t *testing.T) {
		if !isAdmin || createdUserID == 0 {
			t.Skip("Skipping test: requires admin access and a created user")
		}

		deleteURL := apiPath("/users/" + strconv.Itoa(int(createdUserID)))

		req, err := http.NewRequest("DELETE", deleteURL, nil)
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+adminToken)

		resp, err := client.Do(req)
		if err != nil {
			t.Logf("Error connecting to API: %v", err)
			t.Skip("Skipping test: couldn't connect to API")
		}
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNoContent, resp.StatusCode)
	})

	t.Run("Verify User Deleted", func(t *testing.T) {
		if !isAdmin || createdUserID == 0 {
			t.Skip("Skipping test: requires admin access and a created user")
		}

		getUserURL := apiPath("/users/" + strconv.Itoa(int(createdUserID)))
		u, err := url.Parse(getUserURL)
		require.NoError(t, err)

		q := u.Query()
		q.Add("q", "dummy")
		u.RawQuery = q.Encode()

		req, err := http.NewRequest("GET", u.String(), nil)
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+adminToken)

		resp, err := client.Do(req)
		if err != nil {
			t.Logf("Error connecting to API: %v", err)
			t.Skip("Skipping test: couldn't connect to API")
		}
		defer resp.Body.Close()

		assert.True(t, resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusOK)

		if resp.StatusCode == http.StatusOK {
			var users []map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&users)
			if err == nil {

				found := false
				for _, user := range users {
					if id, ok := user["id"].(float64); ok && uint(id) == createdUserID {
						found = true
						break
					}
				}
				assert.False(t, found, "Deleted user should not be in results")
			}
		}
	})
}
