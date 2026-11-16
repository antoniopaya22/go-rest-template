package users

import (
	"context"
	"fmt"
	"strings"

	"github.com/antoniopaya22/go-rest-template/internal/auth"
	"github.com/antoniopaya22/go-rest-template/internal/platform/errs"
	"github.com/antoniopaya22/go-rest-template/internal/platform/httpx"
)

// Store is the persistence the service needs. Repository implements it.
type Store interface {
	List(ctx context.Context, filter Filter, page httpx.PageRequest) ([]User, error)
	GetByID(ctx context.Context, id int64) (User, error)
	GetByUsername(ctx context.Context, username string) (User, error)
	Create(ctx context.Context, user User) (User, error)
	Update(ctx context.Context, user User) (User, error)
	Delete(ctx context.Context, id int64) error
}

// PasswordHasher hashes and verifies passwords. auth.Hasher implements it.
type PasswordHasher interface {
	Hash(password string) (string, error)
	Matches(hash string, password string) bool
	Burn(password string)
}

// Service holds the business rules of user accounts.
type Service struct {
	store     Store
	passwords PasswordHasher
}

// NewService creates a user service.
func NewService(store Store, passwords PasswordHasher) *Service {
	return &Service{store: store, passwords: passwords}
}

// List returns one page of users matching filter.
func (s *Service) List(ctx context.Context, filter Filter, page httpx.PageRequest) ([]User, error) {
	if filter.Role != "" && !filter.Role.Valid() {
		return nil, errs.InvalidField("role", "must be admin or user")
	}
	return s.store.List(ctx, filter, page)
}

// Get returns a user by identifier.
func (s *Service) Get(ctx context.Context, id int64) (User, error) {
	return s.store.GetByID(ctx, id)
}

// Create validates input, hashes the password and stores the user.
func (s *Service) Create(ctx context.Context, input CreateInput) (User, error) {
	input.Username = strings.ToLower(strings.TrimSpace(input.Username))
	input.FirstName = strings.TrimSpace(input.FirstName)
	input.LastName = strings.TrimSpace(input.LastName)
	if input.Role == "" {
		input.Role = auth.RoleUser
	}

	switch {
	case len(input.Username) < 3:
		return User{}, errs.InvalidField("username", "must be at least 3 characters")
	case input.FirstName == "":
		return User{}, errs.InvalidField("first_name", "is required")
	case input.LastName == "":
		return User{}, errs.InvalidField("last_name", "is required")
	case !input.Role.Valid():
		return User{}, errs.InvalidField("role", "must be admin or user")
	}
	if err := auth.ValidatePassword(input.Password); err != nil {
		return User{}, err
	}

	hash, err := s.passwords.Hash(input.Password)
	if err != nil {
		return User{}, err
	}

	// The unique index on username is what guarantees uniqueness; the
	// repository reports a duplicate as a conflict.
	created, err := s.store.Create(ctx, User{
		Username:     input.Username,
		FirstName:    input.FirstName,
		LastName:     input.LastName,
		PasswordHash: hash,
		Role:         input.Role,
	})
	if errs.Is(err, errs.CodeConflict) {
		return User{}, errs.Wrap(errs.CodeConflict, "username already exists", err)
	}
	return created, err
}

// Update applies a partial update to current, which the caller has already
// loaded and authorized.
func (s *Service) Update(ctx context.Context, current User, input UpdateInput) (User, error) {
	if input.FirstName != nil {
		current.FirstName = strings.TrimSpace(*input.FirstName)
		if current.FirstName == "" {
			return User{}, errs.InvalidField("first_name", "cannot be blank")
		}
	}

	if input.LastName != nil {
		current.LastName = strings.TrimSpace(*input.LastName)
		if current.LastName == "" {
			return User{}, errs.InvalidField("last_name", "cannot be blank")
		}
	}

	if input.Role != nil {
		if !input.Role.Valid() {
			return User{}, errs.InvalidField("role", "must be admin or user")
		}
		current.Role = *input.Role
	}

	if input.Password != nil {
		if err := auth.ValidatePassword(*input.Password); err != nil {
			return User{}, err
		}
		hash, err := s.passwords.Hash(*input.Password)
		if err != nil {
			return User{}, err
		}
		current.PasswordHash = hash
	}

	return s.store.Update(ctx, current)
}

// Delete removes a user and, through the database, everything they own.
func (s *Service) Delete(ctx context.Context, id int64) error {
	return s.store.Delete(ctx, id)
}

// EnsureAdmin creates the first administrator when none exists. It reports
// whether an account was created.
func (s *Service) EnsureAdmin(ctx context.Context, input CreateInput) (bool, error) {
	admins, err := s.store.List(ctx, Filter{Role: auth.RoleAdmin}, httpx.PageRequest{Limit: 1})
	if err != nil {
		return false, fmt.Errorf("look for an existing administrator: %w", err)
	}
	if len(admins) > 0 {
		return false, nil
	}

	input.Role = auth.RoleAdmin
	if _, err := s.Create(ctx, input); err != nil {
		return false, fmt.Errorf("create administrator: %w", err)
	}
	return true, nil
}

// Authenticate verifies a username and password. It implements auth.UserDirectory.
func (s *Service) Authenticate(ctx context.Context, username string, password string) (auth.Identity, error) {
	user, err := s.store.GetByUsername(ctx, strings.ToLower(strings.TrimSpace(username)))
	if err != nil {
		if errs.Is(err, errs.CodeNotFound) {
			// Spend the same time as a real check so timing does not
			// reveal which usernames exist.
			s.passwords.Burn(password)
			return auth.Identity{}, errs.Unauthorized("invalid credentials")
		}
		return auth.Identity{}, err
	}

	if !s.passwords.Matches(user.PasswordHash, password) {
		return auth.Identity{}, errs.Unauthorized("invalid credentials")
	}
	return identityOf(user), nil
}

// Identity returns the current identity of a user. It implements auth.UserDirectory.
func (s *Service) Identity(ctx context.Context, userID int64) (auth.Identity, error) {
	user, err := s.store.GetByID(ctx, userID)
	if err != nil {
		return auth.Identity{}, err
	}
	return identityOf(user), nil
}

func identityOf(user User) auth.Identity {
	return auth.Identity{UserID: user.ID, Username: user.Username, Role: user.Role}
}
