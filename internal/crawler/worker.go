package crawler

import (
	"context"
	"errors"
	"log"
	"time"

	"gorm.io/gorm"

	"github.com/fuzumoe/linkTorch-api/internal/analyzer"
	"github.com/fuzumoe/linkTorch-api/internal/model"
	"github.com/fuzumoe/linkTorch-api/internal/repository"
)

type worker struct {
	id           int
	ctx          context.Context
	repo         repository.URLRepository
	analyzer     analyzer.Analyzer
	crawlTimeout time.Duration
	results      chan<- CrawlResult
}

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

func NewWorker(id int, ctx context.Context, r repository.URLRepository, a analyzer.Analyzer, crawlTimeout time.Duration, results chan<- CrawlResult) *worker {
	return newWorker(id, ctx, r, a, crawlTimeout, results)
}

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

func (w *worker) runWithPriority(high, normal, low <-chan uint) {
	for {

		if w.ctx.Done() != nil {
			select {
			case <-w.ctx.Done():
				return
			default:
				return
			}
		}

		select {
		case <-w.ctx.Done():
			return

		case id, ok := <-high:
			if !ok {
				continue
			}
			if id == 0 {
				continue
			}
			w.process(id)

		default:
			select {
			case <-w.ctx.Done():
				return
			case id, ok := <-normal:
				if !ok {
					continue
				}
				if id == 0 {
					continue
				}
				w.process(id)
			case id, ok := <-low:
				if !ok {
					continue
				}
				if id == 0 {
					continue
				}
				w.process(id)
			default:
				time.Sleep(50 * time.Millisecond)
			}
		}
	}
}

func (w *worker) Run(tasks <-chan uint) {
	w.run(tasks)
}

func (w *worker) process(id uint) {
	logf := func(fmtStr string, v ...any) {
		log.Printf("[crawler:%d] id=%d – "+fmtStr, append([]any{id}, v...)...)
	}

	start := time.Now()
	result := CrawlResult{
		URLID:    id,
		Status:   model.StatusRunning,
		Duration: 0,
	}

	defer func() {
		result.Duration = time.Since(start)
		if w.results != nil {
			select {
			case <-w.ctx.Done():
			case w.results <- result:
			default:
				logf("results channel full - dropping result")
			}
		}
	}()

	if err := w.repo.UpdateStatus(id, model.StatusRunning); err != nil {
		logf("cannot set running: %v", err)
		result.Error = err
		return
	}

	rec, err := w.repo.FindByID(id)
	if err != nil {
		setErr(w.repo, id, err)
		logf("lookup: %v", err)
		result.Error = err
		result.Status = model.StatusError
		return
	}

	result.URL = rec.OriginalURL

	if rec.Status == model.StatusStopped {
		logf("aborting analysis because status is 'stopped'")
		result.Status = model.StatusStopped
		return
	}

	timeoutCtx, cancel := context.WithTimeout(w.ctx, w.crawlTimeout)
	defer cancel()

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

	result.LinkCount = len(links)
	result.Links = links

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

func setErr(repo repository.URLRepository, id uint, err error) {
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		_ = repo.UpdateStatus(id, model.StatusError)
	}
}
