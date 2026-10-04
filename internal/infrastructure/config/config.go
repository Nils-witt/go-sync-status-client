// Package config loads this application's on-disk configuration: where to
// reach the go-backup-tool dashboard API and how to authenticate to it.
package config

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
)

// defaultBaseURL is used for the implicit server created when no servers
// are configured (missing config file, or a present-but-empty "servers"
// list).
const defaultBaseURL = "http://localhost:8081"

// defaultRefreshIntervalSeconds is used when the config file is missing, or
// present but leaves refresh_interval_seconds unset or non-positive.
const defaultRefreshIntervalSeconds = 60

// ServerConfig identifies one go-backup-tool dashboard instance to fetch
// sync status from.
type ServerConfig struct {
	// Name labels this server in the tray UI and in logs. If left empty, it
	// is derived from BaseURL's host.
	Name string `json:"name"`
	// BaseURL is the go-backup-tool instance's dashboard address, e.g.
	// "http://localhost:8081".
	BaseURL string `json:"base_url"`
	// BearerToken authenticates dashboard requests. Only required when the
	// target instance has webui.username or OIDC configured.
	BearerToken string `json:"bearer_token"`
}

// Config holds the settings needed to reach one or more go-backup-tool
// dashboard APIs.
type Config struct {
	// Servers lists every go-backup-tool instance to fetch sync status
	// from. If empty, a single implicit server pointing at defaultBaseURL
	// is used.
	Servers []ServerConfig `json:"servers"`
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
func Load(path string, logger *slog.Logger) (Config, error) {
	data, err := os.ReadFile(path) //nolint:gosec // path is a fixed, user-supplied config location, not attacker-controlled input
	if os.IsNotExist(err) {
		logger.Warn("config: config file does not exist", "path", path)
		cfg, ok, err := loadFromRegistry(logger)
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

	cfg, err := parseJSON(data, path)
	if err != nil {
		return Config{}, err
	}
	return applyDefaults(cfg), nil
}

// parseJSON decodes config.json-formatted data. source names where data came
// from (a file path or registry value) for the error message.
func parseJSON(data []byte, source string) (Config, error) {
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("config: parse %s: %w", source, err)
	}
	return cfg, nil
}

// applyDefaults fills in defaults for any field cfg left unset, regardless
// of whether cfg came from the JSON file or the Windows registry.
func applyDefaults(cfg Config) Config {
	if len(cfg.Servers) == 0 {
		cfg.Servers = []ServerConfig{{Name: "default", BaseURL: defaultBaseURL}}
	}
	cfg.Servers = applyServerDefaults(cfg.Servers)
	if cfg.RefreshIntervalSeconds <= 0 {
		cfg.RefreshIntervalSeconds = defaultRefreshIntervalSeconds
	}
	return cfg
}

// applyServerDefaults fills in a missing BaseURL/Name for each server and
// deduplicates names, so every server is always safe to use as a map key
// or display prefix without a hard config-validation error.
func applyServerDefaults(servers []ServerConfig) []ServerConfig {
	seen := make(map[string]int, len(servers))
	result := make([]ServerConfig, len(servers))
	for i, sc := range servers {
		if sc.BaseURL == "" {
			sc.BaseURL = defaultBaseURL
		}
		if sc.Name == "" {
			sc.Name = hostOf(sc.BaseURL)
			if sc.Name == "" {
				sc.Name = fmt.Sprintf("server-%d", i+1)
			}
		}
		if n := seen[sc.Name]; n > 0 {
			seen[sc.Name] = n + 1
			sc.Name = fmt.Sprintf("%s-%d", sc.Name, n+1)
		} else {
			seen[sc.Name] = 1
		}
		result[i] = sc
	}
	return result
}

// hostOf returns baseURL's host, or "" if it can't be parsed or has none.
func hostOf(baseURL string) string {
	u, err := url.Parse(baseURL)
	if err != nil {
		return ""
	}
	return u.Host
}
