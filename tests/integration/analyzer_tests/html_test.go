package analyzer_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fuzumoe/linkTorch-api/internal/analyzer"
)

func TestHTMLAnalyzer_Integration(t *testing.T) {
	// Define a sample HTML document to be served by the test server.
	htmlContent := `<!DOCTYPE html>
<html>
  <head>
    <title>Integration Test Page</title>
  </head>
  <body>
    <h1>Main Header</h1>
    <h2>Secondary Header</h2>
    <form>
      <input type="text" name="user" />
      <input type="password" name="pass" />
    </form>
    <a href="/internal/page">Internal Link</a>
    <a href="https://external.example.com/page">External Link</a>
  </body>
</html>`

	// Start a test HTTP server that serves the above HTML.
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// For any request, return the defined HTML content.
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(htmlContent))
	}))
	defer ts.Close()

	// Parse the test server URL.
	baseURL, err := url.Parse(ts.URL)
	require.NoError(t, err)

	// Create a new HTML analyzer instance.
	analyzerInstance := analyzer.NewHTMLAnalyzer()

	// Execute the analysis.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	result, links, err := analyzerInstance.Analyze(ctx, baseURL)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Subtest: Verify Metrics.
	t.Run("Verify Metrics", func(t *testing.T) {
		assert.Equal(t, "HTML 5", result.HTMLVersion, "Should detect HTML 5 doctype")
		assert.Equal(t, "Integration Test Page", result.Title, "Should extract the title correctly")
		assert.True(t, result.HasLoginForm, "Should detect the login form")
	})

	// Subtest: Verify Headings.
	t.Run("Verify Headings", func(t *testing.T) {
		assert.Equal(t, 1, result.H1Count, "Should count one h1 element")
		assert.Equal(t, 1, result.H2Count, "Should count one h2 element")
		assert.Equal(t, 0, result.H3Count, "Should count zero h3 elements")
		assert.Equal(t, 0, result.H4Count, "Should count zero h4 elements")
		assert.Equal(t, 0, result.H5Count, "Should count zero h5 elements")
		assert.Equal(t, 0, result.H6Count, "Should count zero h6 elements")
	})

	// Subtest: Verify Links.
	t.Run("Verify Links", func(t *testing.T) {
		// Expect two unique links.
		require.Len(t, links, 2, "Expected two unique links to be extracted")
		var internalFound, externalFound bool
		for _, l := range links {
			if strings.Contains(l.Href, ts.URL) {
				internalFound = true
			} else if strings.Contains(l.Href, "external.example.com") {
				externalFound = true
			}
		}
		assert.True(t, internalFound, "Internal link should be present")
		assert.True(t, externalFound, "External link should be present")
	})
}
