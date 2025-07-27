package crawler

import (
	"time"

	"github.com/fuzumoe/linkTorch-api/internal/model"
)

type CrawlResult struct {
	URLID     uint
	URL       string
	Status    string
	Error     error
	LinkCount int
	Duration  time.Duration `json:"duration" swaggertype:"integer" format:"int64" example:"1500000000"` // Duration in nanoseconds
	Links     []model.Link
}

type PriorityTask struct {
	URLID    uint
	Priority int
}

type ControlCommand struct {
	Action string
	Count  int
}
