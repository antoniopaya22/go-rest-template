package middleware

import (
	"net/http"
	"time"

	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httprate"

	"github.com/antoniopaya22/go-rest-template/internal/platform/errs"
	"github.com/antoniopaya22/go-rest-template/internal/platform/httpx"
)

// RateLimit allows a number of requests per client IP in each window. The
// counters live in process memory: behind several replicas each one enforces
// its own budget. Swap the httprate counter for a shared store if you need a
// global limit.
func RateLimit(requests int, window time.Duration) func(http.Handler) http.Handler {
	return httprate.LimitBy(requests, window,
		func(r *http.Request) (string, error) {
			return chimiddleware.GetClientIP(r.Context()), nil
		},
		httprate.WithLimitHandler(func(w http.ResponseWriter, r *http.Request) {
			httpx.WriteError(w, r, errs.New(errs.CodeRateLimited, "too many requests, retry later"))
		}),
	)
}
