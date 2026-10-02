package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/Aryan-Desai12/sentinel-ai/services/api-gateway/internal/logger"
	"github.com/Aryan-Desai12/sentinel-ai/services/api-gateway/internal/node"
)

type NodeRegistrar interface {
	CreateNode(
		ctx context.Context,
		nodeID string,
		apiKey string,
	) error
}

type RegisterHandler struct {
	Repository NodeRegistrar
	Logger     *logger.Logger
}

type RegisterResponse struct {
	NodeID string `json:"node_id"`
	APIKey string `json:"api_key"`
}

func (h RegisterHandler) HandleRegister(
	w http.ResponseWriter,
	r *http.Request,
) {
	nodeID := node.GenerateNodeID()

	apiKey, err := node.GenerateAPIKey(nodeID)
	if err != nil {
		http.Error(
			w,
			`{"error":"failed to generate API key"}`,
			http.StatusInternalServerError,
		)
		return
	}

	err = h.Repository.CreateNode(
		r.Context(),
		nodeID,
		apiKey,
	)

	if err != nil {
		if h.Logger != nil {
			h.Logger.Error("node_registration_failed", map[string]any{
				"node_id": nodeID,
			})
		}

		http.Error(
			w,
			`{"error":"failed to register node"}`,
			http.StatusInternalServerError,
		)
		return
	}

	if h.Logger != nil {
		h.Logger.Info("node_registered", map[string]any{
			"node_id": nodeID,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	json.NewEncoder(w).Encode(RegisterResponse{
		NodeID: nodeID,
		APIKey: apiKey,
	})
}

func ExtractNodeID(apiKey string) string {
	parts := strings.SplitN(apiKey, ".", 2)

	if len(parts) != 2 {
		return ""
	}

	return parts[0]
}
