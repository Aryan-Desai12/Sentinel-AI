package scheduler

import (
	"context"
	"log/slog"
	"time"

	"sentinel-ai/agents/synthetic-monitor/internal/config"
)

type Job struct {
	Target config.Target
}

type Scheduler struct {
	target config.Target
	jobs   chan<- Job //This scheduler is allowed to send jobs, but cannot receive from the channel.
}

func New(target config.Target, jobs chan<- Job) *Scheduler {
	return &Scheduler{
		target: target,
		jobs:   jobs,
	}
}

func (s *Scheduler) Run(ctx context.Context) {
	ticker := time.NewTicker( //creates the target's periodic schedule.
		time.Duration(s.target.IntervalSeconds) * time.Second,
	)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return

		case <-ticker.C:
			select {
			case s.jobs <- Job{Target: s.target}:
				// Job successfully scheduled.

			case <-ctx.Done():
				return

			default:
				slog.Warn(
					"synthetic check job dropped",
					"service", "synthetic-monitor",
					"event", "synthetic_check_job_dropped",
					"url", s.target.URL,
					"reason", "jobs_channel_full",
				)
			}
		}
	}
}
