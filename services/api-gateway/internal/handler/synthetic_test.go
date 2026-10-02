package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type fakeSyntheticPublisher struct {
	called bool
}

func (f *fakeSyntheticPublisher) Publish(
	ctx context.Context,
	topic string,
	key []byte,
	value []byte,
) error {
	f.called = true
	return nil
}

func TestHandleSyntheticCheck(t *testing.T) {
	tests := []struct {
		name           string
		body           string
		expectedStatus int
	}{
		{
			name: "valid payload",
			body: `{
				"url": "https://example.com/health",
				"service": "checkout-api",
				"status": "up",
				"latency_ms": 120,
				"ttfb_ms": 40,
				"http_status": 200,
				"error_type": "",
				"timestamp": "2026-09-28T10:00:00Z"
			}`,
			expectedStatus: http.StatusAccepted,
		},
		{
			name:           "malformed JSON",
			body:           `{"url":`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "invalid status",
			body: `{
				"url": "https://example.com",
				"service": "test",
				"status": "unknown",
				"timestamp": "2026-09-28T10:00:00Z"
			}`,
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pub := &fakeSyntheticPublisher{}

			h := SyntheticHandler{
				Publisher: pub,
			}

			req := httptest.NewRequest(
				http.MethodPost,
				"/checks/synthetic",
				strings.NewReader(tt.body),
			)

			rec := httptest.NewRecorder()

			h.HandleSyntheticCheck(rec, req)

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