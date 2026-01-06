package main

import (
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
)

// Config represents the GateRelay service configuration
type Config struct {
	ListenAddr         string `toml:"listen_addr"`
	ListenPort         int    `toml:"listen_port"`
	TargetAddr         string `toml:"target_addr"`
	TargetPort         int    `toml:"target_port"`
	MaxConns           int    `toml:"max_conns"`
	IdleTimeoutSecs    int    `toml:"idle_timeout_secs"`
	ConnectTimeoutSecs int    `toml:"connect_timeout_secs"`
	ShutdownGraceSecs  int    `toml:"shutdown_grace_secs"`
	LogLevel           string `toml:"log_level"`
}

// DefaultConfig returns a config with sensible defaults
func DefaultConfig() *Config {
	return &Config{
		ListenAddr:         "0.0.0.0",
		ListenPort:         4000,
		TargetAddr:         "127.0.0.1",
		TargetPort:         5000,
		MaxConns:           200,
		IdleTimeoutSecs:    60,
		ConnectTimeoutSecs: 5,
		ShutdownGraceSecs:  10,
		LogLevel:           "info",
	}
}

// LoadConfig loads configuration from a TOML file
func LoadConfig(path string) (*Config, error) {
	cfg := DefaultConfig()

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	if err := toml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return cfg, nil
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.ListenPort < 1 || c.ListenPort > 65535 {
		return fmt.Errorf("listen_port must be between 1 and 65535, got %d", c.ListenPort)
	}
	if c.TargetPort < 1 || c.TargetPort > 65535 {
		return fmt.Errorf("target_port must be between 1 and 65535, got %d", c.TargetPort)
	}
	if c.TargetAddr == "" {
		return fmt.Errorf("target_addr cannot be empty")
	}
	if c.MaxConns < 1 {
		return fmt.Errorf("max_conns must be at least 1, got %d", c.MaxConns)
	}
	if c.IdleTimeoutSecs < 0 {
		return fmt.Errorf("idle_timeout_secs cannot be negative")
	}
	if c.ConnectTimeoutSecs < 0 {
		return fmt.Errorf("connect_timeout_secs cannot be negative")
	}
	if c.ShutdownGraceSecs < 0 {
		return fmt.Errorf("shutdown_grace_secs cannot be negative")
	}

	validLogLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
	}
	if !validLogLevels[c.LogLevel] {
		return fmt.Errorf("log_level must be one of: debug, info, warn, error; got %s", c.LogLevel)
	}

	return nil
}
