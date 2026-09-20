package ssrf

import (
	"context"
	"fmt"
	"net"
	"testing"
)

// Fake resolver for testing purposes

type fakeResolver struct {
	ips []net.IP
	err error
}

func (f fakeResolver) LookupIP(
	ctx context.Context,
	network string,
	host string,
) ([]net.IP, error) {
	return f.ips, f.err
}

// Test that URLs resolving to private IPs are rejected.
func TestValidateURL_RejectsPrivateIP(t *testing.T) {
	resolver := fakeResolver{
		ips: []net.IP{
			net.ParseIP("10.0.0.5"),
		},
	}

	validator := NewValidator(resolver)

	err := validator.ValidateURL(
		context.Background(),
		"https://example.com",
	)

	if err == nil {
		t.Fatal("expected private IP to be rejected")
	}
}

// public ip address for testing purposes
func TestValidateURL_AllowsPublicIP(t *testing.T) {
	resolver := fakeResolver{
		ips: []net.IP{
			net.ParseIP("8.8.8.8"),
		},
	}

	validator := NewValidator(resolver)

	err := validator.ValidateURL(
		context.Background(),
		"https://example.com",
	)

	if err != nil {
		t.Fatalf("expected public IP to be allowed, got: %v", err)
	}
}

//Test all important blocked ranges

func TestValidateURL_RejectsBlockedIPs(t *testing.T) {
	tests := []struct {
		name string
		ip   string
	}{
		{
			name: "loopback",
			ip:   "127.0.0.1",
		},
		{
			name: "link local",
			ip:   "169.254.1.1",
		},
		{
			name: "private 10",
			ip:   "10.0.0.1",
		},
		{
			name: "private 172",
			ip:   "172.16.0.1",
		},
		{
			name: "private 192",
			ip:   "192.168.1.1",
		},
		{
			name: "cloud metadata",
			ip:   "169.254.169.254",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolver := fakeResolver{
				ips: []net.IP{
					net.ParseIP(tt.ip),
				},
			}

			validator := NewValidator(resolver)

			err := validator.ValidateURL(
				context.Background(),
				"https://example.com",
			)

			if err == nil {
				t.Fatalf(
					"expected IP %s to be rejected",
					tt.ip,
				)
			}
		})
	}
}

//Test the important multi-IP case

func TestValidateURL_RejectsIfAnyResolvedIPIsBlocked(t *testing.T) {
	resolver := fakeResolver{
		ips: []net.IP{
			net.ParseIP("8.8.8.8"),
			net.ParseIP("10.0.0.5"),
		},
	}

	validator := NewValidator(resolver)

	err := validator.ValidateURL(
		context.Background(),
		"https://example.com",
	)

	if err == nil {
		t.Fatal("expected URL to be rejected")
	}
}

// Testing if the dns fails then how the validator handles it. This is important to ensure that the system can gracefully handle DNS resolution failures without crashing or allowing potentially unsafe URLs through.
func TestValidateURL_DNSFailure(t *testing.T) {
	resolver := fakeResolver{
		err: fmt.Errorf("DNS unavailable"),
	}

	validator := NewValidator(resolver)

	err := validator.ValidateURL(
		context.Background(),
		"https://example.com",
	)

	if err == nil {
		t.Fatal("expected DNS error")
	}
}

func TestResolveAndValidate_ReturnsValidatedIPs(t *testing.T) {
	resolver := &fakeResolver{
		ips: []net.IP{
			net.ParseIP("93.184.216.34"),
			net.ParseIP("93.184.216.35"),
		},
	}

	validator := NewValidator(resolver)

	ips, err := validator.ResolveAndValidate(
		context.Background(),
		"example.com",
	)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(ips) != 2 {
		t.Fatalf("expected 2 IPs, got %d", len(ips))
	}

	if !ips[0].Equal(net.ParseIP("93.184.216.34")) {
		t.Fatalf("unexpected first IP: %s", ips[0])
	}

	if !ips[1].Equal(net.ParseIP("93.184.216.35")) {
		t.Fatalf("unexpected second IP: %s", ips[1])
	}
}
