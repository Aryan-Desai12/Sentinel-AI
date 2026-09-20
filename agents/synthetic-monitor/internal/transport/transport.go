package transport

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"sentinel-ai/agents/synthetic-monitor/internal/ssrf"
)

type DialFunc func(
	ctx context.Context,
	network string,
	address string,
) (net.Conn, error)

type Transport struct {
	httpTransport *http.Transport
}

func New(resolver ssrf.Resolver) *Transport {
	dialer := &net.Dialer{
		Timeout:   10 * time.Second,
		KeepAlive: 30 * time.Second,
	}

	return newTransport(resolver, dialer.DialContext)
}

func newTransport(
	resolver ssrf.Resolver,
	dialFunc DialFunc,
) *Transport {
	validator := ssrf.NewValidator(resolver)

	httpTransport := &http.Transport{
		DialContext: func(
			ctx context.Context,
			network string,
			address string,
		) (net.Conn, error) {
			host, port, err := net.SplitHostPort(address)
			if err != nil {
				return nil, fmt.Errorf(
					"split dial address %q: %w",
					address,
					err,
				)
			}

			ips, err := validator.ResolveAndValidate(ctx, host)
			if err != nil {
				return nil, fmt.Errorf(
					"dial-time SSRF validation failed for %q: %w",
					host,
					err,
				)
			}

			if len(ips) == 0 {
				return nil, fmt.Errorf(
					"hostname %q resolved to no IP addresses",
					host,
				)
			}

			ip := ips[0]

			targetAddress := net.JoinHostPort(
				ip.String(),
				port,
			)

			return dialFunc(ctx, network, targetAddress)
		},

		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   10,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		ForceAttemptHTTP2:     true,
	}

	return &Transport{
		httpTransport: httpTransport,
	}
}

func (t *Transport) HTTPTransport() *http.Transport {
	return t.httpTransport
}

func NewClient(resolver ssrf.Resolver) *http.Client {
	transport := New(resolver)

	return &http.Client{
		Transport: transport.HTTPTransport(),
	}
}

// Ensure the package uses strconv and strings only when needed by future
// address parsing changes.
var (
	_ = strconv.Itoa
	_ = strings.TrimSpace
)
