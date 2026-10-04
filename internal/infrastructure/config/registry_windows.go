//go:build windows

package config

import (
	"errors"
	"fmt"
	"log/slog"
	"math"
	"slices"
	"strings"

	"golang.org/x/sys/windows/registry"
)

// registryKeyPath is where Windows deployments can provision configuration
// without a per-user config.json — e.g. via Group Policy Preferences or an
// install script. Value names mirror the Config JSON field names.
//
// Layout:
//
//	Software\go-sync-status-client
//	    ConfigJSON               (REG_SZ or REG_MULTI_SZ, optional) — the full
//	                             config.json content; if set, every other
//	                             value below is ignored
//	    BaseURL, BearerToken     (REG_SZ, optional)    — legacy single server
//	    RefreshIntervalSeconds   (REG_DWORD, optional)
//	    Servers\<subkey>         — one subkey per server, in subkey-name order
//	        Name                 (REG_SZ, optional; defaults to <subkey>)
//	        BaseURL, BearerToken (REG_SZ)
const registryKeyPath = `Software\go-sync-status-client`

// registryConfigJSONValue holds the full config.json content as a single
// registry value, for deployments that would rather ship one JSON blob than
// map each setting to its own value.
const registryConfigJSONValue = "ConfigJSON"

// registryServersSubkey holds one child key per configured server.
const registryServersSubkey = "Servers"

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
		cfg, err := readRegistryKey(key, root.name, logger)
		_ = key.Close()
		if err != nil {
			return Config{}, false, err
		}
		return cfg, true, nil
	}
	return Config{}, false, nil
}

// readRegistryKey parses Config from an open registryKeyPath key in hive.
// A non-blank registryConfigJSONValue wins outright (and must be valid
// JSON); otherwise the individual values are read via readRegistryValues.
func readRegistryKey(key registry.Key, hive string, logger *slog.Logger) (Config, error) {
	data, ok := readConfigJSON(key)
	if !ok {
		return readRegistryValues(key, logger), nil
	}
	logger.Info("config: using registry value", "hive", hive, "value", registryConfigJSONValue)
	return parseJSON([]byte(data), hive+`\`+registryKeyPath+`\`+registryConfigJSONValue)
}

// readConfigJSON returns the registryConfigJSONValue content from key. It
// accepts REG_SZ/REG_EXPAND_SZ, or REG_MULTI_SZ (lines joined with "\n") so
// pretty-printed JSON can be pasted into regedit's multi-string editor. ok
// is false when the value is missing or blank.
func readConfigJSON(key registry.Key) (data string, ok bool) {
	data, _, err := key.GetStringValue(registryConfigJSONValue)
	if errors.Is(err, registry.ErrUnexpectedType) {
		var lines []string
		lines, _, err = key.GetStringsValue(registryConfigJSONValue)
		data = strings.Join(lines, "\n")
	}
	if err != nil || strings.TrimSpace(data) == "" {
		return "", false
	}
	return data, true
}

// readRegistryValues parses Config values from an open registry key. The
// legacy top-level BaseURL/BearerToken pair, if set, becomes the first
// server, followed by every child key of registryServersSubkey.
func readRegistryValues(key registry.Key, logger *slog.Logger) Config {
	var cfg Config
	if sc := readServerValues(key); sc != (ServerConfig{}) {
		cfg.Servers = append(cfg.Servers, sc)
	}
	cfg.Servers = append(cfg.Servers, readRegistryServers(key, logger)...)
	if v, _, err := key.GetIntegerValue("RefreshIntervalSeconds"); err == nil && v <= math.MaxInt32 {
		cfg.RefreshIntervalSeconds = int(v)
	}
	return cfg
}

// readRegistryServers reads one ServerConfig per child key of
// registryServersSubkey under parent, sorted by subkey name so the order is
// deterministic. A missing Servers key yields nil; child keys that can't be
// opened are logged and skipped rather than failing the whole load.
func readRegistryServers(parent registry.Key, logger *slog.Logger) []ServerConfig {
	serversKey, err := registry.OpenKey(parent, registryServersSubkey, registry.ENUMERATE_SUB_KEYS)
	if err != nil {
		if !errors.Is(err, registry.ErrNotExist) {
			logger.Warn("config: open registry servers key", "key", registryServersSubkey, "error", err)
		}
		return nil
	}
	defer func() { _ = serversKey.Close() }()

	names, err := serversKey.ReadSubKeyNames(-1)
	if err != nil {
		logger.Warn("config: list registry servers", "key", registryServersSubkey, "error", err)
		return nil
	}
	slices.Sort(names)

	servers := make([]ServerConfig, 0, len(names))
	for _, name := range names {
		key, err := registry.OpenKey(serversKey, name, registry.QUERY_VALUE)
		if err != nil {
			logger.Warn("config: open registry server key", "server", name, "error", err)
			continue
		}
		sc := readServerValues(key)
		_ = key.Close()
		if sc.Name == "" {
			sc.Name = name
		}
		servers = append(servers, sc)
	}
	return servers
}

// readServerValues reads the per-server values (Name, BaseURL, BearerToken)
// from key, leaving any missing value zero-valued.
func readServerValues(key registry.Key) ServerConfig {
	var sc ServerConfig
	if v, _, err := key.GetStringValue("Name"); err == nil {
		sc.Name = v
	}
	if v, _, err := key.GetStringValue("BaseURL"); err == nil {
		sc.BaseURL = v
	}
	if v, _, err := key.GetStringValue("BearerToken"); err == nil {
		sc.BearerToken = v
	}
	return sc
}
