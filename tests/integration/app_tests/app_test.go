package app_test

import (

	// Add this import
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/require"

	"github.com/fuzumoe/urlinsight-backend/configs"
	"github.com/fuzumoe/urlinsight-backend/internal/analyzer"
	"github.com/fuzumoe/urlinsight-backend/internal/app"
	"github.com/fuzumoe/urlinsight-backend/internal/crawler"
	"github.com/fuzumoe/urlinsight-backend/internal/repository"
	"github.com/fuzumoe/urlinsight-backend/tests/utils"
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

// dummyCrawlerPool is a testing implementation of crawler.Pool.
type dummyCrawlerPool struct {
	startFunc           func(ctx context.Context)
	EnqueueFunc         func(id uint)
	EnqueuePriorityFunc func(id uint, priority int)
	ShutdownFunc        func()
	GetResultsFunc      func() <-chan crawler.CrawlResult
	AdjustWorkersFunc   func(cmd crawler.ControlCommand)
}

func (d *dummyCrawlerPool) Start(ctx context.Context) {
	if d.startFunc != nil {
		d.startFunc(ctx)
	}
}

func (d *dummyCrawlerPool) Enqueue(id uint) {
	if d.EnqueueFunc != nil {
		d.EnqueueFunc(id)
	}
}

func (d *dummyCrawlerPool) EnqueueWithPriority(id uint, priority int) {
	if d.EnqueuePriorityFunc != nil {
		d.EnqueuePriorityFunc(id, priority)
	}
}

func (d *dummyCrawlerPool) Shutdown() {
	if d.ShutdownFunc != nil {
		d.ShutdownFunc()
	}
}

func (d *dummyCrawlerPool) GetResults() <-chan crawler.CrawlResult {
	if d.GetResultsFunc != nil {
		return d.GetResultsFunc()
	}
	return make(chan crawler.CrawlResult)
}

func (d *dummyCrawlerPool) AdjustWorkers(cmd crawler.ControlCommand) {
	if d.AdjustWorkersFunc != nil {
		d.AdjustWorkersFunc(cmd)
	}
}

func TestAppRun_Integration(t *testing.T) {
	// Setup test database just once
	utils.SetupTest(t)
	defer utils.CleanTestData(t)

	var port int
	var baseURL string

	t.Run("AppIntegration", func(t *testing.T) {
		setupAppHelper(t, &port, &baseURL)
		healthEndpointsHelper(t, baseURL)
	})
}

// Helper to setup the app and assign port/baseURL
func setupAppHelper(t *testing.T, port *int, baseURL *string) {
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
	dbName := "urlinsight_test"
	testDBDSN := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true",
		dbUser, dbPassword, dbHost, dbPort, dbName)
	t.Logf("Using test DSN: %s", testDBDSN)

	listener, err := net.Listen("tcp", "localhost:0")
	require.NoError(t, err)
	*port = listener.Addr().(*net.TCPAddr).Port
	listener.Close()
	*baseURL = fmt.Sprintf("http://127.0.0.1:%d", *port)

	patches := gomonkey.ApplyFunc(app.LoadConfig, func() (*configs.Config, error) {
		return &configs.Config{
			DatabaseURL:     testDBDSN,
			JWTSecret:       "test-secret",
			ServerHost:      "127.0.0.1",
			ServerPort:      fmt.Sprintf("%d", *port),
			ServerMode:      "debug",           // Enable debug mode
			DevUserEmail:    "dev@example.com", // Set dev user email
			DevUserPassword: "devpassword123",  // Set dev user password
			DevUserName:     "devuser",         // Set dev username
			JWTLifetime:     24 * time.Hour,    // JWT lifetime for token generation
		}, nil
	})
	t.Cleanup(func() {
		patches.Reset()
	})

	errChan := make(chan error, 1)
	go func() {
		t.Log("Starting app.Run()...")
		err := app.Run()
		t.Logf("app.Run() returned with error: %v", err)
		errChan <- err
	}()
	select {
	case err := <-errChan:
		t.Fatalf("App failed to start: %v", err)
	case <-time.After(100 * time.Millisecond):
	}
	t.Log("Waiting for app to start...")
	time.Sleep(3 * time.Second)
}

// Helper for health endpoint tests
func healthEndpointsHelper(t *testing.T, baseURL string) {
	healthPaths := []string{
		"/api/v1/health",
		"/api/v1/status",
	}
	type EndpointResult struct {
		Path       string
		StatusCode int
		Response   map[string]interface{}
		Success    bool
	}
	endpointResults := make([]EndpointResult, 0, len(healthPaths))
	for _, path := range healthPaths {
		t.Run(fmt.Sprintf("Path_%s", strings.ReplaceAll(path[1:], "/", "_")), func(t *testing.T) {
			result := EndpointResult{
				Path:    path,
				Success: false,
			}
			healthURL := fmt.Sprintf("%s%s", baseURL, path)
			t.Logf("Testing endpoint at %s", healthURL)
			client := &http.Client{Timeout: 5 * time.Second}
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
				return
			}
			result.StatusCode = resp.StatusCode
			if resp.StatusCode < 200 || resp.StatusCode > 299 {
				t.Logf("Non-success status code from %s: %d", path, resp.StatusCode)
				resp.Body.Close()
				endpointResults = append(endpointResults, result)
				return
			}
			body, err := io.ReadAll(resp.Body)
			resp.Body.Close()
			if err != nil {
				t.Logf("Error reading response body: %v", err)
				result.Success = true
				endpointResults = append(endpointResults, result)
				return
			}
			t.Logf("Raw response from %s: %s", path, string(body))
			var responseData map[string]interface{}
			err = json.Unmarshal(body, &responseData)
			if err != nil {
				bodyText := string(body)
				t.Logf("Response from %s (not JSON): %s", path, bodyText)
				if strings.Contains(strings.ToLower(bodyText), "ok") ||
					strings.Contains(strings.ToLower(bodyText), "healthy") ||
					strings.Contains(strings.ToLower(bodyText), "urlinsight") {
					result.Success = true
					t.Logf("Found endpoint at %s with text response", path)
				}
			} else {
				t.Logf("Found endpoint at %s with JSON response: %v", path, responseData)
				result.Response = responseData
				result.Success = true
			}
			endpointResults = append(endpointResults, result)
		})
	}
	successCount := 0
	for _, result := range endpointResults {
		if result.Success {
			successCount++
			t.Logf("Endpoint %s is working (status: %d)", result.Path, result.StatusCode)
			if result.Response != nil {
				for _, field := range []string{"status", "health", "state", "result", "message"} {
					if status, exists := result.Response[field]; exists {
						t.Logf("Found field '%s' with value: %v in %s", field, status, result.Path)
						statusStr := strings.ToLower(fmt.Sprintf("%v", status))
						if !contains(statusStr, []string{"ok", "up", "healthy", "alive", "running", "hello", "urlinsight"}) {
							t.Logf("Warning: Unexpected status value in %s: %s", result.Path, statusStr)
						}
					}
				}
			}
		} else {
			t.Logf("Endpoint %s is NOT working", result.Path)
		}
	}
	require.GreaterOrEqual(t, successCount, 1, "At least one health endpoint should be working")
}
func TestCrawlerIsRunning(t *testing.T) {
	// Setup test database and cleanup.
	utils.SetupTest(t)
	defer utils.CleanTestData(t)

	var crawlerStarted bool

	// Create a dummy crawler pool that records when Start() is called.
	dummyPool := &dummyCrawlerPool{
		startFunc: func(ctx context.Context) {
			crawlerStarted = true
			// Block until context is cancelled.
			<-ctx.Done()
		},
		EnqueueFunc:  func(id uint) {},
		ShutdownFunc: func() {},
	}

	// Override crawler.New to return our dummy pool.
	patches := gomonkey.ApplyFunc(crawler.New, func(_ repository.URLRepository, _ analyzer.Analyzer, workers, buf int) crawler.Pool {
		return dummyPool
	})
	t.Cleanup(func() {
		patches.Reset()
	})

	// Find a free port - more reliable method
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()

	// Set up custom config without using setupAppHelper
	testDBDSN := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true",
		"urlinsight_user", "secret", "localhost", "3309", "urlinsight_test")

	configPatches := gomonkey.ApplyFunc(app.LoadConfig, func() (*configs.Config, error) {
		return &configs.Config{
			DatabaseURL:     testDBDSN,
			JWTSecret:       "test-secret",
			ServerHost:      "127.0.0.1",
			ServerPort:      fmt.Sprintf("%d", port),
			ServerMode:      "debug",
			DevUserEmail:    "dev@example.com",
			DevUserPassword: "devpassword123",
			DevUserName:     "devuser",
			JWTLifetime:     24 * time.Hour,
		}, nil
	})
	t.Cleanup(func() {
		configPatches.Reset()
	})

	// Run the app in a separate goroutine.
	errChan := make(chan error, 1)
	go func() {
		t.Log("Starting app.Run()...")
		errChan <- app.Run()
	}()

	// Give sufficient time for app.Run() to call our dummyPool.Start.
	time.Sleep(500 * time.Millisecond)
	require.True(t, crawlerStarted, "Crawler pool should have been started")

	// Signal graceful shutdown by sending SIGTERM.
	p, err := os.FindProcess(os.Getpid())
	require.NoError(t, err)
	p.Signal(syscall.SIGTERM)

	// Wait for app.Run() to exit with a reasonable timeout
	select {
	case err := <-errChan:
		require.NoError(t, err, "app.Run() should exit without error")
	case <-time.After(5 * time.Second):
		t.Fatal("Timeout waiting for app.Run() to exit")
	}
}
