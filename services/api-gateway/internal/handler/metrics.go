package handler

import (
	"encoding/json"
	"net/http"

	"github.com/Aryan-Desai12/sentinel-ai/services/api-gateway/internal/auth"
	"github.com/Aryan-Desai12/sentinel-ai/services/api-gateway/internal/dedupe"
	"github.com/Aryan-Desai12/sentinel-ai/services/api-gateway/internal/logger"
	"github.com/Aryan-Desai12/sentinel-ai/services/api-gateway/internal/model"
	"github.com/Aryan-Desai12/sentinel-ai/services/api-gateway/internal/publisher"
	"github.com/Aryan-Desai12/sentinel-ai/services/api-gateway/internal/ratelimit"
	"github.com/Aryan-Desai12/sentinel-ai/services/api-gateway/internal/validation"
)

type MetricsHandler struct {
	Publisher   publisher.EventPublisher
	DedupeStore *dedupe.Store
	RateLimiter *ratelimit.Limiter
	Logger      *logger.Logger
}

func (h MetricsHandler) HandleNodeMetrics(
	w http.ResponseWriter,
	r *http.Request,
) {
	var telemetry model.Telemetry

	decoder := json.NewDecoder(r.Body)

	if err := decoder.Decode(&telemetry); err != nil {
		http.Error(
			w,
			`{"error":"malformed JSON payload"}`,
			http.StatusBadRequest,
		)
		return
	}

	if err := validation.ValidateTelemetry(telemetry); err != nil {
		http.Error(
			w,
			`{"error":"`+err.Error()+`"}`,
			http.StatusBadRequest,
		)
		return
	}

	telemetry.EventType = "node_metric"

	authenticatedNodeID := auth.NodeIDFromContext(r.Context())

	allowed, err := h.RateLimiter.Allow(
		r.Context(),
		authenticatedNodeID,
	)

	if err != nil {
		http.Error(
			w,
			`{"error":"rate limiter unavailable"}`,
			http.StatusServiceUnavailable,
		)
		return
	}

	if !allowed {
		if h.Logger != nil {
			h.Logger.Warn("rate_limit_exceeded", map[string]any{
				"node_id": authenticatedNodeID,
			})
		}

		http.Error(
			w,
			`{"error":"rate limit exceeded"}`,
			http.StatusTooManyRequests,
		)
		return
	}

	if telemetry.NodeID != authenticatedNodeID {
		http.Error(
			w,
			`{"error":"node_id does not match API key"}`,
			http.StatusUnauthorized,
		)
		return
	}

	duplicate, err := h.DedupeStore.Exists(
		r.Context(),
		telemetry.NodeID,
		telemetry.Timestamp,
	)

	if err != nil {
		http.Error(
			w,
			`{"error":"deduplication check failed"}`,
			http.StatusServiceUnavailable,
		)
		return
	}

	if duplicate {
		if h.Logger != nil {
			h.Logger.Info("duplicate_telemetry", map[string]any{
				"node_id":   telemetry.NodeID,
				"timestamp": telemetry.Timestamp,
			})
		}

		w.WriteHeader(http.StatusAccepted)
		return
	}

	payload, err := json.Marshal(telemetry)
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
		[]byte(telemetry.NodeID),
		payload,
	)

	if err != nil {
		if h.Logger != nil {
			h.Logger.Error("kafka_publish_failed", map[string]any{
				"node_id": telemetry.NodeID,
				"topic":   "metrics.raw",
			})
		}

		http.Error(
			w,
			`{"error":"failed to publish event"}`,
			http.StatusServiceUnavailable,
		)
		return
	}

	err = h.DedupeStore.Mark(
		r.Context(),
		telemetry.NodeID,
		telemetry.Timestamp,
	)

	if err != nil {
		http.Error(
			w,
			`{"error":"failed to record deduplication state"}`,
			http.StatusInternalServerError,
		)
		return
	}

	if h.Logger != nil {
		h.Logger.Info("telemetry_published", map[string]any{
			"node_id":   telemetry.NodeID,
			"timestamp": telemetry.Timestamp,
			"topic":     "metrics.raw",
		})
	}

	w.WriteHeader(http.StatusAccepted)
}
