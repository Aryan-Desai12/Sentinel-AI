package worker

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"sentinel-ai/agents/synthetic-monitor/internal/checker"
	"sentinel-ai/agents/synthetic-monitor/internal/config"
	"sentinel-ai/agents/synthetic-monitor/internal/scheduler"
)

func noopResultHandler(checker.Result) {}

func TestPool_RespectsWorkerLimit(t *testing.T) {
	const (
		workerCount = 3
		jobCount    = 10
	)

	jobs := make(chan scheduler.Job, jobCount)

	for i := 0; i < jobCount; i++ {
		target := config.Target{
			URL:             fmt.Sprintf("https://example-%d.com", i),
			IntervalSeconds: 10,
			TimeoutMs:       2000,
			ExpectedStatus:  200,
		}

		jobs <- scheduler.Job{Target: target}
	}

	close(jobs)

	var current int32
	var maxConcurrent int32
	var completed int32

	check := func(ctx context.Context, target config.Target) checker.Result {
		currentNow := atomic.AddInt32(&current, 1)

		for {
			maxNow := atomic.LoadInt32(&maxConcurrent)

			if currentNow <= maxNow {
				break
			}

			if atomic.CompareAndSwapInt32(
				&maxConcurrent,
				maxNow,
				currentNow,
			) {
				break
			}
		}

		time.Sleep(50 * time.Millisecond)

		atomic.AddInt32(&current, -1)
		atomic.AddInt32(&completed, 1)

		return checker.Result{}
	}

	pool := New(
	workerCount,
	workerCount,
	jobs,
	check,
	noopResultHandler,
)

	ctx := context.Background()

	pool.Run(ctx)

	if got := atomic.LoadInt32(&completed); got != jobCount {
		t.Fatalf(
			"expected %d completed jobs, got %d",
			jobCount,
			got,
		)
	}

	if got := atomic.LoadInt32(&maxConcurrent); got > workerCount {
		t.Fatalf(
			"worker limit exceeded: maximum concurrency was %d, limit was %d",
			got,
			workerCount,
		)
	}
}

func TestPool_RespectsGlobalInFlightLimit(t *testing.T) {
	const (
		workerCount = 5
		maxInFlight = 2
		jobCount    = 10
	)

	jobs := make(chan scheduler.Job, jobCount)

	for i := 0; i < jobCount; i++ {
		target := config.Target{
			URL:             fmt.Sprintf("https://example-%d.com", i),
			IntervalSeconds: 10,
			TimeoutMs:       2000,
			ExpectedStatus:  200,
		}

		jobs <- scheduler.Job{Target: target}
	}

	close(jobs)

	var current int32
	var maxConcurrent int32
	var completed int32

	check := func(ctx context.Context, target config.Target) checker.Result {
		currentNow := atomic.AddInt32(&current, 1)

		for {
			maxNow := atomic.LoadInt32(&maxConcurrent)

			if currentNow <= maxNow {
				break
			}

			if atomic.CompareAndSwapInt32(
				&maxConcurrent,
				maxNow,
				currentNow,
			) {
				break
			}
		}

		time.Sleep(50 * time.Millisecond)

		atomic.AddInt32(&current, -1)
		atomic.AddInt32(&completed, 1)

		return checker.Result{}
	}

	pool := New(
	workerCount,
	maxInFlight,
	jobs,
	check,
	noopResultHandler,
)

	ctx := context.Background()

	pool.Run(ctx)

	if got := atomic.LoadInt32(&completed); got != jobCount {
		t.Fatalf(
			"expected %d completed jobs, got %d",
			jobCount,
			got,
		)
	}

	if got := atomic.LoadInt32(&maxConcurrent); got > maxInFlight {
		t.Fatalf(
			"global in-flight limit exceeded: maximum concurrency was %d, limit was %d",
			got,
			maxInFlight,
		)
	}
}

func TestPool_PreventsOverlappingChecksForSameTarget(t *testing.T) {
	const (
		workerCount = 5
		maxInFlight = 5
		jobCount    = 10
	)

	jobs := make(chan scheduler.Job, jobCount)

	target := config.Target{
		URL:             "https://example.com",
		IntervalSeconds: 1,
		TimeoutMs:       2000,
		ExpectedStatus:  200,
	}

	for i := 0; i < jobCount; i++ {
		jobs <- scheduler.Job{Target: target}
	}

	close(jobs)

	var current int32
	var maxConcurrent int32
	var completed int32

	check := func(ctx context.Context, target config.Target) checker.Result {
		currentNow := atomic.AddInt32(&current, 1)

		for {
			maxNow := atomic.LoadInt32(&maxConcurrent)

			if currentNow <= maxNow {
				break
			}

			if atomic.CompareAndSwapInt32(
				&maxConcurrent,
				maxNow,
				currentNow,
			) {
				break
			}
		}

		time.Sleep(50 * time.Millisecond)

		atomic.AddInt32(&current, -1)
		atomic.AddInt32(&completed, 1)

		return checker.Result{}
	}

	pool := New(
	workerCount,
	maxInFlight,
	jobs,
	check,
	noopResultHandler,
)

	ctx := context.Background()

	pool.Run(ctx)

	if got := atomic.LoadInt32(&completed); got != 1 {
		t.Fatalf(
			"expected exactly 1 check for the same target, got %d",
			got,
		)
	}

	if got := atomic.LoadInt32(&maxConcurrent); got > 1 {
		t.Fatalf(
			"target checks overlapped: maximum concurrency was %d",
			got,
		)
	}
}

//The -race test is particularly useful here because we're deliberately testing concurrent access to our counters.

// workers = 5
// maxInFlight = 2

//          10 jobs
//             ↓
//    ┌─────────────────┐
//    │   5 Workers     │
//    │                 │
//    │ W1 W2 W3 W4 W5 │
//    └────────┬────────┘
//             ↓
//       ┌───────────┐
//       │ 2 slots   │
//       └─────┬─────┘
//             ↓
//       ┌─────┴─────┐
//       │           │
//     Check A     Check B
//     active      active
//       │           │
//       └─────┬─────┘
//             ↓
//        slot freed
//             ↓
//          Check C
