//go:build !windows

package config

import (
	"log/slog"
)

// loadFromRegistry is a no-op on non-Windows platforms — there's no
// registry to read from, so Load always falls through to its built-in
// defaults when config.json is missing.
func loadFromRegistry(logger *slog.Logger) (cfg Config, ok bool, err error) {
	logger.Warn("config: no registry on this platform")
	return Config{}, false, nil
}
