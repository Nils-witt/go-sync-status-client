//go:build windows

package config

import (
	"path/filepath"
	"testing"

	"golang.org/x/sys/windows/registry"
)

func TestLoad_MissingFileFallsBackToRegistry(t *testing.T) {
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

	cfg, err := Load(filepath.Join(t.TempDir(), "does-not-exist.json"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.BaseURL != "http://registry.example.com" || cfg.BearerToken != "reg-tok" || cfg.RefreshIntervalSeconds != 42 {
		t.Errorf("Load() = %+v, want values from registry", cfg)
	}
}

func TestLoad_MissingFileAndRegistryKeyUsesDefaults(t *testing.T) {
	_ = registry.DeleteKey(registry.CURRENT_USER, registryKeyPath)

	cfg, err := Load(filepath.Join(t.TempDir(), "does-not-exist.json"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.BaseURL != defaultBaseURL {
		t.Errorf("BaseURL = %q, want %q", cfg.BaseURL, defaultBaseURL)
	}
	if cfg.RefreshIntervalSeconds != defaultRefreshIntervalSeconds {
		t.Errorf("RefreshIntervalSeconds = %d, want %d", cfg.RefreshIntervalSeconds, defaultRefreshIntervalSeconds)
	}
}
