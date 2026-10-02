package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

type fakeKeyStore struct {
	validKey string
}

func (f fakeKeyStore) ValidateAPIKey(
	ctx context.Context,
	nodeID string,
	apiKey string,
) bool {
	return apiKey == f.validKey
}

func TestAuthenticate(t *testing.T) {
	store := fakeKeyStore{
		validKey: "node-123.test-api-key",
	}

	middleware := Middleware{
		Keys: store,
	}

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
	})

	handler := middleware.Authenticate(next)

	tests := []struct {
		name           string
		apiKey         string
		expectedStatus int
	}{
		{
			name:           "missing API key",
			apiKey:         "",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "invalid API key",
			apiKey:         "wrong-key",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "valid API key",
			apiKey:         "node-123.test-api-key",
			expectedStatus: http.StatusAccepted,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(
				http.MethodPost,
				"/metrics/node",
				nil,
			)

			if tt.apiKey != "" {
				req.Header.Set("X-API-Key", tt.apiKey)
			}

			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Fatalf(
					"expected status %d, got %d",
					tt.expectedStatus,
					rec.Code,
				)
			}
		})
	}
}
