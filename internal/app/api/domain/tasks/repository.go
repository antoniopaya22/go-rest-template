package tasks

import "context"

// Repository defines persistent storage operations for tasks.
type Repository interface {
	List(ctx context.Context, filter Filter) ([]Task, error)
	GetByID(ctx context.Context, id int64) (Task, error)
	Create(ctx context.Context, task Task) (Task, error)
	Update(ctx context.Context, task Task) (Task, error)
	Delete(ctx context.Context, id int64) error
}
