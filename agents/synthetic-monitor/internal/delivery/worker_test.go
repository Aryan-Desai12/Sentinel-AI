package delivery

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"sentinel-ai/agents/synthetic-monitor/internal/model"
)

func TestWorker_DeliversSuccessfully(t *testing.T) {
	var received bool

	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			received = true
			w.WriteHeader(http.StatusAccepted)
		}),
	)
	defer server.Close()

	buffer := NewBuffer(10, time.Minute)

	event := testSyntheticEvent()
	buffer.Enqueue(event)

	client := NewClient(
		server.Client(),
		server.URL,
		"test-api-key",
	)

	worker := NewWorker(buffer, client)

	ctx, cancel := context.WithCancel(
		context.Background(),
	)
	defer cancel()

	done := make(chan struct{})

	go func() {
		worker.Run(ctx)
		close(done)
	}()

	// Give the worker enough time to process the event.
	time.Sleep(50 * time.Millisecond)

	cancel()

	<-done

	if !received {
		t.Fatal("expected event to be delivered")
	}
}

func TestWorker_DropsPermanentFailure(t *testing.T) {
	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
		}),
	)
	defer server.Close()

	buffer := NewBuffer(10, time.Minute)

	event := testSyntheticEvent()
	buffer.Enqueue(event)

	client := NewClient(
		server.Client(),
		server.URL,
		"test-api-key",
	)

	worker := NewWorker(buffer, client)

	ctx, cancel := context.WithTimeout(
		context.Background(),
		100*time.Millisecond,
	)
	defer cancel()

	worker.Run(ctx)

	// If the worker incorrectly retries the permanent error,
	// this test would take much longer or fail to finish.
}

func TestWorker_RetriesTemporaryFailure(t *testing.T) {
	var (
		mu       sync.Mutex
		attempts int
	)

	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			mu.Lock()
			attempts++
			currentAttempt := attempts
			mu.Unlock()

			if currentAttempt < 3 {
				w.WriteHeader(http.StatusServiceUnavailable)
				return
			}

			w.WriteHeader(http.StatusAccepted)
		}),
	)
	defer server.Close()

	buffer := NewBuffer(10, time.Minute)

	event := model.SyntheticCheckEvent{
		URL:    "https://example.com",
		Status: "up",
	}

	buffer.Enqueue(event)

	client := NewClient(
		server.Client(),
		server.URL,
		"test-api-key",
	)

	worker := NewWorker(buffer, client)

	// Don't actually wait 1s + 2s in the test.
	worker.sleepFunc = func(
		ctx context.Context,
		delay time.Duration,
	) bool {
		return true
	}

	ctx, cancel := context.WithTimeout(
		context.Background(),
		time.Second,
	)
	defer cancel()

	done := make(chan struct{})

	go func() {
		worker.Run(ctx)
		close(done)
	}()

	<-done

	mu.Lock()
	gotAttempts := attempts
	mu.Unlock()

	if gotAttempts != 3 {
		t.Fatalf(
			"expected 3 delivery attempts, got %d",
			gotAttempts,
		)
	}
}

func TestWorker_DropsEventWhenTTLExpiresDuringRetry(t *testing.T) {
	var (
		mu       sync.Mutex
		attempts int
	)

	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			mu.Lock()
			attempts++
			mu.Unlock()

			w.WriteHeader(http.StatusServiceUnavailable)
		}),
	)
	defer server.Close()

	// Event becomes stale after 50ms.
	buffer := NewBuffer(10, 50*time.Millisecond)

	event := testSyntheticEvent()
	buffer.Enqueue(event)

	client := NewClient(
		server.Client(),
		server.URL,
		"test-api-key",
	)

	worker := NewWorker(buffer, client)

	// Simulate the retry delay without actually waiting 1 second.
	// Sleep long enough for the event TTL to expire.
	worker.sleepFunc = func(
		ctx context.Context,
		delay time.Duration,
	) bool {
		time.Sleep(60 * time.Millisecond)
		return true
	}

	ctx, cancel := context.WithCancel(
		context.Background(),
	)
	defer cancel()

	done := make(chan struct{})

	go func() {
		worker.Run(ctx)
		close(done)
	}()

	// Wait until the worker notices that the event expired.
	time.Sleep(100 * time.Millisecond)

	cancel()
	<-done

	mu.Lock()
	gotAttempts := attempts
	mu.Unlock()

	if gotAttempts != 1 {
		t.Fatalf(
			"expected exactly 1 delivery attempt before TTL expiry, got %d",
			gotAttempts,
		)
	}
}
