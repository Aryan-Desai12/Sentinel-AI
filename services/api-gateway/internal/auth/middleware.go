package auth

import (
	"context"
	"net/http"
	"strings"

	"github.com/Aryan-Desai12/sentinel-ai/services/api-gateway/internal/logger"
)

type contextKey string

const nodeIDKey contextKey = "node_id"

type KeyStore interface {
	ValidateAPIKey(ctx context.Context, nodeID string, apiKey string) bool
}

type Middleware struct {
	Keys   KeyStore
	Logger *logger.Logger
}

func (m Middleware) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiKey := r.Header.Get("X-API-Key")

		if apiKey == "" {
			if m.Logger != nil {
				m.Logger.Warn("authentication_failed", map[string]any{
					"reason": "missing_api_key",
				})
			}

			http.Error(
				w,
				`{"error":"invalid or missing API key"}`,
				http.StatusUnauthorized,
			)
			return
		}

		parts := strings.SplitN(apiKey, ".", 2)

		if len(parts) != 2 || parts[0] == "" {
			if m.Logger != nil {
				m.Logger.Warn("authentication_failed", map[string]any{
					"reason": "malformed_api_key",
				})
			}

			http.Error(
				w,
				`{"error":"invalid API key"}`,
				http.StatusUnauthorized,
			)
			return
		}

		nodeID := parts[0]

		if !m.Keys.ValidateAPIKey(r.Context(), nodeID, apiKey) {
			if m.Logger != nil {
				m.Logger.Warn("authentication_failed", map[string]any{
					"reason":  "invalid_api_key",
					"node_id": nodeID,
				})
			}

			http.Error(
				w,
				`{"error":"invalid API key"}`,
				http.StatusUnauthorized,
			)
			return
		}

		if m.Logger != nil {
			m.Logger.Info("authentication_success", map[string]any{
				"node_id": nodeID,
			})
		}

		ctx := context.WithValue(r.Context(), nodeIDKey, nodeID)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func NodeIDFromContext(ctx context.Context) string {
	nodeID, _ := ctx.Value(nodeIDKey).(string)
	return nodeID
}

func WithNodeID(ctx context.Context, nodeID string) context.Context {
	return context.WithValue(ctx, nodeIDKey, nodeID)
}
