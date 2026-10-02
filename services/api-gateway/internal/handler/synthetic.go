package handler

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/Aryan-Desai12/sentinel-ai/services/api-gateway/internal/logger"
	"github.com/Aryan-Desai12/sentinel-ai/services/api-gateway/internal/model"
	"github.com/Aryan-Desai12/sentinel-ai/services/api-gateway/internal/publisher"
)

type SyntheticHandler struct {
	Publisher publisher.EventPublisher
	Logger    *logger.Logger
}

func validateSyntheticCheck(check model.SyntheticCheck) error {
	if check.URL == "" {
		return fmt.Errorf("url is required")
	}

	if check.Status == "" {
		return fmt.Errorf("status is required")
	}

	switch check.Status {
	case "up", "degraded", "down":
	default:
		return fmt.Errorf("status must be one of: up, degraded, down")
	}

	if check.LatencyMs < 0 {
		return fmt.Errorf("latency_ms cannot be negative")
	}

	if check.TTFBMs < 0 {
		return fmt.Errorf("ttfb_ms cannot be negative")
	}

	if check.HTTPStatus < 0 {
		return fmt.Errorf("http_status cannot be negative")
	}

	if check.Timestamp.IsZero() {
		return fmt.Errorf("timestamp is required")
	}

	return nil
}

func (h SyntheticHandler) HandleSyntheticCheck(
	w http.ResponseWriter,
	r *http.Request,
) {
	var check model.SyntheticCheck

	decoder := json.NewDecoder(r.Body)

	if err := decoder.Decode(&check); err != nil {
		http.Error(
			w,
			`{"error":"malformed JSON payload"}`,
			http.StatusBadRequest,
		)
		return
	}

	if err := validateSyntheticCheck(check); err != nil {
		http.Error(
			w,
			`{"error":"`+err.Error()+`"}`,
			http.StatusBadRequest,
		)
		return
	}

	check.EventType = "synthetic_check"

	payload, err := json.Marshal(check)
	if err != nil {
		http.Error(
			w,
			`{"error":"failed to encode payload"}`,
			http.StatusInternalServerError,
		)
		return
	}

	err = h.Publisher.Publish(
		r.Context(),
		"metrics.raw",
		[]byte(check.Service),
		payload,
	)

	if err != nil {
		if h.Logger != nil {
			h.Logger.Error("kafka_publish_failed", map[string]any{
				"service": check.Service,
				"topic":   "metrics.raw",
				"type":    "synthetic_check",
			})
		}

		http.Error(
			w,
			`{"error":"failed to publish event"}`,
			http.StatusServiceUnavailable,
		)
		return
	}

	if h.Logger != nil {
		h.Logger.Info("synthetic_check_published", map[string]any{
			"service":   check.Service,
			"status":    check.Status,
			"timestamp": check.Timestamp,
			"topic":     "metrics.raw",
		})
	}

	w.WriteHeader(http.StatusAccepted)
}
