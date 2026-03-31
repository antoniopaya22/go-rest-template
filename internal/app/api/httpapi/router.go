// Package httpapi provides the HTTP transport for the API template.
package httpapi

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/antonioalfa22/go-rest-template/internal/app/api/apicontract"
	"github.com/antonioalfa22/go-rest-template/internal/app/api/auth"
	"github.com/antonioalfa22/go-rest-template/internal/app/api/domain/tasks"
	"github.com/antonioalfa22/go-rest-template/internal/app/api/domain/users"
	authmiddleware "github.com/antonioalfa22/go-rest-template/internal/app/api/httpapi/middleware"
	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Dependencies groups the services required by the router.
type Dependencies struct {
	Logger         *slog.Logger
	Users          *users.Service
	Tasks          *tasks.Service
	Tokens         *auth.TokenManager
	Contract       *apicontract.Contract
	DatabaseHealth func(context.Context) error
	PprofEnabled   bool
	DocsEnabled    bool
	RequestTimeout time.Duration
}

// NewRouter builds the API router and middleware stack.
func NewRouter(deps Dependencies) http.Handler {
	logger := deps.Logger
	if logger == nil {
		logger = slog.Default()
	}

	if deps.RequestTimeout <= 0 {
		deps.RequestTimeout = 30 * time.Second
	}

	handler := &handler{
		logger:         logger,
		users:          deps.Users,
		tasks:          deps.Tasks,
		tokens:         deps.Tokens,
		databaseHealth: deps.DatabaseHealth,
	}

	authenticator := authmiddleware.NewAuthenticator(deps.Tokens)

	router := chi.NewRouter()
	router.Use(chimiddleware.RequestID)
	router.Use(chimiddleware.RealIP)
	router.Use(chimiddleware.Recoverer)
	router.Use(chimiddleware.Timeout(deps.RequestTimeout))
	router.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Request-ID"},
		ExposedHeaders:   []string{"X-Request-ID"},
		AllowCredentials: false,
		MaxAge:           300,
	}))
	router.Use(requestLogger(logger))

	router.Get("/health/live", handler.live)
	router.Get("/health/ready", handler.ready)
	router.Handle("/metrics", promhttp.Handler())
	if deps.Contract != nil {
		router.Get("/openapi/openapi.yaml", deps.Contract.SpecHandler())
		if deps.DocsEnabled {
			router.Get("/docs", deps.Contract.DocsHandler())
		}
	}

	if deps.PprofEnabled {
		router.Mount("/debug", chimiddleware.Profiler())
	}

	router.Route("/api/v1", func(r chi.Router) {
		if deps.Contract != nil {
			r.Use(deps.Contract.Middleware(logger))
		}

		r.Post("/auth/login", handler.login)
		r.Post("/users", handler.createUser)

		r.Group(func(r chi.Router) {
			r.Use(authenticator.Require)
			r.Get("/users/me", handler.currentUser)
			r.Get("/users", handler.listUsers)
			r.Get("/users/{userID}", handler.getUser)
			r.Patch("/users/{userID}", handler.updateUser)
			r.Delete("/users/{userID}", handler.deleteUser)

			r.Get("/tasks", handler.listTasks)
			r.Post("/tasks", handler.createTask)
			r.Get("/tasks/{taskID}", handler.getTask)
			r.Patch("/tasks/{taskID}", handler.updateTask)
			r.Delete("/tasks/{taskID}", handler.deleteTask)
		})
	})

	return router
}

func requestLogger(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			startedAt := time.Now()
			recorder := &statusRecorder{ResponseWriter: w, status: http.StatusOK}

			next.ServeHTTP(recorder, r)

			logger.Info(
				"http request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", recorder.status,
				"remote_addr", r.RemoteAddr,
				"request_id", chimiddleware.GetReqID(r.Context()),
				"duration", time.Since(startedAt),
			)
		})
	}
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}
