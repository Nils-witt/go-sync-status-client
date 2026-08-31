package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_MissingFileUsesDefaults(t *testing.T) {
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

func TestLoad_FilePresentOverridesDefaults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	writeFile(t, path, `{"base_url":"http://example.com","bearer_token":"tok","refresh_interval_seconds":5}`)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.BaseURL != "http://example.com" || cfg.BearerToken != "tok" || cfg.RefreshIntervalSeconds != 5 {
		t.Errorf("Load() = %+v, want overrides applied", cfg)
	}
}

func TestLoad_FilePresentButEmptyFieldsFallBackToDefaults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	writeFile(t, path, `{"refresh_interval_seconds":0}`)

	cfg, err := Load(path)
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

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
