package crawler

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/fuzumoe/linkTorch-api/internal/analyzer"
	"github.com/fuzumoe/linkTorch-api/internal/repository"
)

type Pool interface {
	Start(ctx context.Context)
	Enqueue(id uint)
	EnqueueWithPriority(id uint, priority int)
	Shutdown()
	GetResults() <-chan CrawlResult
	AdjustWorkers(cmd ControlCommand)
}

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

func (p *pool) Start(ctx context.Context) {
	childCtx, cancel := context.WithCancel(ctx)
	p.ctx = childCtx
	defer cancel()

	for i := 0; i < p.workers; i++ {
		w := newWorker(i+1, p.ctx, p.repo, p.analyzer, p.crawlTimeout, p.results)
		p.wg.Add(1)
		go func() {
			defer p.wg.Done()
			w.runWithPriority(p.highPriority, p.normalPriority, p.lowPriority)
		}()
	}

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
					toRemove := min(cmd.Count, p.workers-1)
					if toRemove > 0 {
						log.Printf("[crawler] removing %d workers", toRemove)
						p.workers = p.workers - toRemove
					}
				}
			}
		}
	}()

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

	<-p.ctx.Done()
	p.Shutdown()
}

func (p *pool) Enqueue(id uint) {
	select {
	case <-p.ctx.Done():
	case p.normalPriority <- id:
	default:
		log.Printf("[crawler] queue full – dropping id=%d", id)
	}
}

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

func (p *pool) GetResults() <-chan CrawlResult {
	return p.results
}

func (p *pool) AdjustWorkers(cmd ControlCommand) {
	select {
	case <-p.ctx.Done():
	case p.controlChan <- cmd:
	default:
		log.Printf("[crawler] control channel full – dropping command %v", cmd)
	}
}

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
