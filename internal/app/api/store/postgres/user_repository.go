package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/antoniopaya22/go-rest-template/internal/app/api/domain/users"
)

// UserRepository stores users in PostgreSQL.
type UserRepository struct {
	db *sql.DB
}

// NewUserRepository creates a PostgreSQL-backed user repository.
func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

// List returns users matching the provided filter.
func (r *UserRepository) List(ctx context.Context, filter users.Filter) ([]users.User, error) {
	query := `SELECT id, username, first_name, last_name, password_hash, role, created_at, updated_at FROM users WHERE 1 = 1`
	args := make([]any, 0, 2)
	position := 1

	if filter.Username != "" {
		query += fmt.Sprintf(` AND username ILIKE $%d`, position)
		args = append(args, "%"+strings.TrimSpace(filter.Username)+"%")
		position++
	}

	if filter.Role != "" {
		query += fmt.Sprintf(` AND role = $%d`, position)
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
	row := r.db.QueryRowContext(ctx, `SELECT id, username, first_name, last_name, password_hash, role, created_at, updated_at FROM users WHERE id = $1`, id)
	user, err := scanUser(row)
	if err != nil {
		return users.User{}, wrapQueryError(err, "user not found", "")
	}

	return user, nil
}

// GetByUsername returns a user by username.
func (r *UserRepository) GetByUsername(ctx context.Context, username string) (users.User, error) {
	row := r.db.QueryRowContext(ctx, `SELECT id, username, first_name, last_name, password_hash, role, created_at, updated_at FROM users WHERE username = $1`, strings.ToLower(strings.TrimSpace(username)))
	user, err := scanUser(row)
	if err != nil {
		return users.User{}, wrapQueryError(err, "user not found", "")
	}

	return user, nil
}

// Create inserts a new user.
func (r *UserRepository) Create(ctx context.Context, user users.User) (users.User, error) {
	row := r.db.QueryRowContext(
		ctx,
		`INSERT INTO users (username, first_name, last_name, password_hash, role, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 RETURNING id, username, first_name, last_name, password_hash, role, created_at, updated_at`,
		strings.ToLower(user.Username),
		user.FirstName,
		user.LastName,
		user.PasswordHash,
		user.Role,
		time.Now().UTC(),
		time.Now().UTC(),
	)

	created, err := scanUser(row)
	if err != nil {
		return users.User{}, wrapQueryError(err, "create user failed", "username already exists")
	}

	return created, nil
}

// Update stores an updated user state.
func (r *UserRepository) Update(ctx context.Context, user users.User) (users.User, error) {
	row := r.db.QueryRowContext(
		ctx,
		`UPDATE users
		 SET first_name = $1, last_name = $2, password_hash = $3, role = $4, updated_at = $5
		 WHERE id = $6
		 RETURNING id, username, first_name, last_name, password_hash, role, created_at, updated_at`,
		user.FirstName,
		user.LastName,
		user.PasswordHash,
		user.Role,
		time.Now().UTC(),
		user.ID,
	)

	updated, err := scanUser(row)
	if err != nil {
		return users.User{}, wrapQueryError(err, "user not found", "username already exists")
	}

	return updated, nil
}

// Delete removes a user.
func (r *UserRepository) Delete(ctx context.Context, id int64) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM users WHERE id = $1`, id)
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
		user users.User
		role string
	)

	if err := scanner.Scan(&user.ID, &user.Username, &user.FirstName, &user.LastName, &user.PasswordHash, &role, &user.CreatedAt, &user.UpdatedAt); err != nil {
		return users.User{}, err
	}

	user.Role = users.Role(role)
	return user, nil
}
