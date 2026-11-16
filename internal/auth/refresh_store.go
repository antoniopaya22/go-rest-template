package auth

import (
	"context"
	"database/sql"
	"time"

	"github.com/antoniopaya22/go-rest-template/internal/platform/database"
)

// refreshToken is a stored refresh token. Only the hash of the secret is kept.
type refreshToken struct {
	ID        int64
	UserID    int64
	FamilyID  string
	ExpiresAt time.Time
	RevokedAt *time.Time
}

// refreshStore persists refresh tokens.
type refreshStore struct {
	db *database.DB
}

func (s *refreshStore) create(ctx context.Context, userID int64, familyID string, hash string, now time.Time, expiresAt time.Time) error {
	_, err := s.db.Exec(ctx,
		`INSERT INTO refresh_tokens (user_id, family_id, token_hash, expires_at, created_at) VALUES (?, ?, ?, ?, ?)`,
		userID, familyID, hash, expiresAt, now)
	return err
}

func (s *refreshStore) findByHash(ctx context.Context, hash string) (refreshToken, error) {
	var (
		token     refreshToken
		revokedAt sql.NullTime
	)

	err := s.db.QueryRow(ctx,
		`SELECT id, user_id, family_id, expires_at, revoked_at FROM refresh_tokens WHERE token_hash = ?`, hash).
		Scan(&token.ID, &token.UserID, &token.FamilyID, &token.ExpiresAt, &revokedAt)
	if err != nil {
		return refreshToken{}, s.db.Translate(err, "refresh token")
	}

	if revokedAt.Valid {
		token.RevokedAt = &revokedAt.Time
	}
	return token, nil
}

func (s *refreshStore) revoke(ctx context.Context, id int64, now time.Time) error {
	_, err := s.db.Exec(ctx, `UPDATE refresh_tokens SET revoked_at = ? WHERE id = ? AND revoked_at IS NULL`, now, id)
	return err
}

func (s *refreshStore) revokeFamily(ctx context.Context, familyID string, now time.Time) error {
	_, err := s.db.Exec(ctx, `UPDATE refresh_tokens SET revoked_at = ? WHERE family_id = ? AND revoked_at IS NULL`, now, familyID)
	return err
}

func (s *refreshStore) deleteExpired(ctx context.Context, userID int64, now time.Time) error {
	_, err := s.db.Exec(ctx, `DELETE FROM refresh_tokens WHERE user_id = ? AND expires_at < ?`, userID, now)
	return err
}
