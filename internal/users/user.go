// Package users manages user accounts: the model, its business rules, its
// storage and its HTTP endpoints.
package users

import (
	"time"

	"github.com/antoniopaya22/go-rest-template/v3/internal/auth"
)

// User is a user account.
type User struct {
	ID           int64     `json:"id"`
	Username     string    `json:"username"`
	FirstName    string    `json:"first_name"`
	LastName     string    `json:"last_name"`
	PasswordHash string    `json:"-"`
	Role         auth.Role `json:"role"`
	Version      int64     `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Filter narrows a user listing.
type Filter struct {
	// Username matches accounts whose username contains this text.
	Username string
	Role     auth.Role
}

// CreateInput is the data required to create a user.
type CreateInput struct {
	Username  string    `json:"username"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	Password  string    `json:"password"`
	Role      auth.Role `json:"role"`
}

// UpdateInput holds the fields of a partial update. Nil fields are left unchanged.
type UpdateInput struct {
	FirstName *string    `json:"first_name"`
	LastName  *string    `json:"last_name"`
	Password  *string    `json:"password"`
	Role      *auth.Role `json:"role"`
}
