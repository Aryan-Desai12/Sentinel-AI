package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/Aryan-Desai12/sentinel-ai/services/api-gateway/internal/read"
)

type HistoryHandler struct {
	Repository read.MetricsRepository
}

func (h HistoryHandler) HandleHistory(
	w http.ResponseWriter,
	r *http.Request,
) {
	query := r.URL.Query()

	nodeID := query.Get("node_id")
	fromStr := query.Get("from")
	toStr := query.Get("to")

	if nodeID == "" {
		http.Error(
			w,
			`{"error":"node_id is required"}`,
			http.StatusBadRequest,
		)
		return
	}

	from, err := time.Parse(time.RFC3339, fromStr)
	if err != nil {
		http.Error(
			w,
			`{"error":"invalid from timestamp"}`,
			http.StatusBadRequest,
		)
		return
	}

	to, err := time.Parse(time.RFC3339, toStr)
	if err != nil {
		http.Error(
			w,
			`{"error":"invalid to timestamp"}`,
			http.StatusBadRequest,
		)
		return
	}

	if from.After(to) {
		http.Error(
			w,
			`{"error":"from must be before to"}`,
			http.StatusBadRequest,
		)
		return
	}

	metrics, err := h.Repository.GetHistory(
		r.Context(),
		nodeID,
		from,
		to,
	)

	if err != nil {
		http.Error(
			w,
			`{"error":"failed to fetch history"}`,
			http.StatusInternalServerError,
		)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(metrics); err != nil {
		return
	}
}
