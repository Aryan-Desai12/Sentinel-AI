package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Aryan-Desai12/sentinel-ai/services/api-gateway/internal/read"
)

type fakeIncidentsRepository struct {
	incidents []read.Incident
}

func (f *fakeIncidentsRepository) GetIncidents(
	ctx context.Context,
	from time.Time,
	to time.Time,
) ([]read.Incident, error) {
	return f.incidents, nil
}

func TestHandleIncidents_ValidRequest(t *testing.T) {
	repo := &fakeIncidentsRepository{
		incidents: []read.Incident{
			{
				ID:       "incident-001",
				Title:    "High CPU usage",
				Severity: "critical",
				Status:   "open",
				StartedAt: time.Date(
					2026, 9, 1, 10, 0, 0, 0, time.UTC,
				),
			},
		},
	}

	handler := IncidentsHandler{
		Repository: repo,
	}

	req := httptest.NewRequest(
		http.MethodGet,
		"/incidents?from=2026-09-01T00:00:00Z&to=2026-09-02T00:00:00Z",
		nil,
	)

	rec := httptest.NewRecorder()

	handler.HandleIncidents(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestHandleIncidents_InvalidFrom(t *testing.T) {
	handler := IncidentsHandler{
		Repository: &fakeIncidentsRepository{},
	}

	req := httptest.NewRequest(
		http.MethodGet,
		"/incidents?from=invalid&to=2026-09-02T00:00:00Z",
		nil,
	)

	rec := httptest.NewRecorder()

	handler.HandleIncidents(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestHandleIncidents_InvalidTo(t *testing.T) {
	handler := IncidentsHandler{
		Repository: &fakeIncidentsRepository{},
	}

	req := httptest.NewRequest(
		http.MethodGet,
		"/incidents?from=2026-09-01T00:00:00Z&to=invalid",
		nil,
	)

	rec := httptest.NewRecorder()

	handler.HandleIncidents(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestHandleIncidents_FromAfterTo(t *testing.T) {
	handler := IncidentsHandler{
		Repository: &fakeIncidentsRepository{},
	}

	req := httptest.NewRequest(
		http.MethodGet,
		"/incidents?from=2026-09-03T00:00:00Z&to=2026-09-02T00:00:00Z",
		nil,
	)

	rec := httptest.NewRecorder()

	handler.HandleIncidents(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}
