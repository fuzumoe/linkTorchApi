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
}

// newWorker creates a new worker instance with crawlTimeout.
func newWorker(id int, ctx context.Context, r repository.URLRepository, a analyzer.Analyzer, crawlTimeout time.Duration) *worker {
	return &worker{
		id:           id,
		ctx:          ctx,
		repo:         r,
		analyzer:     a,
		crawlTimeout: crawlTimeout,
	}
}

// NewWorker is an exported constructor for worker.
func NewWorker(id int, ctx context.Context, r repository.URLRepository, a analyzer.Analyzer, crawlTimeout time.Duration) *worker {
	return newWorker(id, ctx, r, a, crawlTimeout)
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

// Run is an exported wrapper around the unexported run method.
func (w *worker) Run(tasks <-chan uint) {
	w.run(tasks)
}

// process handles a single URL analysis task.
func (w *worker) process(id uint) {
	logf := func(fmtStr string, v ...any) {
		log.Printf("[crawler:%d] id=%d – "+fmtStr, append([]any{id}, v...)...)
	}

	// Update status to running.
	if err := w.repo.UpdateStatus(id, model.StatusRunning); err != nil {
		logf("cannot set running: %v", err)
		return
	}

	// Fetch the record.
	rec, err := w.repo.FindByID(id)
	if err != nil {
		setErr(w.repo, id, err)
		logf("lookup: %v", err)
		return
	}

	// Allow a stop request to take precedence.
	if rec.Status == model.StatusStopped {
		logf("aborting analysis because status is 'stopped'")
		return
	}

	// Create a context with the worker's crawl timeout.
	timeoutCtx, cancel := context.WithTimeout(w.ctx, w.crawlTimeout)
	defer cancel()

	start := time.Now()

	// Perform the analysis.
	res, links, err := w.analyzer.Analyze(timeoutCtx, rec.URL())
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
			_ = w.repo.UpdateStatus(id, model.StatusStopped)
			logf("stopped by timeout or cancellation")
			return
		}
		setErr(w.repo, id, err)
		logf("analyze: %v", err)
		return
	}

	// Persist results.
	if err := w.repo.SaveResults(id, res, links); err != nil {
		setErr(w.repo, id, err)
		logf("save: %v", err)
		return
	}

	updated, err := w.repo.FindByID(id)
	if err != nil {
		logf("lookup after analysis failed: %v", err)
		return
	}
	if updated.Status != model.StatusStopped {
		_ = w.repo.UpdateStatus(id, model.StatusDone)
	}
	logf("done in %s (links=%d)", time.Since(start).Truncate(time.Millisecond), len(links))
}

// setErr updates the URL status to "error" if the error is not a record not found.
func setErr(repo repository.URLRepository, id uint, err error) {
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		_ = repo.UpdateStatus(id, model.StatusError)
	}
}
