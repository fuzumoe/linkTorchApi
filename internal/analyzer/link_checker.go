package analyzer

import (
	"context"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/temoto/robotstxt"

	"github.com/fuzumoe/urlinsight-backend/internal/model"
)

// robots is a cache for robots.txt data to avoid repeated requests.
var robots sync.Map

// linkChecker checks the status of links concurrently.
type linkChecker struct {
	conc    int
	timeout time.Duration
	client  *http.Client
}

// newLinkChecker creates a new link checker with the specified concurrency and timeout.
func newLinkChecker(conc int, timeout time.Duration) *linkChecker {
	return &linkChecker{
		conc:    conc,
		timeout: timeout,
		client:  &http.Client{Timeout: timeout},
	}
}

// NewLinkChecker is the exported constructor for linkChecker.
func NewLinkChecker(conc int, timeout time.Duration) *linkChecker {
	return newLinkChecker(conc, timeout)
}

// run checks the status of links in the provided analysis result.
func (lc *linkChecker) run(ctx context.Context, links []model.Link) []model.Link {
	in := make(chan *model.Link)
	var wg sync.WaitGroup

	for i := 0; i < lc.conc; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for l := range in {
				l.StatusCode = lc.head(ctx, l.Href)
			}
		}()
	}

	go func() {
		for i := range links {
			in <- &links[i]
		}
		close(in)
	}()

	wg.Wait()
	return links
}

// Run is the exported wrapper for run, so that external packages can call it.
func (lc *linkChecker) Run(ctx context.Context, links []model.Link) []model.Link {
	return lc.run(ctx, links)
}

// head performs a HEAD request to check the link status, respecting robots.txt rules.
func (lc *linkChecker) head(ctx context.Context, raw string) int {
	u, _ := url.Parse(raw)
	if !robotsAllowed(lc.client, u) {
		return http.StatusForbidden
	}

	req, _ := http.NewRequestWithContext(ctx, http.MethodHead, raw, nil)
	resp, err := lc.client.Do(req)
	if err != nil {
		return 0
	}
	resp.Body.Close()

	if resp.StatusCode == http.StatusMethodNotAllowed {
		req.Method = http.MethodGet
		resp2, err := lc.client.Do(req)
		if err != nil {
			return 0
		}
		resp2.Body.Close()
		return resp2.StatusCode
	}
	return resp.StatusCode
}

// robotsAllowed checks if the link is allowed by robots.txt rules.
func robotsAllowed(c *http.Client, u *url.URL) bool {
	if u.Host == "" {
		return true
	}
	if val, ok := robots.Load(u.Host); ok {
		if val == nil {
			return true
		}
		return val.(*robotstxt.RobotsData).TestAgent(u.Path, "*")
	}

	resp, err := c.Get(u.Scheme + "://" + u.Host + "/robots.txt")
	if err != nil {
		robots.Store(u.Host, nil)
		return true
	}
	defer resp.Body.Close()

	data, err := robotstxt.FromResponse(resp)
	if err != nil {
		robots.Store(u.Host, nil)
		return true
	}
	robots.Store(u.Host, data)
	return data.TestAgent(u.Path, "*")
}
