// Package config loads this application's on-disk configuration: where to
// reach the go-backup-tool dashboard API and how to authenticate to it.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// defaultBaseURL is used when the config file is missing, or present but
// leaves base_url empty.
const defaultBaseURL = "http://localhost:8081"

// defaultRefreshIntervalSeconds is used when the config file is missing, or
// present but leaves refresh_interval_seconds unset or non-positive.
const defaultRefreshIntervalSeconds = 60

// Config holds the settings needed to reach the go-backup-tool dashboard
// API.
type Config struct {
	// BaseURL is the go-backup-tool instance's dashboard address, e.g.
	// "http://localhost:8081".
	BaseURL string `json:"base_url"`
	// BearerToken authenticates dashboard requests. Only required when the
	// target instance has webui.username or OIDC configured.
	BearerToken string `json:"bearer_token"`
	// RefreshIntervalSeconds is how often the tray automatically re-checks
	// sync status, in seconds.
	RefreshIntervalSeconds int `json:"refresh_interval_seconds"`
}

// DefaultPath returns the standard per-user config file location:
// $XDG_CONFIG_HOME/go-sync-status-client/config.json (~/.config/... on
// Linux and macOS, %AppData%\... on Windows).
func DefaultPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("config: resolve user config dir: %w", err)
	}
	return filepath.Join(dir, "go-sync-status-client", "config.json"), nil
}

// Load reads configuration from path. A missing file is not an error: on
// Windows, it falls back to values provisioned in the registry (see
// loadFromRegistry), which lets managed deployments push config without
// writing a per-user file; on other platforms, and when no registry values
// are present either, it yields a Config with just the defaults, so the
// tray can still run against an unauthenticated local instance with no
// config file at all.
func Load(path string) (Config, error) {
	data, err := os.ReadFile(path) //nolint:gosec // path is a fixed, user-supplied config location, not attacker-controlled input
	if os.IsNotExist(err) {
		cfg, ok, err := loadFromRegistry()
		if err != nil {
			return Config{}, err
		}
		if !ok {
			cfg = Config{}
		}
		return applyDefaults(cfg), nil
	}
	if err != nil {
		return Config{}, fmt.Errorf("config: read %s: %w", path, err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("config: parse %s: %w", path, err)
	}
	return applyDefaults(cfg), nil
}

// applyDefaults fills in defaultBaseURL/defaultRefreshIntervalSeconds for
// any field cfg left unset, regardless of whether cfg came from the JSON
// file or the Windows registry.
func applyDefaults(cfg Config) Config {
	if cfg.BaseURL == "" {
		cfg.BaseURL = defaultBaseURL
	}
	if cfg.RefreshIntervalSeconds <= 0 {
		cfg.RefreshIntervalSeconds = defaultRefreshIntervalSeconds
	}
	return cfg
}
