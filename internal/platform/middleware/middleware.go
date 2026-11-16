// Package middleware contains the transport-level HTTP middleware.
package middleware

import (
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"

	"github.com/antoniopaya22/go-rest-template/internal/platform/errs"
	"github.com/antoniopaya22/go-rest-template/internal/platform/httpx"
	"github.com/antoniopaya22/go-rest-template/internal/platform/logging"
)

// Logger attaches a request-scoped logger to the context and logs one record
// per request once it completes.
func Logger(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			startedAt := time.Now()
			requestLogger := logger.With("request_id", chimiddleware.GetReqID(r.Context()))
			recorder := chimiddleware.NewWrapResponseWriter(w, r.ProtoMajor)

			next.ServeHTTP(recorder, r.WithContext(logging.WithContext(r.Context(), requestLogger)))

			requestLogger.LogAttrs(r.Context(), levelFor(recorder.Status()), "http request",
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.String("route", RoutePattern(r)),
				slog.Int("status", recorder.Status()),
				slog.Int("bytes", recorder.BytesWritten()),
				slog.Duration("duration", time.Since(startedAt)),
				slog.String("client_ip", chimiddleware.GetClientIP(r.Context())),
			)
		})
	}
}

func levelFor(status int) slog.Level {
	switch {
	case status >= http.StatusInternalServerError:
		return slog.LevelError
	case status >= http.StatusBadRequest:
		return slog.LevelWarn
	default:
		return slog.LevelInfo
	}
}

// RoutePattern returns the matched route template, such as /api/v1/users/{userID}.
// It is only complete after the request has been routed.
func RoutePattern(r *http.Request) string {
	if routeContext := chi.RouteContext(r.Context()); routeContext != nil {
		if pattern := routeContext.RoutePattern(); pattern != "" {
			return pattern
		}
	}
	return "unmatched"
}

// Recover turns a panic into a 500 problem response and logs the stack trace.
func Recover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			recovered := recover()
			if recovered == nil {
				return
			}
			if recovered == http.ErrAbortHandler { //nolint:errorlint // sentinel panic value, re-raised as documented.
				panic(recovered)
			}

			logging.FromContext(r.Context()).ErrorContext(r.Context(), "panic recovered",
				"panic", fmt.Sprint(recovered), "stack", string(debug.Stack()))
			httpx.WriteError(w, r, errs.New(errs.CodeInternal, "internal error"))
		}()

		next.ServeHTTP(w, r)
	})
}

// SecurityHeaders sets conservative response headers for a JSON API.
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := w.Header()
		header.Set("X-Content-Type-Options", "nosniff")
		header.Set("X-Frame-Options", "DENY")
		header.Set("Referrer-Policy", "no-referrer")
		header.Set("Cross-Origin-Resource-Policy", "same-origin")
		if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
			header.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}

		next.ServeHTTP(w, r)
	})
}

// NotFound and MethodNotAllowed keep unmatched routes on the problem format.

// NotFound writes a 404 problem response.
func NotFound(w http.ResponseWriter, r *http.Request) {
	httpx.WriteError(w, r, errs.NotFound("resource not found"))
}

// MethodNotAllowed writes a 405 problem response.
func MethodNotAllowed(w http.ResponseWriter, r *http.Request) {
	httpx.WriteProblem(w, httpx.Problem{
		Type:      "about:blank",
		Title:     http.StatusText(http.StatusMethodNotAllowed),
		Status:    http.StatusMethodNotAllowed,
		Code:      errs.CodeInvalidArgument,
		Detail:    "method not allowed for this resource",
		RequestID: chimiddleware.GetReqID(r.Context()),
	})
}
