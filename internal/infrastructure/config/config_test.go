package config

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
)

var testLogger = slog.New(slog.NewTextHandler(io.Discard, nil))

func TestLoad_MissingFileUsesDefaults(t *testing.T) {
	cfg, err := Load(filepath.Join(t.TempDir(), "does-not-exist.json"), testLogger)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(cfg.Servers) != 1 || cfg.Servers[0].BaseURL != defaultBaseURL {
		t.Errorf("Servers = %+v, want single default server with BaseURL %q", cfg.Servers, defaultBaseURL)
	}
	if cfg.RefreshIntervalSeconds != defaultRefreshIntervalSeconds {
		t.Errorf("RefreshIntervalSeconds = %d, want %d", cfg.RefreshIntervalSeconds, defaultRefreshIntervalSeconds)
	}
}

func TestLoad_FilePresentOverridesDefaults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	writeFile(t, path, `{"servers":[{"name":"primary","base_url":"http://example.com","bearer_token":"tok"}],"refresh_interval_seconds":5}`)

	cfg, err := Load(path, testLogger)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(cfg.Servers) != 1 || cfg.Servers[0].Name != "primary" || cfg.Servers[0].BaseURL != "http://example.com" ||
		cfg.Servers[0].BearerToken != "tok" || cfg.RefreshIntervalSeconds != 5 {
		t.Errorf("Load() = %+v, want overrides applied", cfg)
	}
}

func TestLoad_FilePresentButEmptyFieldsFallBackToDefaults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	writeFile(t, path, `{"refresh_interval_seconds":0}`)

	cfg, err := Load(path, testLogger)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(cfg.Servers) != 1 || cfg.Servers[0].BaseURL != defaultBaseURL {
		t.Errorf("Servers = %+v, want single default server with BaseURL %q", cfg.Servers, defaultBaseURL)
	}
	if cfg.RefreshIntervalSeconds != defaultRefreshIntervalSeconds {
		t.Errorf("RefreshIntervalSeconds = %d, want %d", cfg.RefreshIntervalSeconds, defaultRefreshIntervalSeconds)
	}
}

func TestLoad_MultipleServers(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	writeFile(t, path, `{"servers":[
		{"name":"one","base_url":"http://one.example.com"},
		{"base_url":"http://two.example.com"},
		{"base_url":"not a url"}
	]}`)

	cfg, err := Load(path, testLogger)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(cfg.Servers) != 3 {
		t.Fatalf("Servers = %+v, want 3 entries", cfg.Servers)
	}
	if cfg.Servers[0].Name != "one" {
		t.Errorf("Servers[0].Name = %q, want %q", cfg.Servers[0].Name, "one")
	}
	if cfg.Servers[1].Name != "two.example.com" {
		t.Errorf("Servers[1].Name = %q, want derived host %q", cfg.Servers[1].Name, "two.example.com")
	}
	if cfg.Servers[2].Name == "" {
		t.Errorf("Servers[2].Name = %q, want a non-empty fallback name", cfg.Servers[2].Name)
	}
}

func TestLoad_DuplicateServerNamesAreDeduplicated(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	writeFile(t, path, `{"servers":[
		{"name":"dup","base_url":"http://one.example.com"},
		{"name":"dup","base_url":"http://two.example.com"}
	]}`)

	cfg, err := Load(path, testLogger)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(cfg.Servers) != 2 {
		t.Fatalf("Servers = %+v, want 2 entries", cfg.Servers)
	}
	if cfg.Servers[0].Name != "dup" || cfg.Servers[1].Name != "dup-2" {
		t.Errorf("Servers names = %q, %q, want %q, %q", cfg.Servers[0].Name, cfg.Servers[1].Name, "dup", "dup-2")
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
