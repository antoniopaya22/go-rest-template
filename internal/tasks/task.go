// Package tasks is the example resource of the template: a model owned by a
// user, with its business rules, storage and HTTP endpoints. Use it as the
// reference when adding your own resources, or generate a copy with
// `make resource name=<singular>`.
package tasks

import "time"

// Status is the workflow state of a task.
type Status string

// Supported statuses.
const (
	StatusPending    Status = "pending"
	StatusInProgress Status = "in_progress"
	StatusDone       Status = "done"
)

// Valid reports whether the status is supported.
func (s Status) Valid() bool {
	return s == StatusPending || s == StatusInProgress || s == StatusDone
}

// Task is a unit of work owned by a user.
type Task struct {
	ID          int64     `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Status      Status    `json:"status"`
	UserID      int64     `json:"user_id"`
	Version     int64     `json:"-"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Filter narrows a task listing.
type Filter struct {
	// UserID restricts the listing to the tasks of one user.
	UserID *int64
	Status Status
}

// CreateInput is the data required to create a task.
type CreateInput struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Status      Status `json:"status"`
	// UserID is the owner. Regular users can only create tasks for themselves.
	UserID *int64 `json:"user_id"`
}

// UpdateInput holds the fields of a partial update. Nil fields are left unchanged.
type UpdateInput struct {
	Title       *string `json:"title"`
	Description *string `json:"description"`
	Status      *Status `json:"status"`
}
