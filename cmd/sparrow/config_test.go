package main

import (
	"os"
	"path/filepath"
	"testing"
)

func writeConfigFile(t *testing.T, content string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("SPARROW_CONFIG", path)
}

func TestResolveConfigPrecedence(t *testing.T) {
	writeConfigFile(t, "server_url: http://file:1\napi_key: filekey\nnamespace: filens\n")
	t.Setenv("SPARROW_URL", "")
	t.Setenv("SPARROW_API_KEY", "")
	t.Setenv("SPARROW_NAMESPACE", "")

	// File only.
	cfg, err := resolveConfig("", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ServerURL != "http://file:1" || cfg.APIKey != "filekey" || cfg.Namespace != "filens" {
		t.Fatalf("file values not loaded: %+v", cfg)
	}

	// Flags override file.
	cfg, _ = resolveConfig("http://flag:2", "flagkey", "flagns")
	if cfg.ServerURL != "http://flag:2" || cfg.APIKey != "flagkey" || cfg.Namespace != "flagns" {
		t.Fatalf("flags should override file: %+v", cfg)
	}

	// Env overrides both flags and file.
	t.Setenv("SPARROW_URL", "http://env:3")
	t.Setenv("SPARROW_API_KEY", "envkey")
	t.Setenv("SPARROW_NAMESPACE", "envns")
	cfg, _ = resolveConfig("http://flag:2", "flagkey", "flagns")
	if cfg.ServerURL != "http://env:3" || cfg.APIKey != "envkey" || cfg.Namespace != "envns" {
		t.Fatalf("env should win: %+v", cfg)
	}
}

func TestResolveConfigDefaults(t *testing.T) {
	t.Setenv("SPARROW_CONFIG", filepath.Join(t.TempDir(), "missing.yaml"))
	t.Setenv("SPARROW_URL", "")
	t.Setenv("SPARROW_API_KEY", "")
	t.Setenv("SPARROW_NAMESPACE", "")
	cfg, err := resolveConfig("", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ServerURL != defaultServerURL || cfg.Namespace != defaultNamespace || cfg.APIKey != "" {
		t.Fatalf("defaults not applied: %+v", cfg)
	}
}

func TestSaveConfigPermissions(t *testing.T) {
	t.Setenv("SPARROW_CONFIG", filepath.Join(t.TempDir(), "sub", "config.yaml"))
	path, err := saveConfig(config{ServerURL: "http://x", Namespace: "n"})
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Fatalf("config file mode = %o, want 600", perm)
	}
}
