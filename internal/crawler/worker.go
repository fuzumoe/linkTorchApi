package crawler

import (
	"context"
	"errors"
	"log"
	"time"

	"gorm.io/gorm"

	"github.com/fuzumoe/urlinsight-backend/internal/analyzer"
	"github.com/fuzumoe/urlinsight-backend/internal/model"
	"github.com/fuzumoe/urlinsight-backend/internal/repository"
)

// worker manages a single crawling job with an individual timeout.
type worker struct {
	id           int
	ctx          context.Context
	repo         repository.URLRepository
	analyzer     analyzer.Analyzer
	crawlTimeout time.Duration
	results      chan<- CrawlResult
}

// newWorker creates a new worker instance with crawlTimeout.
func newWorker(id int, ctx context.Context, r repository.URLRepository, a analyzer.Analyzer, crawlTimeout time.Duration, results chan<- CrawlResult) *worker {
	return &worker{
		id:           id,
		ctx:          ctx,
		repo:         r,
		analyzer:     a,
		crawlTimeout: crawlTimeout,
		results:      results,
	}
}

// NewWorker is an exported constructor for worker.
func NewWorker(id int, ctx context.Context, r repository.URLRepository, a analyzer.Analyzer, crawlTimeout time.Duration, results chan<- CrawlResult) *worker {
	return newWorker(id, ctx, r, a, crawlTimeout, results)
}

// run starts the worker loop.
func (w *worker) run(tasks <-chan uint) {
	for {
		select {
		case <-w.ctx.Done():
			return
		case id, ok := <-tasks:
			if !ok {
				return
			}
			if id == 0 {
				continue
			}
			w.process(id)
		}
	}
}

// runWithPriority starts the worker loop with priority queues
func (w *worker) runWithPriority(high, normal, low <-chan uint) {
	for {
		// First check if the context is cancelled
		if w.ctx.Done() != nil {
			select {
			case <-w.ctx.Done():
				return
			default:
				// Context not cancelled, proceed
			}
		}

		// Check queues in order of priority
		select {
		case <-w.ctx.Done():
			return

		// Check high priority first
		case id, ok := <-high:
			if !ok {
				continue
			}
			if id == 0 {
				continue
			}
			w.process(id)

		// Check if there's anything in normal or low queues using a default case
		default:
			select {
			case <-w.ctx.Done():
				return
			// Then check normal priority
			case id, ok := <-normal:
				if !ok {
					continue
				}
				if id == 0 {
					continue
				}
				w.process(id)
			// Finally check low priority
			case id, ok := <-low:
				if !ok {
					continue
				}
				if id == 0 {
					continue
				}
				w.process(id)
			default:
				// No tasks available in any queue, sleep briefly to avoid CPU spinning
				time.Sleep(50 * time.Millisecond)
			}
		}
	}
}

// Run is an exported wrapper around the unexported run method.
func (w *worker) Run(tasks <-chan uint) {
	w.run(tasks)
}

// process handles a single URL analysis task.
func (w *worker) process(id uint) {
	logf := func(fmtStr string, v ...any) {
		log.Printf("[crawler:%d] id=%d – "+fmtStr, append([]any{id}, v...)...)
	}

	// Create a result structure to track the crawl
	start := time.Now()
	result := CrawlResult{
		URLID:    id,
		Status:   model.StatusRunning,
		Duration: 0,
	}

	// Send the result when we're done
	defer func() {
		result.Duration = time.Since(start)
		// Only send result if we have a results channel
		if w.results != nil {
			select {
			case <-w.ctx.Done():
			case w.results <- result:
			default:
				logf("results channel full - dropping result")
			}
		}
	}()

	// Update status to running.
	if err := w.repo.UpdateStatus(id, model.StatusRunning); err != nil {
		logf("cannot set running: %v", err)
		result.Error = err
		return
	}

	// Fetch the record.
	rec, err := w.repo.FindByID(id)
	if err != nil {
		setErr(w.repo, id, err)
		logf("lookup: %v", err)
		result.Error = err
		result.Status = model.StatusError
		return
	}

	// Store the URL in the result
	result.URL = rec.OriginalURL

	// Allow a stop request to take precedence.
	if rec.Status == model.StatusStopped {
		logf("aborting analysis because status is 'stopped'")
		result.Status = model.StatusStopped
		return
	}

	// Create a context with the worker's crawl timeout.
	timeoutCtx, cancel := context.WithTimeout(w.ctx, w.crawlTimeout)
	defer cancel()

	// Perform the analysis.
	res, links, err := w.analyzer.Analyze(timeoutCtx, rec.URL())
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
			_ = w.repo.UpdateStatus(id, model.StatusStopped)
			logf("stopped by timeout or cancellation")
			result.Status = model.StatusStopped
			result.Error = err
			return
		}
		setErr(w.repo, id, err)
		logf("analyze: %v", err)
		result.Status = model.StatusError
		result.Error = err
		return
	}

	// Update the result with link info
	result.LinkCount = len(links)
	result.Links = links

	// Persist results.
	if err := w.repo.SaveResults(id, res, links); err != nil {
		setErr(w.repo, id, err)
		logf("save: %v", err)
		result.Status = model.StatusError
		result.Error = err
		return
	}

	updated, err := w.repo.FindByID(id)
	if err != nil {
		logf("lookup after analysis failed: %v", err)
		result.Error = err
		return
	}
	if updated.Status != model.StatusStopped {
		_ = w.repo.UpdateStatus(id, model.StatusDone)
		result.Status = model.StatusDone
	} else {
		result.Status = model.StatusStopped
	}
	logf("done in %s (links=%d)", time.Since(start).Truncate(time.Millisecond), len(links))
}

// setErr updates the URL status to "error" if the error is not a record not found.
func setErr(repo repository.URLRepository, id uint, err error) {
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		_ = repo.UpdateStatus(id, model.StatusError)
	}
}
