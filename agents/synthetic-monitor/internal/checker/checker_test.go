package checker

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"syscall"
	"testing"
	"time"

	"sentinel-ai/agents/synthetic-monitor/internal/config"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(
	req *http.Request,
) (*http.Response, error) {
	return f(req)
}

func TestCheck_Up(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		w.WriteHeader(http.StatusOK)
	}))

	defer server.Close()

	client := server.Client()
	checker := New(client)

	target := config.Target{
		URL:            server.URL,
		TimeoutMs:      2000,
		ExpectedStatus: 200,
	}

	result := checker.Check(context.Background(), target)

	if result.Status != "up" {
		t.Fatalf("expected status up, got %s", result.Status)
	}

	if result.HTTPStatus != 200 {
		t.Fatalf("expected HTTP status 200, got %d", result.HTTPStatus)
	}

	if result.ErrorType != "" {
		t.Fatalf("expected no error type, got %s", result.ErrorType)
	}

	if result.LatencyMs < 0 {
		t.Fatalf("expected non-negative latency, got %d", result.LatencyMs)
	}

	if result.TTFBMs < 0 {
		t.Fatalf("expected non-negative TTFB, got %d", result.TTFBMs)
	}
}

func TestCheck_Degraded(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))

	defer server.Close()

	client := server.Client()
	checker := New(client)

	degradedThreshold := 50

	target := config.Target{
		URL:               server.URL,
		TimeoutMs:         2000,
		ExpectedStatus:    200,
		DegradedLatencyMs: &degradedThreshold,
	}

	result := checker.Check(context.Background(), target)

	if result.Status != "degraded" {
		t.Fatalf("expected degraded, got %s", result.Status)
	}

	if result.HTTPStatus != 200 {
		t.Fatalf("expected HTTP status 200, got %d", result.HTTPStatus)
	}

	if result.LatencyMs <= int64(degradedThreshold) {
		t.Fatalf(
			"expected latency > %dms, got %dms",
			degradedThreshold,
			result.LatencyMs,
		)
	}
}

func TestCheck_Down_StatusMismatch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		w.WriteHeader(http.StatusInternalServerError)
	}))

	defer server.Close()

	client := server.Client()
	checker := New(client)

	target := config.Target{
		URL:            server.URL,
		TimeoutMs:      2000,
		ExpectedStatus: 200,
	}

	result := checker.Check(context.Background(), target)

	if result.Status != "down" {
		t.Fatalf("expected down, got %s", result.Status)
	}

	if result.ErrorType != "status_mismatch" {
		t.Fatalf(
			"expected status_mismatch, got %s",
			result.ErrorType,
		)
	}

	if result.HTTPStatus != 500 {
		t.Fatalf(
			"expected HTTP status 500, got %d",
			result.HTTPStatus,
		)
	}
}

func TestCheck_Down_Timeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		time.Sleep(500 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))

	defer server.Close()

	client := server.Client()
	checker := New(client)

	target := config.Target{
		URL:            server.URL,
		TimeoutMs:      50,
		ExpectedStatus: 200,
	}

	result := checker.Check(context.Background(), target)

	if result.Status != "down" {
		t.Fatalf("expected down, got %s", result.Status)
	}

	if result.ErrorType != "timeout" {
		t.Fatalf(
			"expected timeout, got %s",
			result.ErrorType,
		)
	}
}

func TestCheck_TTFBIsMeasured(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		time.Sleep(50 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))

	defer server.Close()

	client := server.Client()
	checker := New(client)

	target := config.Target{
		URL:            server.URL,
		TimeoutMs:      2000,
		ExpectedStatus: 200,
	}

	result := checker.Check(context.Background(), target)

	if result.TTFBMs <= 0 {
		t.Fatalf(
			"expected TTFB to be measured, got %dms",
			result.TTFBMs,
		)
	}

	if result.LatencyMs < result.TTFBMs {
		t.Fatalf(
			"expected total latency >= TTFB, latency=%dms ttfb=%dms",
			result.LatencyMs,
			result.TTFBMs,
		)
	}
}

func TestCheck_Down_DNSError(t *testing.T) {
	client := &http.Client{
		Transport: roundTripFunc(func(
			req *http.Request,
		) (*http.Response, error) {
			return nil, &net.DNSError{
				Err:  "no such host",
				Name: "does-not-exist.example",
			}
		}),
	}

	checker := New(client)

	target := config.Target{
		URL:            "https://does-not-exist.example",
		TimeoutMs:      2000,
		ExpectedStatus: 200,
	}

	result := checker.Check(context.Background(), target)

	if result.Status != "down" {
		t.Fatalf("expected down, got %s", result.Status)
	}

	if result.ErrorType != "dns_error" {
		t.Fatalf(
			"expected dns_error, got %s",
			result.ErrorType,
		)
	}
}

func TestCheck_Down_ConnectionRefused(t *testing.T) {
	client := &http.Client{
		Transport: roundTripFunc(func(
			req *http.Request,
		) (*http.Response, error) {
			return nil, &net.OpError{
				Op:  "dial",
				Net: "tcp",
				Err: syscall.ECONNREFUSED,
			}
		}),
	}

	checker := New(client)

	target := config.Target{
		URL:            "https://example.com",
		TimeoutMs:      2000,
		ExpectedStatus: 200,
	}

	result := checker.Check(context.Background(), target)

	if result.Status != "down" {
		t.Fatalf("expected down, got %s", result.Status)
	}

	if result.ErrorType != "connection_refused" {
		t.Fatalf(
			"expected connection_refused, got %s",
			result.ErrorType,
		)
	}
}

func TestCheck_Down_TLSError(t *testing.T) {
	client := &http.Client{
		Transport: roundTripFunc(func(
			req *http.Request,
		) (*http.Response, error) {
			return nil, errors.New(
				"tls: failed to verify certificate",
			)
		}),
	}

	checker := New(client)

	target := config.Target{
		URL:            "https://example.com",
		TimeoutMs:      2000,
		ExpectedStatus: 200,
	}

	result := checker.Check(context.Background(), target)

	if result.Status != "down" {
		t.Fatalf("expected down, got %s", result.Status)
	}

	if result.ErrorType != "tls_error" {
		t.Fatalf(
			"expected tls_error, got %s",
			result.ErrorType,
		)
	}
}
