// Package config provides YAML + environment variable configuration loading
// for the ALMS server.
package config

import (
	"fmt"
)

// ServerConfig holds HTTP server settings.
type ServerConfig struct {
	Host string `yaml:"host" json:"host"`
	Port int    `yaml:"port" json:"port"`
}

// Addr returns the host:port address string.
func (s ServerConfig) Addr() string {
	return fmt.Sprintf("%s:%d", s.Host, s.Port)
}

// DatabaseConfig holds PostgreSQL connection settings.
type DatabaseConfig struct {
	DSN string `yaml:"dsn" json:"dsn"`
}

// AuthConfig holds authentication settings.
type AuthConfig struct {
	Token string `yaml:"token" json:"token"`
}

// Config is the top-level configuration for ALMS.
type Config struct {
	Server   ServerConfig   `yaml:"server" json:"server"`
	Database DatabaseConfig `yaml:"database" json:"database"`
	Auth     AuthConfig     `yaml:"auth" json:"auth"`
}

// DefaultConfig returns a Config populated with sensible defaults.
func DefaultConfig() Config {
	return Config{
		Server: ServerConfig{
			Host: "127.0.0.1",
			Port: 8001,
		},
		Database: DatabaseConfig{
			DSN: "postgres://alms:alms@localhost:5432/alms_db?sslmode=disable", //nolint:gosec
		},
		Auth: AuthConfig{
			Token: "",
		},
	}
}

// Load reads configuration from the first available YAML file and applies
// environment variable overrides. If no file exists, it returns defaults.
func Load(cfgPath string) Config {
	cfg := DefaultConfig()

	if cfgPath != "" {
		loadConfigFile(&cfg, cfgPath)
	} else {
		loadFirstAvailableConfig(&cfg)
	}

	applyEnvironmentOverrides(&cfg)
	return cfg
}
