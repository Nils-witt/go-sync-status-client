//go:build windows

package config

import (
	"path/filepath"
	"slices"
	"testing"

	"golang.org/x/sys/windows/registry"
)

// skipIfMachineKeyExists skips t when HKLM already holds registryKeyPath,
// since loadFromRegistry prefers it over the HKCU key these tests control.
func skipIfMachineKeyExists(t *testing.T) {
	t.Helper()
	key, err := registry.OpenKey(registry.LOCAL_MACHINE, registryKeyPath, registry.QUERY_VALUE)
	if err == nil {
		_ = key.Close()
		t.Skipf("HKLM\\%s exists and would shadow the HKCU test key", registryKeyPath)
	}
}

func TestLoad_MissingFileFallsBackToRegistry(t *testing.T) {
	skipIfMachineKeyExists(t)
	key, _, err := registry.CreateKey(registry.CURRENT_USER, registryKeyPath, registry.SET_VALUE|registry.QUERY_VALUE)
	if err != nil {
		t.Fatalf("create registry key: %v", err)
	}
	t.Cleanup(func() {
		_ = key.Close()
		_ = registry.DeleteKey(registry.CURRENT_USER, registryKeyPath)
	})

	if err := key.SetStringValue("BaseURL", "http://registry.example.com"); err != nil {
		t.Fatalf("SetStringValue BaseURL: %v", err)
	}
	if err := key.SetStringValue("BearerToken", "reg-tok"); err != nil {
		t.Fatalf("SetStringValue BearerToken: %v", err)
	}
	if err := key.SetDWordValue("RefreshIntervalSeconds", 42); err != nil {
		t.Fatalf("SetDWordValue RefreshIntervalSeconds: %v", err)
	}

	cfg, err := Load(filepath.Join(t.TempDir(), "does-not-exist.json"), testLogger)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(cfg.Servers) != 1 || cfg.Servers[0].BaseURL != "http://registry.example.com" || cfg.Servers[0].BearerToken != "reg-tok" || cfg.RefreshIntervalSeconds != 42 {
		t.Errorf("Load() = %+v, want values from registry", cfg)
	}
}

func TestLoad_MissingFileFallsBackToRegistryServers(t *testing.T) {
	skipIfMachineKeyExists(t)
	root, _, err := registry.CreateKey(registry.CURRENT_USER, registryKeyPath, registry.SET_VALUE|registry.QUERY_VALUE)
	if err != nil {
		t.Fatalf("create registry key: %v", err)
	}
	serverKeys := []string{"b-second", "a-first"}
	t.Cleanup(func() {
		_ = root.Close()
		for _, name := range serverKeys {
			_ = registry.DeleteKey(registry.CURRENT_USER, registryKeyPath+`\`+registryServersSubkey+`\`+name)
		}
		_ = registry.DeleteKey(registry.CURRENT_USER, registryKeyPath+`\`+registryServersSubkey)
		_ = registry.DeleteKey(registry.CURRENT_USER, registryKeyPath)
	})

	if err := root.SetStringValue("BaseURL", "http://legacy.example.com"); err != nil {
		t.Fatalf("SetStringValue BaseURL: %v", err)
	}
	values := map[string]map[string]string{
		"b-second": {"BaseURL": "http://second.example.com", "BearerToken": "tok-2"},
		"a-first":  {"BaseURL": "http://first.example.com", "Name": "Primary"},
	}
	for _, name := range serverKeys {
		key, _, err := registry.CreateKey(registry.CURRENT_USER, registryKeyPath+`\`+registryServersSubkey+`\`+name, registry.SET_VALUE)
		if err != nil {
			t.Fatalf("create server key %s: %v", name, err)
		}
		for k, v := range values[name] {
			if err := key.SetStringValue(k, v); err != nil {
				t.Fatalf("SetStringValue %s/%s: %v", name, k, err)
			}
		}
		_ = key.Close()
	}

	cfg, err := Load(filepath.Join(t.TempDir(), "does-not-exist.json"), testLogger)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	want := []ServerConfig{
		{Name: "legacy.example.com", BaseURL: "http://legacy.example.com"},
		{Name: "Primary", BaseURL: "http://first.example.com"},
		{Name: "b-second", BaseURL: "http://second.example.com", BearerToken: "tok-2"},
	}
	if !slices.Equal(cfg.Servers, want) {
		t.Errorf("Servers = %+v, want %+v", cfg.Servers, want)
	}
}

func TestLoad_MissingFileAndRegistryKeyUsesDefaults(t *testing.T) {
	skipIfMachineKeyExists(t)
	_ = registry.DeleteKey(registry.CURRENT_USER, registryKeyPath)

	cfg, err := Load(filepath.Join(t.TempDir(), "does-not-exist.json"), testLogger)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(cfg.Servers) != 1 || cfg.Servers[0].BaseURL != defaultBaseURL {
		t.Errorf("Servers = %+v, want default BaseURL %q", cfg.Servers, defaultBaseURL)
	}
	if cfg.RefreshIntervalSeconds != defaultRefreshIntervalSeconds {
		t.Errorf("RefreshIntervalSeconds = %d, want %d", cfg.RefreshIntervalSeconds, defaultRefreshIntervalSeconds)
	}
}
