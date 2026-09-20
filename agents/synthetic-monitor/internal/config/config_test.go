package config

import (
	"context"
	"net"
	"os"
	"path/filepath"
	"testing"
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

func writeTempConfig(t *testing.T, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "config.yaml")

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write test config: %v", err)
	}

	return path
}

// baseConfig returns the common configuration
// required by the current Config.Validate().
func baseConfig(targets string) string {
	return `
check:
  workers: 5
  max_in_flight: 10

delivery:
  gateway_url: "http://localhost:8080"
  buffer_capacity: 100
  buffer_ttl_seconds: 300
  workers: 3

targets:
` + targets
}

func TestLoad_KeepsValidTargets(t *testing.T) {
	configPath := writeTempConfig(t, baseConfig(`
  - url: "https://example.com"
    interval_seconds: 30
    timeout_ms: 2000
    expected_status: 200
`))

	resolver := &fakeResolver{
		ips: map[string][]net.IP{
			"example.com": {
				net.ParseIP("93.184.216.34"),
			},
		},
	}

	cfg, err := load(configPath, resolver)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(cfg.Targets) != 1 {
		t.Fatalf("expected 1 valid target, got %d", len(cfg.Targets))
	}

	if cfg.Targets[0].URL != "https://example.com" {
		t.Fatalf("unexpected target URL: %s", cfg.Targets[0].URL)
	}
}

func TestLoad_RejectsBlockedTarget(t *testing.T) {
	configPath := writeTempConfig(t, baseConfig(`
  - url: "http://internal.example"
    interval_seconds: 30
    timeout_ms: 2000
    expected_status: 200
`))

	resolver := &fakeResolver{
		ips: map[string][]net.IP{
			"internal.example": {
				net.ParseIP("10.0.0.10"),
			},
		},
	}

	cfg, err := load(configPath, resolver)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(cfg.Targets) != 0 {
		t.Fatalf("expected 0 valid targets, got %d", len(cfg.Targets))
	}
}

func TestLoad_KeepsValidAndRejectsBlockedTargets(t *testing.T) {
	configPath := writeTempConfig(t, baseConfig(`
  - url: "https://public.example"
    interval_seconds: 30
    timeout_ms: 2000
    expected_status: 200

  - url: "http://internal.example"
    interval_seconds: 30
    timeout_ms: 2000
    expected_status: 200
`))

	resolver := &fakeResolver{
		ips: map[string][]net.IP{
			"public.example": {
				net.ParseIP("93.184.216.34"),
			},
			"internal.example": {
				net.ParseIP("192.168.1.10"),
			},
		},
	}

	cfg, err := load(configPath, resolver)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(cfg.Targets) != 1 {
		t.Fatalf("expected 1 valid target, got %d", len(cfg.Targets))
	}

	if cfg.Targets[0].URL != "https://public.example" {
		t.Fatalf("unexpected target retained: %s", cfg.Targets[0].URL)
	}
}

func TestLoad_AllTargetsRejected(t *testing.T) {
	configPath := writeTempConfig(t, baseConfig(`
  - url: "http://internal-one.example"
    interval_seconds: 30
    timeout_ms: 2000
    expected_status: 200

  - url: "http://internal-two.example"
    interval_seconds: 30
    timeout_ms: 2000
    expected_status: 200
`))

	resolver := &fakeResolver{
		ips: map[string][]net.IP{
			"internal-one.example": {
				net.ParseIP("10.0.0.10"),
			},
			"internal-two.example": {
				net.ParseIP("172.16.0.10"),
			},
		},
	}

	cfg, err := load(configPath, resolver)
	if err != nil {
		t.Fatalf("expected config loading to succeed, got %v", err)
	}

	if len(cfg.Targets) != 0 {
		t.Fatalf("expected 0 valid targets, got %d", len(cfg.Targets))
	}
}