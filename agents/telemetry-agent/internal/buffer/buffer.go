// We should not send telemetry directly from the collector to the Gateway.
// The buffer gives us a temporary place to hold valid telemetry while the transport retries.
// For our current MVP, we'll keep it simple:
// In-memory bounded buffer
// FIFO behavior
// Configurable maximum size
// No database yet
// No external queue yet
// This is enough to demonstrate the reliability principle without introducing unnecessary infrastructure.
//
//              Telemetry Agent
//                    │
//          ┌─────────┴─────────┐
//          ↓                   ↓
//   Collection loop       Transport loop
//     every 10 sec          continuously
//          │                   │
//          ↓                   ↓
//       Add()               Get()
//          │                   │
//          └────── Buffer ─────┘

// The collection loop produces telemetry.
// The transport loop consumes telemetry and sends it to the Gateway.
// These can execute at the same time.

// buffer = shared queue, with producers on one side and consumers on the other.
//
// A Mutex is basically a lock.

// Think of the buffer as a room with one key.

//              Buffer
//                🔒
//               /   \
//        Collection  Transport

// Only one goroutine can enter the critical section at a time.

// We aren't protecting the events from each other. We are protecting the shared queue from simultaneous modification.

// In Sentinel, an event is a structured piece of information representing something that happened or was observed at a particular point in time.

package buffer

import (
	"fmt"
	"sync"

	"github.com/Aryan-Desai12/sentinel-ai/agents/telemetry-agent/internal/model"
)

type Buffer struct {
	mu       sync.Mutex
	events   []model.Telemetry
	capacity int
}

func New(capacity int) *Buffer {
	return &Buffer{
		events:   make([]model.Telemetry, 0, capacity),
		capacity: capacity,
	}
}

func (b *Buffer) Add(event model.Telemetry) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if len(b.events) >= b.capacity {
		return fmt.Errorf("buffer is full")
	}

	b.events = append(b.events, event)

	return nil
}

// Peek returns the oldest event without removing it.
func (b *Buffer) Peek() (model.Telemetry, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if len(b.events) == 0 {
		return model.Telemetry{}, false
	}

	return b.events[0], true
}

// Remove removes the oldest event after successful delivery.
func (b *Buffer) Remove() (model.Telemetry, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if len(b.events) == 0 {
		return model.Telemetry{}, false
	}

	event := b.events[0]
	b.events = b.events[1:]

	return event, true
}
