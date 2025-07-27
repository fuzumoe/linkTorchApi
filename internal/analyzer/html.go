package analyzer

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"golang.org/x/net/html"

	"github.com/fuzumoe/linkTorch-api/internal/model"
)

// HTMLAnalyzer analyzes HTML documents for various metrics.
type htmlAnalyzer struct {
	client *http.Client
	check  *linkChecker
}

// NewHTMLAnalyzer creates a new HTML analyzer with default settings.
func NewHTMLAnalyzer() *htmlAnalyzer {
	return &htmlAnalyzer{
		client: &http.Client{Timeout: 10 * time.Second},
		check:  newLinkChecker(12, 5*time.Second),
	}
}

// Analyze fetches the HTML document from the URL and extracts various metrics.
func (a *htmlAnalyzer) Analyze(
	ctx context.Context,
	u *url.URL,
) (*model.AnalysisResult, []model.Link, error) {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	resp, err := a.client.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, nil, err
	}

	res := &model.AnalysisResult{
		HTMLVersion:  detectHTMLVersion(doc),
		Title:        strings.TrimSpace(doc.Find("title").First().Text()),
		HasLoginForm: doc.Find("form input[type='password']").Length() > 0,
	}

	// headings
	doc.Find("h1,h2,h3,h4,h5,h6").Each(func(_ int, s *goquery.Selection) {
		switch strings.ToLower(goquery.NodeName(s)) {
		case "h1":
			res.H1Count++
		case "h2":
			res.H2Count++
		case "h3":
			res.H3Count++
		case "h4":
			res.H4Count++
		case "h5":
			res.H5Count++
		case "h6":
			res.H6Count++
		}
	})

	seen := make(map[string]struct{})
	var links []model.Link
	doc.Find("a[href]").Each(func(_ int, a *goquery.Selection) {
		href, _ := a.Attr("href")
		abs := resolve(u, href)
		if abs == "" {
			return
		}
		if _, ok := seen[abs]; ok {
			return
		}
		seen[abs] = struct{}{}

		lnk := model.Link{
			Href:       abs,
			IsExternal: !sameHost(u, abs),
		}
		links = append(links, lnk)
	})

	links = a.check.run(ctx, links)
	for _, l := range links {
		if l.IsExternal {
			res.ExternalLinkCount++
		} else {
			res.InternalLinkCount++
		}
		if l.StatusCode >= 400 && l.StatusCode < 600 {
			res.BrokenLinkCount++
		}
	}
	return res, links, nil
}

// detectHTMLVersion checks the doctype of the HTML document to determine its version.
func detectHTMLVersion(doc *goquery.Document) string {
	if n := doc.Nodes[0].FirstChild; n != nil && n.Type == html.DoctypeNode {
		d := strings.ToLower(strings.TrimSpace(n.Data))
		if strings.HasPrefix(d, "html") {
			return "HTML 5"
		}
		return d
	}
	return "unknown"
}

// resolve resolves a relative URL against a base URL.
func resolve(base *url.URL, href string) string {
	p, err := url.Parse(strings.TrimSpace(href))
	if err != nil {
		return ""
	}
	return base.ResolveReference(p).String()
}

// sameHost checks if the given raw URL has the same hostname as the base URL.
func sameHost(a *url.URL, raw string) bool {
	b, err := url.Parse(raw)
	return err == nil && a.Hostname() == b.Hostname()
}
