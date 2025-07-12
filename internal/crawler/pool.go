package crawler

import (
	"context"
	"log"
	"os/signal"
	"sync"
	"syscall"

	"github.com/fuzumoe/urlinsight-backend/internal/analyzer"
	"github.com/fuzumoe/urlinsight-backend/internal/repository"
)

// Pool is injected into url_service so handlers can queue jobs.
type Pool interface {
	Start()
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

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)

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
	repo repository.URLRepository
	anal analyzer.Analyzer

	workers int
	tasks   chan uint

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// Start spins up background workers.
func (p *pool) Start() {
	for i := 0; i < p.workers; i++ {
		w := newWorker(i+1, p.ctx, p.repo, p.anal)
		p.wg.Add(1)
		go func() {
			defer p.wg.Done()
			w.run(p.tasks)
		}()
	}
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

// Shutdown flushes the queue then stops workers.
func (p *pool) Shutdown() {
	p.cancel()
	p.wg.Wait()
	close(p.tasks)
}
