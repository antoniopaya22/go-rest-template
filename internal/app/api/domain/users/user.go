// Package users contains the user domain model and business logic.
package users

import "time"

// Role identifies the authorization role of a user.
type Role string

const (
	// RoleAdmin grants administrative permissions.
	RoleAdmin Role = "admin"
	// RoleUser grants regular user permissions.
	RoleUser Role = "user"
)

// Valid reports whether the role is supported.
func (r Role) Valid() bool {
	return r == RoleAdmin || r == RoleUser
}

// User represents a persisted user account.
type User struct {
	ID           int64     `json:"id"`
	Username     string    `json:"username"`
	FirstName    string    `json:"first_name"`
	LastName     string    `json:"last_name"`
	PasswordHash string    `json:"-"`
	Role         Role      `json:"role"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Filter defines query criteria for listing users.
type Filter struct {
	Username string
	Role     Role
}

// CreateInput contains the data required to create a user.
type CreateInput struct {
	Username  string `json:"username"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Password  string `json:"password"`
	Role      Role   `json:"role"`
}

// UpdateInput contains mutable user fields.
type UpdateInput struct {
	FirstName string  `json:"first_name"`
	LastName  string  `json:"last_name"`
	Password  *string `json:"password,omitempty"`
	Role      *Role   `json:"role,omitempty"`
}
