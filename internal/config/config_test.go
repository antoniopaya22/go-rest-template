package config

import (
	"os"
	"strings"
	"testing"
	"time"
)

func TestDefaults(t *testing.T) {
	t.Parallel()

	cfg, err := FromMap(map[string]string{"DATABASE_URL": "postgres://localhost/app"})
	if err != nil {
		t.Fatalf("defaults must be valid: %v", err)
	}

	if !cfg.OpenAPI.ValidateResponses {
		t.Error("responses must be validated by default in development")
	}
	if cfg.Log.Format != "text" {
		t.Errorf("development log format = %q, want text", cfg.Log.Format)
	}
	if cfg.Auth.Issuer != cfg.App.Name {
		t.Errorf("issuer = %q, want the application name", cfg.Auth.Issuer)
	}
	if cfg.HTTP.Address() != "0.0.0.0:8080" {
		t.Errorf("address = %q", cfg.HTTP.Address())
	}
}

// init:sqlite:begin

func TestDevelopmentDefaultsToSQLite(t *testing.T) {
	t.Parallel()

	cfg, err := FromMap(nil)
	if err != nil {
		t.Fatalf("defaults must be valid: %v", err)
	}
	if cfg.Database.Driver != DriverSQLite {
		t.Errorf("development driver = %q, want sqlite", cfg.Database.Driver)
	}
}

// init:sqlite:end

func TestProductionDefaults(t *testing.T) {
	t.Parallel()

	cfg, err := FromMap(map[string]string{
		"APP_ENV":      "production",
		"AUTH_SECRET":  strings.Repeat("s", 32),
		"DATABASE_URL": "postgres://localhost/app",
	})
	if err != nil {
		t.Fatalf("FromMap: %v", err)
	}

	// init:postgres:begin
	if cfg.Database.Driver != DriverPostgres {
		t.Errorf("production driver = %q, want postgres", cfg.Database.Driver)
	}
	// init:postgres:end
	if cfg.OpenAPI.ValidateResponses {
		t.Error("responses must not be validated by default in production")
	}
	if cfg.Log.Format != "json" {
		t.Errorf("production log format = %q, want json", cfg.Log.Format)
	}
}

func TestOverrides(t *testing.T) {
	t.Parallel()

	cfg, err := FromMap(map[string]string{
		"HTTP_PORT":                 " 9000 ",
		"HTTP_REQUEST_TIMEOUT":      "45s",
		"HTTP_CORS_ALLOWED_ORIGINS": "https://a.example, https://b.example",
		"LOG_LEVEL":                 "debug",
		"ADMIN_PORT":                "0",
		"DATABASE_URL":              "postgres://localhost/app",
	})
	if err != nil {
		t.Fatalf("FromMap: %v", err)
	}

	if cfg.HTTP.Port != 9000 || cfg.HTTP.RequestTimeout != 45*time.Second {
		t.Errorf("HTTP = %+v", cfg.HTTP)
	}
	if got := strings.Join(cfg.HTTP.CORSAllowedOrigins, "|"); got != "https://a.example|https://b.example" {
		t.Errorf("origins = %q", got)
	}
	if cfg.Admin.Enabled() {
		t.Error("ADMIN_PORT=0 must disable the admin server")
	}
}

func TestInvalidConfiguration(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		values map[string]string
		want   string
	}{
		{"not an integer", map[string]string{"HTTP_PORT": "eighty"}, "HTTP_PORT"},
		{"port out of range", map[string]string{"HTTP_PORT": "70000"}, "HTTP_PORT"},
		{"not a duration", map[string]string{"HTTP_READ_TIMEOUT": "soon"}, "HTTP_READ_TIMEOUT"},
		{"not a boolean", map[string]string{"DATABASE_AUTO_MIGRATE": "maybe"}, "DATABASE_AUTO_MIGRATE"},
		{"unknown enum value", map[string]string{"APP_ENV": "staging"}, "APP_ENV"},
		{"unknown log level", map[string]string{"LOG_LEVEL": "loud"}, "LOG_LEVEL"},
		{"same ports", map[string]string{"ADMIN_PORT": "8080"}, "ADMIN_PORT"},
		{"postgres without URL", map[string]string{"DATABASE_DRIVER": "postgres"}, "DATABASE_URL"}, // init:postgres
		{"weak bcrypt cost", map[string]string{"AUTH_BCRYPT_COST": "4"}, "AUTH_BCRYPT_COST"},
		{"refresh shorter than access", map[string]string{"AUTH_REFRESH_TOKEN_TTL": "1m"}, "AUTH_REFRESH_TOKEN_TTL"},
		{"EdDSA without key", map[string]string{"AUTH_ALGORITHM": "EdDSA"}, "AUTH_PRIVATE_KEY_FILE"},
		{"default secret in production", map[string]string{"APP_ENV": "production", "DATABASE_URL": "x"}, "AUTH_SECRET"},
		{"short secret in production", map[string]string{"APP_ENV": "production", "DATABASE_URL": "x", "AUTH_SECRET": "short"}, "AUTH_SECRET"},
		{"bootstrap without password", map[string]string{"BOOTSTRAP_ADMIN_ENABLED": "true", "BOOTSTRAP_ADMIN_USERNAME": "a", "BOOTSTRAP_ADMIN_FIRST_NAME": "a", "BOOTSTRAP_ADMIN_LAST_NAME": "a"}, "BOOTSTRAP_ADMIN_PASSWORD"},
		{"rate limit of zero", map[string]string{"RATE_LIMIT_REQUESTS": "0"}, "RATE_LIMIT_REQUESTS"}, // init:ratelimit
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := FromMap(tt.values)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want one mentioning %s", err, tt.want)
			}
		})
	}
}

func TestEveryProblemIsReported(t *testing.T) {
	t.Parallel()

	_, err := FromMap(map[string]string{"HTTP_PORT": "x", "LOG_LEVEL": "y", "AUTH_BCRYPT_COST": "1"})
	if err == nil {
		t.Fatal("expected an error")
	}
	for _, name := range []string{"HTTP_PORT", "LOG_LEVEL", "AUTH_BCRYPT_COST"} {
		if !strings.Contains(err.Error(), name) {
			t.Errorf("error does not mention %s: %v", name, err)
		}
	}
}

func TestReferenceListsEveryVariableOnce(t *testing.T) {
	t.Parallel()

	seen := map[string]bool{}
	for _, v := range Reference() {
		if seen[v.Name] {
			t.Errorf("%s is declared twice", v.Name)
		}
		seen[v.Name] = true
		if v.Description == "" || v.Group == "" {
			t.Errorf("%s has no description or group", v.Name)
		}
	}
	if !seen["HTTP_PORT"] || !seen["DATABASE_URL"] {
		t.Error("reference is missing well-known variables")
	}
}

// TestGeneratedFilesAreCurrent fails when a variable changed and
// `make generate` was not run.
func TestGeneratedFilesAreCurrent(t *testing.T) {
	t.Parallel()

	files := map[string]string{
		"../../configs/env.example":             EnvExample(),
		"../../docs/reference/configuration.md": Markdown(), // init:site
	}
	for path, want := range files {
		got, err := os.ReadFile(path)
		if err != nil {
			t.Errorf("read %s: %v", path, err)
			continue
		}
		if string(got) != want {
			t.Errorf("%s is out of date; run `make generate`", path)
		}
	}
}
