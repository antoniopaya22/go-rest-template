// Package config loads and validates runtime configuration from the environment.
//
// Every variable is declared once, in load, with its default and description.
// The same declarations produce configs/env.example and the configuration
// reference in the documentation (see cmd/api -print-config).
package config

import (
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// Environment names.
const (
	EnvDevelopment = "development"
	EnvProduction  = "production"
)

// Database driver names.
const (
	DriverSQLite   = "sqlite"
	DriverPostgres = "postgres"
)

// Token signing algorithms.
const (
	AlgorithmHS256 = "HS256"
	AlgorithmEdDSA = "EdDSA"
)

const defaultAuthSecret = "change-me-in-production"

// Config groups all runtime configuration.
type Config struct {
	App       App
	HTTP      HTTP
	Admin     Admin
	Auth      Auth
	Database  Database
	Bootstrap Bootstrap
	OpenAPI   OpenAPI
	RateLimit RateLimit // init:ratelimit
	Log       Log
	Metrics   Metrics // init:metrics
	Tracing   Tracing // init:otel
}

// App contains top-level application settings.
type App struct {
	Name            string
	Environment     string
	ShutdownTimeout time.Duration
}

// IsProduction reports whether the application runs in production mode.
func (a App) IsProduction() bool { return a.Environment == EnvProduction }

// HTTP contains the public API server settings.
type HTTP struct {
	Host               string
	Port               int
	ReadTimeout        time.Duration
	WriteTimeout       time.Duration
	IdleTimeout        time.Duration
	RequestTimeout     time.Duration
	TrustedProxies     int
	MaxBodyBytes       int
	CORSAllowedOrigins []string
}

// Address returns the listen address of the API server.
func (c HTTP) Address() string { return net.JoinHostPort(c.Host, strconv.Itoa(c.Port)) }

// Admin contains the settings of the internal server that exposes metrics and
// profiling. It must not be reachable from the public network.
type Admin struct {
	Host         string
	Port         int
	PprofEnabled bool
}

// Enabled reports whether the admin server should be started.
func (c Admin) Enabled() bool { return c.Port > 0 }

// Address returns the listen address of the admin server.
func (c Admin) Address() string { return net.JoinHostPort(c.Host, strconv.Itoa(c.Port)) }

// Auth contains authentication settings.
type Auth struct {
	Algorithm           string
	Secret              string
	PrivateKeyFile      string
	Issuer              string
	AccessTokenTTL      time.Duration
	RefreshTokenTTL     time.Duration
	BcryptCost          int
	RegistrationEnabled bool
}

// Database contains relational database settings.
type Database struct {
	Driver          string
	URL             string
	SQLitePath      string
	AutoMigrate     bool
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxIdleTime time.Duration
	ConnMaxLifetime time.Duration
}

// Bootstrap contains the credentials of the first administrator, created at
// startup when no administrator exists.
type Bootstrap struct {
	AdminEnabled   bool
	AdminUsername  string
	AdminFirstName string
	AdminLastName  string
	AdminPassword  string
}

// OpenAPI contains contract validation settings.
type OpenAPI struct {
	ValidateRequests  bool
	ValidateResponses bool
	DocsEnabled       bool
}

// RateLimit contains request throttling settings.
type RateLimit struct {
	Enabled       bool
	Requests      int
	Window        time.Duration
	LoginRequests int
}

// Log contains logging settings.
type Log struct {
	Level  slog.Level
	Format string
}

// Metrics contains Prometheus settings.
type Metrics struct {
	Enabled bool
}

// Tracing contains OpenTelemetry settings. The exporter itself is configured
// with the standard OTEL_EXPORTER_OTLP_* variables.
type Tracing struct {
	Enabled     bool
	ServiceName string
}

// Load reads the configuration from the process environment. When a .env file
// exists in the working directory it is read first; variables already present
// in the environment take precedence over the file.
func Load() (Config, error) {
	fileValues, err := godotenv.Read()
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return Config{}, fmt.Errorf("read .env file: %w", err)
	}

	return FromLookup(func(name string) (string, bool) {
		if value, ok := os.LookupEnv(name); ok {
			return value, true
		}
		value, ok := fileValues[name]
		return value, ok
	})
}

// FromMap builds a configuration from explicit values, with defaults for the
// rest. It is meant for tests.
func FromMap(values map[string]string) (Config, error) {
	return FromLookup(func(name string) (string, bool) {
		value, ok := values[name]
		return value, ok
	})
}

// FromLookup builds a configuration from an arbitrary variable source.
// It returns every problem found, joined into a single error.
func FromLookup(lookup func(string) (string, bool)) (Config, error) {
	l := &loader{lookup: lookup}
	cfg := load(l)
	l.errs = append(l.errs, validate(cfg)...)

	if len(l.errs) > 0 {
		return Config{}, fmt.Errorf("invalid configuration:\n  - %w", joinLines(l.errs))
	}
	return cfg, nil
}

// Reference returns every variable the application reads, in declaration order.
func Reference() []Var {
	l := &loader{lookup: func(string) (string, bool) { return "", false }}
	load(l)
	return l.vars
}

func load(l *loader) Config {
	var cfg Config

	l.section("Application")
	cfg.App = App{
		Name:            l.str("APP_NAME", "go-rest-template", "Service name used in logs, token issuer and traces."),
		Environment:     l.enum("APP_ENV", EnvDevelopment, []string{EnvDevelopment, EnvProduction}, "Runtime environment. Production tightens defaults and refuses insecure settings."),
		ShutdownTimeout: l.duration("APP_SHUTDOWN_TIMEOUT", 10*time.Second, "Maximum time to wait for in-flight requests on shutdown."),
	}
	production := cfg.App.IsProduction()

	l.section("HTTP server")
	cfg.HTTP = HTTP{
		Host:               l.str("HTTP_HOST", "0.0.0.0", "Interface the API listens on."),
		Port:               l.integer("HTTP_PORT", 8080, "Port the API listens on."),
		ReadTimeout:        l.duration("HTTP_READ_TIMEOUT", 5*time.Second, "Maximum time to read a request, including the body."),
		WriteTimeout:       l.duration("HTTP_WRITE_TIMEOUT", 10*time.Second, "Maximum time to write a response."),
		IdleTimeout:        l.duration("HTTP_IDLE_TIMEOUT", 60*time.Second, "Maximum time to keep an idle keep-alive connection open."),
		RequestTimeout:     l.duration("HTTP_REQUEST_TIMEOUT", 30*time.Second, "Deadline applied to the context of every request."),
		TrustedProxies:     l.integer("HTTP_TRUSTED_PROXIES", 0, "Number of reverse proxies in front of the API whose X-Forwarded-For entries are trusted. 0 uses the TCP peer address."),
		MaxBodyBytes:       l.integer("HTTP_MAX_BODY_BYTES", 1<<20, "Maximum accepted request body size in bytes."),
		CORSAllowedOrigins: l.list("HTTP_CORS_ALLOWED_ORIGINS", "*", "Comma-separated origins allowed by CORS. Use explicit origins in production."),
	}

	l.section("Admin server")
	cfg.Admin = Admin{
		Host:         l.str("ADMIN_HOST", "0.0.0.0", "Interface the admin server (metrics, profiling) listens on. Keep it off the public network."),
		Port:         l.integer("ADMIN_PORT", 9090, "Port of the admin server. 0 disables it."),
		PprofEnabled: l.boolean("ADMIN_PPROF_ENABLED", false, "Expose net/http/pprof under /debug/pprof on the admin server."),
	}

	l.section("Authentication")
	cfg.Auth = Auth{
		Algorithm:           l.enum("AUTH_ALGORITHM", AlgorithmHS256, []string{AlgorithmHS256, AlgorithmEdDSA}, "Access token signing algorithm. EdDSA publishes the public key at /.well-known/jwks.json."),
		Secret:              l.str("AUTH_SECRET", defaultAuthSecret, "HMAC signing key for HS256. At least 32 characters in production."),
		PrivateKeyFile:      l.str("AUTH_PRIVATE_KEY_FILE", "", "Path to a PEM-encoded PKCS#8 Ed25519 private key. Required for EdDSA."),
		Issuer:              l.str("AUTH_ISSUER", cfg.App.Name, "Value of the iss claim. Defaults to APP_NAME."),
		AccessTokenTTL:      l.duration("AUTH_ACCESS_TOKEN_TTL", 15*time.Minute, "Lifetime of access tokens."),
		RefreshTokenTTL:     l.duration("AUTH_REFRESH_TOKEN_TTL", 720*time.Hour, "Lifetime of refresh tokens."),
		BcryptCost:          l.integer("AUTH_BCRYPT_COST", 12, "bcrypt work factor, between 10 and 16."),
		RegistrationEnabled: l.boolean("AUTH_REGISTRATION_ENABLED", true, "Allow anonymous callers to create regular user accounts."),
	}

	l.section("Database")
	defaultDriver := DriverSQLite
	if production {
		defaultDriver = DriverPostgres
	}
	cfg.Database = Database{
		Driver:          l.enum("DATABASE_DRIVER", defaultDriver, []string{DriverSQLite, DriverPostgres}, "Database backend. Defaults to sqlite in development and postgres in production."),
		URL:             l.str("DATABASE_URL", "", "PostgreSQL connection string. Required when DATABASE_DRIVER=postgres."),
		SQLitePath:      l.str("SQLITE_PATH", "data/app.db", "SQLite database file. Used when DATABASE_DRIVER=sqlite."),
		AutoMigrate:     l.boolean("DATABASE_AUTO_MIGRATE", true, "Apply pending migrations at startup."),
		MaxOpenConns:    l.integer("DATABASE_MAX_OPEN_CONNS", 20, "Maximum open connections (PostgreSQL)."),
		MaxIdleConns:    l.integer("DATABASE_MAX_IDLE_CONNS", 5, "Maximum idle connections (PostgreSQL)."),
		ConnMaxIdleTime: l.duration("DATABASE_CONN_MAX_IDLE_TIME", 5*time.Minute, "Maximum time a connection may stay idle (PostgreSQL)."),
		ConnMaxLifetime: l.duration("DATABASE_CONN_MAX_LIFETIME", time.Hour, "Maximum time a connection may be reused (PostgreSQL)."),
	}

	l.section("Bootstrap")
	cfg.Bootstrap = Bootstrap{
		AdminEnabled:   l.boolean("BOOTSTRAP_ADMIN_ENABLED", false, "Create an administrator at startup when none exists."),
		AdminUsername:  l.str("BOOTSTRAP_ADMIN_USERNAME", "", "Username of the bootstrap administrator."),
		AdminFirstName: l.str("BOOTSTRAP_ADMIN_FIRST_NAME", "", "First name of the bootstrap administrator."),
		AdminLastName:  l.str("BOOTSTRAP_ADMIN_LAST_NAME", "", "Last name of the bootstrap administrator."),
		AdminPassword:  l.str("BOOTSTRAP_ADMIN_PASSWORD", "", "Password of the bootstrap administrator, at least 12 characters."),
	}

	l.section("OpenAPI")
	cfg.OpenAPI = OpenAPI{
		ValidateRequests:  l.boolean("OPENAPI_VALIDATE_REQUESTS", true, "Reject requests that do not match the OpenAPI contract."),
		ValidateResponses: l.boolean("OPENAPI_VALIDATE_RESPONSES", !production, "Fail responses that do not match the contract. Buffers every response, so it defaults to true in development and false in production."),
		DocsEnabled:       l.boolean("OPENAPI_DOCS_ENABLED", true, "Serve the interactive API reference at /docs."),
	}

	// init:ratelimit:begin
	l.section("Rate limiting")
	cfg.RateLimit = RateLimit{
		Enabled:       l.boolean("RATE_LIMIT_ENABLED", true, "Throttle requests per client IP."),
		Requests:      l.integer("RATE_LIMIT_REQUESTS", 100, "Requests allowed per client IP in each window."),
		Window:        l.duration("RATE_LIMIT_WINDOW", time.Minute, "Length of the rate limit window."),
		LoginRequests: l.integer("RATE_LIMIT_LOGIN_REQUESTS", 10, "Login and token refresh attempts allowed per client IP in each window."),
	}
	// init:ratelimit:end

	l.section("Logging")
	cfg.Log = Log{
		Level:  l.level("LOG_LEVEL", slog.LevelInfo, "Minimum level of emitted log records."),
		Format: l.enum("LOG_FORMAT", "auto", []string{"auto", "text", "json"}, "Log encoding. auto selects text in development and json in production."),
	}
	if cfg.Log.Format == "auto" {
		cfg.Log.Format = "text"
		if production {
			cfg.Log.Format = "json"
		}
	}

	// init:metrics:begin
	l.section("Metrics")
	cfg.Metrics = Metrics{
		Enabled: l.boolean("METRICS_ENABLED", true, "Expose Prometheus metrics at /metrics on the admin server."),
	}
	// init:metrics:end

	// init:otel:begin
	l.section("Tracing")
	cfg.Tracing = Tracing{
		Enabled:     l.boolean("OTEL_ENABLED", false, "Export OpenTelemetry traces over OTLP/HTTP. The endpoint is set with OTEL_EXPORTER_OTLP_ENDPOINT."),
		ServiceName: l.str("OTEL_SERVICE_NAME", cfg.App.Name, "Service name reported in traces. Defaults to APP_NAME."),
	}
	// init:otel:end

	return cfg
}

func validate(cfg Config) []error {
	var problems []error
	fail := func(problem string) {
		problems = append(problems, errors.New(problem))
	}

	if cfg.HTTP.Port < 1 || cfg.HTTP.Port > 65535 {
		fail("HTTP_PORT: must be between 1 and 65535")
	}
	if cfg.Admin.Port < 0 || cfg.Admin.Port > 65535 {
		fail("ADMIN_PORT: must be between 0 and 65535")
	}
	if cfg.Admin.Enabled() && cfg.Admin.Port == cfg.HTTP.Port {
		fail("ADMIN_PORT: must differ from HTTP_PORT")
	}
	if cfg.HTTP.MaxBodyBytes < 1 {
		fail("HTTP_MAX_BODY_BYTES: must be greater than zero")
	}
	if cfg.HTTP.TrustedProxies < 0 {
		fail("HTTP_TRUSTED_PROXIES: must not be negative")
	}
	if len(cfg.HTTP.CORSAllowedOrigins) == 0 {
		fail("HTTP_CORS_ALLOWED_ORIGINS: at least one origin is required")
	}

	if cfg.Auth.AccessTokenTTL <= 0 {
		fail("AUTH_ACCESS_TOKEN_TTL: must be greater than zero")
	}
	if cfg.Auth.RefreshTokenTTL <= cfg.Auth.AccessTokenTTL {
		fail("AUTH_REFRESH_TOKEN_TTL: must be longer than AUTH_ACCESS_TOKEN_TTL")
	}
	if cfg.Auth.BcryptCost < 10 || cfg.Auth.BcryptCost > 16 {
		fail("AUTH_BCRYPT_COST: must be between 10 and 16")
	}
	switch cfg.Auth.Algorithm {
	case AlgorithmHS256:
		if cfg.App.IsProduction() && cfg.Auth.Secret == defaultAuthSecret {
			fail("AUTH_SECRET: the default value is not allowed in production")
		}
		if cfg.App.IsProduction() && len(cfg.Auth.Secret) < 32 {
			fail("AUTH_SECRET: must be at least 32 characters in production")
		}
	case AlgorithmEdDSA:
		if cfg.Auth.PrivateKeyFile == "" {
			fail("AUTH_PRIVATE_KEY_FILE: required when AUTH_ALGORITHM=EdDSA")
		}
	}

	switch cfg.Database.Driver {
	case DriverSQLite:
		if cfg.Database.SQLitePath == "" {
			fail("SQLITE_PATH: required when DATABASE_DRIVER=sqlite")
		}
	case DriverPostgres:
		if cfg.Database.URL == "" {
			fail("DATABASE_URL: required when DATABASE_DRIVER=postgres")
		}
	}

	if cfg.Bootstrap.AdminEnabled {
		if cfg.Bootstrap.AdminUsername == "" || cfg.Bootstrap.AdminFirstName == "" || cfg.Bootstrap.AdminLastName == "" {
			fail("BOOTSTRAP_ADMIN_*: username, first name and last name are required when BOOTSTRAP_ADMIN_ENABLED=true")
		}
		if len(cfg.Bootstrap.AdminPassword) < 12 {
			fail("BOOTSTRAP_ADMIN_PASSWORD: must be at least 12 characters")
		}
	}

	// init:ratelimit:begin
	if cfg.RateLimit.Enabled {
		if cfg.RateLimit.Requests < 1 || cfg.RateLimit.LoginRequests < 1 {
			fail("RATE_LIMIT_REQUESTS, RATE_LIMIT_LOGIN_REQUESTS: must be greater than zero")
		}
		if cfg.RateLimit.Window <= 0 {
			fail("RATE_LIMIT_WINDOW: must be greater than zero")
		}
	}
	// init:ratelimit:end

	return problems
}

func joinLines(problems []error) error {
	lines := make([]string, 0, len(problems))
	for _, problem := range problems {
		lines = append(lines, problem.Error())
	}
	return errors.New(strings.Join(lines, "\n  - "))
}
