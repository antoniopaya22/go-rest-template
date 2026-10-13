package users

import (
	"context"
	"fmt"
	"strings"

	"github.com/antoniopaya22/go-rest-template/internal/pkg/errs"
)

type passwordManager interface {
	Hash(password string) (string, error)
	Compare(hash string, password string) error
}

// Service encapsulates user business rules.
type Service struct {
	repository Repository
	passwords  passwordManager
}

// NewService creates a user service.
func NewService(repository Repository, passwords passwordManager) *Service {
	return &Service{repository: repository, passwords: passwords}
}

// List returns users matching the provided filter.
func (s *Service) List(ctx context.Context, filter Filter) ([]User, error) {
	return s.repository.List(ctx, filter)
}

// Get returns a user by identifier.
func (s *Service) Get(ctx context.Context, id int64) (User, error) {
	if id <= 0 {
		return User{}, errs.New(errs.CodeInvalidArgument, "user id must be greater than zero", nil)
	}

	return s.repository.GetByID(ctx, id)
}

// Create validates input, hashes the password and stores the user.
func (s *Service) Create(ctx context.Context, input CreateInput) (User, error) {
	normalized, err := normalizeCreateInput(input)
	if err != nil {
		return User{}, err
	}

	_, err = s.repository.GetByUsername(ctx, normalized.Username)
	if err == nil {
		return User{}, errs.New(errs.CodeConflict, "username already exists", nil)
	}

	if errs.CodeOf(err) != errs.CodeNotFound {
		return User{}, fmt.Errorf("check existing user: %w", err)
	}

	hash, err := s.passwords.Hash(normalized.Password)
	if err != nil {
		return User{}, fmt.Errorf("hash password: %w", err)
	}

	return s.repository.Create(ctx, User{
		Username:     normalized.Username,
		FirstName:    normalized.FirstName,
		LastName:     normalized.LastName,
		PasswordHash: hash,
		Role:         normalized.Role,
	})
}

// Update applies changes to an existing user.
func (s *Service) Update(ctx context.Context, id int64, input UpdateInput) (User, error) {
	if id <= 0 {
		return User{}, errs.New(errs.CodeInvalidArgument, "user id must be greater than zero", nil)
	}

	current, err := s.repository.GetByID(ctx, id)
	if err != nil {
		return User{}, err
	}

	if strings.TrimSpace(input.FirstName) != "" {
		current.FirstName = strings.TrimSpace(input.FirstName)
	}

	if strings.TrimSpace(input.LastName) != "" {
		current.LastName = strings.TrimSpace(input.LastName)
	}

	if input.Role != nil {
		if !input.Role.Valid() {
			return User{}, errs.New(errs.CodeInvalidArgument, "role must be admin or user", nil)
		}

		current.Role = *input.Role
	}

	if input.Password != nil {
		hash, hashErr := s.passwords.Hash(*input.Password)
		if hashErr != nil {
			return User{}, fmt.Errorf("hash password: %w", hashErr)
		}

		current.PasswordHash = hash
	}

	return s.repository.Update(ctx, current)
}

// Delete removes a user by identifier.
func (s *Service) Delete(ctx context.Context, id int64) error {
	if id <= 0 {
		return errs.New(errs.CodeInvalidArgument, "user id must be greater than zero", nil)
	}

	return s.repository.Delete(ctx, id)
}

// Authenticate verifies credentials and returns the matching user.
func (s *Service) Authenticate(ctx context.Context, username string, password string) (User, error) {
	normalizedUsername := strings.ToLower(strings.TrimSpace(username))
	if normalizedUsername == "" || strings.TrimSpace(password) == "" {
		return User{}, errs.New(errs.CodeInvalidArgument, "username and password are required", nil)
	}

	user, err := s.repository.GetByUsername(ctx, normalizedUsername)
	if err != nil {
		return User{}, errs.New(errs.CodeUnauthorized, "invalid credentials", err)
	}

	if err := s.passwords.Compare(user.PasswordHash, password); err != nil {
		return User{}, err
	}

	return user, nil
}

// BootstrapAdmin creates the first administrative user when none exists.
func (s *Service) BootstrapAdmin(ctx context.Context, input CreateInput) (User, bool, error) {
	admins, err := s.repository.List(ctx, Filter{Role: RoleAdmin})
	if err != nil {
		return User{}, false, fmt.Errorf("list existing admins: %w", err)
	}

	if len(admins) > 0 {
		return admins[0], false, nil
	}

	input.Role = RoleAdmin
	created, err := s.Create(ctx, input)
	if err != nil {
		return User{}, false, fmt.Errorf("bootstrap admin: %w", err)
	}

	return created, true, nil
}

func normalizeCreateInput(input CreateInput) (CreateInput, error) {
	username := strings.ToLower(strings.TrimSpace(input.Username))
	firstName := strings.TrimSpace(input.FirstName)
	lastName := strings.TrimSpace(input.LastName)

	if username == "" {
		return CreateInput{}, errs.New(errs.CodeInvalidArgument, "username is required", nil)
	}

	if firstName == "" {
		return CreateInput{}, errs.New(errs.CodeInvalidArgument, "first_name is required", nil)
	}

	if lastName == "" {
		return CreateInput{}, errs.New(errs.CodeInvalidArgument, "last_name is required", nil)
	}

	if len(strings.TrimSpace(input.Password)) < 8 {
		return CreateInput{}, errs.New(errs.CodeInvalidArgument, "password must be at least 8 characters", nil)
	}

	role := input.Role
	if role == "" {
		role = RoleUser
	}

	if !role.Valid() {
		return CreateInput{}, errs.New(errs.CodeInvalidArgument, "role must be admin or user", nil)
	}

	return CreateInput{
		Username:  username,
		FirstName: firstName,
		LastName:  lastName,
		Password:  input.Password,
		Role:      role,
	}, nil
}
