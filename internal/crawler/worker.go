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

// Pool is injected into url_service so handlers can queue jobs.
type worker struct {
	id       int
	ctx      context.Context
	repo     repository.URLRepository
	analyzer analyzer.Analyzer
}

// NewWorker creates a new worker instance.
func newWorker(id int, ctx context.Context, r repository.URLRepository, a analyzer.Analyzer) *worker {
	return &worker{id: id, ctx: ctx, repo: r, analyzer: a}
}

// NewWorker creates and returns a new worker instance.
func NewWorker(id int, ctx context.Context, r repository.URLRepository, a analyzer.Analyzer) *worker {
	return newWorker(id, ctx, r, a)
}

// Start runs the worker in a loop, processing tasks from the channel.
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

// process fetches a URL, analyzes it, and saves the results.
func (w *worker) process(id uint) {
	logf := func(fmt string, v ...any) {
		log.Printf("[crawler:%d] id=%d – "+fmt, append([]any{id}, v...)...)
	}

	// status → running.
	if err := w.repo.UpdateStatus(id, model.StatusRunning); err != nil {
		logf("cannot set running: %v", err)
		return
	}

	// Fetch row (need OriginalURL).
	rec, err := w.repo.FindByID(id)
	if err != nil {
		setErr(w.repo, id, err)
		logf("lookup: %v", err)
		return
	}

	// NEW: Check if the URL status was externally set to 'stopped' already.
	// This allows a stop request to take precedence.
	if rec.Status == model.StatusStopped {
		logf("aborting analysis because status is 'stopped'")
		return
	}

	// analyze (with ctx).
	start := time.Now()
	res, links, err := w.analyzer.Analyze(w.ctx, rec.URL())
	if err != nil {
		if errors.Is(err, context.Canceled) {
			_ = w.repo.UpdateStatus(id, model.StatusStopped)
			logf("stopped by ctx")
			return
		}
		setErr(w.repo, id, err)
		logf("analyze: %v", err)
		return
	}

	// persist atomically.
	if err := w.repo.SaveResults(id, res, links); err != nil {
		setErr(w.repo, id, err)
		logf("save: %v", err)
		return
	}

	// Only mark as done if the status wasn't changed to stopped meanwhile.
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

// setErr updates the status to Error if the error is not a record not found.
func setErr(repo repository.URLRepository, id uint, err error) {
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		_ = repo.UpdateStatus(id, model.StatusError)
	}
}
