package tasks_test

import (
	"context"
	"testing"
	"time"

	"github.com/antonioalfa22/go-rest-template/internal/app/api/domain/tasks"
	"github.com/antonioalfa22/go-rest-template/internal/app/api/domain/users"
	"github.com/antonioalfa22/go-rest-template/internal/pkg/errs"
)

type stubTaskRepository struct {
	nextID int64
	items  map[int64]tasks.Task
}

func newStubTaskRepository() *stubTaskRepository {
	return &stubTaskRepository{nextID: 1, items: make(map[int64]tasks.Task)}
}

func (r *stubTaskRepository) List(_ context.Context, filter tasks.Filter) ([]tasks.Task, error) {
	list := make([]tasks.Task, 0, len(r.items))
	for _, task := range r.items {
		if filter.OwnerID != nil && task.UserID != *filter.OwnerID {
			continue
		}

		if filter.Status != "" && task.Status != filter.Status {
			continue
		}

		list = append(list, task)
	}

	return list, nil
}

func (r *stubTaskRepository) GetByID(_ context.Context, id int64) (tasks.Task, error) {
	task, ok := r.items[id]
	if !ok {
		return tasks.Task{}, errs.New(errs.CodeNotFound, "task not found", nil)
	}

	return task, nil
}

func (r *stubTaskRepository) Create(_ context.Context, task tasks.Task) (tasks.Task, error) {
	task.ID = r.nextID
	r.nextID++
	task.CreatedAt = time.Now().UTC()
	task.UpdatedAt = task.CreatedAt
	r.items[task.ID] = task
	return task, nil
}

func (r *stubTaskRepository) Update(_ context.Context, task tasks.Task) (tasks.Task, error) {
	if _, ok := r.items[task.ID]; !ok {
		return tasks.Task{}, errs.New(errs.CodeNotFound, "task not found", nil)
	}

	task.UpdatedAt = time.Now().UTC()
	r.items[task.ID] = task
	return task, nil
}

func (r *stubTaskRepository) Delete(_ context.Context, id int64) error {
	if _, ok := r.items[id]; !ok {
		return errs.New(errs.CodeNotFound, "task not found", nil)
	}

	delete(r.items, id)
	return nil
}

type stubUserFinder struct {
	items map[int64]users.User
}

func (f stubUserFinder) GetByID(_ context.Context, id int64) (users.User, error) {
	user, ok := f.items[id]
	if !ok {
		return users.User{}, errs.New(errs.CodeNotFound, "user not found", nil)
	}

	return user, nil
}

func TestServiceCreate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    tasks.CreateInput
		finder   stubUserFinder
		wantCode errs.Code
		want     tasks.Status
	}{
		{
			name: "creates task with default status",
			input: tasks.CreateInput{
				Title:       "Upgrade template",
				Description: "Replace the legacy API starter",
				UserID:      1,
			},
			finder: stubUserFinder{items: map[int64]users.User{1: {ID: 1, Username: "anton"}}},
			want:   tasks.StatusPending,
		},
		{
			name: "rejects unknown owner",
			input: tasks.CreateInput{
				Title:       "Upgrade template",
				Description: "Replace the legacy API starter",
				UserID:      99,
			},
			finder:   stubUserFinder{items: map[int64]users.User{}},
			wantCode: errs.CodeNotFound,
		},
		{
			name: "rejects invalid status",
			input: tasks.CreateInput{
				Title:       "Upgrade template",
				Description: "Replace the legacy API starter",
				Status:      tasks.Status("blocked"),
				UserID:      1,
			},
			finder:   stubUserFinder{items: map[int64]users.User{1: {ID: 1, Username: "anton"}}},
			wantCode: errs.CodeInvalidArgument,
		},
	}

	for _, testCase := range tests {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			service := tasks.NewService(newStubTaskRepository(), testCase.finder)
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

			if created.Status != testCase.want {
				t.Fatalf("expected status %q, got %q", testCase.want, created.Status)
			}
		})
	}
}

func TestServiceUpdateRejectsBlankTitle(t *testing.T) {
	t.Parallel()

	repository := newStubTaskRepository()
	created, err := repository.Create(context.Background(), tasks.Task{
		Title:       "Upgrade template",
		Description: "Replace the legacy API starter",
		Status:      tasks.StatusPending,
		UserID:      1,
	})
	if err != nil {
		t.Fatalf("seed task: %v", err)
	}

	service := tasks.NewService(repository, stubUserFinder{items: map[int64]users.User{1: {ID: 1}}})
	blank := "   "

	_, err = service.Update(context.Background(), created.ID, tasks.UpdateInput{Title: &blank})
	if errs.CodeOf(err) != errs.CodeInvalidArgument {
		t.Fatalf("expected error code %q, got %q", errs.CodeInvalidArgument, errs.CodeOf(err))
	}
}
