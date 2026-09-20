package scheduler

import (
	"context"
	"testing"
	"time"

	"sentinel-ai/agents/synthetic-monitor/internal/config"
)

func TestScheduler_ProducesJobs(t *testing.T) {
	jobs := make(chan Job, 2)

	target := config.Target{
		URL:             "https://example.com",
		IntervalSeconds: 1,
		TimeoutMs:       2000,
		ExpectedStatus:  200,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	s := New(target, jobs)

	go s.Run(ctx)

	select {
	case job := <-jobs:
		if job.Target.URL != target.URL {
			t.Fatalf(
				"expected URL %q, got %q",
				target.URL,
				job.Target.URL,
			)
		}

	case <-time.After(1500 * time.Millisecond):
		t.Fatal("scheduler did not produce a job")
	}
}

func TestScheduler_DropsJobWhenChannelIsFull(t *testing.T) {
	jobs := make(chan Job, 1)

	target := config.Target{
		URL:             "https://example.com",
		IntervalSeconds: 1,
		TimeoutMs:       2000,
		ExpectedStatus:  200,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	s := New(target, jobs)

	// Fill the channel so the scheduler cannot send another job.
	jobs <- Job{Target: target}

	done := make(chan struct{})

	go func() {
		s.Run(ctx)
		close(done)
	}()

	// Wait for at least one scheduler tick.
	time.Sleep(1100 * time.Millisecond)

	cancel()

	select {
	case <-done:
		// Scheduler returned successfully.
	case <-time.After(500 * time.Millisecond):
		t.Fatal("scheduler did not stop after cancellation")
	}

	if got := len(jobs); got != 1 {
		t.Fatalf(
			"expected channel to remain full with 1 job, got %d",
			got,
		)
	}
}
