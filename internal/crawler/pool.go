package crawler

import (
	"context"
	"log"
	"sync"

	"github.com/fuzumoe/urlinsight-backend/internal/analyzer"
	"github.com/fuzumoe/urlinsight-backend/internal/repository"
)

// Pool is injected into url_service so handlers can queue jobs.
type Pool interface {
	// Start runs background workers until the passed context is cancelled.
	Start(ctx context.Context)
	Enqueue(id uint)
	Shutdown()
}

// New creates a new crawler pool with the given repository and analyzer.
func New(repo repository.URLRepository, a analyzer.Analyzer, workers, buf int) Pool {
	if workers <= 0 {
		workers = 4
	}
	if buf <= 0 {
		buf = 128
	}

	// Start with a background context.
	ctx, cancel := context.WithCancel(context.Background())

	return &pool{
		repo:    repo,
		anal:    a,
		workers: workers,
		tasks:   make(chan uint, buf),
		ctx:     ctx,
		cancel:  cancel,
	}
}

// pool manages a set of workers that process URL analysis tasks.
type pool struct {
	repo    repository.URLRepository
	anal    analyzer.Analyzer
	workers int
	tasks   chan uint

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// Start spins up background workers and blocks until ctx is cancelled.
// When ctx.Done() is signaled, it calls Shutdown().
func (p *pool) Start(ctx context.Context) {
	// Create a child context that can be cancelled either by the external ctx or p.cancel.
	childCtx, cancel := context.WithCancel(ctx)
	// Overwrite our internal context with the child context.
	p.ctx = childCtx
	// Ensure that when Start() exits, we cancel the child context.
	defer cancel()

	// Spin up workers.
	for i := 0; i < p.workers; i++ {
		w := newWorker(i+1, p.ctx, p.repo, p.anal)
		p.wg.Add(1)
		go func() {
			defer p.wg.Done()
			w.run(p.tasks)
		}()
	}

	// Block until the external context is cancelled.
	<-p.ctx.Done()
	p.Shutdown()
}

// Enqueue drops a URL-row ID onto the buffered channel.
func (p *pool) Enqueue(id uint) {
	select {
	case <-p.ctx.Done():
	case p.tasks <- id:
	default:
		log.Printf("[crawler] queue full – dropping id=%d", id)
	}
}

// Shutdown cancels the context, waits for all workers to finish, and then closes the tasks channel.
func (p *pool) Shutdown() {
	p.cancel()
	p.wg.Wait()
	close(p.tasks)
}
