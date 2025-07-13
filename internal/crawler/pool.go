package crawler

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/fuzumoe/urlinsight-backend/internal/analyzer"
	"github.com/fuzumoe/urlinsight-backend/internal/repository"
)

// Pool defines the interface for a crawler pool that manages multiple workers.
type Pool interface {
	Start(ctx context.Context)
	Enqueue(id uint)
	Shutdown()
}

// New creates a new crawler pool with the specified number of workers and buffer size.
func New(repo repository.URLRepository, a analyzer.Analyzer, workers, buf int, crawlTimeout time.Duration) Pool {
	if workers <= 0 {
		workers = 4
	}
	if buf <= 0 {
		buf = 128
	}
	if crawlTimeout <= 0 {
		crawlTimeout = 30 * time.Second
	}

	// Start with a background context.
	ctx, cancel := context.WithCancel(context.Background())

	return &pool{
		repo:         repo,
		analyzer:     a,
		workers:      workers,
		tasks:        make(chan uint, buf),
		ctx:          ctx,
		cancel:       cancel,
		crawlTimeout: crawlTimeout,
	}
}

// pool manages a set of workers that process URL analysis tasks.
type pool struct {
	repo         repository.URLRepository
	analyzer     analyzer.Analyzer
	workers      int
	tasks        chan uint
	ctx          context.Context
	cancel       context.CancelFunc
	wg           sync.WaitGroup
	crawlTimeout time.Duration
}

// Start initializes the workers and begins processing tasks.
func (p *pool) Start(ctx context.Context) {
	// Create a child context that can be cancelled either by the external ctx or by p.cancel.
	childCtx, cancel := context.WithCancel(ctx)
	// Overwrite our internal context with the child context.
	p.ctx = childCtx
	// Ensure that when Start() exits, we cancel the child context.
	defer cancel()

	// Spin up workers.
	for i := 0; i < p.workers; i++ {
		w := newWorker(i+1, p.ctx, p.repo, p.analyzer, p.crawlTimeout)
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
