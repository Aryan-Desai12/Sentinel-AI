package transport

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Aryan-Desai12/sentinel-ai/agents/telemetry-agent/internal/model"
)

func testTelemetry() model.Telemetry {
	return model.Telemetry{
		NodeID: "node-001",
		Timestamp: time.Date(
			2026,
			9,
			7,
			10,
			0,
			0,
			0,
			time.UTC,
		),
		CPUPct:  45.5,
		MemPct:  60.2,
		DiskPct: 70.1,
		NetIn:   1000,
		NetOut:  2000,
	}
}

func TestTransportSendSuccess(t *testing.T) {
	event := testTelemetry()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}

		if r.URL.Path != "/metrics/node" {
			t.Errorf("expected /metrics/node, got %s", r.URL.Path)
		}

		if r.Header.Get("X-API-Key") != "test-api-key" {
			t.Errorf("expected API key header")
		}

		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected application/json content type")
		}

		var received model.Telemetry

		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}

		if received != event {
			t.Errorf("received telemetry does not match expected event")
		}

		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	transport := New(server.URL, "test-api-key")

	if err := transport.Send(context.Background(), event); err != nil {
		t.Fatalf("expected successful send, got error: %v", err)
	}
}

func TestTransportSendBadRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()

	transport := New(server.URL, "test-api-key")

	err := transport.Send(context.Background(), testTelemetry())

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	transportErr, ok := err.(*Error)
	if !ok {
		t.Fatalf("expected *Error, got %T", err)
	}

	if transportErr.StatusCode != http.StatusBadRequest {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusBadRequest,
			transportErr.StatusCode,
		)
	}
}

func TestTransportSendUnauthorized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	transport := New(server.URL, "test-api-key")

	err := transport.Send(context.Background(), testTelemetry())

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	transportErr, ok := err.(*Error)
	if !ok {
		t.Fatalf("expected *Error, got %T", err)
	}

	if transportErr.StatusCode != http.StatusUnauthorized {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusUnauthorized,
			transportErr.StatusCode,
		)
	}
}

func TestTransportSendServiceUnavailable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	transport := New(server.URL, "test-api-key")

	err := transport.Send(context.Background(), testTelemetry())

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	transportErr, ok := err.(*Error)
	if !ok {
		t.Fatalf("expected *Error, got %T", err)
	}

	if transportErr.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusServiceUnavailable,
			transportErr.StatusCode,
		)
	}
}

func TestTransportSendUnexpectedStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	transport := New(server.URL, "test-api-key")

	err := transport.Send(context.Background(), testTelemetry())

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	transportErr, ok := err.(*Error)
	if !ok {
		t.Fatalf("expected *Error, got %T", err)
	}

	if transportErr.StatusCode != http.StatusInternalServerError {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusInternalServerError,
			transportErr.StatusCode,
		)
	}
}

func TestTransportSendNetworkError(t *testing.T) {
	transport := New("http://127.0.0.1:1", "test-api-key")

	err := transport.Send(context.Background(), testTelemetry())

	if err == nil {
		t.Fatal("expected network error, got nil")
	}
}

func TestTransportSendContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	defer server.Close()

	transport := New(server.URL, "test-api-key")

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := transport.Send(ctx, testTelemetry())

	if err == nil {
		t.Fatal("expected context cancellation error, got nil")
	}
}
