package model

import (
	"encoding/json"
	"testing"
	"time"

	"sentinel-ai/agents/synthetic-monitor/internal/checker"
)

func TestFromResult(t *testing.T) {
	timestamp := time.Date(
		2026,
		9,
		19,
		12,
		30,
		0,
		0,
		time.UTC,
	)

	result := checker.Result{
		URL:        "https://example.com/health",
		Service:    "checkout-api",
		Status:     "degraded",
		LatencyMs:  3200,
		TTFBMs:     2800,
		HTTPStatus: 200,
		Timestamp:  timestamp,
	}

	event := FromResult(result)

	if event.URL != result.URL {
		t.Fatalf("expected URL %q, got %q", result.URL, event.URL)
	}

	if event.Service != result.Service {
		t.Fatalf("expected service %q, got %q", result.Service, event.Service)
	}

	if event.Status != result.Status {
		t.Fatalf("expected status %q, got %q", result.Status, event.Status)
	}

	if event.LatencyMs != result.LatencyMs {
		t.Fatalf(
			"expected latency %d, got %d",
			result.LatencyMs,
			event.LatencyMs,
		)
	}

	if event.TTFBMs != result.TTFBMs {
		t.Fatalf(
			"expected TTFB %d, got %d",
			result.TTFBMs,
			event.TTFBMs,
		)
	}

	if event.HTTPStatus != result.HTTPStatus {
		t.Fatalf(
			"expected HTTP status %d, got %d",
			result.HTTPStatus,
			event.HTTPStatus,
		)
	}

	if !event.Timestamp.Equal(result.Timestamp) {
		t.Fatalf("timestamp mismatch")
	}
}

func TestSyntheticCheckEvent_JSON(t *testing.T) {
	timestamp := time.Date(
		2026,
		9,
		19,
		12,
		30,
		0,
		0,
		time.UTC,
	)

	event := SyntheticCheckEvent{
		URL:        "https://example.com/health",
		Service:    "checkout-api",
		Status:     "up",
		LatencyMs:  120,
		TTFBMs:     80,
		HTTPStatus: 200,
		Timestamp:  timestamp,
	}

	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("failed to marshal event: %v", err)
	}

	var payload map[string]any

	if err := json.Unmarshal(data, &payload); err != nil {
		t.Fatalf("failed to decode JSON: %v", err)
	}

	expected := map[string]any{
		"url":         "https://example.com/health",
		"service":     "checkout-api",
		"status":      "up",
		"latency_ms":  float64(120),
		"ttfb_ms":     float64(80),
		"http_status": float64(200),
		"timestamp":   "2026-09-19T12:30:00Z",
	}

	for key, want := range expected {
		got, exists := payload[key]

		if !exists {
			t.Fatalf("expected JSON field %q to exist", key)
		}

		if got != want {
			t.Fatalf(
				"field %q: expected %v, got %v",
				key,
				want,
				got,
			)
		}
	}
}
