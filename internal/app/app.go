// Package app wires configuration, infrastructure and transports.
package app

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/antoniopaya22/go-rest-template/internal/app/api/apicontract"
	"github.com/antoniopaya22/go-rest-template/internal/app/api/auth"
	"github.com/antoniopaya22/go-rest-template/internal/app/api/domain/tasks"
	"github.com/antoniopaya22/go-rest-template/internal/app/api/domain/users"
	"github.com/antoniopaya22/go-rest-template/internal/app/api/httpapi"
	"github.com/antoniopaya22/go-rest-template/internal/app/api/store/postgres"
	"github.com/antoniopaya22/go-rest-template/internal/app/api/store/sqlite"
	"github.com/antoniopaya22/go-rest-template/internal/pkg/config"
)

// App represents the running API application.
type App struct {
	config config.Config
	logger *slog.Logger
	db     *sql.DB
	server *http.Server
}

// New creates a fully wired application instance.
func New(ctx context.Context, cfg config.Config) (*App, error) {
	logger := newLogger(cfg)

	db, userRepository, taskRepository, err := openStores(ctx, cfg)
	if err != nil {
		return nil, err
	}

	if cfg.Database.AutoMigrate {
		if err := applyMigrations(ctx, cfg, db); err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("apply migrations: %w", err)
		}
	}

	passwords, err := auth.NewPasswordManager(cfg.Auth.BcryptCost)
	if err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("create password manager: %w", err)
	}

	tokens, err := auth.NewTokenManager(cfg.Auth.Secret, cfg.Auth.Issuer, cfg.Auth.TokenTTL)
	if err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("create token manager: %w", err)
	}

	userService := users.NewService(userRepository, passwords)
	taskService := tasks.NewService(taskRepository, userRepository)

	if cfg.Bootstrap.Admin.Enabled {
		admin, created, err := userService.BootstrapAdmin(ctx, users.CreateInput{
			Username:  cfg.Bootstrap.Admin.Username,
			FirstName: cfg.Bootstrap.Admin.FirstName,
			LastName:  cfg.Bootstrap.Admin.LastName,
			Password:  cfg.Bootstrap.Admin.Password,
			Role:      users.RoleAdmin,
		})
		if err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("bootstrap admin: %w", err)
		}

		if created {
			logger.Info("bootstrap admin created", "username", admin.Username)
		} else {
			logger.Info("bootstrap admin skipped", "username", admin.Username)
		}
	}

	contract, err := apicontract.Load(cfg.OpenAPI)
	if err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("load OpenAPI contract: %w", err)
	}

	router := httpapi.NewRouter(httpapi.Dependencies{
		Logger:         logger,
		Users:          userService,
		Tasks:          taskService,
		Tokens:         tokens,
		Contract:       contract,
		DatabaseHealth: func(ctx context.Context) error { return db.PingContext(ctx) },
		PprofEnabled:   cfg.Observability.PprofEnabled,
		DocsEnabled:    cfg.OpenAPI.ServeDocumentation,
		RequestTimeout: cfg.HTTP.RequestTimeout,
	})

	server := &http.Server{
		Addr:         cfg.HTTP.Address(),
		Handler:      router,
		ReadTimeout:  cfg.HTTP.ReadTimeout,
		WriteTimeout: cfg.HTTP.WriteTimeout,
		IdleTimeout:  cfg.HTTP.IdleTimeout,
	}

	return &App{
		config: cfg,
		logger: logger,
		db:     db,
		server: server,
	}, nil
}

// Handler returns the application HTTP handler.
func (a *App) Handler() http.Handler {
	return a.server.Handler
}

// Run starts the HTTP server and blocks until the context is canceled or the server fails.
func (a *App) Run(ctx context.Context) error {
	errCh := make(chan error, 1)

	go func() {
		a.logger.Info("http server starting", "addr", a.server.Addr, "env", a.config.App.Environment)
		if err := a.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- fmt.Errorf("listen and serve: %w", err)
			return
		}

		errCh <- nil
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		a.logger.Info("shutdown requested")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), a.config.App.ShutdownTimeout)
		defer cancel()
		return a.Shutdown(shutdownCtx)
	}
}

// Shutdown gracefully stops the server and closes the database.
func (a *App) Shutdown(ctx context.Context) error {
	var combined error

	if err := a.server.Shutdown(ctx); err != nil {
		combined = errors.Join(combined, fmt.Errorf("shutdown http server: %w", err))
	}

	if err := a.db.Close(); err != nil {
		combined = errors.Join(combined, fmt.Errorf("close database: %w", err))
	}

	return combined
}

func newLogger(cfg config.Config) *slog.Logger {
	handlerOptions := &slog.HandlerOptions{Level: cfg.Observability.LogLevel}
	if cfg.App.Environment == "development" {
		return slog.New(slog.NewTextHandler(os.Stdout, handlerOptions))
	}

	return slog.New(slog.NewJSONHandler(os.Stdout, handlerOptions))
}

func openStores(ctx context.Context, cfg config.Config) (*sql.DB, users.Repository, tasks.Repository, error) {
	switch cfg.Database.Driver {
	case config.DatabaseDriverSQLite:
		db, err := sqlite.Open(ctx, cfg.Database)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("open sqlite database: %w", err)
		}

		return db, sqlite.NewUserRepository(db), sqlite.NewTaskRepository(db), nil
	case config.DatabaseDriverPostgres:
		db, err := postgres.Open(ctx, cfg.Database)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("open postgres database: %w", err)
		}

		return db, postgres.NewUserRepository(db), postgres.NewTaskRepository(db), nil
	default:
		return nil, nil, nil, fmt.Errorf("unsupported database driver %q", cfg.Database.Driver)
	}
}

func applyMigrations(ctx context.Context, cfg config.Config, db *sql.DB) error {
	switch cfg.Database.Driver {
	case config.DatabaseDriverSQLite:
		return sqlite.ApplyMigrations(ctx, db, cfg.Database.MigrationDir())
	case config.DatabaseDriverPostgres:
		return postgres.ApplyMigrations(ctx, db, cfg.Database.MigrationDir())
	default:
		return fmt.Errorf("unsupported database driver %q", cfg.Database.Driver)
	}
}
