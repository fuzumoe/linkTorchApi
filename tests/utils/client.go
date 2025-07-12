package utils

import (
	"fmt"
	"io"
	"net/http"
	"os"
)

// Client wraps http.Client and knows your API base URL.
type Client struct {
	baseURL string
	client  *http.Client
}

// NewClient reads TEST_PORT (or PORT) and returns a new Client.
func NewClient() *Client {
	port := os.Getenv("TEST_PORT")
	if port == "" {
		port = os.Getenv("PORT")
	}
	base := fmt.Sprintf("http://localhost:%s", port)
	return &Client{
		baseURL: base,
		client:  &http.Client{},
	}
}

// Get issues a GET against baseURL+path.
func (c *Client) Get(path string) (*http.Response, error) {
	return c.client.Get(c.baseURL + path)
}

// Post issues a POST against baseURL+path.
func (c *Client) Post(path, contentType string, body io.Reader) (*http.Response, error) {
	return c.client.Post(c.baseURL+path, contentType, body)
}

// Do issues a custom HTTP request against baseURL+path.
func (c *Client) Do(req *http.Request) (*http.Response, error) {
	if !req.URL.IsAbs() {
		// build full URL
		req.URL.Scheme = "http"
		req.URL.Host = req.Host // set Host first
		full := c.baseURL + req.URL.Path
		newReq, _ := http.NewRequest(req.Method, full, req.Body)
		newReq.Header = req.Header
		req = newReq
	}
	return c.client.Do(req)
}
