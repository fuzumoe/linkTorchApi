package app_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"github.com/fuzumoe/urlinsight-backend/configs"
	"github.com/fuzumoe/urlinsight-backend/internal/app"
	"github.com/fuzumoe/urlinsight-backend/tests/integration"
)

// Helper function to check if a string contains any of the given values.
func contains(s string, values []string) bool {
	for _, v := range values {
		if strings.Contains(s, v) {
			return true
		}
	}
	return false
}

func TestAppRun_Integration(t *testing.T) {
	// Use test mode to reduce noise in logs.
	gin.SetMode(gin.TestMode)

	// Setup test database.
	_ = integration.SetupTest(t)

	// Get database connection details from environment variables.
	dbHost := os.Getenv("DB_HOST")
	if dbHost == "" {
		dbHost = "localhost"
	}

	dbPort := os.Getenv("DB_PORT")
	if dbPort == "" {
		dbPort = "3309"
	}

	dbUser := os.Getenv("DB_USER")
	if dbUser == "" {
		dbUser = "urlinsight_user"
	}

	dbPassword := os.Getenv("DB_PASSWORD")
	if dbPassword == "" {
		dbPassword = "secret"
	}

	// Use the test database name.
	dbName := "urlinsight_test"

	// Construct the DSN.
	testDBDSN := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true",
		dbUser, dbPassword, dbHost, dbPort, dbName)
	t.Logf("Using test DSN: %s", testDBDSN)

	_ = integration.SetupTest(t)
	defer integration.CleanTestData(t)

	t.Run("AppRun_Integration", func(t *testing.T) {
		// Create a listener on a random port.
		listener, err := net.Listen("tcp", "localhost:0")
		require.NoError(t, err)
		port := listener.Addr().(*net.TCPAddr).Port
		listener.Close() // Close it so our app can use it.

		t.Logf("Test using port: %d", port)

		// Mock the config loader to use our test database and the available port.
		patches := gomonkey.ApplyFunc(app.LoadConfig, func() (*configs.Config, error) {
			return &configs.Config{
				DatabaseURL: testDBDSN, // Use the constructed DSN.
				JWTSecret:   "test-secret",
				ServerHost:  "127.0.0.1", // Use loopback explicitly.
				ServerPort:  fmt.Sprintf("%d", port),
			}, nil
		})
		defer patches.Reset()

		// Start the app in a separate goroutine.
		errChan := make(chan error, 1)
		go func() {
			t.Log("Starting app.Run()...")
			err := app.Run()
			t.Logf("app.Run() returned with error: %v", err)
			errChan <- err
		}()

		// Check if the app failed to start (give it a moment).
		select {
		case err := <-errChan:
			t.Fatalf("App failed to start: %v", err)
		case <-time.After(100 * time.Millisecond):
			// App didn't return an immediate error, good.
		}

		// Give the app more time to fully initialize and start listening.
		t.Log("Waiting for app to start...")
		time.Sleep(3 * time.Second)

		// Try different health endpoint paths with preference for your API structure.
		healthPaths := []string{
			"/api/v1/status", // Status endpoint.
			"/api/v1/health", // Health endpoint.
		}

		// Track results for each endpoint.
		type EndpointResult struct {
			Path       string
			StatusCode int
			Response   map[string]interface{}
			Success    bool
		}
		endpointResults := make([]EndpointResult, 0, len(healthPaths))

		// Test all health endpoints.
		for _, path := range healthPaths {
			result := EndpointResult{
				Path:    path,
				Success: false,
			}

			healthURL := fmt.Sprintf("http://127.0.0.1:%d%s", port, path)
			t.Logf("Testing health endpoint at %s", healthURL)

			client := &http.Client{
				Timeout: 5 * time.Second, // Add timeout to prevent hanging.
			}

			// Try up to 3 times with backoff.
			var resp *http.Response
			var connErr error

			for attempt := 0; attempt < 3; attempt++ {
				resp, connErr = client.Get(healthURL)
				if connErr == nil {
					break
				}
				t.Logf("Connection attempt %d failed: %v", attempt+1, connErr)
				time.Sleep(time.Duration(attempt+1) * 500 * time.Millisecond)
			}

			if connErr != nil {
				t.Logf("Could not connect to %s after retries: %v", path, connErr)
				endpointResults = append(endpointResults, result)
				continue
			}

			// Record status code.
			result.StatusCode = resp.StatusCode

			// Check status code - consider 200-299 as success.
			if resp.StatusCode < 200 || resp.StatusCode > 299 {
				t.Logf("Non-success status code from %s: %d", path, resp.StatusCode)
				resp.Body.Close()
				endpointResults = append(endpointResults, result)
				continue
			}

			// Try to parse as JSON.
			body, err := io.ReadAll(resp.Body)
			resp.Body.Close()
			if err != nil {
				t.Logf("Error reading response body: %v", err)
				// Even with read error, we found a working endpoint.
				result.Success = true
				endpointResults = append(endpointResults, result)
				continue
			}

			// Log raw response for debugging.
			t.Logf("Raw response from %s: %s", path, string(body))

			var responseData map[string]interface{}
			err = json.Unmarshal(body, &responseData)
			if err != nil {
				// Not JSON, check if it's a simple text response.
				bodyText := string(body)
				t.Logf("Response from %s (not JSON): %s", path, bodyText)

				// Simple health checks might just return "OK" or similar.
				if strings.Contains(strings.ToLower(bodyText), "ok") ||
					strings.Contains(strings.ToLower(bodyText), "healthy") {
					result.Success = true
					t.Logf("Found health endpoint at %s with text response", path)
				}
			} else {
				// Successfully parsed JSON.
				t.Logf("Found health endpoint at %s with JSON response: %v", path, responseData)
				result.Response = responseData
				result.Success = true
			}

			endpointResults = append(endpointResults, result)
		}

		// Check results for all endpoints.
		successCount := 0
		for _, result := range endpointResults {
			if result.Success {
				successCount++
				t.Logf("Endpoint %s is working (status: %d)", result.Path, result.StatusCode)

				// Check for status fields in the response.
				if result.Response != nil {
					for _, field := range []string{"status", "health", "state", "result", "message"} {
						if status, exists := result.Response[field]; exists {
							t.Logf("Found field '%s' with value: %v in %s", field, status, result.Path)

							// Verify status value looks valid.
							statusStr := strings.ToLower(fmt.Sprintf("%v", status))
							if !contains(statusStr, []string{"ok", "up", "healthy", "alive", "running", "hello"}) {
								t.Logf("Warning: Unexpected status value in %s: %s", result.Path, statusStr)
							}
						}
					}
				}
			} else {
				t.Logf("Endpoint %s is NOT working", result.Path)
			}
		}

		// Verify at least one endpoint is working.
		require.GreaterOrEqual(t, successCount, 1, "At least one health endpoint should be working")

		// Verify specific endpoints are working.
		statusFound := false
		healthFound := false
		for _, result := range endpointResults {
			if result.Path == "/api/v1/status" && result.Success {
				statusFound = true
			}
			if result.Path == "/api/v1/health" && result.Success {
				healthFound = true
			}
		}
		require.True(t, statusFound, "Status endpoint /api/v1/status should be working")
		require.True(t, healthFound, "Health endpoint /api/v1/health should be working")

		t.Log("Test completed successfully")
	})
	integration.CleanTestData(t) // Clean up test data after the test completes.
}
