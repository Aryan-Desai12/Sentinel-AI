// Why do we need buffer_test.go?
// buffer.go contains the actual functionality:
// Collector
//    ↓
// Buffer
//    ↓
// Transport
// A bug in the buffer could cause:

// telemetry to be lost
// events to come out in the wrong order
// data to be read incorrectly
// problems when collector and transport access it simultaneously
// So buffer_test.go acts as an automated check for those behaviors.

//*****buffer_test.go is a developer-time automated test, not something that runs in the actual agent.******//

package buffer

import (
	"testing"

	"github.com/Aryan-Desai12/sentinel-ai/agents/telemetry-agent/internal/model"
)

func TestBufferAddAndPeek(t *testing.T) {
	buffer := New(3)

	event := model.Telemetry{
		NodeID: "node-001",
		CPUPct: 25,
	}

	if err := buffer.Add(event); err != nil {
		t.Fatalf("Add() failed: %v", err)
	}

	got, ok := buffer.Peek()

	if !ok {
		t.Fatal("Peek() returned no event")
	}

	if got.NodeID != event.NodeID {
		t.Errorf("expected node ID %q, got %q", event.NodeID, got.NodeID)
	}

	if got.CPUPct != event.CPUPct {
		t.Errorf("expected CPU %.2f, got %.2f", event.CPUPct, got.CPUPct)
	}
}

func TestBufferFIFO(t *testing.T) {
	buffer := New(3)

	events := []model.Telemetry{
		{NodeID: "node-001", CPUPct: 10},
		{NodeID: "node-001", CPUPct: 20},
		{NodeID: "node-001", CPUPct: 30},
	}

	for _, event := range events {
		if err := buffer.Add(event); err != nil {
			t.Fatalf("Add() failed: %v", err)
		}
	}

	for _, expected := range events {
		got, ok := buffer.Peek()

		if !ok {
			t.Fatal("Peek() returned no event")
		}

		if got.CPUPct != expected.CPUPct {
			t.Errorf(
				"expected CPU %.2f, got %.2f",
				expected.CPUPct,
				got.CPUPct,
			)
		}

		_, ok = buffer.Remove()
		if !ok {
			t.Fatal("Remove() failed")
		}
	}
}

func TestBufferPeekDoesNotRemove(t *testing.T) {
	buffer := New(2)

	event := model.Telemetry{
		NodeID: "node-001",
		CPUPct: 25,
	}

	if err := buffer.Add(event); err != nil {
		t.Fatalf("Add() failed: %v", err)
	}

	_, ok := buffer.Peek()
	if !ok {
		t.Fatal("Peek() returned no event")
	}

	// Peek again. The event should still exist.
	got, ok := buffer.Peek()

	if !ok {
		t.Fatal("event was removed by Peek()")
	}

	if got.CPUPct != event.CPUPct {
		t.Errorf(
			"expected CPU %.2f, got %.2f",
			event.CPUPct,
			got.CPUPct,
		)
	}
}

func TestBufferEmpty(t *testing.T) {
	buffer := New(3)

	_, ok := buffer.Peek()

	if ok {
		t.Fatal("expected buffer to be empty")
	}

	_, ok = buffer.Remove()

	if ok {
		t.Fatal("expected Remove() to fail on empty buffer")
	}
}

func TestBufferCapacity(t *testing.T) {
	buffer := New(2)

	event := model.Telemetry{
		NodeID: "node-001",
	}

	if err := buffer.Add(event); err != nil {
		t.Fatalf("first Add() failed: %v", err)
	}

	if err := buffer.Add(event); err != nil {
		t.Fatalf("second Add() failed: %v", err)
	}

	if err := buffer.Add(event); err == nil {
		t.Fatal("expected third Add() to fail when buffer is full")
	}
}
