package proxy

import (
	"bytes"
	"io"
	"net/http"
	"time"

	"github.com/Aryan-Desai12/sentinel-ai/services/load-balancer/internal/balancer"
)

type Proxy struct {
	Balancer *balancer.RoundRobin
	Client   *http.Client
}

func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	client := p.Client

	if client == nil {
		client = &http.Client{
			Timeout: 5 * time.Second,
		}
	}

	// Read the request body once so it can be recreated for retries.
	var body []byte

	if r.Body != nil {
		var err error

		body, err = io.ReadAll(r.Body)
		if err != nil {
			http.Error(
				w,
				`{"error":"failed to read request body"}`,
				http.StatusBadRequest,
			)
			return
		}

		r.Body.Close()
	}

	for attempt := 0; attempt < 2; attempt++ {
		backend := p.Balancer.Next()

		if backend == nil {
			http.Error(
				w,
				`{"error":"no gateway available"}`,
				http.StatusServiceUnavailable,
			)
			return
		}

		reqURL := *backend.URL
		reqURL.Path = r.URL.Path
		reqURL.RawQuery = r.URL.RawQuery

		req, err := http.NewRequestWithContext(
			r.Context(),
			r.Method,
			reqURL.String(),
			bytes.NewReader(body),
		)
		if err != nil {
			http.Error(
				w,
				`{"error":"failed to create upstream request"}`,
				http.StatusInternalServerError,
			)
			return
		}

		req.Header = r.Header.Clone()
		req.Header.Set("X-Forwarded-Host", r.Host)
		req.Header.Set("X-Forwarded-Proto", "http")

		resp, err := client.Do(req)

		if err != nil {
			p.Balancer.MarkUnhealthy(backend)

			if attempt == 0 {
				continue
			}

			http.Error(
				w,
				`{"error":"gateway unavailable"}`,
				http.StatusServiceUnavailable,
			)
			return
		}

		defer resp.Body.Close()

		for key, values := range resp.Header {
			for _, value := range values {
				w.Header().Add(key, value)
			}
		}

		w.WriteHeader(resp.StatusCode)

		_, _ = io.Copy(w, resp.Body)

		return
	}
}
