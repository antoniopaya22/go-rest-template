package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/antonioalfa22/go-rest-template/internal/app/api/domain/tasks"
)

// TaskRepository stores tasks in PostgreSQL.
type TaskRepository struct {
	db *sql.DB
}

// NewTaskRepository creates a PostgreSQL-backed task repository.
func NewTaskRepository(db *sql.DB) *TaskRepository {
	return &TaskRepository{db: db}
}

// List returns tasks matching the provided filter.
func (r *TaskRepository) List(ctx context.Context, filter tasks.Filter) ([]tasks.Task, error) {
	query := `SELECT id, title, description, status, user_id, created_at, updated_at FROM tasks WHERE 1 = 1`
	args := make([]any, 0, 2)
	position := 1

	if filter.OwnerID != nil {
		query += fmt.Sprintf(` AND user_id = $%d`, position)
		args = append(args, *filter.OwnerID)
		position++
	}

	if filter.Status != "" {
		query += fmt.Sprintf(` AND status = $%d`, position)
		args = append(args, filter.Status)
	}

	query += ` ORDER BY id ASC`

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query tasks: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	list := make([]tasks.Task, 0)
	for rows.Next() {
		task, err := scanTask(rows)
		if err != nil {
			return nil, err
		}

		list = append(list, task)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate tasks: %w", err)
	}

	return list, nil
}

// GetByID returns a task by identifier.
func (r *TaskRepository) GetByID(ctx context.Context, id int64) (tasks.Task, error) {
	row := r.db.QueryRowContext(ctx, `SELECT id, title, description, status, user_id, created_at, updated_at FROM tasks WHERE id = $1`, id)
	task, err := scanTask(row)
	if err != nil {
		return tasks.Task{}, wrapQueryError(err, "task not found", "")
	}

	return task, nil
}

// Create inserts a new task.
func (r *TaskRepository) Create(ctx context.Context, task tasks.Task) (tasks.Task, error) {
	row := r.db.QueryRowContext(
		ctx,
		`INSERT INTO tasks (title, description, status, user_id, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id, title, description, status, user_id, created_at, updated_at`,
		task.Title,
		task.Description,
		task.Status,
		task.UserID,
		time.Now().UTC(),
		time.Now().UTC(),
	)

	created, err := scanTask(row)
	if err != nil {
		return tasks.Task{}, wrapQueryError(err, "create task failed", "")
	}

	return created, nil
}

// Update stores an updated task state.
func (r *TaskRepository) Update(ctx context.Context, task tasks.Task) (tasks.Task, error) {
	row := r.db.QueryRowContext(
		ctx,
		`UPDATE tasks
		 SET title = $1, description = $2, status = $3, updated_at = $4
		 WHERE id = $5
		 RETURNING id, title, description, status, user_id, created_at, updated_at`,
		task.Title,
		task.Description,
		task.Status,
		time.Now().UTC(),
		task.ID,
	)

	updated, err := scanTask(row)
	if err != nil {
		return tasks.Task{}, wrapQueryError(err, "task not found", "")
	}

	return updated, nil
}

// Delete removes a task.
func (r *TaskRepository) Delete(ctx context.Context, id int64) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM tasks WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete task: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read deleted task rows affected: %w", err)
	}

	return rowsAffectedOrNotFound(affected, "task not found")
}

func scanTask(scanner interface{ Scan(dest ...any) error }) (tasks.Task, error) {
	var task tasks.Task
	var status string

	if err := scanner.Scan(&task.ID, &task.Title, &task.Description, &status, &task.UserID, &task.CreatedAt, &task.UpdatedAt); err != nil {
		return tasks.Task{}, err
	}

	task.Status = tasks.Status(status)
	return task, nil
}
