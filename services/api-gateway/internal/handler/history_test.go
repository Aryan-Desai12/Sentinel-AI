package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Aryan-Desai12/sentinel-ai/services/api-gateway/internal/model"
)

type fakeMetricsRepository struct {
	metrics []model.Telemetry
	err     error
}

func (f fakeMetricsRepository) GetHistory(
	ctx context.Context,
	nodeID string,
	from time.Time,
	to time.Time,
) ([]model.Telemetry, error) {
	return f.metrics, f.err
}

func TestHandleHistory(t *testing.T) {
	repository := fakeMetricsRepository{
		metrics: []model.Telemetry{
			{
				NodeID:    "node-001",
				Timestamp: time.Date(2026, 10, 1, 10, 0, 0, 0, time.UTC),
				CPUPct:    72.4,
				MemPct:    61.2,
				DiskPct:   45.8,
			},
		},
	}

	handler := HistoryHandler{
		Repository: repository,
	}

	req := httptest.NewRequest(
		http.MethodGet,
		"/metrics/history?node_id=node-001&from=2026-10-01T09:00:00Z&to=2026-10-01T11:00:00Z",
		nil,
	)

	rec := httptest.NewRecorder()

	handler.HandleHistory(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusOK,
			rec.Code,
		)
	}
}

func TestHandleHistoryValidation(t *testing.T) {
	handler := HistoryHandler{
		Repository: fakeMetricsRepository{},
	}

	tests := []struct {
		name string
		url  string
	}{
		{
			name: "missing node id",
			url:  "/metrics/history?from=2026-10-01T09:00:00Z&to=2026-10-01T11:00:00Z",
		},
		{
			name: "invalid from",
			url:  "/metrics/history?node_id=node-001&from=invalid&to=2026-10-01T11:00:00Z",
		},
		{
			name: "invalid to",
			url:  "/metrics/history?node_id=node-001&from=2026-10-01T09:00:00Z&to=invalid",
		},
		{
			name: "from after to",
			url:  "/metrics/history?node_id=node-001&from=2026-10-01T12:00:00Z&to=2026-10-01T11:00:00Z",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(
				http.MethodGet,
				tt.url,
				nil,
			)

			rec := httptest.NewRecorder()

			handler.HandleHistory(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Fatalf(
					"expected status %d, got %d",
					http.StatusBadRequest,
					rec.Code,
				)
			}
		})
	}
}
