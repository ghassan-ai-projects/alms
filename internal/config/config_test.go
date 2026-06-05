package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestServerAddr(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		cfg  ServerConfig
		want string
	}{
		{
			name: "default localhost",
			cfg:  ServerConfig{Host: "127.0.0.1", Port: 8001},
			want: "127.0.0.1:8001",
		},
		{
			name: "custom host and port",
			cfg:  ServerConfig{Host: "0.0.0.0", Port: 9000},
			want: "0.0.0.0:9000",
		},
		{
			name: "empty host",
			cfg:  ServerConfig{Port: 8080},
			want: ":8080",
		},
		{
			name: "port 0",
			cfg:  ServerConfig{Host: "localhost", Port: 0},
			want: "localhost:0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.cfg.Addr()
			if got != tt.want {
				t.Errorf("Addr() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Server.Host != "127.0.0.1" {
		t.Errorf("DefaultConfig() Server.Host = %q, want %q", cfg.Server.Host, "127.0.0.1")
	}
	if cfg.Server.Port != 8001 {
		t.Errorf("DefaultConfig() Server.Port = %d, want %d", cfg.Server.Port, 8001)
	}
	if cfg.Database.DSN != "postgres://alms:alms@localhost:5432/alms_db?sslmode=disable" {
		t.Errorf("DefaultConfig() Database.DSN = %q, want default DSN", cfg.Database.DSN)
	}
	if cfg.Auth.Token != "" {
		t.Errorf("DefaultConfig() Auth.Token = %q, want empty", cfg.Auth.Token)
	}
}

func TestLoadDefaults(t *testing.T) {
	// Load with empty path and no env overrides — should return defaults
	cfg := Load("")

	if cfg.Server.Host != "127.0.0.1" {
		t.Errorf("Load(\"\") Server.Host = %q, want %q", cfg.Server.Host, "127.0.0.1")
	}
	if cfg.Server.Port != 8001 {
		t.Errorf("Load(\"\") Server.Port = %d, want %d", cfg.Server.Port, 8001)
	}
}

func TestLoadFromFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "alms.yaml")

	content := `
server:
  host: 0.0.0.0
  port: 9999
database:
  dsn: "postgres://test:test@localhost:5432/test_db?sslmode=disable"
auth:
  token: "test-token"
`
	err := os.WriteFile(cfgPath, []byte(content), 0600)
	if err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	cfg := Load(cfgPath)

	if cfg.Server.Host != "0.0.0.0" {
		t.Errorf("Load() Server.Host = %q, want %q", cfg.Server.Host, "0.0.0.0")
	}
	if cfg.Server.Port != 9999 {
		t.Errorf("Load() Server.Port = %d, want %d", cfg.Server.Port, 9999)
	}
	if cfg.Database.DSN != "postgres://test:test@localhost:5432/test_db?sslmode=disable" {
		t.Errorf("Load() Database.DSN = %q, want test DSN", cfg.Database.DSN)
	}
	if cfg.Auth.Token != "test-token" {
		t.Errorf("Load() Auth.Token = %q, want %q", cfg.Auth.Token, "test-token")
	}
}

func TestLoadFromEnv(t *testing.T) {

	// Load from file to set defaults, then override with env vars
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "alms.yaml")

	content := `server:
  host: 127.0.0.1
  port: 8001
database:
  dsn: "postgres://original:pass@localhost:5432/orig_db?sslmode=disable"
auth:
  token: ""
`
	err := os.WriteFile(cfgPath, []byte(content), 0600)
	if err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	t.Setenv("ALMS_PG_DSN", "postgres://env:pass@localhost:5432/env_db?sslmode=disable")
	t.Setenv("ALMS_AUTH_TOKEN", "env-token")

	cfg := Load(cfgPath)

	if cfg.Database.DSN != "postgres://env:pass@localhost:5432/env_db?sslmode=disable" {
		t.Errorf("Load() Database.DSN = %q, want env DSN", cfg.Database.DSN)
	}
	if cfg.Auth.Token != "env-token" {
		t.Errorf("Load() Auth.Token = %q, want %q", cfg.Auth.Token, "env-token")
	}
}

func TestLoadBrokenConfigFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "alms.yaml")

	err := os.WriteFile(cfgPath, []byte("{invalid: yaml: [broken"), 0600)
	if err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	cfg := Load(cfgPath)
	// Should fall back to defaults
	if cfg.Server.Host != "127.0.0.1" {
		t.Errorf("Load(broken) Server.Host = %q, want default", cfg.Server.Host)
	}
}

func TestLoadNonExistentFile(t *testing.T) {
	t.Parallel()

	cfg := Load("/nonexistent/path/alms.yaml")
	// Should return defaults
	if cfg.Server.Host != "127.0.0.1" {
		t.Errorf("Load(nonexistent) Server.Host = %q, want default", cfg.Server.Host)
	}
}
