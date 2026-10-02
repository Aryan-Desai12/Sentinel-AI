package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/Aryan-Desai12/sentinel-ai/services/api-gateway/internal/read"
)

type IncidentsHandler struct {
	Repository read.IncidentsRepository
}

func (h IncidentsHandler) HandleIncidents(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	fromStr := query.Get("from")
	toStr := query.Get("to")

	from, err := time.Parse(time.RFC3339, fromStr)
	if err != nil {
		http.Error(w, `{"error":"invalid from"}`, http.StatusBadRequest)
		return
	}

	to, err := time.Parse(time.RFC3339, toStr)
	if err != nil {
		http.Error(w, `{"error":"invalid to"}`, http.StatusBadRequest)
		return
	}

	if from.After(to) {
		http.Error(w, `{"error":"from must be before to"}`, http.StatusBadRequest)
		return
	}

	incidents, err := h.Repository.GetIncidents(
		r.Context(),
		from,
		to,
	)
	if err != nil {
		http.Error(
			w,
			`{"error":"failed to fetch incidents"}`,
			http.StatusInternalServerError,
		)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(incidents); err != nil {
		return
	}
}
