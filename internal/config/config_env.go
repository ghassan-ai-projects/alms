package config

import "os"

func applyEnvironmentOverrides(cfg *Config) {
	if dsn := os.Getenv("ALMS_PG_DSN"); dsn != "" {
		cfg.Database.DSN = dsn
	}
	if token := os.Getenv("ALMS_AUTH_TOKEN"); token != "" {
		cfg.Auth.Token = token
	}
}
