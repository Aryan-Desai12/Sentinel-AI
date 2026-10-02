package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/Aryan-Desai12/sentinel-ai/services/api-gateway/internal/auth"
	"github.com/Aryan-Desai12/sentinel-ai/services/api-gateway/internal/dedupe"
	"github.com/Aryan-Desai12/sentinel-ai/services/api-gateway/internal/publisher"
	"github.com/Aryan-Desai12/sentinel-ai/services/api-gateway/internal/ratelimit"
)

type fakePublisher struct {
	err error
}

func (f fakePublisher) Publish(
	ctx context.Context,
	topic string,
	key []byte,
	value []byte,
) error {
	return f.err
}

func TestHandleNodeMetrics(t *testing.T) {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	ctx := context.Background()

	if err := client.Ping(ctx).Err(); err != nil {
		t.Skip("Redis is not running")
	}

	dedupeStore := dedupe.NewStore(client, time.Minute)
	rateLimiter := ratelimit.NewLimiter(client, 100)

	tests := []struct {
		name           string
		payload        string
		authenticated  string
		publisherError error
		expectedStatus int
	}{
		{
			name: "valid payload",
			payload: `{
				"node_id": "node-001",
				"timestamp": "2026-09-29T19:00:00Z",
				"cpu_pct": 72.4,
				"mem_pct": 61.2,
				"disk_pct": 45.8,
				"net_in": 123456,
				"net_out": 654321
			}`,
			authenticated:  "node-001",
			expectedStatus: http.StatusAccepted,
		},
		{
			name:           "malformed JSON",
			payload:        `{invalid-json}`,
			authenticated:  "node-001",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "invalid telemetry",
			payload: `{
				"node_id": "node-001",
				"timestamp": "2026-09-29T19:00:00Z",
				"cpu_pct": 150,
				"mem_pct": 61.2,
				"disk_pct": 45.8,
				"net_in": 123456,
				"net_out": 654321
			}`,
			authenticated:  "node-001",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "node ID does not match API key",
			payload: `{
				"node_id": "node-002",
				"timestamp": "2026-09-29T19:00:00Z",
				"cpu_pct": 72.4,
				"mem_pct": 61.2,
				"disk_pct": 45.8,
				"net_in": 123456,
				"net_out": 654321
			}`,
			authenticated:  "node-001",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "publisher failure",
			payload: `{
				"node_id": "node-001",
				"timestamp": "2026-09-29T19:00:00Z",
				"cpu_pct": 72.4,
				"mem_pct": 61.2,
				"disk_pct": 45.8,
				"net_in": 123456,
				"net_out": 654321
			}`,
			authenticated:  "node-001",
			publisherError: context.DeadlineExceeded,
			expectedStatus: http.StatusServiceUnavailable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			// Clean rate-limit state for this test.
			client.Del(
				ctx,
				"sentinel:ratelimit:"+tt.authenticated,
			)

			client.Del(
				ctx,
				"sentinel:dedupe:"+tt.authenticated+":2026-09-29T19:00:00Z",
			)

			req := httptest.NewRequest(
				http.MethodPost,
				"/metrics/node",
				strings.NewReader(tt.payload),
			)

			req = req.WithContext(
				auth.WithNodeID(
					req.Context(),
					tt.authenticated,
				),
			)

			rec := httptest.NewRecorder()

			handler := MetricsHandler{
				Publisher: fakePublisher{
					err: tt.publisherError,
				},
				DedupeStore: dedupeStore,
				RateLimiter: rateLimiter,
			}

			handler.HandleNodeMetrics(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Fatalf(
					"expected status %d, got %d",
					tt.expectedStatus,
					rec.Code,
				)
			}
		})
	}
}

func TestHandleNodeMetricsDuplicate(t *testing.T) {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	ctx := context.Background()

	if err := client.Ping(ctx).Err(); err != nil {
		t.Skip("Redis is not running")
	}

	store := dedupe.NewStore(client, time.Minute)
	rateLimiter := ratelimit.NewLimiter(client, 100)

	nodeID := "node-duplicate"
	timestamp := time.Date(
		2026,
		9,
		29,
		19,
		0,
		0,
		0,
		time.UTC,
	)

	// Clean both Redis states before the test.
	client.Del(
		ctx,
		"sentinel:ratelimit:"+nodeID,
	)

	client.Del(
		ctx,
		"sentinel:dedupe:"+nodeID+":"+timestamp.Format(time.RFC3339Nano),
	)

	// Mark the event as already processed.
	if err := store.Mark(
		ctx,
		nodeID,
		timestamp,
	); err != nil {
		t.Fatalf("failed to mark test event: %v", err)
	}

	telemetry := `{
		"node_id": "node-duplicate",
		"timestamp": "2026-09-29T19:00:00Z",
		"cpu_pct": 72.4,
		"mem_pct": 61.2,
		"disk_pct": 45.8,
		"net_in": 123456,
		"net_out": 654321
	}`

	handler := MetricsHandler{
		Publisher:   fakePublisher{},
		DedupeStore: store,
		RateLimiter: rateLimiter,
	}

	req := httptest.NewRequest(
		http.MethodPost,
		"/metrics/node",
		strings.NewReader(telemetry),
	)

	req = req.WithContext(
		auth.WithNodeID(
			req.Context(),
			nodeID,
		),
	)

	rec := httptest.NewRecorder()

	handler.HandleNodeMetrics(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusAccepted,
			rec.Code,
		)
	}

	// Cleanup.
	client.Del(
		ctx,
		"sentinel:ratelimit:"+nodeID,
	)

	client.Del(
		ctx,
		"sentinel:dedupe:"+nodeID+":"+timestamp.Format(time.RFC3339Nano),
	)
}

func TestHandleNodeMetricsRateLimit(t *testing.T) {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	ctx := context.Background()

	if err := client.Ping(ctx).Err(); err != nil {
		t.Skip("Redis is not running")
	}

	nodeID := "node-rate-limit"

	// Clean rate-limit state before the test.
	client.Del(
		ctx,
		"sentinel:ratelimit:"+nodeID,
	)

	dedupeStore := dedupe.NewStore(client, time.Minute)

	// Allow only 3 requests in the current window.
	rateLimiter := ratelimit.NewLimiter(client, 3)

	handler := MetricsHandler{
		Publisher:   fakePublisher{},
		DedupeStore: dedupeStore,
		RateLimiter: rateLimiter,
	}

	payload := `{
		"node_id": "node-rate-limit",
		"timestamp": "2026-10-01T10:00:00Z",
		"cpu_pct": 72.4,
		"mem_pct": 61.2,
		"disk_pct": 45.8,
		"net_in": 123456,
		"net_out": 654321
	}`

	for i := 1; i <= 3; i++ {
		req := httptest.NewRequest(
			http.MethodPost,
			"/metrics/node",
			strings.NewReader(payload),
		)

		req = req.WithContext(
			auth.WithNodeID(
				req.Context(),
				nodeID,
			),
		)

		rec := httptest.NewRecorder()

		handler.HandleNodeMetrics(rec, req)

		if rec.Code != http.StatusAccepted {
			t.Fatalf(
				"request %d: expected status %d, got %d",
				i,
				http.StatusAccepted,
				rec.Code,
			)
		}
	}

	// Fourth request should exceed the limit.
	req := httptest.NewRequest(
		http.MethodPost,
		"/metrics/node",
		strings.NewReader(payload),
	)

	req = req.WithContext(
		auth.WithNodeID(
			req.Context(),
			nodeID,
		),
	)

	rec := httptest.NewRecorder()

	handler.HandleNodeMetrics(rec, req)

	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusTooManyRequests,
			rec.Code,
		)
	}

	// Cleanup.
	client.Del(
		ctx,
		"sentinel:ratelimit:"+nodeID,
	)
}

var _ publisher.EventPublisher = fakePublisher{}
