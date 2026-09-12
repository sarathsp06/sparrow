package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const (
	defaultServerURL = "http://localhost:8080"
	defaultNamespace = "default"
)

// config is the resolved CLI configuration.
// Precedence: environment > flags > ~/.sparrow/config.yaml > defaults.
type config struct {
	ServerURL string `yaml:"server_url"`
	APIKey    string `yaml:"api_key,omitempty"`
	Namespace string `yaml:"namespace"`
}

// configPath returns ~/.sparrow/config.yaml, honoring SPARROW_CONFIG for tests.
func configPath() (string, error) {
	if p := os.Getenv("SPARROW_CONFIG"); p != "" {
		return p, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home dir: %w", err)
	}
	return filepath.Join(home, ".sparrow", "config.yaml"), nil
}

// configFlags registers the shared connection flags on fs and returns the
// destination pointers. Empty string means "not set via flag".
func configFlags(fs *flag.FlagSet) (url, apiKey, namespace *string) {
	url = fs.String("url", "", "Sparrow server URL (overrides config file)")
	apiKey = fs.String("api-key", "", "API key sent as X-API-Key (overrides config file)")
	namespace = fs.String("namespace", "", "tenant namespace (overrides config file)")
	return url, apiKey, namespace
}

// resolveConfig merges file < flags < env, then applies defaults.
func resolveConfig(flagURL, flagAPIKey, flagNamespace string) (config, error) {
	var cfg config
	path, err := configPath()
	if err != nil {
		return cfg, err
	}
	if data, err := os.ReadFile(path); err == nil {
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			return cfg, fmt.Errorf("parse %s: %w", path, err)
		}
	}
	overlay := func(dst *string, flagVal, envKey string) {
		if flagVal != "" {
			*dst = flagVal
		}
		if v := os.Getenv(envKey); v != "" {
			*dst = v
		}
	}
	overlay(&cfg.ServerURL, flagURL, "SPARROW_URL")
	overlay(&cfg.APIKey, flagAPIKey, "SPARROW_API_KEY")
	overlay(&cfg.Namespace, flagNamespace, "SPARROW_NAMESPACE")
	if cfg.ServerURL == "" {
		cfg.ServerURL = defaultServerURL
	}
	if cfg.Namespace == "" {
		cfg.Namespace = defaultNamespace
	}
	return cfg, nil
}

// saveConfig writes cfg to the config file with 0600 permissions.
func saveConfig(cfg config) (string, error) {
	path, err := configPath()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return "", err
	}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return "", err
	}
	return path, os.WriteFile(path, data, 0o600)
}
