package auth

import (
	"crypto/rand"
	"fmt"

	"golang.org/x/crypto/bcrypt"

	"github.com/antoniopaya22/go-rest-template/v3/internal/platform/errs"
)

// Password length limits. bcrypt ignores everything after 72 bytes, so longer
// passwords are rejected instead of being silently truncated.
const (
	MinPasswordLength = 8
	MaxPasswordBytes  = 72
)

// Hasher hashes and verifies passwords with bcrypt.
type Hasher struct {
	cost  int
	dummy []byte
}

// NewHasher creates a hasher with the given bcrypt cost.
func NewHasher(cost int) (*Hasher, error) {
	if cost < bcrypt.MinCost || cost > bcrypt.MaxCost {
		return nil, fmt.Errorf("bcrypt cost must be between %d and %d", bcrypt.MinCost, bcrypt.MaxCost)
	}

	dummy, err := bcrypt.GenerateFromPassword([]byte(rand.Text()), cost)
	if err != nil {
		return nil, fmt.Errorf("generate dummy hash: %w", err)
	}
	return &Hasher{cost: cost, dummy: dummy}, nil
}

// ValidatePassword checks a new password against the length policy.
func ValidatePassword(password string) error {
	if len([]rune(password)) < MinPasswordLength {
		return errs.InvalidField("password", fmt.Sprintf("must be at least %d characters", MinPasswordLength))
	}
	if len(password) > MaxPasswordBytes {
		return errs.InvalidField("password", fmt.Sprintf("must be at most %d bytes", MaxPasswordBytes))
	}
	return nil
}

// Hash returns the bcrypt hash of a password.
func (h *Hasher) Hash(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), h.cost)
	if err != nil {
		return "", fmt.Errorf("hash password: %w", err)
	}
	return string(hash), nil
}

// Matches reports whether password corresponds to hash.
func (h *Hasher) Matches(hash string, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

// Burn spends the time of one verification without a real hash. Call it when
// the username does not exist so that response time does not reveal it.
func (h *Hasher) Burn(password string) {
	_ = bcrypt.CompareHashAndPassword(h.dummy, []byte(password))
}
