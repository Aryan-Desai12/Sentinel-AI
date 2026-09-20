package checker

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptrace"
	"strings"
	"syscall"
	"time"

	"sentinel-ai/agents/synthetic-monitor/internal/config"
)

type Result struct {
	URL        string
	Service    string
	Status     string
	LatencyMs  int64
	TTFBMs     int64
	HTTPStatus int
	ErrorType  string
	Timestamp  time.Time
}

type Checker struct {
	client *http.Client
}

func New(client *http.Client) *Checker {
	return &Checker{
		client: client,
	}
}

func (c *Checker) Check(
	ctx context.Context,
	target config.Target,
) Result {
	result := Result{
		URL:       target.URL,
		Service:   target.Service,
		Timestamp: time.Now().UTC(),
	}

	start := time.Now()

	var ttfb time.Time

	trace := &httptrace.ClientTrace{
		GotFirstResponseByte: func() {
			ttfb = time.Now()
		},
	}

	reqCtx := httptrace.WithClientTrace(ctx, trace)

	timeout := time.Duration(target.TimeoutMs) * time.Millisecond

	reqCtx, cancel := context.WithTimeout(reqCtx, timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(
		reqCtx,
		http.MethodGet,
		target.URL,
		nil,
	)
	if err != nil {
		result.Status = "down"
		result.ErrorType = classifyError(err)
		return result
	}

	resp, err := c.client.Do(req)

	result.LatencyMs = time.Since(start).Milliseconds()

	if !ttfb.IsZero() {
		result.TTFBMs = ttfb.Sub(start).Milliseconds()
	}

	if err != nil {
		result.Status = "down"
		result.ErrorType = classifyError(err)

		return result
	}

	defer resp.Body.Close()

	result.HTTPStatus = resp.StatusCode

	if resp.StatusCode != target.ExpectedStatus {
		result.Status = "down"
		result.ErrorType = "status_mismatch"

		return result
	}

	if target.DegradedLatencyMs != nil &&
		result.LatencyMs > int64(*target.DegradedLatencyMs) {
		result.Status = "degraded"
		return result
	}

	result.Status = "up"

	return result
}

func classifyError(err error) string {
	if err == nil {
		return ""
	}

	// Request context exceeded its deadline.
	if errors.Is(err, context.DeadlineExceeded) {
		return "timeout"
	}

	// Network-level timeout.
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return "timeout"
	}

	// DNS resolution failure.
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return "dns_error"
	}

	// Connection refused.
	if errors.Is(err, syscall.ECONNREFUSED) {
		return "connection_refused"
	}

	// TLS errors.
	if strings.Contains(
		strings.ToLower(err.Error()),
		"tls",
	) {
		return "tls_error"
	}

	// Unknown error.
	return ""
}

//                 Check()
//                    │
//                    ▼
//              HTTP Request
//                    │
//       ┌────────────┼────────────┐
//       │            │            │
//    timeout       DNS          TLS
//       │            │            │
//       ▼            ▼            ▼
//    DOWN         DOWN          DOWN
//   timeout     dns_error     tls_error

//              connection
//               refused
//                  │
//                  ▼
//                DOWN
//           connection_refused

//              wrong status
//                  │
//                  ▼
//                DOWN
//           status_mismatch

//           expected status
//                  │
//           ┌──────┴──────┐
//           │             │
//     slow response    normal
//           │             │
//           ▼             ▼
//       DEGRADED          UP
