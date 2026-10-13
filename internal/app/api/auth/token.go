package auth

import (
	"fmt"
	"strings"
	"time"

	"github.com/antoniopaya22/go-rest-template/internal/app/api/domain/users"
	"github.com/antoniopaya22/go-rest-template/internal/pkg/errs"
	"github.com/golang-jwt/jwt/v5"
)

// Claims represents the JWT claims used by the API.
type Claims struct {
	UserID   int64      `json:"user_id"`
	Username string     `json:"username"`
	Role     users.Role `json:"role"`
	jwt.RegisteredClaims
}

// Token contains the serialized access token metadata.
type Token struct {
	AccessToken string    `json:"access_token"`
	TokenType   string    `json:"token_type"`
	ExpiresAt   time.Time `json:"expires_at"`
}

// TokenManager issues and validates JWT access tokens.
type TokenManager struct {
	secret []byte
	issuer string
	ttl    time.Duration
}

// NewTokenManager creates a new token manager.
func NewTokenManager(secret string, issuer string, ttl time.Duration) (*TokenManager, error) {
	if strings.TrimSpace(secret) == "" {
		return nil, errs.New(errs.CodeInvalidArgument, "AUTH_SECRET is required", nil)
	}

	if strings.TrimSpace(issuer) == "" {
		return nil, errs.New(errs.CodeInvalidArgument, "AUTH_ISSUER is required", nil)
	}

	if ttl <= 0 {
		return nil, errs.New(errs.CodeInvalidArgument, "AUTH_TOKEN_TTL must be greater than zero", nil)
	}

	return &TokenManager{
		secret: []byte(secret),
		issuer: issuer,
		ttl:    ttl,
	}, nil
}

// Issue creates a signed access token for a user.
func (m *TokenManager) Issue(userID int64, username string, role users.Role) (Token, error) {
	now := time.Now().UTC()
	expiresAt := now.Add(m.ttl)

	claims := Claims{
		UserID:   userID,
		Username: username,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    m.issuer,
			Subject:   fmt.Sprintf("%d", userID),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	}

	encoded, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(m.secret)
	if err != nil {
		return Token{}, fmt.Errorf("sign access token: %w", err)
	}

	return Token{
		AccessToken: encoded,
		TokenType:   "Bearer",
		ExpiresAt:   expiresAt,
	}, nil
}

// Parse validates and decodes an access token.
func (m *TokenManager) Parse(raw string) (*Claims, error) {
	claims := &Claims{}

	token, err := jwt.ParseWithClaims(raw, claims, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errs.New(errs.CodeUnauthorized, "unexpected signing method", nil)
		}

		return m.secret, nil
	}, jwt.WithIssuer(m.issuer), jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	if err != nil {
		return nil, errs.New(errs.CodeUnauthorized, "invalid token", err)
	}

	if !token.Valid {
		return nil, errs.New(errs.CodeUnauthorized, "invalid token", nil)
	}

	return claims, nil
}
