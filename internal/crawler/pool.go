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
	EnqueueWithPriority(id uint, priority int)
	Shutdown()
	GetResults() <-chan CrawlResult
	AdjustWorkers(cmd ControlCommand)
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
		repo:           repo,
		analyzer:       a,
		workers:        workers,
		tasks:          make(chan uint, buf),
		highPriority:   make(chan uint, buf/4),
		normalPriority: make(chan uint, buf/2),
		lowPriority:    make(chan uint, buf/4),
		results:        make(chan CrawlResult, buf),
		controlChan:    make(chan ControlCommand, 10),
		ctx:            ctx,
		cancel:         cancel,
		crawlTimeout:   crawlTimeout,
	}
}

// pool manages a set of workers that process URL analysis tasks.
type pool struct {
	repo           repository.URLRepository
	analyzer       analyzer.Analyzer
	workers        int
	tasks          chan uint
	highPriority   chan uint
	normalPriority chan uint
	lowPriority    chan uint
	results        chan CrawlResult
	controlChan    chan ControlCommand
	ctx            context.Context
	cancel         context.CancelFunc
	wg             sync.WaitGroup
	crawlTimeout   time.Duration
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
		w := newWorker(i+1, p.ctx, p.repo, p.analyzer, p.crawlTimeout, p.results)
		p.wg.Add(1)
		go func() {
			defer p.wg.Done()
			w.runWithPriority(p.highPriority, p.normalPriority, p.lowPriority)
		}()
	}

	// Start a goroutine to handle control commands
	go func() {
		for {
			select {
			case <-p.ctx.Done():
				return
			case cmd := <-p.controlChan:
				switch cmd.Action {
				case "add":
					log.Printf("[crawler] adding %d new workers", cmd.Count)
					for i := 0; i < cmd.Count; i++ {
						w := newWorker(p.workers+i+1, p.ctx, p.repo, p.analyzer, p.crawlTimeout, p.results)
						p.wg.Add(1)
						go func() {
							defer p.wg.Done()
							w.runWithPriority(p.highPriority, p.normalPriority, p.lowPriority)
						}()
					}
					p.workers += cmd.Count
				case "remove":
					// Workers will automatically exit when context is cancelled
					// So we just need to update the count
					toRemove := min(cmd.Count, p.workers-1)
					if toRemove > 0 {
						log.Printf("[crawler] removing %d workers", toRemove)
						p.workers = p.workers - toRemove
					}
				}
			}
		}
	}()

	// Start a background task to forward regular tasks to normal priority
	go func() {
		for {
			select {
			case <-p.ctx.Done():
				return
			case id, ok := <-p.tasks:
				if !ok {
					return
				}
				select {
				case <-p.ctx.Done():
					return
				case p.normalPriority <- id:
				default:
					log.Printf("[crawler] normal priority queue full – dropping id=%d", id)
				}
			}
		}
	}()

	// Block until the external context is cancelled.
	<-p.ctx.Done()
	p.Shutdown()
}

// Enqueue drops a URL-row ID onto the buffered channel.
func (p *pool) Enqueue(id uint) {
	select {
	case <-p.ctx.Done():
	case p.normalPriority <- id:
	default:
		log.Printf("[crawler] queue full – dropping id=%d", id)
	}
}

// EnqueueWithPriority enqueues a URL with the specified priority
func (p *pool) EnqueueWithPriority(id uint, priority int) {
	var targetQueue chan uint

	switch {
	case priority > 7:
		targetQueue = p.highPriority
	case priority < 3:
		targetQueue = p.lowPriority
	default:
		targetQueue = p.normalPriority
	}

	select {
	case <-p.ctx.Done():
	case targetQueue <- id:
	default:
		log.Printf("[crawler] priority queue %d full – dropping id=%d", priority, id)
	}
}

// GetResults returns the channel that emits crawl results
func (p *pool) GetResults() <-chan CrawlResult {
	return p.results
}

// AdjustWorkers allows dynamically adding or removing workers
func (p *pool) AdjustWorkers(cmd ControlCommand) {
	select {
	case <-p.ctx.Done():
	case p.controlChan <- cmd:
	default:
		log.Printf("[crawler] control channel full – dropping command %v", cmd)
	}
}

// Shutdown cancels the context, waits for all workers to finish, and then closes the channels.
func (p *pool) Shutdown() {
	p.cancel()
	p.wg.Wait()
	close(p.tasks)
	close(p.highPriority)
	close(p.normalPriority)
	close(p.lowPriority)
	close(p.results)
	close(p.controlChan)
}
