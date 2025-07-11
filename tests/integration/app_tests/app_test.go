package app_test

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fuzumoe/urlinsight-backend/configs"
	"github.com/fuzumoe/urlinsight-backend/internal/app"
	"github.com/fuzumoe/urlinsight-backend/tests/integration"
)

func TestAppRun_Integration(t *testing.T) {
	// Use test mode to reduce noise in logs
	gin.SetMode(gin.TestMode)

	// Get database connection details from environment variables
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

	// Use the test database name
	dbName := "urlinsight_test"

	// Construct the DSN
	testDBDSN := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true",
		dbUser, dbPassword, dbHost, dbPort, dbName)
	t.Logf("Using test DSN: %s", testDBDSN)

	_ = integration.SetupTest(t)
	defer integration.CleanTestData(t)

	t.Run("AppRun_Integration", func(t *testing.T) {
		// Create a listener on a random port
		listener, err := net.Listen("tcp", "localhost:0")
		require.NoError(t, err)
		port := listener.Addr().(*net.TCPAddr).Port
		listener.Close() // Close it so our app can use it

		t.Logf("Test using port: %d", port)

		// Mock the config loader to use our test database and the available port
		patches := gomonkey.ApplyFunc(app.LoadConfig, func() (*configs.Config, error) {
			return &configs.Config{
				DatabaseURL: testDBDSN, // Use the constructed DSN
				JWTSecret:   "test-secret",
				ServerHost:  "127.0.0.1", // Use loopback explicitly
				ServerPort:  fmt.Sprintf("%d", port),
			}, nil
		})
		defer patches.Reset()

		// Start the app in a separate goroutine
		errChan := make(chan error, 1)
		go func() {
			t.Log("Starting app.Run()...")
			err := app.Run()
			t.Logf("app.Run() returned with error: %v", err)
			errChan <- err
		}()

		// Check if the app failed to start (give it a moment)
		select {
		case err := <-errChan:
			t.Fatalf("App failed to start: %v", err)
		case <-time.After(100 * time.Millisecond):
			// App didn't return an immediate error, good
		}

		// Give the app more time to fully initialize and start listening
		t.Log("Waiting for app to start...")
		time.Sleep(3 * time.Second)

		// Test the health endpoint
		healthURL := fmt.Sprintf("http://127.0.0.1:%d/health", port)
		t.Logf("Testing health endpoint at %s", healthURL)

		// Add retries for connection
		var resp *http.Response
		var connErr error
		for i := 0; i < 3; i++ {
			resp, connErr = http.Get(healthURL)
			if connErr == nil {
				break
			}
			t.Logf("Connection attempt %d failed: %v, retrying...", i+1, connErr)
			time.Sleep(1 * time.Second)
		}

		if connErr != nil {
			t.Fatalf("Failed to connect after retries: %v", connErr)
		}

		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var healthData map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&healthData)
		require.NoError(t, err)

		// Check health response - just verify the status is "ok"
		assert.Equal(t, "ok", healthData["status"])
		t.Logf("Health check response: %v", healthData)

		// Test the home endpoint
		homeURL := fmt.Sprintf("http://127.0.0.1:%d/", port)
		t.Logf("Testing home endpoint at %s", homeURL)
		resp, err = http.Get(homeURL)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var homeData map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&homeData)
		require.NoError(t, err)

		// Updated expectation for the actual message
		assert.Equal(t, "Welcome to URL Insight Backend!", homeData["message"])
		t.Logf("Home endpoint response: %v", homeData)

		// Since the server doesn't shut down properly with our mock approach,
		// we'll just consider the test successful if we got this far
		t.Log("Test completed successfully")
	})
}
