package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/antoniopaya22/go-rest-template/internal/app/api/domain/users"
)

// UserRepository stores users in SQLite.
type UserRepository struct {
	db *sql.DB
}

// NewUserRepository creates a SQLite-backed user repository.
func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

// List returns users matching the provided filter.
func (r *UserRepository) List(ctx context.Context, filter users.Filter) ([]users.User, error) {
	query := `SELECT id, username, first_name, last_name, password_hash, role, created_at, updated_at FROM users WHERE 1 = 1`
	args := make([]any, 0, 2)

	if filter.Username != "" {
		query += ` AND username LIKE ?`
		args = append(args, "%"+strings.ToLower(filter.Username)+"%")
	}

	if filter.Role != "" {
		query += ` AND role = ?`
		args = append(args, filter.Role)
	}

	query += ` ORDER BY id ASC`

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query users: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	list := make([]users.User, 0)
	for rows.Next() {
		user, err := scanUser(rows)
		if err != nil {
			return nil, err
		}

		list = append(list, user)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate users: %w", err)
	}

	return list, nil
}

// GetByID returns a user by identifier.
func (r *UserRepository) GetByID(ctx context.Context, id int64) (users.User, error) {
	row := r.db.QueryRowContext(ctx, `SELECT id, username, first_name, last_name, password_hash, role, created_at, updated_at FROM users WHERE id = ?`, id)
	user, err := scanUser(row)
	if err != nil {
		return users.User{}, wrapQueryError(err, "user not found", "")
	}

	return user, nil
}

// GetByUsername returns a user by username.
func (r *UserRepository) GetByUsername(ctx context.Context, username string) (users.User, error) {
	row := r.db.QueryRowContext(ctx, `SELECT id, username, first_name, last_name, password_hash, role, created_at, updated_at FROM users WHERE username = ?`, strings.ToLower(username))
	user, err := scanUser(row)
	if err != nil {
		return users.User{}, wrapQueryError(err, "user not found", "")
	}

	return user, nil
}

// Create inserts a new user.
func (r *UserRepository) Create(ctx context.Context, user users.User) (users.User, error) {
	now := time.Now().UTC()
	result, err := r.db.ExecContext(
		ctx,
		`INSERT INTO users (username, first_name, last_name, password_hash, role, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		strings.ToLower(user.Username),
		user.FirstName,
		user.LastName,
		user.PasswordHash,
		user.Role,
		formatTimestamp(now),
		formatTimestamp(now),
	)
	if err != nil {
		return users.User{}, wrapQueryError(err, "create user failed", "username already exists")
	}

	id, err := result.LastInsertId()
	if err != nil {
		return users.User{}, fmt.Errorf("read inserted user id: %w", err)
	}

	return r.GetByID(ctx, id)
}

// Update stores an updated user state.
func (r *UserRepository) Update(ctx context.Context, user users.User) (users.User, error) {
	result, err := r.db.ExecContext(
		ctx,
		`UPDATE users SET first_name = ?, last_name = ?, password_hash = ?, role = ?, updated_at = ? WHERE id = ?`,
		user.FirstName,
		user.LastName,
		user.PasswordHash,
		user.Role,
		formatTimestamp(time.Now()),
		user.ID,
	)
	if err != nil {
		return users.User{}, wrapQueryError(err, "update user failed", "username already exists")
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return users.User{}, fmt.Errorf("read updated user rows affected: %w", err)
	}

	if err := rowsAffectedOrNotFound(affected, "user not found"); err != nil {
		return users.User{}, err
	}

	return r.GetByID(ctx, user.ID)
}

// Delete removes a user.
func (r *UserRepository) Delete(ctx context.Context, id int64) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM users WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete user: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read deleted user rows affected: %w", err)
	}

	return rowsAffectedOrNotFound(affected, "user not found")
}

func scanUser(scanner interface{ Scan(dest ...any) error }) (users.User, error) {
	var (
		user       users.User
		role       string
		createdRaw string
		updatedRaw string
	)

	if err := scanner.Scan(&user.ID, &user.Username, &user.FirstName, &user.LastName, &user.PasswordHash, &role, &createdRaw, &updatedRaw); err != nil {
		return users.User{}, err
	}

	createdAt, err := parseTimestamp(createdRaw)
	if err != nil {
		return users.User{}, err
	}

	updatedAt, err := parseTimestamp(updatedRaw)
	if err != nil {
		return users.User{}, err
	}

	user.Role = users.Role(role)
	user.CreatedAt = createdAt
	user.UpdatedAt = updatedAt

	return user, nil
}
