package utils

import (
	"fmt"
	"io"
	"net/http"
	"os"
)

type Client struct {
	baseURL string
	client  *http.Client
}

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

func (c *Client) Get(path string) (*http.Response, error) {
	return c.client.Get(c.baseURL + path)
}

func (c *Client) Post(path, contentType string, body io.Reader) (*http.Response, error) {
	return c.client.Post(c.baseURL+path, contentType, body)
}

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
