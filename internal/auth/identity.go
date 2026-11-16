// Package auth implements authentication: password hashing, access and
// refresh tokens, and the middleware that binds a caller to a request.
//
// The package does not know how users are stored. It depends on a
// UserDirectory, which the users package implements.
package auth

import (
	"context"

	"github.com/antoniopaya22/go-rest-template/internal/platform/errs"
)

// Role identifies the authorization role of a user.
type Role string

// Supported roles.
const (
	RoleAdmin Role = "admin"
	RoleUser  Role = "user"
)

// Valid reports whether the role is supported.
func (r Role) Valid() bool { return r == RoleAdmin || r == RoleUser }

// Identity is the authenticated caller of a request.
type Identity struct {
	UserID   int64
	Username string
	Role     Role
}

// IsAdmin reports whether the caller is an administrator.
func (i Identity) IsAdmin() bool { return i.Role == RoleAdmin }

// Authorize allows administrators and the owner of a resource, and rejects
// everyone else.
func (i Identity) Authorize(ownerID int64) error {
	if i.IsAdmin() || i.UserID == ownerID {
		return nil
	}
	return errs.Forbidden("insufficient permissions")
}

// UserDirectory is what authentication needs to know about users.
type UserDirectory interface {
	// Authenticate verifies a username and password.
	Authenticate(ctx context.Context, username string, password string) (Identity, error)
	// Identity returns the current identity of a user. It is called on every
	// authenticated request, so role changes and deletions apply immediately.
	Identity(ctx context.Context, userID int64) (Identity, error)
}

type identityKey struct{}

// WithIdentity returns a context carrying the authenticated caller.
func WithIdentity(ctx context.Context, identity Identity) context.Context {
	return context.WithValue(ctx, identityKey{}, identity)
}

// IdentityFrom returns the authenticated caller, if the request has one.
func IdentityFrom(ctx context.Context) (Identity, bool) {
	identity, ok := ctx.Value(identityKey{}).(Identity)
	return identity, ok
}

// MustIdentity returns the authenticated caller. It panics when the route is
// not behind Middleware.Authenticate, which is a wiring mistake.
func MustIdentity(ctx context.Context) Identity {
	identity, ok := IdentityFrom(ctx)
	if !ok {
		panic("auth: handler reached without an authenticated identity")
	}
	return identity
}
