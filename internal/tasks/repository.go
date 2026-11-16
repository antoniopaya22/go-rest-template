package tasks

import (
	"context"
	"time"

	"github.com/antoniopaya22/go-rest-template/internal/platform/database"
	"github.com/antoniopaya22/go-rest-template/internal/platform/errs"
	"github.com/antoniopaya22/go-rest-template/internal/platform/httpx"
)

const columns = `id, title, description, status, user_id, version, created_at, updated_at`

// Repository stores tasks in the relational database.
type Repository struct {
	db *database.DB
}

// NewRepository creates a task repository.
func NewRepository(db *database.DB) *Repository {
	return &Repository{db: db}
}

// List returns up to page.Limit+1 tasks after the page cursor, ordered by id.
func (r *Repository) List(ctx context.Context, filter Filter, page httpx.PageRequest) ([]Task, error) {
	query := `SELECT ` + columns + ` FROM tasks WHERE id > ?`
	args := []any{page.After}

	if filter.UserID != nil {
		query += ` AND user_id = ?`
		args = append(args, *filter.UserID)
	}
	if filter.Status != "" {
		query += ` AND status = ?`
		args = append(args, string(filter.Status))
	}

	query += ` ORDER BY id LIMIT ?`
	args = append(args, page.Limit+1)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	list := make([]Task, 0, page.Limit)
	for rows.Next() {
		task, err := scan(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, task)
	}
	return list, rows.Err()
}

// GetByID returns a task by identifier.
func (r *Repository) GetByID(ctx context.Context, id int64) (Task, error) {
	task, err := scan(r.db.QueryRow(ctx, `SELECT `+columns+` FROM tasks WHERE id = ?`, id))
	return task, r.db.Translate(err, "task")
}

// Create inserts a task and returns it with its generated fields.
func (r *Repository) Create(ctx context.Context, task Task) (Task, error) {
	now := timestamp()
	created, err := scan(r.db.QueryRow(ctx,
		`INSERT INTO tasks (title, description, status, user_id, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?)
		 RETURNING `+columns,
		task.Title, task.Description, string(task.Status), task.UserID, now, now))
	return created, r.db.Translate(err, "task")
}

// Update stores a modified task. The write only succeeds when the row still
// has the version that was read, so concurrent updates cannot overwrite each other.
func (r *Repository) Update(ctx context.Context, task Task) (Task, error) {
	updated, err := scan(r.db.QueryRow(ctx,
		`UPDATE tasks
		 SET title = ?, description = ?, status = ?, version = version + 1, updated_at = ?
		 WHERE id = ? AND version = ?
		 RETURNING `+columns,
		task.Title, task.Description, string(task.Status), timestamp(), task.ID, task.Version))
	if err = r.db.Translate(err, "task"); errs.Is(err, errs.CodeNotFound) {
		return Task{}, errs.Wrap(errs.CodeConflict, "task was modified by another request", err)
	}
	return updated, err
}

// Delete removes a task.
func (r *Repository) Delete(ctx context.Context, id int64) error {
	affected, err := r.db.Exec(ctx, `DELETE FROM tasks WHERE id = ?`, id)
	if err != nil {
		return err
	}
	if affected == 0 {
		return errs.NotFound("task not found")
	}
	return nil
}

func scan(row interface{ Scan(dest ...any) error }) (Task, error) {
	var task Task
	err := row.Scan(&task.ID, &task.Title, &task.Description, &task.Status, &task.UserID,
		&task.Version, &task.CreatedAt, &task.UpdatedAt)
	// Drivers return times in different locations; the API always speaks UTC.
	task.CreatedAt, task.UpdatedAt = task.CreatedAt.UTC(), task.UpdatedAt.UTC()
	return task, err
}

// timestamp returns the current time at the precision the database keeps.
func timestamp() time.Time {
	return time.Now().UTC().Truncate(time.Microsecond)
}
