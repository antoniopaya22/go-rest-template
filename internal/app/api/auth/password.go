// Package auth contains authentication primitives.
package auth

import (
	"fmt"
	"strings"

	"github.com/antonioalfa22/go-rest-template/internal/pkg/errs"
	"golang.org/x/crypto/bcrypt"
)

// PasswordManager hashes and verifies user passwords.
type PasswordManager struct {
	cost int
}

// NewPasswordManager creates a password manager with the configured cost.
func NewPasswordManager(cost int) (*PasswordManager, error) {
	if cost < bcrypt.MinCost || cost > bcrypt.MaxCost {
		return nil, errs.New(errs.CodeInvalidArgument, fmt.Sprintf("bcrypt cost must be between %d and %d", bcrypt.MinCost, bcrypt.MaxCost), nil)
	}

	return &PasswordManager{cost: cost}, nil
}

// Hash generates a bcrypt hash for a plain-text password.
func (m *PasswordManager) Hash(password string) (string, error) {
	trimmed := strings.TrimSpace(password)
	if trimmed == "" {
		return "", errs.New(errs.CodeInvalidArgument, "password is required", nil)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(trimmed), m.cost)
	if err != nil {
		return "", fmt.Errorf("generate password hash: %w", err)
	}

	return string(hash), nil
}

// Compare checks a plain-text password against a stored hash.
func (m *PasswordManager) Compare(hash string, password string) error {
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); err != nil {
		return errs.New(errs.CodeUnauthorized, "invalid credentials", err)
	}

	return nil
}
