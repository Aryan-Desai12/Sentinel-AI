package config

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os"

	"gopkg.in/yaml.v3"
	"sentinel-ai/agents/synthetic-monitor/internal/ssrf"
)

type Config struct {
	Check    CheckConfig    `yaml:"check"`
	Delivery DeliveryConfig `yaml:"delivery"`
	Targets  []Target       `yaml:"targets"`
}

type CheckConfig struct {
	Workers      int `yaml:"workers"`
	MaxInFlight  int `yaml:"max_in_flight"`
}

type DeliveryConfig struct {
	GatewayURL       string `yaml:"gateway_url"`
	BufferCapacity   int    `yaml:"buffer_capacity"`
	BufferTTLSeconds int    `yaml:"buffer_ttl_seconds"`
	Workers          int    `yaml:"workers"`
}

func (d DeliveryConfig) Validate() error {
	if d.GatewayURL == "" {
		return fmt.Errorf("gateway_url is required")
	}

	if d.BufferCapacity <= 0 {
		return fmt.Errorf("buffer_capacity must be greater than 0")
	}

	if d.BufferTTLSeconds <= 0 {
		return fmt.Errorf(
			"buffer_ttl_seconds must be greater than 0",
		)
	}

	if d.Workers <= 0 {
		return fmt.Errorf("workers must be greater than 0")
	}

	

	return nil
}

type Target struct {
	URL               string `yaml:"url"`
	IntervalSeconds   int    `yaml:"interval_seconds"`
	TimeoutMs         int    `yaml:"timeout_ms"`
	ExpectedStatus    int    `yaml:"expected_status"`
	DegradedLatencyMs *int   `yaml:"degraded_latency_ms,omitempty"`
	Service           string `yaml:"service,omitempty"`
}

func Load(path string) (Config, error) {
	return load(path, net.DefaultResolver)
}

func load(path string, resolver ssrf.Resolver) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read config: %w", err)
	}

	var cfg Config

	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return Config{}, fmt.Errorf("validate config: %w", err)
	}

	validator := ssrf.NewValidator(resolver)

	validTargets := make([]Target, 0, len(cfg.Targets))

	for _, target := range cfg.Targets {
		err := validator.ValidateURL(
			context.Background(),
			target.URL,
		)

		if err != nil {
			var rejection ssrf.Rejection

			if errors.As(err, &rejection) {
				slog.Error(
					"synthetic target rejected",
					"service", "synthetic-monitor",
					"event", "synthetic_target_rejected",
					"url", target.URL,
					"resolved_ip", rejection.ResolvedIP.String(),
					"blocked_rule", rejection.Rule,
					"reason", rejection.Reason,
				)
			} else {
				slog.Error(
					"synthetic target rejected",
					"service", "synthetic-monitor",
					"event", "synthetic_target_rejected",
					"url", target.URL,
					"reason", err.Error(),
				)
			}

			continue
		}

		validTargets = append(validTargets, target)
	}

	cfg.Targets = validTargets

	return cfg, nil
}

func (c Config) Validate() error {
	if err := c.Check.Validate(); err != nil {
		return fmt.Errorf("check: %w", err)
	}

	if err := c.Delivery.Validate(); err != nil {
		return fmt.Errorf("delivery: %w", err)
	}

	if len(c.Targets) == 0 {
		return fmt.Errorf("no targets configured")
	}

	for i, target := range c.Targets {
		if err := target.Validate(); err != nil {
			return fmt.Errorf(
				"target %d: %w",
				i,
				err,
			)
		}
	}

	return nil
}

func (c CheckConfig) Validate() error {
	if c.Workers <= 0 {
		return fmt.Errorf("workers must be greater than 0")
	}

	if c.MaxInFlight <= 0 {
		return fmt.Errorf("max_in_flight must be greater than 0")
	}
	return nil
}

func (t Target) Validate() error {
	if t.URL == "" {
		return fmt.Errorf("url is required")
	}

	if t.IntervalSeconds <= 0 {
		return fmt.Errorf("interval_seconds must be greater than 0")
	}

	if t.TimeoutMs <= 0 {
		return fmt.Errorf("timeout_ms must be greater than 0")
	}

	if t.ExpectedStatus < 100 || t.ExpectedStatus > 599 {
		return fmt.Errorf("expected_status must be between 100 and 599")
	}

	if t.DegradedLatencyMs != nil && *t.DegradedLatencyMs <= 0 {
		return fmt.Errorf("degraded_latency_ms must be greater than 0")
	}

	return nil
}
