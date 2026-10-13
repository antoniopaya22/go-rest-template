package tasks

import (
	"context"
	"strings"

	"github.com/antoniopaya22/go-rest-template/internal/app/api/domain/users"
	"github.com/antoniopaya22/go-rest-template/internal/pkg/errs"
)

type userFinder interface {
	GetByID(ctx context.Context, id int64) (users.User, error)
}

// Service encapsulates task business rules.
type Service struct {
	repository Repository
	users      userFinder
}

// NewService creates a task service.
func NewService(repository Repository, users userFinder) *Service {
	return &Service{repository: repository, users: users}
}

// List returns tasks matching the filter.
func (s *Service) List(ctx context.Context, filter Filter) ([]Task, error) {
	return s.repository.List(ctx, filter)
}

// Get returns a task by identifier.
func (s *Service) Get(ctx context.Context, id int64) (Task, error) {
	if id <= 0 {
		return Task{}, errs.New(errs.CodeInvalidArgument, "task id must be greater than zero", nil)
	}

	return s.repository.GetByID(ctx, id)
}

// Create validates input and stores the task.
func (s *Service) Create(ctx context.Context, input CreateInput) (Task, error) {
	normalized, err := normalizeCreateInput(input)
	if err != nil {
		return Task{}, err
	}

	if _, err := s.users.GetByID(ctx, normalized.UserID); err != nil {
		return Task{}, err
	}

	return s.repository.Create(ctx, Task{
		Title:       normalized.Title,
		Description: normalized.Description,
		Status:      normalized.Status,
		UserID:      normalized.UserID,
	})
}

// Update applies changes to an existing task.
func (s *Service) Update(ctx context.Context, id int64, input UpdateInput) (Task, error) {
	if id <= 0 {
		return Task{}, errs.New(errs.CodeInvalidArgument, "task id must be greater than zero", nil)
	}

	current, err := s.repository.GetByID(ctx, id)
	if err != nil {
		return Task{}, err
	}

	if input.Title != nil {
		title := strings.TrimSpace(*input.Title)
		if title == "" {
			return Task{}, errs.New(errs.CodeInvalidArgument, "title cannot be blank", nil)
		}

		current.Title = title
	}

	if input.Description != nil {
		description := strings.TrimSpace(*input.Description)
		if description == "" {
			return Task{}, errs.New(errs.CodeInvalidArgument, "description cannot be blank", nil)
		}

		current.Description = description
	}

	if input.Status != nil {
		if !input.Status.Valid() {
			return Task{}, errs.New(errs.CodeInvalidArgument, "status must be pending, in_progress or done", nil)
		}

		current.Status = *input.Status
	}

	return s.repository.Update(ctx, current)
}

// Delete removes a task by identifier.
func (s *Service) Delete(ctx context.Context, id int64) error {
	if id <= 0 {
		return errs.New(errs.CodeInvalidArgument, "task id must be greater than zero", nil)
	}

	return s.repository.Delete(ctx, id)
}

func normalizeCreateInput(input CreateInput) (CreateInput, error) {
	title := strings.TrimSpace(input.Title)
	description := strings.TrimSpace(input.Description)
	status := input.Status

	if title == "" {
		return CreateInput{}, errs.New(errs.CodeInvalidArgument, "title is required", nil)
	}

	if description == "" {
		return CreateInput{}, errs.New(errs.CodeInvalidArgument, "description is required", nil)
	}

	if input.UserID <= 0 {
		return CreateInput{}, errs.New(errs.CodeInvalidArgument, "user_id must be greater than zero", nil)
	}

	if status == "" {
		status = StatusPending
	}

	if !status.Valid() {
		return CreateInput{}, errs.New(errs.CodeInvalidArgument, "status must be pending, in_progress or done", nil)
	}

	return CreateInput{
		Title:       title,
		Description: description,
		Status:      status,
		UserID:      input.UserID,
	}, nil
}
