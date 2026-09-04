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
// install script writing HKCU directly. Value names mirror the Config JSON
// field names.
const registryKeyPath = `Software\go-sync-status-client`

// loadFromRegistry reads Config values from
// HKEY_CURRENT_MACHINE\Software\go-sync-status-client. ok is false (with a nil
// error) when the key itself doesn't exist, so Load can fall back to
// defaults; a missing individual value within an existing key is likewise
// left zero-valued rather than treated as an error. Any other failure to
// read the key is returned as an error.
func loadFromRegistry(logger *slog.Logger) (cfg Config, ok bool, err error) {
	logger.Info("config: loading from registry key", "key", registryKeyPath)
	key, err := registry.OpenKey(registry.LOCAL_MACHINE, registryKeyPath, registry.QUERY_VALUE)
	if err != nil {
		if errors.Is(err, registry.ErrNotExist) {
			return Config{}, false, nil
		}
		return Config{}, false, fmt.Errorf("config: open registry key %s: %w", registryKeyPath, err)
	}
	logger.Info("config: loaded from registry key", "key", registryKeyPath)
	defer func() { _ = key.Close() }()

	logger.Info("config: parsing registry values")
	if v, _, err := key.GetStringValue("BaseURL"); err == nil {
		cfg.BaseURL = v
	}
	if v, _, err := key.GetStringValue("BearerToken"); err == nil {
		cfg.BearerToken = v
	}
	if v, _, err := key.GetIntegerValue("RefreshIntervalSeconds"); err == nil && v <= math.MaxInt32 {
		cfg.RefreshIntervalSeconds = int(v)
	}
	return cfg, true, nil
}
