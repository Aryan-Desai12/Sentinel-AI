package transport

import (
	"context"
	"errors"
	"net"
	"strings"
	"testing"
	"time"

	"sentinel-ai/agents/synthetic-monitor/internal/ssrf"
)

type fakeResolver struct {
	ips map[string][]net.IP
	err error
}

func (r *fakeResolver) LookupIP(
	ctx context.Context,
	network string,
	host string,
) ([]net.IP, error) {
	if r.err != nil {
		return nil, r.err
	}

	return r.ips[host], nil
}

func startTestListener(t *testing.T) (net.Listener, string) {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("start listener: %v", err)
	}

	return listener, listener.Addr().String()
}

func TestDialContext_RejectsBlockedIP(t *testing.T) {
	resolver := &fakeResolver{
		ips: map[string][]net.IP{
			"internal.example": {
				net.ParseIP("10.0.0.10"),
			},
		},
	}

	transport := New(resolver)

	ctx, cancel := context.WithTimeout(
		context.Background(),
		time.Second,
	)
	defer cancel()

	_, err := transport.HTTPTransport().DialContext(
		ctx,
		"tcp",
		"internal.example:80",
	)

	if err == nil {
		t.Fatal("expected blocked IP to be rejected")
	}

	if !strings.Contains(err.Error(), "dial-time SSRF validation failed") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDialContext_RejectsIfAnyResolvedIPIsBlocked(t *testing.T) {
	resolver := &fakeResolver{
		ips: map[string][]net.IP{
			"mixed.example": {
				net.ParseIP("93.184.216.34"),
				net.ParseIP("192.168.1.10"),
			},
		},
	}

	transport := New(resolver)

	ctx, cancel := context.WithTimeout(
		context.Background(),
		time.Second,
	)
	defer cancel()

	_, err := transport.HTTPTransport().DialContext(
		ctx,
		"tcp",
		"mixed.example:80",
	)

	if err == nil {
		t.Fatal("expected connection to be rejected")
	}
}

func TestDialContext_ConnectsToValidatedIP(t *testing.T) {
	listener, address := startTestListener(t)
	defer listener.Close()

	_, port, err := net.SplitHostPort(address)
	if err != nil {
		t.Fatalf("split listener address: %v", err)
	}

	resolver := &fakeResolver{
		ips: map[string][]net.IP{
			"public.example": {
				net.ParseIP("127.0.0.1"),
			},
		},
	}

	// This test cannot use 127.0.0.1 because Sentinel's SSRF policy
	// intentionally blocks loopback addresses.
	_ = port

	transport := New(resolver)

	ctx, cancel := context.WithTimeout(
		context.Background(),
		time.Second,
	)
	defer cancel()

	_, err = transport.HTTPTransport().DialContext(
		ctx,
		"tcp",
		"public.example:"+port,
	)

	if err == nil {
		t.Fatal("expected loopback IP to be rejected by SSRF policy")
	}

	var rejection ssrf.Rejection
	if !errors.As(err, &rejection) {
		t.Fatalf("expected SSRF rejection, got %v", err)
	}
}

func TestDialContext_ReturnsErrorForInvalidAddress(t *testing.T) {
	resolver := &fakeResolver{}

	transport := New(resolver)

	ctx, cancel := context.WithTimeout(
		context.Background(),
		time.Second,
	)
	defer cancel()

	_, err := transport.HTTPTransport().DialContext(
		ctx,
		"tcp",
		"invalid-address",
	)

	if err == nil {
		t.Fatal("expected invalid address to return an error")
	}
}

func TestDialContext_PropagatesDNSFailure(t *testing.T) {
	resolver := &fakeResolver{
		err: errors.New("DNS unavailable"),
	}

	transport := New(resolver)

	ctx, cancel := context.WithTimeout(
		context.Background(),
		time.Second,
	)
	defer cancel()

	_, err := transport.HTTPTransport().DialContext(
		ctx,
		"tcp",
		"example.com:443",
	)

	if err == nil {
		t.Fatal("expected DNS error")
	}

	if !strings.Contains(err.Error(), "DNS unavailable") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDialContext_DialsValidatedIP(t *testing.T) {
	expectedIP := net.ParseIP("93.184.216.34")

	resolver := &fakeResolver{
		ips: map[string][]net.IP{
			"example.com": {
				expectedIP,
			},
		},
	}

	var dialedAddress string

	dialFunc := func(
		ctx context.Context,
		network string,
		address string,
	) (net.Conn, error) {
		dialedAddress = address

		return nil, errors.New("stop test")
	}

	transport := newTransport(resolver, dialFunc)

	ctx := context.Background()

	_, err := transport.HTTPTransport().DialContext(
		ctx,
		"tcp",
		"example.com:443",
	)

	if err == nil {
		t.Fatal("expected test dial error")
	}

	expectedAddress := net.JoinHostPort(
		expectedIP.String(),
		"443",
	)

	if dialedAddress != expectedAddress {
		t.Fatalf(
			"expected dial address %q, got %q",
			expectedAddress,
			dialedAddress,
		)
	}
}
