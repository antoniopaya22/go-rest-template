package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/antoniopaya22/go-rest-template/v3/internal/platform/database"
	"github.com/antoniopaya22/go-rest-template/v3/internal/platform/errs"
)

// TokenPair is the response of a successful login or refresh.
type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"`
}

// Service issues, rotates and revokes tokens.
type Service struct {
	db         *database.DB
	users      UserDirectory
	signer     *Signer
	tokens     *refreshStore
	refreshTTL time.Duration
	now        func() time.Time
}

// NewService creates the authentication service.
func NewService(db *database.DB, users UserDirectory, signer *Signer, refreshTTL time.Duration) *Service {
	return &Service{
		db:         db,
		users:      users,
		signer:     signer,
		tokens:     &refreshStore{db: db},
		refreshTTL: refreshTTL,
		now:        func() time.Time { return time.Now().UTC() },
	}
}

// Login verifies credentials and starts a new token family.
func (s *Service) Login(ctx context.Context, username string, password string) (TokenPair, error) {
	identity, err := s.users.Authenticate(ctx, username, password)
	if err != nil {
		return TokenPair{}, err
	}

	now := s.now()
	if err := s.tokens.deleteExpired(ctx, identity.UserID, now); err != nil {
		return TokenPair{}, fmt.Errorf("delete expired refresh tokens: %w", err)
	}

	return s.issue(ctx, identity, rand.Text(), now)
}

// Refresh exchanges a refresh token for a new pair. Each refresh token works
// once: presenting it a second time means it leaked, so the whole family is
// revoked and the user has to log in again.
func (s *Service) Refresh(ctx context.Context, refreshToken string) (TokenPair, error) {
	var (
		pair   TokenPair
		reused bool
	)

	err := s.db.InTx(ctx, func(ctx context.Context) error {
		now := s.now()

		stored, err := s.tokens.findByHash(ctx, hashToken(refreshToken))
		if err != nil {
			if errs.Is(err, errs.CodeNotFound) {
				return errs.Unauthorized("invalid refresh token")
			}
			return err
		}

		if stored.RevokedAt != nil {
			reused = true
			return s.tokens.revokeFamily(ctx, stored.FamilyID, now)
		}

		if !now.Before(stored.ExpiresAt) {
			return errs.Unauthorized("refresh token expired")
		}

		identity, err := s.users.Identity(ctx, stored.UserID)
		if err != nil {
			return errs.Wrap(errs.CodeUnauthorized, "invalid refresh token", err)
		}

		if err := s.tokens.revoke(ctx, stored.ID, now); err != nil {
			return err
		}

		pair, err = s.issue(ctx, identity, stored.FamilyID, now)
		return err
	})
	if err != nil {
		return TokenPair{}, err
	}
	if reused {
		return TokenPair{}, errs.Unauthorized("refresh token already used")
	}
	return pair, nil
}

// Logout revokes the family of a refresh token. Unknown tokens are ignored so
// the endpoint does not reveal whether a token exists.
func (s *Service) Logout(ctx context.Context, refreshToken string) error {
	stored, err := s.tokens.findByHash(ctx, hashToken(refreshToken))
	if err != nil {
		if errs.Is(err, errs.CodeNotFound) {
			return nil
		}
		return err
	}
	return s.tokens.revokeFamily(ctx, stored.FamilyID, s.now())
}

func (s *Service) issue(ctx context.Context, identity Identity, familyID string, now time.Time) (TokenPair, error) {
	accessToken, err := s.signer.Issue(identity, now)
	if err != nil {
		return TokenPair{}, err
	}

	refreshToken := rand.Text() + rand.Text()
	if err := s.tokens.create(ctx, identity.UserID, familyID, hashToken(refreshToken), now, now.Add(s.refreshTTL)); err != nil {
		return TokenPair{}, fmt.Errorf("store refresh token: %w", err)
	}

	return TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    int64(s.signer.TTL().Seconds()),
	}, nil
}

// hashToken returns the value stored for a refresh token. The tokens are
// random and long, so a plain SHA-256 is enough and lookups stay indexed.
func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
