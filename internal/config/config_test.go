package config

import (
	"log/slog"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"
)

func validConfig() *Config {
	return &Config{
		HTTPPort:                 "8080",
		DatabaseURL:              "postgres://localhost/riverqueue?sslmode=disable",
		EncryptionKeys:           []string{"new=" + strings.Repeat("ab", 32)},
		EncryptionPrimaryKeyID:   "new",
		MaxBodyBytes:             5242880,
		MaxCapturedResponseBytes: 1 << 20,
		QueueDBMaxConns:          50,
		DBMaxConns:               25,
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*Config)
		wantErr string
	}{
		{"valid", func(c *Config) {}, ""},
		{"production without api key", func(c *Config) { c.Environment = "production" }, "SPARROW_API_KEY"},
		{"production with short api key", func(c *Config) { c.Environment = "production"; c.APIKey = "k" }, "at least 32"},
		{"production with api key", func(c *Config) { c.Environment = "production"; c.APIKey = strings.Repeat("k", 32) }, ""},
		{"captured response cap too small", func(c *Config) { c.MaxCapturedResponseBytes = 10 }, "SPARROW_MAX_CAPTURED_RESPONSE_BYTES"},
		{"negative token ttl", func(c *Config) { c.TokenDefaultTTL = -time.Hour }, "SPARROW_TOKEN_DEFAULT_TTL"},
		{"invalid allowed networks", func(c *Config) { c.AllowedNetworks = []string{"nope"} }, "SPARROW_ALLOWED_NETWORKS"},
		{"invalid log level", func(c *Config) { c.LogLevel = "verbose" }, "SPARROW_LOG_LEVEL"},
		{"warn log level", func(c *Config) { c.LogLevel = "WARN" }, ""},
		{"valid allowed networks", func(c *Config) { c.AllowedNetworks = []string{"10.0.0.0/8", "fd00::1"} }, ""},
		{"zero queue pool", func(c *Config) { c.QueueDBMaxConns = 0 }, "SPARROW_QUEUE_DB_MAX_CONNS"},
		{"zero app pool", func(c *Config) { c.DBMaxConns = 0 }, "SPARROW_DB_MAX_CONNS"},
		{"missing encryption keyring", func(c *Config) { c.EncryptionKeys = nil }, "SPARROW_ENCRYPTION_KEYS"},
		{"legacy single encryption key no longer satisfies config", func(c *Config) {
			c.EncryptionKeys = nil
			c.EncryptionPrimaryKeyID = ""
		}, "SPARROW_ENCRYPTION_KEYS"},
		{"invalid keyring hex", func(c *Config) { c.EncryptionKeys = []string{"new=" + strings.Repeat("zz", 32)} }, "SPARROW_ENCRYPTION_KEYS"},
		{"short keyring key", func(c *Config) { c.EncryptionKeys = []string{"new=abcd"} }, "SPARROW_ENCRYPTION_KEYS"},
		{"valid encryption keyring", func(c *Config) {
			c.EncryptionKeys = []string{
				"old=" + strings.Repeat("ab", 32),
				"new=" + strings.Repeat("cd", 32),
			}
			c.EncryptionPrimaryKeyID = "new"
		}, ""},
		{"single-key primary missing", func(c *Config) {
			c.EncryptionKeys = []string{"main=" + strings.Repeat("ab", 32)}
			c.EncryptionPrimaryKeyID = ""
		}, "SPARROW_ENCRYPTION_PRIMARY_KEY_ID"},
		{"keyring primary missing", func(c *Config) {
			c.EncryptionKeys = []string{
				"old=" + strings.Repeat("ab", 32),
				"new=" + strings.Repeat("cd", 32),
			}
			c.EncryptionPrimaryKeyID = ""
		}, "SPARROW_ENCRYPTION_PRIMARY_KEY_ID"},
		{"keyring primary not found", func(c *Config) {
			c.EncryptionKeys = []string{
				"old=" + strings.Repeat("ab", 32),
				"new=" + strings.Repeat("cd", 32),
			}
			c.EncryptionPrimaryKeyID = "missing"
		}, "SPARROW_ENCRYPTION_PRIMARY_KEY_ID"},
		{"negative auto-disable window", func(c *Config) { c.AutoDisableAfter = -time.Hour }, "SPARROW_AUTO_DISABLE_AFTER"},
		{"auto-disable off", func(c *Config) { c.AutoDisableAfter = 0 }, ""},
		{"negative auto-disable failure minimum", func(c *Config) { c.AutoDisableMinFailures = -1 }, "SPARROW_AUTO_DISABLE_MIN_FAILURES"},
		{"body limit below minimum", func(c *Config) { c.MaxBodyBytes = 1024 }, "SPARROW_MAX_BODY_BYTES"},
		{"all-zeros key in production", func(c *Config) {
			c.Environment = "production"
			c.APIKey = strings.Repeat("k", 32)
			c.EncryptionKeys = []string{"old=" + strings.Repeat("0", 64), "new=" + strings.Repeat("ab", 32)}
		}, "all zeros"},
		{"all-zeros key outside production", func(c *Config) {
			c.EncryptionKeys = []string{"new=" + strings.Repeat("0", 64)}
		}, ""},
		{"negative shutdown timeout", func(c *Config) { c.ShutdownTimeout = -time.Second }, "SPARROW_SHUTDOWN_TIMEOUT"},
		{"custom shutdown timeout", func(c *Config) { c.ShutdownTimeout = 25 * time.Second }, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := validConfig()
			tt.mutate(cfg)
			err := cfg.Validate()
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("Validate() = %v, want nil", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("Validate() = %v, want error containing %q", err, tt.wantErr)
			}
		})
	}
}

func TestWarnings(t *testing.T) {
	cfg := validConfig()
	if w := cfg.Warnings(); len(w) != 0 {
		t.Fatalf("Warnings() for localhost = %v, want none", w)
	}

	cfg.DatabaseURL = "postgres://db.internal.example.com/riverqueue?sslmode=disable"
	w := cfg.Warnings()
	if len(w) != 1 || !strings.Contains(w[0], "sslmode=disable") {
		t.Fatalf("Warnings() for remote sslmode=disable = %v, want one advisory", w)
	}
}

func TestUIInjectKeyIsRetired(t *testing.T) {
	t.Setenv("SPARROW_UI_INJECT_KEY", "true")
	cfg := &Config{DatabaseURL: "postgres://localhost/db"}
	w := cfg.Warnings()
	if len(w) != 1 || !strings.Contains(w[0], "SPARROW_UI_INJECT_KEY") {
		t.Fatalf("Warnings() with SPARROW_UI_INJECT_KEY set = %v, want one retirement notice", w)
	}
}

func TestAIConfig(t *testing.T) {
	base := func() *Config {
		return &Config{HTTPPort: "8080", DatabaseURL: "postgres://localhost/x", EncryptionKeys: []string{"main=" + strings.Repeat("00", 32)}, EncryptionPrimaryKeyID: "main", MaxCapturedResponseBytes: 1024, MaxBodyBytes: 1 << 20, QueueDBMaxConns: 50, DBMaxConns: 25}
	}
	cases := []struct {
		name       string
		mut        func(c *Config)
		enabled    bool
		problem    string // substring of AIConfigError, "" for none
		model      string
		warnAIVars bool
	}{
		{"off by default", func(c *Config) {}, false, "", "claude-haiku-4-5", false},
		{"a key alone does not turn it on", func(c *Config) { c.AIAPIKey = "k" }, false, "", "claude-haiku-4-5", true},
		{"anthropic on with key", func(c *Config) { c.AIDrafting = true; c.AIAPIKey = "k" }, true, "", "claude-haiku-4-5", false},
		{"anthropic explicit model", func(c *Config) { c.AIDrafting = true; c.AIAPIKey = "k"; c.AIModel = "claude-sonnet-5-5" }, true, "", "claude-sonnet-5-5", false},
		{"anthropic on without key", func(c *Config) { c.AIDrafting = true }, true, "SPARROW_AI_API_KEY is not set", "claude-haiku-4-5", false},
		{"openai on without base url", func(c *Config) { c.AIDrafting = true; c.AIProvider = "openai"; c.AIAPIKey = "k"; c.AIModel = "m" }, true, "SPARROW_AI_BASE_URL is not set", "m", false},
		{"openai on without model", func(c *Config) {
			c.AIDrafting = true
			c.AIProvider = "openai"
			c.AIBaseURL = "http://localhost:11434/v1"
		}, true, "SPARROW_AI_MODEL is not set", "", false},
		{"openai local server, no key", func(c *Config) {
			c.AIDrafting = true
			c.AIProvider = "openai"
			c.AIBaseURL = "http://localhost:11434/v1"
			c.AIModel = "llama3.2"
		}, true, "", "llama3.2", false},
		{"unknown provider", func(c *Config) { c.AIDrafting = true; c.AIProvider = "gemini" }, true, "not supported", "", false},
		{"bad base url", func(c *Config) { c.AIDrafting = true; c.AIAPIKey = "k"; c.AIBaseURL = "localhost:11434" }, true, "not an absolute URL", "claude-haiku-4-5", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := base()
			tc.mut(c)
			if err := c.Validate(); err != nil {
				t.Fatalf("Validate() = %v; AI settings must never stop startup", err)
			}
			if c.AIEnabled() != tc.enabled {
				t.Fatalf("AIEnabled() = %v, want %v", c.AIEnabled(), tc.enabled)
			}
			err := c.AIConfigError()
			switch {
			case tc.problem == "" && err != nil:
				t.Fatalf("AIConfigError() = %v", err)
			case tc.problem != "" && (err == nil || !strings.Contains(err.Error(), tc.problem)):
				t.Fatalf("AIConfigError() = %v, want containing %q", err, tc.problem)
			}
			if c.AIModelOrDefault() != tc.model {
				t.Fatalf("AIModelOrDefault() = %q, want %q", c.AIModelOrDefault(), tc.model)
			}
			warned := strings.Contains(strings.Join(c.Warnings(), "\n"), "SPARROW_AI_ENABLED is not true")
			if warned != tc.warnAIVars {
				t.Fatalf("warning about AI vars without the switch = %v, want %v", warned, tc.warnAIVars)
			}
		})
	}
}

func TestLoadTreatsBlankAsUnset(t *testing.T) {
	for _, blank := range []string{"", "-", "  ", " - "} {
		t.Run(strconv.Quote(blank), func(t *testing.T) {
			t.Setenv("SPARROW_AI_API_KEY", blank)
			t.Setenv("SPARROW_AI_ENABLED", blank)
			t.Setenv("SPARROW_AI_TIMEOUT", blank)
			t.Setenv("SPARROW_HTTP_PORT", blank)
			t.Setenv("PORT", blank)
			t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", blank)
			c, err := Load()
			if err != nil {
				t.Fatalf("Load() = %v", err)
			}
			if c.AIEnabled() || c.AIAPIKey != "" {
				t.Fatalf("AI drafting on with SPARROW_AI_API_KEY=%q", blank)
			}
			if c.AITimeout != 3*time.Minute || c.HTTPPort != "8080" {
				t.Fatalf("defaults not applied: AITimeout=%s HTTPPort=%q", c.AITimeout, c.HTTPPort)
			}
			if _, set := os.LookupEnv("OTEL_EXPORTER_OTLP_ENDPOINT"); set {
				t.Fatalf("OTEL_EXPORTER_OTLP_ENDPOINT=%q still set; the OTel exporters read it directly", blank)
			}
		})
	}
}

func TestLoadHTTPPortFallsBackToPORT(t *testing.T) {
	tests := []struct {
		name, sparrowPort, port, want string
	}{
		{"neither set", "", "", "8080"},
		{"PORT only", "", "9000", "9000"},
		{"SPARROW_HTTP_PORT wins", "8089", "9000", "8089"},
		{"blank SPARROW_HTTP_PORT is unset", "-", "9000", "9000"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("SPARROW_HTTP_PORT", tc.sparrowPort)
			t.Setenv("PORT", tc.port)
			c, err := Load()
			if err != nil {
				t.Fatalf("Load() = %v", err)
			}
			if c.HTTPPort != tc.want {
				t.Fatalf("HTTPPort = %q, want %q", c.HTTPPort, tc.want)
			}
		})
	}
}

func TestSlogLevel(t *testing.T) {
	for in, want := range map[string]slog.Level{
		"": slog.LevelInfo, "info": slog.LevelInfo, "debug": slog.LevelDebug,
		"warn": slog.LevelWarn, "Warning": slog.LevelWarn, " ERROR ": slog.LevelError,
	} {
		got, err := (&Config{LogLevel: in}).SlogLevel()
		if err != nil || got != want {
			t.Errorf("SlogLevel(%q) = %v, %v; want %v", in, got, err, want)
		}
	}
	if _, err := (&Config{LogLevel: "trace"}).SlogLevel(); err == nil {
		t.Error("SlogLevel(\"trace\") should fail")
	}
}

func TestShutdownTimeoutOrDefault(t *testing.T) {
	if got := (&Config{}).ShutdownTimeoutOrDefault(); got != DefaultShutdownTimeout {
		t.Errorf("unset = %s, want %s", got, DefaultShutdownTimeout)
	}
	if got := (&Config{ShutdownTimeout: 25 * time.Second}).ShutdownTimeoutOrDefault(); got != 25*time.Second {
		t.Errorf("25s = %s", got)
	}
}

// Outside production an all-zeros key starts, but with a warning naming it.
func TestWarningsZeroEncryptionKey(t *testing.T) {
	cfg := validConfig()
	cfg.EncryptionKeys = []string{"new=" + strings.Repeat("0", 64)}
	var found bool
	for _, w := range cfg.Warnings() {
		found = found || (strings.Contains(w, "all zeros") && strings.Contains(w, "new"))
	}
	if !found {
		t.Fatalf("Warnings() = %v, want one about the all-zeros key", cfg.Warnings())
	}
	if w := validConfig().Warnings(); len(w) != 0 {
		t.Fatalf("a real key should not warn: %v", w)
	}
}

func TestProductionMatchesAnyCase(t *testing.T) {
	for v, want := range map[string]bool{
		"production": true, "Production": true, " PRODUCTION ": true,
		"prod": false, "": false, "development": false, "production-eu": false,
	} {
		if got := (&Config{Environment: v}).IsProduction(); got != want {
			t.Errorf("IsProduction(%q) = %v, want %v", v, got, want)
		}
	}
}

func TestWarnsOnProductionLookalikesAndRenamedVars(t *testing.T) {
	warned := func(c *Config, sub string) bool {
		return strings.Contains(strings.Join(c.Warnings(), "\n"), sub)
	}
	if !warned(&Config{Environment: "prod"}, `SPARROW_ENVIRONMENT="prod" is not "production"`) {
		t.Error("prod should warn that the production checks are off")
	}
	if warned(&Config{Environment: "development"}, "is not \"production\"") {
		t.Error("development should not warn")
	}

	t.Setenv("ENVIRONMENT", "production")
	t.Setenv("CORS_ALLOWED_ORIGINS", "https://ui.example.com")
	c := &Config{}
	if !warned(c, "ENVIRONMENT is set but no longer read; Sparrow reads SPARROW_ENVIRONMENT") ||
		!warned(c, "CORS_ALLOWED_ORIGINS is set but no longer read; Sparrow reads SPARROW_CORS_ALLOWED_ORIGINS") {
		t.Errorf("old names should warn, got %v", c.Warnings())
	}
	t.Setenv("SPARROW_ENVIRONMENT", "production")
	if warned(c, "ENVIRONMENT is set but no longer read") {
		t.Error("no warning once the new name is set too")
	}
}
