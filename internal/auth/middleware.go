package auth

import (
	"net/http"
	"slices"
	"strings"

	"github.com/antoniopaya22/go-rest-template/v3/internal/platform/errs"
	"github.com/antoniopaya22/go-rest-template/v3/internal/platform/httpx"
)

// Middleware authenticates requests carrying a bearer access token.
type Middleware struct {
	signer *Signer
	users  UserDirectory
}

// NewMiddleware creates the authentication middleware.
func NewMiddleware(signer *Signer, users UserDirectory) *Middleware {
	return &Middleware{signer: signer, users: users}
}

// Authenticate rejects requests without a valid access token and stores the
// caller in the request context.
func (m *Middleware) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		scheme, token, found := strings.Cut(r.Header.Get("Authorization"), " ")
		if !found || !strings.EqualFold(scheme, "Bearer") || strings.TrimSpace(token) == "" {
			w.Header().Set("WWW-Authenticate", `Bearer`)
			httpx.WriteError(w, r, errs.Unauthorized("missing bearer token"))
			return
		}

		userID, err := m.signer.Verify(strings.TrimSpace(token))
		if err != nil {
			w.Header().Set("WWW-Authenticate", `Bearer error="invalid_token"`)
			httpx.WriteError(w, r, err)
			return
		}

		identity, err := m.users.Identity(r.Context(), userID)
		if err != nil {
			if errs.Is(err, errs.CodeNotFound) {
				err = errs.Unauthorized("invalid or expired token")
			}
			httpx.WriteError(w, r, err)
			return
		}

		next.ServeHTTP(w, r.WithContext(WithIdentity(r.Context(), identity)))
	})
}

// RequireRole rejects authenticated callers whose role is not listed.
func RequireRole(roles ...Role) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !slices.Contains(roles, MustIdentity(r.Context()).Role) {
				httpx.WriteError(w, r, errs.Forbidden("insufficient permissions"))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
