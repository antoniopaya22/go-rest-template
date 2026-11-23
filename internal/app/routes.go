package app

import (
	"context"
	"net/http"
	"net/http/pprof"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/prometheus/client_golang/prometheus/promhttp" // init:metrics

	"github.com/antoniopaya22/go-rest-template/v3/internal/platform/errs"
	"github.com/antoniopaya22/go-rest-template/v3/internal/platform/httpx"
	"github.com/antoniopaya22/go-rest-template/v3/internal/platform/middleware"
	"github.com/antoniopaya22/go-rest-template/v3/internal/platform/tracing" // init:otel
	"github.com/antoniopaya22/go-rest-template/v3/internal/platform/version"
)

// routes builds the public API: global middleware, operational endpoints and
// the versioned API with every feature mounted on it.
func (a *App) routes() http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	if a.cfg.HTTP.TrustedProxies > 0 {
		r.Use(chimiddleware.ClientIPFromXFFTrustedProxies(a.cfg.HTTP.TrustedProxies))
	} else {
		r.Use(chimiddleware.ClientIPFromRemoteAddr)
	}
	// init:otel:begin
	if a.cfg.Tracing.Enabled {
		r.Use(tracing.Middleware(a.cfg.Tracing.ServiceName))
	}
	// init:otel:end
	r.Use(middleware.Logger(a.logger))
	r.Use(middleware.Recover)
	// init:metrics:begin
	if a.cfg.Metrics.Enabled {
		r.Use(middleware.Metrics(a.registry))
	}
	// init:metrics:end
	r.Use(middleware.SecurityHeaders)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins: a.cfg.HTTP.CORSAllowedOrigins,
		AllowedMethods: []string{http.MethodGet, http.MethodPost, http.MethodPatch, http.MethodDelete, http.MethodOptions},
		AllowedHeaders: []string{"Accept", "Authorization", "Content-Type", "If-Match", "X-Request-ID"},
		ExposedHeaders: []string{"ETag", "X-Request-ID"},
		MaxAge:         300,
	}))
	r.Use(chimiddleware.RequestSize(int64(a.cfg.HTTP.MaxBodyBytes)))
	r.Use(chimiddleware.Timeout(a.cfg.HTTP.RequestTimeout))

	r.NotFound(middleware.NotFound)
	r.MethodNotAllowed(middleware.MethodNotAllowed)

	r.Get("/health/live", a.live)
	r.Get("/health/ready", a.ready)
	r.Get("/version", func(w http.ResponseWriter, _ *http.Request) {
		httpx.WriteJSON(w, http.StatusOK, version.Get())
	})
	r.Get("/openapi.yaml", a.contract.SpecHandler())
	if a.cfg.OpenAPI.DocsEnabled {
		r.Get("/docs", a.contract.DocsHandler(a.cfg.App.Name+" API", "/openapi.yaml"))
	}
	if jwks, ok := a.auth.JWKS(); ok {
		r.Get("/.well-known/jwks.json", jwks)
	}

	r.Route("/api/v1", func(r chi.Router) {
		var loginLimiter func(http.Handler) http.Handler
		// init:ratelimit:begin
		if a.cfg.RateLimit.Enabled {
			r.Use(middleware.RateLimit(a.cfg.RateLimit.Requests, a.cfg.RateLimit.Window))
			loginLimiter = middleware.RateLimit(a.cfg.RateLimit.LoginRequests, a.cfg.RateLimit.Window)
		}
		// init:ratelimit:end
		r.Use(a.contract.Middleware)

		a.auth.Routes(r, loginLimiter)
		a.users.Routes(r, a.authn)
		a.tasks.Routes(r, a.authn) // init:example
		// scaffold:routes
	})

	return r
}

// adminRoutes builds the internal server. It carries no authentication, so it
// must only be reachable from a trusted network.
func (a *App) adminRoutes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health/live", a.live)

	// init:metrics:begin
	if a.cfg.Metrics.Enabled {
		mux.Handle("GET /metrics", promhttp.HandlerFor(a.registry, promhttp.HandlerOpts{Registry: a.registry}))
	}
	// init:metrics:end

	if a.cfg.Admin.PprofEnabled {
		mux.HandleFunc("/debug/pprof/", pprof.Index)
		mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
		mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
		mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
		mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
	}

	return mux
}

// live reports that the process is running.
func (a *App) live(w http.ResponseWriter, _ *http.Request) {
	httpx.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// ready reports whether the service can take traffic: the database answers
// and the process is not shutting down.
func (a *App) ready(w http.ResponseWriter, r *http.Request) {
	if a.draining.Load() {
		httpx.WriteError(w, r, errs.New(errs.CodeUnavailable, "shutting down"))
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	if err := a.db.Ping(ctx); err != nil {
		httpx.WriteError(w, r, errs.Wrap(errs.CodeUnavailable, "database not ready", err))
		return
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}
