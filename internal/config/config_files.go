package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

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

func loadFirstAvailableConfig(cfg *Config) {
	for _, path := range configFilePaths() {
		if loadConfigFile(cfg, path) {
			return
		}
	}
}

func loadConfigFile(cfg *Config, path string) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to parse config %s: %v\n", path, err)
	}
	return true
}
