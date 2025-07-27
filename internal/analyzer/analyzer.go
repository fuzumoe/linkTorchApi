package analyzer

import (
	"context"
	"net/url"

	"github.com/fuzumoe/linkTorch-api/internal/model"
)

// Analyzer defines the interface for analyzing URLs.
type Analyzer interface {
	Analyze(ctx context.Context, u *url.URL) (*model.AnalysisResult, []model.Link, error)
}

// New creates a new HTML analyzer instance.
func New() Analyzer { return NewHTMLAnalyzer() }
