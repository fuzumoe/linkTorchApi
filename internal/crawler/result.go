package crawler

import (
	"time"

	"github.com/fuzumoe/linkTorch-api/internal/model"
)

// CrawlResult represents the outcome of a crawling operation
type CrawlResult struct {
	URLID     uint
	URL       string
	Status    string
	Error     error
	LinkCount int
	Duration  time.Duration `json:"duration" swaggertype:"integer" format:"int64" example:"1500000000"` // Duration in nanoseconds
	Links     []model.Link  // Optional: include the actual links if needed
}

// PriorityTask represents a URL crawling task with priority
type PriorityTask struct {
	URLID    uint
	Priority int // Higher number means higher priority
}

// ControlCommand represents an instruction to modify the crawler's behavior
type ControlCommand struct {
	Action string // "add" or "remove"
	Count  int    // Number of workers to add or remove
}
