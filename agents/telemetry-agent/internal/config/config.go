package config

import (
	"fmt"
	"net/url"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	NodeID          string `yaml:"node_id"`
	APIKey          string `yaml:"api_key"`
	GatewayURL      string `yaml:"gateway_url"`
	IntervalSeconds int    `yaml:"interval_seconds"`
}

func Load(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read config file: %w", err)
	}

	var cfg Config

	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse config file: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func (c Config) Validate() error {
	if c.NodeID == "" {
		return fmt.Errorf("node_id is required")
	}

	if c.APIKey == "" {
		return fmt.Errorf("api_key is required")
	}

	if c.GatewayURL == "" {
		return fmt.Errorf("gateway_url is required")
	}

	u, err := url.Parse(c.GatewayURL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return fmt.Errorf("gateway_url must be a valid HTTP/HTTPS URL")
	}

	if c.IntervalSeconds <= 0 {
		return fmt.Errorf("interval_seconds must be greater than 0")
	}

	return nil
}