package users_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/antonioalfa22/go-rest-template/internal/app/api/domain/users"
	"github.com/antonioalfa22/go-rest-template/internal/pkg/errs"
)

type stubPasswordManager struct{}

func (stubPasswordManager) Hash(password string) (string, error) {
	return "hash:" + password, nil
}

func (stubPasswordManager) Compare(hash string, password string) error {
	if hash != "hash:"+password {
		return errs.New(errs.CodeUnauthorized, "invalid credentials", errors.New("password mismatch"))
	}

	return nil
}

type stubUserRepository struct {
	nextID int64
	items  map[int64]users.User
}

func newStubUserRepository() *stubUserRepository {
	return &stubUserRepository{
		nextID: 1,
		items:  make(map[int64]users.User),
	}
}

func (r *stubUserRepository) List(_ context.Context, filter users.Filter) ([]users.User, error) {
	list := make([]users.User, 0, len(r.items))
	for _, user := range r.items {
		if filter.Username != "" && user.Username != filter.Username {
			continue
		}

		if filter.Role != "" && user.Role != filter.Role {
			continue
		}

		list = append(list, user)
	}

	return list, nil
}

func (r *stubUserRepository) GetByID(_ context.Context, id int64) (users.User, error) {
	user, ok := r.items[id]
	if !ok {
		return users.User{}, errs.New(errs.CodeNotFound, "user not found", nil)
	}

	return user, nil
}

func (r *stubUserRepository) GetByUsername(_ context.Context, username string) (users.User, error) {
	for _, user := range r.items {
		if user.Username == username {
			return user, nil
		}
	}

	return users.User{}, errs.New(errs.CodeNotFound, "user not found", nil)
}

func (r *stubUserRepository) Create(_ context.Context, user users.User) (users.User, error) {
	user.ID = r.nextID
	r.nextID++
	user.CreatedAt = time.Now().UTC()
	user.UpdatedAt = user.CreatedAt
	r.items[user.ID] = user
	return user, nil
}

func (r *stubUserRepository) Update(_ context.Context, user users.User) (users.User, error) {
	if _, ok := r.items[user.ID]; !ok {
		return users.User{}, errs.New(errs.CodeNotFound, "user not found", nil)
	}

	user.UpdatedAt = time.Now().UTC()
	r.items[user.ID] = user
	return user, nil
}

func (r *stubUserRepository) Delete(_ context.Context, id int64) error {
	if _, ok := r.items[id]; !ok {
		return errs.New(errs.CodeNotFound, "user not found", nil)
	}

	delete(r.items, id)
	return nil
}

func TestServiceCreate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		seed     func(*stubUserRepository)
		input    users.CreateInput
		wantCode errs.Code
		wantRole users.Role
	}{
		{
			name: "creates regular user with default role",
			input: users.CreateInput{
				Username:  "anton",
				FirstName: "Anton",
				LastName:  "Alfa",
				Password:  "super-secure",
			},
			wantRole: users.RoleUser,
		},
		{
			name: "rejects duplicate username",
			seed: func(repo *stubUserRepository) {
				_, _ = repo.Create(context.Background(), users.User{
					Username:     "anton",
					FirstName:    "Existing",
					LastName:     "User",
					PasswordHash: "hash:super-secure",
					Role:         users.RoleUser,
				})
			},
			input: users.CreateInput{
				Username:  "anton",
				FirstName: "Anton",
				LastName:  "Alfa",
				Password:  "super-secure",
			},
			wantCode: errs.CodeConflict,
		},
		{
			name: "rejects invalid role",
			input: users.CreateInput{
				Username:  "anton",
				FirstName: "Anton",
				LastName:  "Alfa",
				Password:  "super-secure",
				Role:      users.Role("root"),
			},
			wantCode: errs.CodeInvalidArgument,
		},
	}

	for _, testCase := range tests {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			repository := newStubUserRepository()
			if testCase.seed != nil {
				testCase.seed(repository)
			}

			service := users.NewService(repository, stubPasswordManager{})
			created, err := service.Create(context.Background(), testCase.input)

			if testCase.wantCode != "" {
				if errs.CodeOf(err) != testCase.wantCode {
					t.Fatalf("expected error code %q, got %q", testCase.wantCode, errs.CodeOf(err))
				}

				return
			}

			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}

			if created.Role != testCase.wantRole {
				t.Fatalf("expected role %q, got %q", testCase.wantRole, created.Role)
			}

			if created.PasswordHash == "" {
				t.Fatal("expected password hash to be populated")
			}
		})
	}
}

func TestServiceAuthenticate(t *testing.T) {
	t.Parallel()

	repository := newStubUserRepository()
	service := users.NewService(repository, stubPasswordManager{})

	created, err := service.Create(context.Background(), users.CreateInput{
		Username:  "anton",
		FirstName: "Anton",
		LastName:  "Alfa",
		Password:  "super-secure",
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	tests := []struct {
		name     string
		username string
		password string
		wantCode errs.Code
	}{
		{name: "valid credentials", username: created.Username, password: "super-secure"},
		{name: "wrong password", username: created.Username, password: "incorrect", wantCode: errs.CodeUnauthorized},
		{name: "missing user", username: "missing", password: "super-secure", wantCode: errs.CodeUnauthorized},
	}

	for _, testCase := range tests {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			user, err := service.Authenticate(context.Background(), testCase.username, testCase.password)
			if testCase.wantCode != "" {
				if errs.CodeOf(err) != testCase.wantCode {
					t.Fatalf("expected error code %q, got %q", testCase.wantCode, errs.CodeOf(err))
				}

				return
			}

			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}

			if user.ID == 0 {
				t.Fatal("expected authenticated user id to be populated")
			}
		})
	}
}

func TestServiceBootstrapAdmin(t *testing.T) {
	t.Parallel()

	t.Run("creates admin when none exists", func(t *testing.T) {
		t.Parallel()

		repository := newStubUserRepository()
		service := users.NewService(repository, stubPasswordManager{})

		admin, created, err := service.BootstrapAdmin(context.Background(), users.CreateInput{
			Username:  "bootstrap-admin",
			FirstName: "Bootstrap",
			LastName:  "Admin",
			Password:  "super-secure-admin",
		})
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if !created {
			t.Fatal("expected bootstrap flow to create an admin")
		}

		if admin.Role != users.RoleAdmin {
			t.Fatalf("expected admin role, got %q", admin.Role)
		}
	})

	t.Run("does not create a second admin", func(t *testing.T) {
		t.Parallel()

		repository := newStubUserRepository()
		_, _ = repository.Create(context.Background(), users.User{
			Username:     "existing-admin",
			FirstName:    "Existing",
			LastName:     "Admin",
			PasswordHash: "hash:super-secure-admin",
			Role:         users.RoleAdmin,
		})

		service := users.NewService(repository, stubPasswordManager{})
		admin, created, err := service.BootstrapAdmin(context.Background(), users.CreateInput{
			Username:  "ignored-admin",
			FirstName: "Ignored",
			LastName:  "Admin",
			Password:  "super-secure-admin",
		})
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if created {
			t.Fatal("expected bootstrap flow to skip creation when an admin already exists")
		}

		if admin.Username != "existing-admin" {
			t.Fatalf("expected existing admin to be returned, got %q", admin.Username)
		}
	})
}
