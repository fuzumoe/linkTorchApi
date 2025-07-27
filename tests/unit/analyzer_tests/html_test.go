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

func TestHTMLAnalyzer_Analyze(t *testing.T) {
	// Define a sample HTML document.
	htmlContent := `<!DOCTYPE html>
	<html>
	  <head>
		<title>Test Page</title>
	  </head>
	  <body>
		<h1>Header1</h1>
		<h2>Subheader1</h2>
		<h2>Subheader2</h2>
		<form>
		  <input type="password" />
		</form>
		<a href="/internal">Internal Link</a>
		<a href="http://external.com/page">External Link</a>
	  </body>
	</html>`

	// Start a test HTTP server to serve the HTML content.
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(htmlContent))
	}))
	defer ts.Close()

	// Parse the test server URL.
	baseURL, err := url.Parse(ts.URL)
	require.NoError(t, err)

	// Create a new HTML analyzer instance.
	ha := analyzer.NewHTMLAnalyzer()

	// Analyze the document at the test server URL.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	result, links, err := ha.Analyze(ctx, baseURL)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Subtest: HTML Metrics.
	t.Run("HTML Metrics", func(t *testing.T) {
		assert.Equal(t, "HTML 5", result.HTMLVersion, "HTML version should be parsed as HTML 5")
		assert.Equal(t, "Test Page", result.Title, "Title should be parsed correctly")
		assert.True(t, result.HasLoginForm, "Login form should be detected")
	})

	// Subtest: Heading Counts.
	t.Run("Heading Counts", func(t *testing.T) {
		assert.Equal(t, 1, result.H1Count, "There should be one h1 element")
		assert.Equal(t, 2, result.H2Count, "There should be two h2 elements")
		assert.Equal(t, 0, result.H3Count, "There should be zero h3 elements")
		assert.Equal(t, 0, result.H4Count, "There should be zero h4 elements")
		assert.Equal(t, 0, result.H5Count, "There should be zero h5 elements")
		assert.Equal(t, 0, result.H6Count, "There should be zero h6 elements")
	})

	// Subtest: Link Extraction.
	t.Run("Link Extraction", func(t *testing.T) {
		assert.Len(t, links, 2, "There should be two unique links")
		var internalFound, externalFound bool
		for _, l := range links {
			if strings.Contains(l.Href, ts.URL) {
				internalFound = true
			} else if strings.Contains(l.Href, "external.com") {
				externalFound = true
			}
		}
		assert.True(t, internalFound, "Internal link should be present")
		assert.True(t, externalFound, "External link should be present")
	})
}
