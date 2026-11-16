// Package app wires configuration, infrastructure and features into a running service.
//
// This file is the composition root: it is the only place that knows about
// every package. Adding a feature means constructing it in New and mounting
// its routes in routes.go.
package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"sync/atomic"

	"github.com/prometheus/client_golang/prometheus"            // init:metrics
	"github.com/prometheus/client_golang/prometheus/collectors" // init:metrics

	"github.com/antoniopaya22/go-rest-template/api"
	"github.com/antoniopaya22/go-rest-template/internal/auth"
	"github.com/antoniopaya22/go-rest-template/internal/config"
	"github.com/antoniopaya22/go-rest-template/internal/platform/contract"
	"github.com/antoniopaya22/go-rest-template/internal/platform/database"
	"github.com/antoniopaya22/go-rest-template/internal/platform/tracing" // init:otel
	"github.com/antoniopaya22/go-rest-template/internal/tasks"            // init:example
	"github.com/antoniopaya22/go-rest-template/internal/users"
)

// App is a fully wired service.
type App struct {
	cfg    config.Config
	logger *slog.Logger
	db     *database.DB

	contract *contract.Contract
	authn    *auth.Middleware
	auth     *auth.Handler
	users    *users.Handler
	tasks    *tasks.Handler // init:example

	registry *prometheus.Registry // init:metrics

	api      *http.Server
	admin    *http.Server
	draining atomic.Bool
	cleanup  []func(context.Context) error
}

// New connects to the database, applies migrations when configured and builds
// every component. The caller owns the returned App and must call Close.
func New(ctx context.Context, cfg config.Config, logger *slog.Logger) (_ *App, err error) {
	a := &App{cfg: cfg, logger: logger}
	defer func() {
		if err != nil {
			_ = a.Close(ctx)
		}
	}()

	// init:otel:begin
	if cfg.Tracing.Enabled {
		shutdown, err := tracing.Setup(ctx, cfg.Tracing.ServiceName)
		if err != nil {
			return nil, fmt.Errorf("set up tracing: %w", err)
		}
		a.cleanup = append(a.cleanup, shutdown)
	}
	// init:otel:end

	a.db, err = database.Open(ctx, cfg.Database)
	if err != nil {
		return nil, err
	}
	a.cleanup = append(a.cleanup, func(context.Context) error { return a.db.Close() })

	if cfg.Database.AutoMigrate {
		if err := a.db.Migrate(ctx); err != nil {
			return nil, err
		}
	}

	hasher, err := auth.NewHasher(cfg.Auth.BcryptCost)
	if err != nil {
		return nil, fmt.Errorf("create password hasher: %w", err)
	}

	signer, err := auth.NewSigner(cfg.Auth)
	if err != nil {
		return nil, fmt.Errorf("create token signer: %w", err)
	}

	userService := users.NewService(users.NewRepository(a.db), hasher)
	a.users = users.NewHandler(userService, cfg.Auth.RegistrationEnabled)
	a.authn = auth.NewMiddleware(signer, userService)
	a.auth = auth.NewHandler(auth.NewService(a.db, userService, signer, cfg.Auth.RefreshTokenTTL), signer)
	a.tasks = tasks.NewHandler(tasks.NewService(tasks.NewRepository(a.db))) // init:example

	if cfg.Bootstrap.AdminEnabled {
		created, err := userService.EnsureAdmin(ctx, users.CreateInput{
			Username:  cfg.Bootstrap.AdminUsername,
			FirstName: cfg.Bootstrap.AdminFirstName,
			LastName:  cfg.Bootstrap.AdminLastName,
			Password:  cfg.Bootstrap.AdminPassword,
		})
		if err != nil {
			return nil, fmt.Errorf("bootstrap administrator: %w", err)
		}
		if created {
			logger.Info("bootstrap administrator created", "username", cfg.Bootstrap.AdminUsername)
		}
	}

	a.contract, err = contract.Load(api.Spec, contract.Options{
		ValidateRequests:  cfg.OpenAPI.ValidateRequests,
		ValidateResponses: cfg.OpenAPI.ValidateResponses,
	})
	if err != nil {
		return nil, err
	}

	// init:metrics:begin
	a.registry = prometheus.NewRegistry()
	a.registry.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
	)
	// init:metrics:end

	a.api = &http.Server{
		Addr:              cfg.HTTP.Address(),
		Handler:           a.routes(),
		ReadTimeout:       cfg.HTTP.ReadTimeout,
		ReadHeaderTimeout: cfg.HTTP.ReadTimeout,
		WriteTimeout:      cfg.HTTP.WriteTimeout,
		IdleTimeout:       cfg.HTTP.IdleTimeout,
	}

	if cfg.Admin.Enabled() {
		a.admin = &http.Server{
			Addr:              cfg.Admin.Address(),
			Handler:           a.adminRoutes(),
			ReadHeaderTimeout: cfg.HTTP.ReadTimeout,
		}
	}

	return a, nil
}

// Handler returns the public API handler. Tests use it with httptest.
func (a *App) Handler() http.Handler { return a.api.Handler }

// AdminHandler returns the handler of the admin server.
func (a *App) AdminHandler() http.Handler { return a.adminRoutes() }

// Run serves until ctx is canceled or a server fails, then shuts down gracefully.
func (a *App) Run(ctx context.Context) error {
	failed := make(chan error, 2)
	serve := func(name string, server *http.Server) {
		a.logger.Info("server listening", "server", name, "addr", server.Addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			failed <- fmt.Errorf("%s server: %w", name, err)
		}
	}

	go serve("api", a.api)
	if a.admin != nil {
		go serve("admin", a.admin)
	}

	var runErr error
	select {
	case runErr = <-failed:
	case <-ctx.Done():
		a.logger.Info("shutdown requested")
	}

	shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), a.cfg.App.ShutdownTimeout)
	defer cancel()
	return errors.Join(runErr, a.Close(shutdownCtx))
}

// Close stops accepting requests, waits for in-flight ones and releases every
// resource. It is safe to call more than once.
func (a *App) Close(ctx context.Context) error {
	// Fail readiness first so load balancers stop routing here.
	a.draining.Store(true)

	var combined error
	for _, server := range []*http.Server{a.api, a.admin} {
		if server != nil {
			combined = errors.Join(combined, server.Shutdown(ctx))
		}
	}

	for i := len(a.cleanup) - 1; i >= 0; i-- {
		combined = errors.Join(combined, a.cleanup[i](ctx))
	}
	a.cleanup = nil

	return combined
}

// ContractOperations lists the operations of the OpenAPI contract as
// "METHOD /path". Tests use it to check that the router and the contract agree.
func (a *App) ContractOperations() []string { return a.contract.Operations() }
