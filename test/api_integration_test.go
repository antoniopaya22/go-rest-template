package test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/antoniopaya22/go-rest-template/internal/app"
	"github.com/antoniopaya22/go-rest-template/internal/app/api/auth"
	"github.com/antoniopaya22/go-rest-template/internal/app/api/domain/tasks"
	"github.com/antoniopaya22/go-rest-template/internal/app/api/domain/users"
	"github.com/antoniopaya22/go-rest-template/internal/pkg/config"
	_ "github.com/jackc/pgx/v5/stdlib" // Register the pgx driver for PostgreSQL integration tests.
)

type loginResponse struct {
	Token auth.Token `json:"token"`
	User  users.User `json:"user"`
}

func TestAPIIntegration(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		configuration config.Config
		before        func(*testing.T)
		after         func(*testing.T)
	}{
		{
			name:          "sqlite",
			configuration: sqliteConfig(t),
		},
	}

	if dsn := strings.TrimSpace(os.Getenv("TEST_POSTGRES_DSN")); dsn != "" {
		tests = append(tests, struct {
			name          string
			configuration config.Config
			before        func(*testing.T)
			after         func(*testing.T)
		}{
			name:          "postgres",
			configuration: postgresConfig(dsn),
			before: func(t *testing.T) {
				t.Helper()
				resetPostgresDatabase(t, dsn)
			},
			after: func(t *testing.T) {
				t.Helper()
				resetPostgresDatabase(t, dsn)
			},
		})
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if tt.before != nil {
				tt.before(t)
			}

			if tt.after != nil {
				t.Cleanup(func() {
					tt.after(t)
				})
			}

			runAPIFlow(t, tt.configuration)
		})
	}
}

func runAPIFlow(t *testing.T, cfg config.Config) {
	t.Helper()

	application, err := app.New(context.Background(), cfg)
	if err != nil {
		t.Fatalf("create application: %v", err)
	}
	t.Cleanup(func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if shutdownErr := application.Shutdown(shutdownCtx); shutdownErr != nil {
			t.Errorf("shutdown application: %v", shutdownErr)
		}
	})

	server := httptest.NewServer(application.Handler())
	t.Cleanup(server.Close)

	client := server.Client()

	body := doRequest(t, client, http.MethodGet, server.URL+"/openapi/openapi.yaml", nil, "", http.StatusOK)
	if !strings.Contains(string(body), "openapi: 3.1.0") {
		t.Fatalf("expected OpenAPI document, got %q", string(body))
	}

	_ = doRequest(t, client, http.MethodGet, server.URL+"/docs", nil, "", http.StatusOK)
	_ = doRequest(t, client, http.MethodGet, server.URL+"/health/live", nil, "", http.StatusOK)
	_ = doRequest(t, client, http.MethodGet, server.URL+"/health/ready", nil, "", http.StatusOK)

	var adminLogin loginResponse
	decodeResponse(t, doRequest(t, client, http.MethodPost, server.URL+"/api/v1/auth/login", map[string]any{
		"username": cfg.Bootstrap.Admin.Username,
		"password": cfg.Bootstrap.Admin.Password,
	}, "", http.StatusOK), &adminLogin)

	var createdUser users.User
	decodeResponse(t, doRequest(t, client, http.MethodPost, server.URL+"/api/v1/users", map[string]any{
		"username":   "integration-user",
		"first_name": "Integration",
		"last_name":  "Tester",
		"password":   "ultra-secure-user",
		"role":       "user",
	}, "", http.StatusCreated), &createdUser)

	var userLogin loginResponse
	decodeResponse(t, doRequest(t, client, http.MethodPost, server.URL+"/api/v1/auth/login", map[string]any{
		"username": "integration-user",
		"password": "ultra-secure-user",
	}, "", http.StatusOK), &userLogin)

	var ownTask tasks.Task
	decodeResponse(t, doRequest(t, client, http.MethodPost, server.URL+"/api/v1/tasks", map[string]any{
		"title":       "Own task",
		"description": "Created by the integration user",
		"status":      "in_progress",
	}, adminLogin.Token.AccessToken, http.StatusCreated), &ownTask)

	if ownTask.UserID != adminLogin.User.ID {
		t.Fatalf("expected admin-owned task, got owner %d", ownTask.UserID)
	}

	var delegatedTask tasks.Task
	decodeResponse(t, doRequest(t, client, http.MethodPost, server.URL+"/api/v1/tasks", map[string]any{
		"title":       "Delegated task",
		"description": "Created by admin for a regular user",
		"status":      "pending",
		"user_id":     createdUser.ID,
	}, adminLogin.Token.AccessToken, http.StatusCreated), &delegatedTask)

	if delegatedTask.UserID != createdUser.ID {
		t.Fatalf("expected delegated task owner %d, got %d", createdUser.ID, delegatedTask.UserID)
	}

	var userTasks []tasks.Task
	decodeResponse(t, doRequest(t, client, http.MethodGet, server.URL+"/api/v1/tasks", nil, userLogin.Token.AccessToken, http.StatusOK), &userTasks)
	if len(userTasks) != 1 {
		t.Fatalf("expected one visible user task, got %d", len(userTasks))
	}

	if userTasks[0].ID != delegatedTask.ID {
		t.Fatalf("expected delegated task %d, got %d", delegatedTask.ID, userTasks[0].ID)
	}

	var listedUsers []users.User
	decodeResponse(t, doRequest(t, client, http.MethodGet, server.URL+"/api/v1/users?role=user", nil, adminLogin.Token.AccessToken, http.StatusOK), &listedUsers)
	if len(listedUsers) != 1 {
		t.Fatalf("expected one regular user, got %d", len(listedUsers))
	}

	if listedUsers[0].ID != createdUser.ID {
		t.Fatalf("expected listed user %d, got %d", createdUser.ID, listedUsers[0].ID)
	}

	_ = doRequest(t, client, http.MethodPost, server.URL+"/api/v1/tasks", map[string]any{
		"title":       "Broken task",
		"description": "Should fail OpenAPI validation",
		"status":      "invalid-status",
	}, userLogin.Token.AccessToken, http.StatusBadRequest)
	_ = doRequest(t, client, http.MethodGet, server.URL+"/api/v1/users", nil, userLogin.Token.AccessToken, http.StatusForbidden)
}

func sqliteConfig(t *testing.T) config.Config {
	t.Helper()

	return baseConfig(config.DatabaseConfig{
		Driver:          config.DatabaseDriverSQLite,
		MigrationsDir:   filepath.Join("..", "migrations", "sqlite"),
		SQLitePath:      filepath.Join(t.TempDir(), "app.db"),
		AutoMigrate:     true,
		MaxOpenConns:    1,
		MaxIdleConns:    1,
		ConnMaxIdleTime: 5 * time.Minute,
		ConnMaxLifetime: time.Hour,
	})
}

func postgresConfig(dsn string) config.Config {
	return baseConfig(config.DatabaseConfig{
		Driver:          config.DatabaseDriverPostgres,
		MigrationsDir:   filepath.Join("..", "migrations", "postgres"),
		PostgresURL:     dsn,
		AutoMigrate:     true,
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		ConnMaxIdleTime: 5 * time.Minute,
		ConnMaxLifetime: time.Hour,
	})
}

func baseConfig(database config.DatabaseConfig) config.Config {
	return config.Config{
		App: config.AppConfig{
			Name:            "go-rest-template-test",
			Environment:     "development",
			ShutdownTimeout: 5 * time.Second,
		},
		HTTP: config.HTTPConfig{
			Host:           "127.0.0.1",
			Port:           8080,
			ReadTimeout:    5 * time.Second,
			WriteTimeout:   5 * time.Second,
			IdleTimeout:    30 * time.Second,
			RequestTimeout: 5 * time.Second,
		},
		Auth: config.AuthConfig{
			Secret:     "integration-secret-key",
			Issuer:     "go-rest-template-test",
			TokenTTL:   time.Hour,
			BcryptCost: 10,
		},
		Database: database,
		Bootstrap: config.BootstrapConfig{
			Admin: config.AdminBootstrapConfig{
				Enabled:   true,
				Username:  "bootstrap-admin",
				FirstName: "Bootstrap",
				LastName:  "Admin",
				Password:  "super-secure-admin",
			},
		},
		OpenAPI: config.OpenAPIConfig{
			SpecPath:           filepath.Join("..", "api", "openapi.yaml"),
			ValidateRequests:   true,
			ValidateResponses:  true,
			ServeDocumentation: true,
		},
		Observability: config.ObservabilityConfig{},
	}
}

func doRequest(t *testing.T, client *http.Client, method string, url string, payload any, token string, wantStatus int) []byte {
	t.Helper()

	var body io.Reader
	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("marshal payload: %v", err)
		}
		body = bytes.NewReader(encoded)
	}

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		t.Fatalf("create request %s %s: %v", method, url, err)
	}

	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	if strings.TrimSpace(token) != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("perform request %s %s: %v", method, url, err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read response body: %v", err)
	}

	if resp.StatusCode != wantStatus {
		t.Fatalf("unexpected status for %s %s: got %d want %d body=%s", method, url, resp.StatusCode, wantStatus, string(data))
	}

	return data
}

func decodeResponse(t *testing.T, data []byte, dst any) {
	t.Helper()

	if err := json.Unmarshal(data, dst); err != nil {
		t.Fatalf("decode response: %v", err)
	}
}

func resetPostgresDatabase(t *testing.T, dsn string) {
	t.Helper()

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("open postgres for cleanup: %v", err)
	}
	defer func() {
		_ = db.Close()
	}()

	if _, err := db.Exec(`DROP TABLE IF EXISTS tasks CASCADE`); err != nil {
		t.Fatalf("drop tasks table: %v", err)
	}

	if _, err := db.Exec(`DROP TABLE IF EXISTS users CASCADE`); err != nil {
		t.Fatalf("drop users table: %v", err)
	}

	if _, err := db.Exec(`DROP TABLE IF EXISTS schema_migrations CASCADE`); err != nil {
		t.Fatalf("drop schema_migrations table: %v", err)
	}
}
