package delivery

import (
	"container/list"
	"context"
	"log/slog"
	"sync"
	"time"

	"sentinel-ai/agents/synthetic-monitor/internal/model"
)

type queuedEvent struct {
	event    model.SyntheticCheckEvent
	queuedAt time.Time
}

type Buffer struct {
	mu       sync.Mutex
	events   *list.List
	capacity int
	ttl      time.Duration
	notify   chan struct{}
}

func NewBuffer(capacity int, ttl time.Duration) *Buffer {
	return &Buffer{
		events:   list.New(),
		capacity: capacity,
		ttl:      ttl,
		notify:   make(chan struct{}, 1),
	}
}

func (b *Buffer) Enqueue(event model.SyntheticCheckEvent) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.events.Len() >= b.capacity {
		oldest := b.events.Front()

		if oldest != nil {
			dropped := oldest.Value.(queuedEvent)
			b.events.Remove(oldest)

			slog.Warn(
				"synthetic event dropped",
				"service", "synthetic-monitor",
				"event", "synthetic_event_dropped",
				"reason", "buffer_full",
				"policy", "drop_oldest",
				"url", dropped.event.URL,
			)
		}
	}

	b.events.PushBack(queuedEvent{
		event:    event,
		queuedAt: time.Now(),
	})

	select {
	case b.notify <- struct{}{}:
	default:
	}
}

func (b *Buffer) Dequeue(ctx context.Context) (
	model.SyntheticCheckEvent,
	time.Time,
	bool,
) {
	for {
		b.mu.Lock()

		if element := b.events.Front(); element != nil {
			item := element.Value.(queuedEvent)
			b.events.Remove(element)

			b.mu.Unlock()

			if b.ttl > 0 && time.Since(item.queuedAt) > b.ttl {
				slog.Warn(
					"synthetic event expired",
					"service", "synthetic-monitor",
					"event", "synthetic_event_expired",
					"reason", "ttl_exceeded",
					"url", item.event.URL,
					"age", time.Since(item.queuedAt).String(),
					"ttl", b.ttl.String(),
				)

				continue
			}

			return item.event, item.queuedAt, true
		}

		b.mu.Unlock()

		select {
		case <-b.notify:
		case <-ctx.Done():
			return model.SyntheticCheckEvent{}, time.Time{}, false
		}
	}
}

func (b *Buffer) IsExpired(queuedAt time.Time) bool {
	return time.Since(queuedAt) > b.ttl
}

// Synthetic Check
//       ↓
// SyntheticCheckEvent
//       ↓
// ┌─────────────────────┐
// │   Delivery Buffer   │
// │                     │
// │ bounded             │
// │ drop-oldest         │
// │ TTL                 │
// └─────────┬───────────┘
//           ↓
//     Delivery Worker
//           ↓
//     Gateway HTTP API
