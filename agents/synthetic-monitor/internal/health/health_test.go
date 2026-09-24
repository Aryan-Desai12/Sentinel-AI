package health

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandler_ReturnsHealthyStatus(t *testing.T) {
	handler := NewHandler(2)

	req := httptest.NewRequest(
		http.MethodGet,
		"/health",
		nil,
	)

	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf(
			"expected status 200, got %d",
			recorder.Code,
		)
	}

	var response Response

	if err := json.NewDecoder(
		recorder.Body,
	).Decode(&response); err != nil {
		t.Fatalf(
			"failed to decode response: %v",
			err,
		)
	}

	if response.Status != "ok" {
		t.Fatalf(
			"expected status ok, got %q",
			response.Status,
		)
	}

	if response.Service != "synthetic-monitor" {
		t.Fatalf(
			"unexpected service: %q",
			response.Service,
		)
	}

	if response.ActiveChecks != 2 {
		t.Fatalf(
			"expected 2 active checks, got %d",
			response.ActiveChecks,
		)
	}
}

func TestHandler_ReturnsZeroActiveChecks(t *testing.T) {
	handler := NewHandler(0)

	req := httptest.NewRequest(
		http.MethodGet,
		"/health",
		nil,
	)

	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf(
			"expected status 200, got %d",
			recorder.Code,
		)
	}

	var response Response

	if err := json.NewDecoder(
		recorder.Body,
	).Decode(&response); err != nil {
		t.Fatalf(
			"failed to decode response: %v",
			err,
		)
	}

	if response.ActiveChecks != 0 {
		t.Fatalf(
			"expected 0 active checks, got %d",
			response.ActiveChecks,
		)
	}
}

func TestHandler_RejectsNonGET(t *testing.T) {
	handler := NewHandler(2)

	req := httptest.NewRequest(
		http.MethodPost,
		"/health",
		nil,
	)

	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf(
			"expected status 405, got %d",
			recorder.Code,
		)
	}
}