package delivery

import (
	"context"
	"testing"
	"time"

	"sentinel-ai/agents/synthetic-monitor/internal/model"
)

func TestBuffer_EnqueueAndDequeue(t *testing.T) {
	buffer := NewBuffer(2, time.Minute)

	event := model.SyntheticCheckEvent{
		URL:    "https://example.com",
		Status: "up",
	}

	buffer.Enqueue(event)

	ctx := context.Background()

	got, _, ok := buffer.Dequeue(ctx)
	if !ok {
		t.Fatal("expected event to be dequeued")
	}

	if got.URL != event.URL {
		t.Fatalf(
			"expected URL %q, got %q",
			event.URL,
			got.URL,
		)
	}
}

func TestBuffer_DropsOldestWhenFull(t *testing.T) {
	buffer := NewBuffer(2, time.Minute)

	eventA := model.SyntheticCheckEvent{
		URL:    "https://a.example.com",
		Status: "up",
	}

	eventB := model.SyntheticCheckEvent{
		URL:    "https://b.example.com",
		Status: "up",
	}

	eventC := model.SyntheticCheckEvent{
		URL:    "https://c.example.com",
		Status: "down",
	}

	buffer.Enqueue(eventA)
	buffer.Enqueue(eventB)
	buffer.Enqueue(eventC)

	ctx := context.Background()

	got, _, ok := buffer.Dequeue(ctx)
	if !ok {
		t.Fatal("expected event")
	}

	if got.URL != eventB.URL {
		t.Fatalf(
			"expected oldest event to be dropped; got %q",
			got.URL,
		)
	}

	got, _, ok = buffer.Dequeue(ctx)
	if !ok {
		t.Fatal("expected event")
	}

	if got.URL != eventC.URL {
		t.Fatalf(
			"expected newest event to remain; got %q",
			got.URL,
		)
	}
}

func TestBuffer_ExpiresOldEvents(t *testing.T) {
	buffer := NewBuffer(2, 20*time.Millisecond)

	event := model.SyntheticCheckEvent{
		URL:    "https://example.com",
		Status: "up",
	}

	buffer.Enqueue(event)

	time.Sleep(30 * time.Millisecond)

	ctx, cancel := context.WithTimeout(
		context.Background(),
		50*time.Millisecond,
	)
	defer cancel()

	_, _, ok := buffer.Dequeue(ctx)

	if ok {
		t.Fatal("expected expired event to be discarded")
	}
}

func TestBuffer_DequeueStopsOnContextCancellation(t *testing.T) {
	buffer := NewBuffer(2, time.Minute)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, _, ok := buffer.Dequeue(ctx)

	if ok {
		t.Fatal("expected dequeue to stop after context cancellation")
	}
}
