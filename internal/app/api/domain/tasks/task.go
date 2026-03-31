// Package tasks contains the task domain model and business logic.
package tasks

import "time"

// Status identifies the workflow status of a task.
type Status string

const (
	// StatusPending identifies a task that has not started.
	StatusPending Status = "pending"
	// StatusInProgress identifies a task currently in progress.
	StatusInProgress Status = "in_progress"
	// StatusDone identifies a completed task.
	StatusDone Status = "done"
)

// Valid reports whether the status is supported.
func (s Status) Valid() bool {
	return s == StatusPending || s == StatusInProgress || s == StatusDone
}

// Task represents a persisted task record.
type Task struct {
	ID          int64     `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Status      Status    `json:"status"`
	UserID      int64     `json:"user_id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Filter defines query criteria for tasks.
type Filter struct {
	OwnerID *int64
	Status  Status
}

// CreateInput contains the data required to create a task.
type CreateInput struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Status      Status `json:"status"`
	UserID      int64  `json:"user_id"`
}

// UpdateInput contains mutable task fields.
type UpdateInput struct {
	Title       *string `json:"title,omitempty"`
	Description *string `json:"description,omitempty"`
	Status      *Status `json:"status,omitempty"`
}
