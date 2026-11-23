package tasks

import (
	"context"
	"strings"

	"github.com/antoniopaya22/go-rest-template/v3/internal/platform/errs"
	"github.com/antoniopaya22/go-rest-template/v3/internal/platform/httpx"
)

// Store is the persistence the service needs. Repository implements it.
type Store interface {
	List(ctx context.Context, filter Filter, page httpx.PageRequest) ([]Task, error)
	GetByID(ctx context.Context, id int64) (Task, error)
	Create(ctx context.Context, task Task) (Task, error)
	Update(ctx context.Context, task Task) (Task, error)
	Delete(ctx context.Context, id int64) error
}

// Service holds the business rules of tasks.
type Service struct {
	store Store
}

// NewService creates a task service.
func NewService(store Store) *Service {
	return &Service{store: store}
}

// List returns one page of tasks matching filter.
func (s *Service) List(ctx context.Context, filter Filter, page httpx.PageRequest) ([]Task, error) {
	if filter.Status != "" && !filter.Status.Valid() {
		return nil, errs.InvalidField("status", "must be pending, in_progress or done")
	}
	return s.store.List(ctx, filter, page)
}

// Get returns a task by identifier.
func (s *Service) Get(ctx context.Context, id int64) (Task, error) {
	return s.store.GetByID(ctx, id)
}

// Create validates input and stores a task owned by ownerID.
func (s *Service) Create(ctx context.Context, ownerID int64, input CreateInput) (Task, error) {
	title := strings.TrimSpace(input.Title)
	description := strings.TrimSpace(input.Description)
	status := input.Status
	if status == "" {
		status = StatusPending
	}

	switch {
	case title == "":
		return Task{}, errs.InvalidField("title", "is required")
	case description == "":
		return Task{}, errs.InvalidField("description", "is required")
	case !status.Valid():
		return Task{}, errs.InvalidField("status", "must be pending, in_progress or done")
	}

	// A missing owner surfaces as a foreign key violation, which the
	// repository reports as invalid input.
	return s.store.Create(ctx, Task{
		Title:       title,
		Description: description,
		Status:      status,
		UserID:      ownerID,
	})
}

// Update applies a partial update to current, which the caller has already
// loaded and authorized.
func (s *Service) Update(ctx context.Context, current Task, input UpdateInput) (Task, error) {
	if input.Title != nil {
		current.Title = strings.TrimSpace(*input.Title)
		if current.Title == "" {
			return Task{}, errs.InvalidField("title", "cannot be blank")
		}
	}

	if input.Description != nil {
		current.Description = strings.TrimSpace(*input.Description)
		if current.Description == "" {
			return Task{}, errs.InvalidField("description", "cannot be blank")
		}
	}

	if input.Status != nil {
		if !input.Status.Valid() {
			return Task{}, errs.InvalidField("status", "must be pending, in_progress or done")
		}
		current.Status = *input.Status
	}

	return s.store.Update(ctx, current)
}

// Delete removes a task.
func (s *Service) Delete(ctx context.Context, id int64) error {
	return s.store.Delete(ctx, id)
}
