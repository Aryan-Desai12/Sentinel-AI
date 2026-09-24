package health

import (
	"encoding/json"
	"net/http"
)

type Response struct {
	Status       string `json:"status"`
	Service      string `json:"service"`
	ActiveChecks int    `json:"active_checks"`
}

type Handler struct {
	activeChecks int
}

func NewHandler(activeChecks int) *Handler {
	return &Handler{
		activeChecks: activeChecks,
	}
}

func (h *Handler) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodGet {
		http.Error(
			w,
			"method not allowed",
			http.StatusMethodNotAllowed,
		)
		return
	}

	response := Response{
		Status:       "ok",
		Service:      "synthetic-monitor",
		ActiveChecks: h.activeChecks,
	}

	w.Header().Set(
		"Content-Type",
		"application/json",
	)

	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		return
	}
}