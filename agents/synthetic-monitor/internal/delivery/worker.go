package delivery

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"sentinel-ai/agents/synthetic-monitor/internal/model"
)

const (
	initialRetryDelay = 1 * time.Second
	maxRetryDelay     = 30 * time.Second
)

type Worker struct {
	buffer    *Buffer
	client    *Client
	sleepFunc func(context.Context, time.Duration) bool
}

func NewWorker(
	buffer *Buffer,
	client *Client,
) *Worker {
	return &Worker{
		buffer:    buffer,
		client:    client,
		sleepFunc: sleepWithContext,
	}
}

func sleepWithContext(
	ctx context.Context,
	delay time.Duration,
) bool {
	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-timer.C:
		return true

	case <-ctx.Done():
		return false
	}
}

func (w *Worker) Run(ctx context.Context) {
	for {
		event, queuedAt, ok := w.buffer.Dequeue(ctx)
		if !ok {
			return
		}

		w.deliver(ctx, event, queuedAt)
	}
}

func (w *Worker) deliver(
	ctx context.Context,
	event model.SyntheticCheckEvent,
	queuedAt time.Time,
) {
	delay := initialRetryDelay

	for {
		if w.buffer.IsExpired(queuedAt) {
			slog.Warn(
				"synthetic event expired during delivery retry",
				"service", "synthetic-monitor",
				"event", "synthetic_event_expired",
				"url", event.URL,
			)
			return
		}

		err := w.client.Send(ctx, event)

		if err == nil {
			slog.Debug(
				"synthetic event delivered",
				"service", "synthetic-monitor",
				"event", "synthetic_event_delivered",
				"url", event.URL,
			)

			return
		}

		if errors.Is(err, context.Canceled) ||
			errors.Is(err, context.DeadlineExceeded) ||
			ctx.Err() != nil {
			return
		}

		if !IsRetryable(err) {
			slog.Warn(
				"synthetic event dropped",
				"service", "synthetic-monitor",
				"event", "synthetic_event_dropped",
				"reason", "permanent_delivery_failure",
				"url", event.URL,
				"error", err,
			)

			return
		}

		slog.Warn(
			"synthetic event delivery failed",
			"service", "synthetic-monitor",
			"event", "synthetic_event_retry",
			"url", event.URL,
			"retry_in", delay.String(),
			"error", err,
		)

		if !w.sleepFunc(ctx, delay) {
			return
		}

		delay *= 2

		if delay > maxRetryDelay {
			delay = maxRetryDelay
		}
	}
}

//The worker's responsibility is:

// Dequeue
//    ↓
// Check TTL
//    ↓
// Send
//    ↓
// ┌───────────────┐
// │               │
// 202           Failure
// │               │
// Remove       classify
//                 │
//         ┌───────┴────────┐
//         │                │
//    permanent         temporary
//         │                │
//       drop          backoff
//                          │
//                     retry same event

//               Gateway Down
//                    │
//           ┌────────┴─────────┐
//           │                  │
//      Existing event      New events
//           │                  │
//    exponential retry      buffer
//           │                  │
//           │             drop oldest
//           │                  │
//           └────────┬─────────┘
//                    ↓
//              Gateway recovers
//                    ↓
//             latest events
//              get delivered

// So now we have created a delivery worker that can handle the delivery of synthetic check events to the gateway. It will retry on temporary failures with exponential backoff and drop events on permanent failures. The worker will also respect the TTL of events in the buffer, ensuring that expired events are not delivered.

// 	Buffer
//   ↓
// Worker
//   ↓
// Client
//   ↓
// Gateway

// failure
//   ↓
// classify
//   ├── permanent → discard
//   └── temporary → exponential backoff → retry
