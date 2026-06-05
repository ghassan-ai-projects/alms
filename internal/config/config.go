// Package config provides YAML + environment variable configuration loading
// for the ALMS server.
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
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

// configFilePaths returns the list of candidate config file paths in order
// of priority (highest first).
func configFilePaths() []string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "/root"
	}
	return []string{
		filepath.Join(home, ".alms", "alms.yaml"),
		"/etc/alms/alms.yaml",
		"/opt/alms/alms.yaml",
	}
}

// Load reads configuration from the first available YAML file and applies
// environment variable overrides. If no file exists, it returns defaults.
func Load(cfgPath string) Config {
	cfg := DefaultConfig()

	if cfgPath != "" {
		if data, err := os.ReadFile(cfgPath); err == nil {
			if err := yaml.Unmarshal(data, &cfg); err != nil {
				fmt.Fprintf(os.Stderr, "warning: failed to parse config %s: %v\n", cfgPath, err)
			}
		}
	} else {
		for _, path := range configFilePaths() {
			if data, err := os.ReadFile(path); err == nil {
				if err := yaml.Unmarshal(data, &cfg); err != nil {
					fmt.Fprintf(os.Stderr, "warning: failed to parse config %s: %v\n", path, err)
				}
				break
			}
		}
	}

	if dsn := os.Getenv("ALMS_PG_DSN"); dsn != "" {
		cfg.Database.DSN = dsn
	}
	if token := os.Getenv("ALMS_AUTH_TOKEN"); token != "" {
		cfg.Auth.Token = token
	}

	return cfg
}
