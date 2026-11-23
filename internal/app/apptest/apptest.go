// Package apptest starts the whole application for end-to-end tests.
//
// A test gets a real HTTP server backed by a throwaway database: a temporary
// SQLite file, and additionally a private PostgreSQL schema when
// TEST_POSTGRES_DSN is set. Feature packages use it from an external test
// package (package foo_test) to exercise their endpoints through the full
// middleware stack, contract validation included.
package apptest

import (
	"bytes"
	"context"
	"database/sql" // init:postgres
	"encoding/json"
	"fmt" // init:postgres
	"io"
	"log/slog"
	"maps"
	"net/http"
	"net/http/httptest"
	"net/url"       // init:postgres
	"os"            // init:postgres
	"path/filepath" // init:sqlite
	"strings"
	"testing"
	"time"

	// init:postgres:begin
	// Registers the pgx driver, used to create one schema per test.
	_ "github.com/jackc/pgx/v5/stdlib"
	// init:postgres:end

	"github.com/antoniopaya22/go-rest-template/v3/internal/app"
	"github.com/antoniopaya22/go-rest-template/v3/internal/auth"
	"github.com/antoniopaya22/go-rest-template/v3/internal/config"
)

// Credentials of the administrator every environment starts with.
const (
	AdminUsername = "admin"
	AdminPassword = "admin-password-123"
)

// Env is a running application.
type Env struct {
	t   *testing.T
	App *app.App
	URL string
}

// Each runs fn once per available database backend, as parallel subtests.
// Settings override the default test configuration.
func Each(t *testing.T, fn func(t *testing.T, env *Env), settings ...map[string]string) {
	t.Helper()

	backends := Backends()
	if len(backends) == 0 {
		t.Skip("no database backend available; set TEST_POSTGRES_DSN")
	}

	for _, backend := range backends {
		t.Run(backend, func(t *testing.T) {
			t.Parallel()
			fn(t, start(t, backend, settings...))
		})
	}
}

// Backends lists the database drivers the tests can run against.
func Backends() []string {
	backends := []string{}
	backends = append(backends, config.DriverSQLite) // init:sqlite
	// init:postgres:begin
	if os.Getenv("TEST_POSTGRES_DSN") != "" {
		backends = append(backends, config.DriverPostgres)
	}
	// init:postgres:end
	return backends
}

// New starts the application on the first available backend. The test is
// skipped when there is none, which happens in a PostgreSQL-only project when
// TEST_POSTGRES_DSN is not set.
func New(t *testing.T, settings ...map[string]string) *Env {
	t.Helper()

	backends := Backends()
	if len(backends) == 0 {
		t.Skip("no database backend available; set TEST_POSTGRES_DSN")
	}
	return start(t, backends[0], settings...)
}

func start(t *testing.T, backend string, settings ...map[string]string) *Env {
	t.Helper()

	values := map[string]string{
		"DATABASE_DRIVER":            backend,
		"ADMIN_PORT":                 "0",
		"AUTH_BCRYPT_COST":           "10",
		"OPENAPI_VALIDATE_RESPONSES": "true",
		"RATE_LIMIT_ENABLED":         "false", // init:ratelimit
		"BOOTSTRAP_ADMIN_ENABLED":    "true",
		"BOOTSTRAP_ADMIN_USERNAME":   AdminUsername,
		"BOOTSTRAP_ADMIN_FIRST_NAME": "Test",
		"BOOTSTRAP_ADMIN_LAST_NAME":  "Admin",
		"BOOTSTRAP_ADMIN_PASSWORD":   AdminPassword,
	}

	switch backend {
	// init:sqlite:begin
	case config.DriverSQLite:
		values["SQLITE_PATH"] = filepath.Join(t.TempDir(), "test.db")
	// init:sqlite:end
	// init:postgres:begin
	case config.DriverPostgres:
		values["DATABASE_URL"] = postgresSchema(t, os.Getenv("TEST_POSTGRES_DSN"))
	// init:postgres:end
	default:
		t.Fatalf("unknown backend %q", backend)
	}

	for _, override := range settings {
		maps.Copy(values, override)
	}

	cfg, err := config.FromMap(values)
	if err != nil {
		t.Fatalf("build configuration: %v", err)
	}

	application, err := app.New(context.Background(), cfg, slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatalf("create application: %v", err)
	}

	server := httptest.NewServer(application.Handler())
	t.Cleanup(func() {
		server.Close()
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := application.Close(ctx); err != nil {
			t.Errorf("close application: %v", err)
		}
	})

	return &Env{t: t, App: application, URL: server.URL}
}

// init:postgres:begin

// postgresSchema creates a schema private to the test and returns a DSN whose
// search_path points at it, so tests can run in parallel on one database.
func postgresSchema(t *testing.T, dsn string) string {
	t.Helper()

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("open postgres: %v", err)
	}

	schema := fmt.Sprintf("test_%d", time.Now().UnixNano())
	if _, err := db.ExecContext(context.Background(), "CREATE SCHEMA "+schema); err != nil {
		t.Fatalf("create schema: %v", err)
	}
	t.Cleanup(func() {
		_, _ = db.ExecContext(context.Background(), "DROP SCHEMA "+schema+" CASCADE")
		_ = db.Close()
	})

	parsed, err := url.Parse(dsn)
	if err != nil {
		t.Fatalf("TEST_POSTGRES_DSN must be a URL: %v", err)
	}
	query := parsed.Query()
	query.Set("search_path", schema)
	parsed.RawQuery = query.Encode()
	return parsed.String()
}

// init:postgres:end

// Response is a recorded HTTP response.
type Response struct {
	Status int
	Header http.Header
	Body   []byte
}

// Decode unmarshals the body into dst and fails the test on error.
func (r Response) Decode(t *testing.T, dst any) {
	t.Helper()
	if err := json.Unmarshal(r.Body, dst); err != nil {
		t.Fatalf("decode %q: %v", r.Body, err)
	}
}

// Code returns the "code" of a problem response.
func (r Response) Code(t *testing.T) string {
	t.Helper()
	var problem struct {
		Code string `json:"code"`
	}
	r.Decode(t, &problem)
	return problem.Code
}

// Request describes one call. Body, when not nil, is sent as JSON.
type Request struct {
	Method string
	Path   string
	Token  string
	Body   any
	Header map[string]string
}

// Do performs a request and fails the test when the status is not want.
func (e *Env) Do(want int, request Request) Response {
	e.t.Helper()

	var body io.Reader
	if raw, ok := request.Body.(string); ok {
		body = strings.NewReader(raw)
	} else if request.Body != nil {
		encoded, err := json.Marshal(request.Body)
		if err != nil {
			e.t.Fatalf("encode request: %v", err)
		}
		body = bytes.NewReader(encoded)
	}

	httpRequest, err := http.NewRequestWithContext(context.Background(), request.Method, e.URL+request.Path, body)
	if err != nil {
		e.t.Fatalf("build request: %v", err)
	}
	if body != nil {
		httpRequest.Header.Set("Content-Type", "application/json")
	}
	if request.Token != "" {
		httpRequest.Header.Set("Authorization", "Bearer "+request.Token)
	}
	for key, value := range request.Header {
		httpRequest.Header.Set(key, value)
	}

	httpResponse, err := http.DefaultClient.Do(httpRequest)
	if err != nil {
		e.t.Fatalf("%s %s: %v", request.Method, request.Path, err)
	}
	defer func() { _ = httpResponse.Body.Close() }()

	payload, err := io.ReadAll(httpResponse.Body)
	if err != nil {
		e.t.Fatalf("read response: %v", err)
	}

	if httpResponse.StatusCode != want {
		e.t.Fatalf("%s %s: status %d, want %d; body: %s", request.Method, request.Path, httpResponse.StatusCode, want, payload)
	}

	return Response{Status: httpResponse.StatusCode, Header: httpResponse.Header, Body: payload}
}

// Login returns the token pair of a user.
func (e *Env) Login(username string, password string) auth.TokenPair {
	e.t.Helper()

	var pair auth.TokenPair
	e.Do(http.StatusOK, Request{
		Method: http.MethodPost,
		Path:   "/api/v1/auth/login",
		Body:   map[string]string{"username": username, "password": password},
	}).Decode(e.t, &pair)
	return pair
}

// AdminToken returns an access token of the bootstrap administrator.
func (e *Env) AdminToken() string {
	e.t.Helper()
	return e.Login(AdminUsername, AdminPassword).AccessToken
}

// User registers a regular user and returns its id and an access token.
func (e *Env) User(username string) (int64, string) {
	e.t.Helper()

	const password = "user-password-123"
	var created struct {
		ID int64 `json:"id"`
	}
	e.Do(http.StatusCreated, Request{
		Method: http.MethodPost,
		Path:   "/api/v1/users",
		Body: map[string]string{
			"username": username, "first_name": "Test", "last_name": "User", "password": password,
		},
	}).Decode(e.t, &created)

	return created.ID, e.Login(username, password).AccessToken
}
