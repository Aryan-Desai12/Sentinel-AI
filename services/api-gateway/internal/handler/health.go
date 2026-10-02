package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
)

type HealthHandler struct {
	DB       *sql.DB
	Redis    *redis.Client
	Instance string
}

func (h HealthHandler) HandleHealth(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	if err := h.DB.PingContext(ctx); err != nil {
		http.Error(
			w,
			`{"status":"unhealthy","dependency":"postgres"}`,
			http.StatusServiceUnavailable,
		)
		return
	}

	if err := h.Redis.Ping(ctx).Err(); err != nil {
		http.Error(
			w,
			`{"status":"unhealthy","dependency":"redis"}`,
			http.StatusServiceUnavailable,
		)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := map[string]string{
		"status":   "healthy",
		"instance": h.Instance,
	}

	json.NewEncoder(w).Encode(response)
}
