// Package config loads and validates runtime configuration.
package config

import (
	"fmt"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/antonioalfa22/go-rest-template/internal/pkg/errs"
)

// Config groups all runtime configuration for the application.
type Config struct {
	App           AppConfig
	HTTP          HTTPConfig
	Auth          AuthConfig
	Database      DatabaseConfig
	Bootstrap     BootstrapConfig
	OpenAPI       OpenAPIConfig
	Observability ObservabilityConfig
}

// AppConfig contains top-level application settings.
type AppConfig struct {
	Name            string
	Environment     string
	ShutdownTimeout time.Duration
}

// HTTPConfig contains HTTP server settings.
type HTTPConfig struct {
	Host           string
	Port           int
	ReadTimeout    time.Duration
	WriteTimeout   time.Duration
	IdleTimeout    time.Duration
	RequestTimeout time.Duration
}

// Address returns the listen address for the HTTP server.
func (c HTTPConfig) Address() string {
	return net.JoinHostPort(c.Host, strconv.Itoa(c.Port))
}

// AuthConfig contains authentication settings.
type AuthConfig struct {
	Secret     string
	Issuer     string
	TokenTTL   time.Duration
	BcryptCost int
}

// DatabaseDriver identifies the configured database backend.
type DatabaseDriver string

const (
	// DatabaseDriverSQLite selects SQLite for local development.
	DatabaseDriverSQLite DatabaseDriver = "sqlite"
	// DatabaseDriverPostgres selects PostgreSQL for production-oriented deployments.
	DatabaseDriverPostgres DatabaseDriver = "postgres"
)

// DatabaseConfig contains relational database settings.
type DatabaseConfig struct {
	Driver          DatabaseDriver
	MigrationsDir   string
	SQLitePath      string
	PostgresURL     string
	AutoMigrate     bool
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxIdleTime time.Duration
	ConnMaxLifetime time.Duration
}

// MigrationDir returns the migration directory for the active database driver.
func (c DatabaseConfig) MigrationDir() string {
	if strings.TrimSpace(c.MigrationsDir) != "" {
		return c.MigrationsDir
	}

	return filepath.Join("migrations", string(c.Driver))
}

// BootstrapConfig controls runtime bootstrapping tasks.
type BootstrapConfig struct {
	Admin AdminBootstrapConfig
}

// AdminBootstrapConfig contains the credentials for the first admin bootstrap flow.
type AdminBootstrapConfig struct {
	Enabled   bool
	Username  string
	FirstName string
	LastName  string
	Password  string
}

// OpenAPIConfig contains OpenAPI specification and validation settings.
type OpenAPIConfig struct {
	SpecPath           string
	ValidateRequests   bool
	ValidateResponses  bool
	ServeDocumentation bool
}

// ObservabilityConfig contains logging and diagnostics settings.
type ObservabilityConfig struct {
	LogLevel     slog.Level
	PprofEnabled bool
}

// Load reads configuration from the process environment.
func Load() (Config, error) {
	var cfg Config

	var err error

	cfg.App = AppConfig{
		Name:            getenv("APP_NAME", "go-rest-template"),
		Environment:     strings.ToLower(getenv("APP_ENV", "development")),
		ShutdownTimeout: mustDuration("APP_SHUTDOWN_TIMEOUT", "10s"),
	}

	cfg.HTTP = HTTPConfig{
		Host:           getenv("HTTP_HOST", "0.0.0.0"),
		Port:           mustInt("HTTP_PORT", 8080),
		ReadTimeout:    mustDuration("HTTP_READ_TIMEOUT", "5s"),
		WriteTimeout:   mustDuration("HTTP_WRITE_TIMEOUT", "10s"),
		IdleTimeout:    mustDuration("HTTP_IDLE_TIMEOUT", "60s"),
		RequestTimeout: mustDuration("HTTP_REQUEST_TIMEOUT", "30s"),
	}

	cfg.Auth = AuthConfig{
		Secret:     getenv("AUTH_SECRET", "change-me-in-production"),
		Issuer:     getenv("AUTH_ISSUER", cfg.App.Name),
		TokenTTL:   mustDuration("AUTH_TOKEN_TTL", "24h"),
		BcryptCost: mustInt("AUTH_BCRYPT_COST", 12),
	}

	databaseDriver := getenv("DATABASE_DRIVER", defaultDatabaseDriver(cfg.App.Environment))

	cfg.Database = DatabaseConfig{
		Driver:          DatabaseDriver(strings.ToLower(databaseDriver)),
		MigrationsDir:   getenv("DATABASE_MIGRATIONS_DIR", ""),
		SQLitePath:      getenv("SQLITE_PATH", "data/app.db"),
		PostgresURL:     getenv("DATABASE_URL", ""),
		AutoMigrate:     mustBool("AUTO_MIGRATE", true),
		MaxOpenConns:    mustInt("DATABASE_MAX_OPEN_CONNS", 20),
		MaxIdleConns:    mustInt("DATABASE_MAX_IDLE_CONNS", 5),
		ConnMaxIdleTime: mustDuration("DATABASE_CONN_MAX_IDLE_TIME", "5m"),
		ConnMaxLifetime: mustDuration("DATABASE_CONN_MAX_LIFETIME", "1h"),
	}

	cfg.Bootstrap = BootstrapConfig{
		Admin: AdminBootstrapConfig{
			Enabled:   mustBool("BOOTSTRAP_ADMIN_ENABLED", false),
			Username:  getenv("BOOTSTRAP_ADMIN_USERNAME", ""),
			FirstName: getenv("BOOTSTRAP_ADMIN_FIRST_NAME", ""),
			LastName:  getenv("BOOTSTRAP_ADMIN_LAST_NAME", ""),
			Password:  getenv("BOOTSTRAP_ADMIN_PASSWORD", ""),
		},
	}

	cfg.OpenAPI = OpenAPIConfig{
		SpecPath:           getenv("OPENAPI_SPEC_PATH", filepath.Join("api", "openapi.yaml")),
		ValidateRequests:   mustBool("OPENAPI_VALIDATE_REQUESTS", true),
		ValidateResponses:  mustBool("OPENAPI_VALIDATE_RESPONSES", true),
		ServeDocumentation: mustBool("OPENAPI_SERVE_DOCUMENTATION", true),
	}

	cfg.Observability = ObservabilityConfig{
		LogLevel:     mustLevel("LOG_LEVEL", slog.LevelInfo),
		PprofEnabled: mustBool("ENABLE_PPROF", false),
	}

	err = validate(cfg)
	if err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func validate(cfg Config) error {
	if cfg.HTTP.Port <= 0 || cfg.HTTP.Port > 65535 {
		return errs.New(errs.CodeInvalidArgument, "HTTP_PORT must be between 1 and 65535", nil)
	}

	if cfg.Auth.TokenTTL <= 0 {
		return errs.New(errs.CodeInvalidArgument, "AUTH_TOKEN_TTL must be greater than zero", nil)
	}

	if cfg.Auth.BcryptCost < 10 || cfg.Auth.BcryptCost > 16 {
		return errs.New(errs.CodeInvalidArgument, "AUTH_BCRYPT_COST must be between 10 and 16", nil)
	}

	switch cfg.Database.Driver {
	case DatabaseDriverSQLite:
		if strings.TrimSpace(cfg.Database.SQLitePath) == "" {
			return errs.New(errs.CodeInvalidArgument, "SQLITE_PATH is required when DATABASE_DRIVER=sqlite", nil)
		}
	case DatabaseDriverPostgres:
		if strings.TrimSpace(cfg.Database.PostgresURL) == "" {
			return errs.New(errs.CodeInvalidArgument, "DATABASE_URL is required when DATABASE_DRIVER=postgres", nil)
		}
	default:
		return errs.New(errs.CodeInvalidArgument, "DATABASE_DRIVER must be sqlite or postgres", nil)
	}

	if cfg.App.Environment == "production" && cfg.Auth.Secret == "change-me-in-production" {
		return errs.New(errs.CodeInvalidArgument, "AUTH_SECRET must be overridden in production", nil)
	}

	if strings.TrimSpace(cfg.OpenAPI.SpecPath) == "" {
		return errs.New(errs.CodeInvalidArgument, "OPENAPI_SPEC_PATH is required", nil)
	}

	if cfg.Bootstrap.Admin.Enabled {
		if strings.TrimSpace(cfg.Bootstrap.Admin.Username) == "" || strings.TrimSpace(cfg.Bootstrap.Admin.FirstName) == "" || strings.TrimSpace(cfg.Bootstrap.Admin.LastName) == "" || strings.TrimSpace(cfg.Bootstrap.Admin.Password) == "" {
			return errs.New(errs.CodeInvalidArgument, "bootstrap admin username, first name, last name and password are required when BOOTSTRAP_ADMIN_ENABLED=true", nil)
		}

		if len(strings.TrimSpace(cfg.Bootstrap.Admin.Password)) < 12 {
			return errs.New(errs.CodeInvalidArgument, "bootstrap admin password must be at least 12 characters", nil)
		}
	}

	return nil
}

func defaultDatabaseDriver(environment string) string {
	if environment == "production" {
		return string(DatabaseDriverPostgres)
	}

	return string(DatabaseDriverSQLite)
}

func getenv(key string, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	return value
}

func mustDuration(key string, fallback string) time.Duration {
	value := getenv(key, fallback)
	duration, err := time.ParseDuration(value)
	if err != nil {
		panic(fmt.Sprintf("parse %s: %v", key, err))
	}

	return duration
}

func mustInt(key string, fallback int) int {
	value := getenv(key, strconv.Itoa(fallback))
	parsed, err := strconv.Atoi(value)
	if err != nil {
		panic(fmt.Sprintf("parse %s: %v", key, err))
	}

	return parsed
}

func mustBool(key string, fallback bool) bool {
	value := getenv(key, strconv.FormatBool(fallback))
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		panic(fmt.Sprintf("parse %s: %v", key, err))
	}

	return parsed
}

func mustLevel(key string, fallback slog.Level) slog.Level {
	value := strings.ToUpper(getenv(key, fallback.String()))

	switch value {
	case "DEBUG":
		return slog.LevelDebug
	case "INFO":
		return slog.LevelInfo
	case "WARN":
		return slog.LevelWarn
	case "ERROR":
		return slog.LevelError
	default:
		panic(fmt.Sprintf("parse %s: unsupported log level %q", key, value))
	}
}
