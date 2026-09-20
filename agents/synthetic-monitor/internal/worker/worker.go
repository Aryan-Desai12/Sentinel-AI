package worker

import (
	"context"
	"sync"

	"sentinel-ai/agents/synthetic-monitor/internal/checker"
	"sentinel-ai/agents/synthetic-monitor/internal/config"
	"sentinel-ai/agents/synthetic-monitor/internal/scheduler"
)

type CheckFunc func(context.Context, config.Target) checker.Result

type ResultHandler func(checker.Result)

type Pool struct {
	workers     int
	jobs        <-chan scheduler.Job
	check       CheckFunc
	onResult    ResultHandler
	inFlightSem chan struct{}
	guards      sync.Map
}

func New(
	workers int,
	maxInFlight int,
	jobs <-chan scheduler.Job,
	check CheckFunc,
	onResult ResultHandler,
) *Pool {
	return &Pool{
		workers:     workers,
		jobs:        jobs,
		check:       check,
		onResult:    onResult,
		inFlightSem: make(chan struct{}, maxInFlight),
	}
}

func (p *Pool) Run(ctx context.Context) {
	var wg sync.WaitGroup

	for i := 0; i < p.workers; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()
			p.worker(ctx)
		}()
	}

	wg.Wait()
}

func (p *Pool) worker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return

		case job, ok := <-p.jobs:
			if !ok {
				return
			}

			guard := p.guardFor(job.Target)

			if !guard.TryAcquire() {
				continue
			}

			if !p.acquire(ctx) {
				guard.Release()
				return
			}

			func() {
				defer guard.Release()
				defer p.release()

				result := p.check(ctx, job.Target)

				if p.onResult != nil {
					p.onResult(result)
				}
			}()
		}
	}
}

func (p *Pool) acquire(ctx context.Context) bool {
	select {
	case p.inFlightSem <- struct{}{}:
		return true

	case <-ctx.Done():
		return false
	}
}

func (p *Pool) release() {
	<-p.inFlightSem
}

func (p *Pool) guardFor(target config.Target) *TargetGuard {
	value, _ := p.guards.LoadOrStore(
		target.URL,
		&TargetGuard{},
	)

	return value.(*TargetGuard)
}

// Why acquire the target guard before the global semaphore?
// Because if Target A is already running, we should immediately skip its duplicate job rather than occupy a worker waiting for the global HTTP slot.
// Job
//  ↓
// Target guard
//  ↓
// Already running?
//  ├── YES → skip
//  └── NO
//       ↓
// Global in-flight semaphore
//       ↓
// Checker

// Worker Pool
// → limits total worker execution

// Global Semaphore
// → limits simultaneous HTTP checks

// Per-Target CAS
// → prevents overlapping checks for the same target

//                   ┌── Scheduler A ──┐
//                   ├── Scheduler B ──┤
// Targets ──────────┼── Scheduler C ──┼──→ Bounded Jobs
//                   └── Scheduler N ──┘        │
//                                              ↓
//                                       ┌──────────────┐
//                                       │ Worker Pool  │
//                                       └──────┬───────┘
//                                              ↓
//                                       Per-Target CAS
//                                              ↓
//                                       Global Semaphore
//                                              ↓
//                                           Checker
//                                              ↓
//                                            Result
