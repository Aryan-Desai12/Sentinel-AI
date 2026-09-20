package ssrf

import (
	"context"
	"fmt"
	"net"
	"net/url"
)

type Resolver interface {
	LookupIP(ctx context.Context, network, host string) ([]net.IP, error)
}

type Validator struct {
	resolver Resolver
}

func NewValidator(resolver Resolver) *Validator {
	return &Validator{resolver: resolver}
}

var blockedNetworks = mustParseNetworks([]string{
	// IPv4
	"127.0.0.0/8",    // loopback
	"169.254.0.0/16",  // link-local / cloud metadata
	"10.0.0.0/8",      // private
	"172.16.0.0/12",   // private
	"192.168.0.0/16",  // private

	// IPv6
	"::1/128",     // loopback
	"fe80::/10",   // link-local
	"fc00::/7",    // unique-local/private
})

type Rejection struct {
	ResolvedIP net.IP
	Rule       string
	Reason     string
}

func (r Rejection) Error() string {
	return fmt.Sprintf(
		"resolved IP %s matches blocked rule %s (%s)",
		r.ResolvedIP,
		r.Rule,
		r.Reason,
	)
}

func mustParseNetworks(networks []string) []*net.IPNet {
	result := make([]*net.IPNet, 0, len(networks))

	for _, network := range networks {
		_, ipNet, err := net.ParseCIDR(network)
		if err != nil {
			panic(fmt.Sprintf("invalid blocked network %q: %v", network, err))
		}

		result = append(result, ipNet)
	}

	return result
}

func blockedRule(ip net.IP) (string, bool) {
	if ipv4 := ip.To4(); ipv4 != nil {
		for _, network := range blockedNetworks {
			if network.Contains(ipv4) {
				return network.String(), true
			}
		}
	}

	for _, network := range blockedNetworks {
		if network.Contains(ip) {
			return network.String(), true
		}
	}

	return "", false
}

func isBlockedIP(ip net.IP) bool {
	_, blocked := blockedRule(ip)
	return blocked
}

// ResolveAndValidate resolves the hostname and validates every resolved IP.
// If any IP belongs to a blocked network, the entire resolution is rejected.
func (v *Validator) ResolveAndValidate(
	ctx context.Context,
	hostname string,
) ([]net.IP, error) {
	ips, err := v.resolver.LookupIP(ctx, "ip", hostname)
	if err != nil {
		return nil, fmt.Errorf(
			"DNS resolution failed for %q: %w",
			hostname,
			err,
		)
	}

	for _, ip := range ips {
		if rule, blocked := blockedRule(ip); blocked {
			return nil, Rejection{
				ResolvedIP: ip,
				Rule:       rule,
				Reason:     "private_or_internal_destination",
			}
		}
	}

	return ips, nil
}

func (v *Validator) ValidateURL(
	ctx context.Context,
	rawURL string,
) error {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf(
			"unsupported URL scheme %q",
			parsed.Scheme,
		)
	}

	hostname := parsed.Hostname()
	if hostname == "" {
		return fmt.Errorf("URL hostname is required")
	}

	_, err = v.ResolveAndValidate(ctx, hostname)
	if err != nil {
		return fmt.Errorf("validate URL %q: %w", rawURL, err)
	}

	return nil
}
