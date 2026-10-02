package proxy

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Aryan-Desai12/sentinel-ai/services/load-balancer/internal/balancer"
)

func TestProxy_POSTBodySurvivesRetry(t *testing.T) {
	backend1 := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "backend unavailable", http.StatusServiceUnavailable)
		}),
	)
	defer backend1.Close()

	backend2 := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatal(err)
			}

			if string(body) != `{"message":"hello"}` {
				t.Fatalf("unexpected body: %s", body)
			}

			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status":"received"}`))
		}),
	)
	defer backend2.Close()

	rr, err := balancer.NewRoundRobin([]string{
		backend1.URL,
		backend2.URL,
	})
	if err != nil {
		t.Fatal(err)
	}

	// Make backend1 the first selected backend.
	rr.Next()

	proxy := &Proxy{
		Balancer: rr,
	}

	req := httptest.NewRequest(
		http.MethodPost,
		"/test",
		strings.NewReader(`{"message":"hello"}`),
	)

	rec := httptest.NewRecorder()

	proxy.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	if rec.Body.String() != `{"status":"received"}` {
		t.Fatalf("unexpected response: %s", rec.Body.String())
	}
}
