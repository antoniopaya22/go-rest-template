package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/antonioalfa22/go-rest-template/internal/app/api/domain/tasks"
)

// TaskRepository stores tasks in SQLite.
type TaskRepository struct {
	db *sql.DB
}

// NewTaskRepository creates a SQLite-backed task repository.
func NewTaskRepository(db *sql.DB) *TaskRepository {
	return &TaskRepository{db: db}
}

// List returns tasks matching the provided filter.
func (r *TaskRepository) List(ctx context.Context, filter tasks.Filter) ([]tasks.Task, error) {
	query := `SELECT id, title, description, status, user_id, created_at, updated_at FROM tasks WHERE 1 = 1`
	args := make([]any, 0, 2)

	if filter.OwnerID != nil {
		query += ` AND user_id = ?`
		args = append(args, *filter.OwnerID)
	}

	if filter.Status != "" {
		query += ` AND status = ?`
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
	row := r.db.QueryRowContext(ctx, `SELECT id, title, description, status, user_id, created_at, updated_at FROM tasks WHERE id = ?`, id)
	task, err := scanTask(row)
	if err != nil {
		return tasks.Task{}, wrapQueryError(err, "task not found", "")
	}

	return task, nil
}

// Create inserts a new task.
func (r *TaskRepository) Create(ctx context.Context, task tasks.Task) (tasks.Task, error) {
	now := time.Now().UTC()
	result, err := r.db.ExecContext(
		ctx,
		`INSERT INTO tasks (title, description, status, user_id, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		task.Title,
		task.Description,
		task.Status,
		task.UserID,
		formatTimestamp(now),
		formatTimestamp(now),
	)
	if err != nil {
		return tasks.Task{}, wrapQueryError(err, "create task failed", "")
	}

	id, err := result.LastInsertId()
	if err != nil {
		return tasks.Task{}, fmt.Errorf("read inserted task id: %w", err)
	}

	return r.GetByID(ctx, id)
}

// Update stores an updated task state.
func (r *TaskRepository) Update(ctx context.Context, task tasks.Task) (tasks.Task, error) {
	result, err := r.db.ExecContext(
		ctx,
		`UPDATE tasks SET title = ?, description = ?, status = ?, updated_at = ? WHERE id = ?`,
		task.Title,
		task.Description,
		task.Status,
		formatTimestamp(time.Now()),
		task.ID,
	)
	if err != nil {
		return tasks.Task{}, wrapQueryError(err, "update task failed", "")
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return tasks.Task{}, fmt.Errorf("read updated task rows affected: %w", err)
	}

	if err := rowsAffectedOrNotFound(affected, "task not found"); err != nil {
		return tasks.Task{}, err
	}

	return r.GetByID(ctx, task.ID)
}

// Delete removes a task.
func (r *TaskRepository) Delete(ctx context.Context, id int64) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM tasks WHERE id = ?`, id)
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
	var (
		task       tasks.Task
		status     string
		createdRaw string
		updatedRaw string
	)

	if err := scanner.Scan(&task.ID, &task.Title, &task.Description, &status, &task.UserID, &createdRaw, &updatedRaw); err != nil {
		return tasks.Task{}, err
	}

	createdAt, err := parseTimestamp(createdRaw)
	if err != nil {
		return tasks.Task{}, err
	}

	updatedAt, err := parseTimestamp(updatedRaw)
	if err != nil {
		return tasks.Task{}, err
	}

	task.Status = tasks.Status(status)
	task.CreatedAt = createdAt
	task.UpdatedAt = updatedAt

	return task, nil
}
