package users

import (
	"context"
	"strings"
	"time"

	"github.com/antoniopaya22/go-rest-template/v3/internal/platform/database"
	"github.com/antoniopaya22/go-rest-template/v3/internal/platform/errs"
	"github.com/antoniopaya22/go-rest-template/v3/internal/platform/httpx"
)

const columns = `id, username, first_name, last_name, password_hash, role, version, created_at, updated_at`

// Repository stores users in the relational database.
type Repository struct {
	db *database.DB
}

// NewRepository creates a user repository.
func NewRepository(db *database.DB) *Repository {
	return &Repository{db: db}
}

// List returns up to page.Limit+1 users after the page cursor, ordered by id.
func (r *Repository) List(ctx context.Context, filter Filter, page httpx.PageRequest) ([]User, error) {
	query := `SELECT ` + columns + ` FROM users WHERE id > ?`
	args := []any{page.After}

	if filter.Username != "" {
		query += ` AND LOWER(username) LIKE ?`
		args = append(args, "%"+strings.ToLower(filter.Username)+"%")
	}
	if filter.Role != "" {
		query += ` AND role = ?`
		args = append(args, string(filter.Role))
	}

	query += ` ORDER BY id LIMIT ?`
	args = append(args, page.Limit+1)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	list := make([]User, 0, page.Limit)
	for rows.Next() {
		user, err := scan(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, user)
	}
	return list, rows.Err()
}

// GetByID returns a user by identifier.
func (r *Repository) GetByID(ctx context.Context, id int64) (User, error) {
	user, err := scan(r.db.QueryRow(ctx, `SELECT `+columns+` FROM users WHERE id = ?`, id))
	return user, r.db.Translate(err, "user")
}

// GetByUsername returns a user by username.
func (r *Repository) GetByUsername(ctx context.Context, username string) (User, error) {
	user, err := scan(r.db.QueryRow(ctx, `SELECT `+columns+` FROM users WHERE username = ?`, username))
	return user, r.db.Translate(err, "user")
}

// Create inserts a user and returns it with its generated fields.
func (r *Repository) Create(ctx context.Context, user User) (User, error) {
	now := timestamp()
	created, err := scan(r.db.QueryRow(ctx,
		`INSERT INTO users (username, first_name, last_name, password_hash, role, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)
		 RETURNING `+columns,
		user.Username, user.FirstName, user.LastName, user.PasswordHash, string(user.Role), now, now))
	return created, r.db.Translate(err, "user")
}

// Update stores a modified user. The write only succeeds when the row still
// has the version that was read, so concurrent updates cannot overwrite each other.
func (r *Repository) Update(ctx context.Context, user User) (User, error) {
	updated, err := scan(r.db.QueryRow(ctx,
		`UPDATE users
		 SET first_name = ?, last_name = ?, password_hash = ?, role = ?, version = version + 1, updated_at = ?
		 WHERE id = ? AND version = ?
		 RETURNING `+columns,
		user.FirstName, user.LastName, user.PasswordHash, string(user.Role), timestamp(), user.ID, user.Version))
	if err = r.db.Translate(err, "user"); errs.Is(err, errs.CodeNotFound) {
		return User{}, errs.Wrap(errs.CodeConflict, "user was modified by another request", err)
	}
	return updated, err
}

// Delete removes a user.
func (r *Repository) Delete(ctx context.Context, id int64) error {
	affected, err := r.db.Exec(ctx, `DELETE FROM users WHERE id = ?`, id)
	if err != nil {
		return err
	}
	if affected == 0 {
		return errs.NotFound("user not found")
	}
	return nil
}

func scan(row interface{ Scan(dest ...any) error }) (User, error) {
	var user User
	err := row.Scan(&user.ID, &user.Username, &user.FirstName, &user.LastName, &user.PasswordHash,
		&user.Role, &user.Version, &user.CreatedAt, &user.UpdatedAt)
	// Drivers return times in different locations; the API always speaks UTC.
	user.CreatedAt, user.UpdatedAt = user.CreatedAt.UTC(), user.UpdatedAt.UTC()
	return user, err
}

// timestamp returns the current time at the precision the database keeps.
func timestamp() time.Time {
	return time.Now().UTC().Truncate(time.Microsecond)
}
