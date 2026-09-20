package delivery

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"sentinel-ai/agents/synthetic-monitor/internal/model"
)

func testSyntheticEvent() model.SyntheticCheckEvent {
	return model.SyntheticCheckEvent{
		URL:        "https://example.com",
		Service:    "example",
		Status:     "up",
		LatencyMs:  120,
		TTFBMs:     80,
		HTTPStatus: 200,
		Timestamp: time.Date(
			2026,
			9,
			19,
			10,
			0,
			0,
			0,
			time.UTC,
		),
	}
}

func TestClientSendSuccess(t *testing.T) {
	event := testSyntheticEvent()

	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				t.Errorf("expected POST, got %s", r.Method)
			}

			if r.URL.Path != "/checks/synthetic" {
				t.Errorf(
					"expected /checks/synthetic, got %s",
					r.URL.Path,
				)
			}

			if r.Header.Get("X-API-Key") != "test-api-key" {
				t.Errorf("expected API key header")
			}

			if r.Header.Get("Content-Type") != "application/json" {
				t.Errorf(
					"expected application/json content type",
				)
			}

			var received model.SyntheticCheckEvent

			if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
				t.Fatalf(
					"failed to decode request body: %v",
					err,
				)
			}

			if received != event {
				t.Errorf(
					"received event does not match expected event",
				)
			}

			w.WriteHeader(http.StatusAccepted)
		}),
	)
	defer server.Close()

	client := NewClient(
		server.Client(),
		server.URL,
		"test-api-key",
	)

	err := client.Send(
		context.Background(),
		event,
	)

	if err != nil {
		t.Fatalf(
			"expected successful send, got error: %v",
			err,
		)
	}
}

func TestClientSendBadRequest(t *testing.T) {
	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
		}),
	)
	defer server.Close()

	client := NewClient(
		server.Client(),
		server.URL,
		"test-api-key",
	)

	err := client.Send(
		context.Background(),
		testSyntheticEvent(),
	)

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	clientErr, ok := err.(*Error)
	if !ok {
		t.Fatalf("expected *Error, got %T", err)
	}

	if clientErr.StatusCode != http.StatusBadRequest {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusBadRequest,
			clientErr.StatusCode,
		)
	}
}

func TestClientSendUnauthorized(t *testing.T) {
	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
		}),
	)
	defer server.Close()

	client := NewClient(
		server.Client(),
		server.URL,
		"test-api-key",
	)

	err := client.Send(
		context.Background(),
		testSyntheticEvent(),
	)

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	clientErr, ok := err.(*Error)
	if !ok {
		t.Fatalf("expected *Error, got %T", err)
	}

	if clientErr.StatusCode != http.StatusUnauthorized {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusUnauthorized,
			clientErr.StatusCode,
		)
	}
}

func TestClientSendServiceUnavailable(t *testing.T) {
	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusServiceUnavailable)
		}),
	)
	defer server.Close()

	client := NewClient(
		server.Client(),
		server.URL,
		"test-api-key",
	)

	err := client.Send(
		context.Background(),
		testSyntheticEvent(),
	)

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	clientErr, ok := err.(*Error)
	if !ok {
		t.Fatalf("expected *Error, got %T", err)
	}

	if clientErr.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusServiceUnavailable,
			clientErr.StatusCode,
		)
	}
}

func TestClientSendUnexpectedStatus(t *testing.T) {
	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}),
	)
	defer server.Close()

	client := NewClient(
		server.Client(),
		server.URL,
		"test-api-key",
	)

	err := client.Send(
		context.Background(),
		testSyntheticEvent(),
	)

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	clientErr, ok := err.(*Error)
	if !ok {
		t.Fatalf("expected *Error, got %T", err)
	}

	if clientErr.StatusCode != http.StatusInternalServerError {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusInternalServerError,
			clientErr.StatusCode,
		)
	}
}

func TestClientSendNetworkError(t *testing.T) {
	client := NewClient(
		&http.Client{
			Timeout: 100 * time.Millisecond,
		},
		"http://127.0.0.1:1",
		"test-api-key",
	)

	err := client.Send(
		context.Background(),
		testSyntheticEvent(),
	)

	if err == nil {
		t.Fatal("expected network error, got nil")
	}
}

func TestClientSendContextCancellation(t *testing.T) {
	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			<-r.Context().Done()
		}),
	)
	defer server.Close()

	client := NewClient(
		server.Client(),
		server.URL,
		"test-api-key",
	)

	ctx, cancel := context.WithCancel(
		context.Background(),
	)
	cancel()

	err := client.Send(
		ctx,
		testSyntheticEvent(),
	)

	if err == nil {
		t.Fatal("expected context cancellation error, got nil")
	}
}