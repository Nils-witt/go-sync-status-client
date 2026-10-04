//go:build windows

package config

import (
	"errors"
	"fmt"
	"log/slog"
	"math"

	"golang.org/x/sys/windows/registry"
)

// registryKeyPath is where Windows deployments can provision configuration
// without a per-user config.json — e.g. via Group Policy Preferences or an
// install script. Value names mirror the Config JSON field names.
const registryKeyPath = `Software\go-sync-status-client`

// registryRoots lists the hives searched for registryKeyPath, in order:
// machine-wide provisioning (HKLM) wins over per-user (HKCU).
var registryRoots = []struct {
	name string
	key  registry.Key
}{
	{name: "HKLM", key: registry.LOCAL_MACHINE},
	{name: "HKCU", key: registry.CURRENT_USER},
}

// loadFromRegistry reads Config values from registryKeyPath under the first
// hive in registryRoots where that key exists. ok is false (with a nil
// error) when the key exists in none of them, so Load can fall back to
// defaults; a missing individual value within an existing key is likewise
// left zero-valued rather than treated as an error. Any other failure to
// read the key is returned as an error.
func loadFromRegistry(logger *slog.Logger) (cfg Config, ok bool, err error) {
	for _, root := range registryRoots {
		logger.Info("config: loading from registry key", "hive", root.name, "key", registryKeyPath)
		key, err := registry.OpenKey(root.key, registryKeyPath, registry.QUERY_VALUE)
		if errors.Is(err, registry.ErrNotExist) {
			continue
		}
		if err != nil {
			return Config{}, false, fmt.Errorf("config: open registry key %s\\%s: %w", root.name, registryKeyPath, err)
		}
		logger.Info("config: loaded from registry key", "hive", root.name, "key", registryKeyPath)
		cfg := readRegistryValues(key)
		_ = key.Close()
		return cfg, true, nil
	}
	return Config{}, false, nil
}

// readRegistryValues parses Config values from an open registry key.
func readRegistryValues(key registry.Key) Config {
	var cfg Config
	var sc ServerConfig
	if v, _, err := key.GetStringValue("BaseURL"); err == nil {
		sc.BaseURL = v
	}
	if v, _, err := key.GetStringValue("BearerToken"); err == nil {
		sc.BearerToken = v
	}
	if sc != (ServerConfig{}) {
		cfg.Servers = []ServerConfig{sc}
	}
	if v, _, err := key.GetIntegerValue("RefreshIntervalSeconds"); err == nil && v <= math.MaxInt32 {
		cfg.RefreshIntervalSeconds = int(v)
	}
	return cfg
}
