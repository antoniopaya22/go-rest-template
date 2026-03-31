package users

import "context"

// Repository defines persistent storage operations for users.
type Repository interface {
	List(ctx context.Context, filter Filter) ([]User, error)
	GetByID(ctx context.Context, id int64) (User, error)
	GetByUsername(ctx context.Context, username string) (User, error)
	Create(ctx context.Context, user User) (User, error)
	Update(ctx context.Context, user User) (User, error)
	Delete(ctx context.Context, id int64) error
}
