//go:build !windows

package config

// loadFromRegistry is a no-op on non-Windows platforms — there's no
// registry to read from, so Load always falls through to its built-in
// defaults when config.json is missing.
func loadFromRegistry() (cfg Config, ok bool, err error) {
	return Config{}, false, nil
}
