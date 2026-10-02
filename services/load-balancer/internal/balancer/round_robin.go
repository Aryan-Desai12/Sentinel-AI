package balancer

import (
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/Aryan-Desai12/sentinel-ai/services/load-balancer/internal/logger"
)

type RoundRobin struct {
	backends []*Backend
	healthy  map[string]bool

	mu     sync.Mutex
	next   int
	Logger *logger.Logger
}

func NewRoundRobin(urls []string) (*RoundRobin, error) {
	backends := make([]*Backend, 0, len(urls))
	healthy := make(map[string]bool)

	for _, rawURL := range urls {
		u, err := url.Parse(rawURL)
		if err != nil {
			return nil, err
		}

		backend := &Backend{
			URL: u,
		}

		backends = append(backends, backend)
		healthy[u.String()] = true
	}

	return &RoundRobin{
		backends: backends,
		healthy:  healthy,
	}, nil
}

func (r *RoundRobin) Next() *Backend {
	r.mu.Lock()
	defer r.mu.Unlock()

	if len(r.backends) == 0 {
		return nil
	}

	for i := 0; i < len(r.backends); i++ {
		backend := r.backends[r.next]

		r.next = (r.next + 1) % len(r.backends)

		if r.healthy[backend.URL.String()] {
			return backend
		}
	}

	return nil
}

func (r *RoundRobin) StartHealthChecks() {
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for {
			r.checkHealth()

			<-ticker.C
		}
	}()
}

func (r *RoundRobin) checkHealth() {
	client := &http.Client{
		Timeout: 2 * time.Second,
	}

	for _, backend := range r.backends {
		resp, err := client.Get(backend.URL.String() + "/health")

		isHealthy := err == nil &&
			resp.StatusCode == http.StatusOK

		if resp != nil {
			resp.Body.Close()
		}

		r.mu.Lock()
		r.healthy[backend.URL.String()] = isHealthy
		r.mu.Unlock()

		if r.Logger != nil {
			r.Logger.Info("backend_health_check", map[string]any{
				"backend": backend.URL.String(),
				"healthy": isHealthy,
			})
		}
	}
}

func (r *RoundRobin) MarkUnhealthy(backend *Backend) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.healthy[backend.URL.String()] = false

	if r.Logger != nil {
		r.Logger.Warn("backend_marked_unhealthy", map[string]any{
			"backend": backend.URL.String(),
		})
	}
}
