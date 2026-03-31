// Package middleware provides HTTP middleware for the API transport.
package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/antonioalfa22/go-rest-template/internal/app/api/auth"
	"github.com/antonioalfa22/go-rest-template/internal/app/api/domain/users"
	"github.com/antonioalfa22/go-rest-template/internal/pkg/errs"
)

type principalKey struct{}

// Principal describes the authenticated subject bound to a request.
type Principal struct {
	UserID   int64
	Username string
	Role     users.Role
}

// Authenticator validates bearer tokens and stores the principal in the request context.
type Authenticator struct {
	tokens *auth.TokenManager
}

// NewAuthenticator creates a new authenticator middleware.
func NewAuthenticator(tokens *auth.TokenManager) *Authenticator {
	return &Authenticator{tokens: tokens}
}

// Require enforces bearer token authentication.
func (a *Authenticator) Require(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := strings.TrimSpace(r.Header.Get("Authorization"))
		parts := strings.SplitN(header, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || strings.TrimSpace(parts[1]) == "" {
			writeUnauthorized(w, errs.New(errs.CodeUnauthorized, "missing bearer token", nil))
			return
		}

		claims, err := a.tokens.Parse(strings.TrimSpace(parts[1]))
		if err != nil {
			writeUnauthorized(w, err)
			return
		}

		principal := Principal{
			UserID:   claims.UserID,
			Username: claims.Username,
			Role:     claims.Role,
		}

		ctx := context.WithValue(r.Context(), principalKey{}, principal)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// PrincipalFromContext extracts the authenticated principal from a request context.
func PrincipalFromContext(ctx context.Context) (Principal, bool) {
	principal, ok := ctx.Value(principalKey{}).(Principal)
	return principal, ok
}

func writeUnauthorized(w http.ResponseWriter, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"code":  string(errs.CodeOf(err)),
		"error": errs.MessageOf(err),
	})
}
