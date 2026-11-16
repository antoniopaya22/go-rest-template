package auth

import (
	"crypto/ed25519"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/antoniopaya22/go-rest-template/internal/config"
	"github.com/antoniopaya22/go-rest-template/internal/platform/errs"
)

// claims is the payload of an access token. The role is informational: the
// middleware always reloads the caller from the user directory.
type claims struct {
	Role Role `json:"role"`
	jwt.RegisteredClaims
}

// Signer issues and verifies JWT access tokens.
type Signer struct {
	method    jwt.SigningMethod
	signKey   any
	verifyKey any
	keyID     string
	issuer    string
	ttl       time.Duration
	jwks      []byte
}

// NewSigner creates a signer for the configured algorithm.
func NewSigner(cfg config.Auth) (*Signer, error) {
	signer := &Signer{issuer: cfg.Issuer, ttl: cfg.AccessTokenTTL}

	switch cfg.Algorithm {
	case config.AlgorithmHS256:
		if cfg.Secret == "" {
			return nil, errors.New("AUTH_SECRET is required for HS256")
		}
		signer.method = jwt.SigningMethodHS256
		signer.signKey = []byte(cfg.Secret)
		signer.verifyKey = []byte(cfg.Secret)
	case config.AlgorithmEdDSA:
		privateKey, err := loadEd25519Key(cfg.PrivateKeyFile)
		if err != nil {
			return nil, err
		}
		publicKey, _ := privateKey.Public().(ed25519.PublicKey)
		thumbprint := sha256.Sum256(publicKey)

		signer.method = jwt.SigningMethodEdDSA
		signer.signKey = privateKey
		signer.verifyKey = publicKey
		signer.keyID = base64.RawURLEncoding.EncodeToString(thumbprint[:8])
		signer.jwks, err = json.Marshal(map[string]any{"keys": []map[string]string{{
			"kty": "OKP",
			"crv": "Ed25519",
			"use": "sig",
			"alg": "EdDSA",
			"kid": signer.keyID,
			"x":   base64.RawURLEncoding.EncodeToString(publicKey),
		}}})
		if err != nil {
			return nil, fmt.Errorf("encode JWKS: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported signing algorithm %q", cfg.Algorithm)
	}

	return signer, nil
}

func loadEd25519Key(path string) (ed25519.PrivateKey, error) {
	raw, err := os.ReadFile(path) //nolint:gosec // operator-provided path.
	if err != nil {
		return nil, fmt.Errorf("read AUTH_PRIVATE_KEY_FILE: %w", err)
	}

	block, _ := pem.Decode(raw)
	if block == nil {
		return nil, errors.New("AUTH_PRIVATE_KEY_FILE does not contain a PEM block")
	}

	parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse AUTH_PRIVATE_KEY_FILE: %w", err)
	}

	key, ok := parsed.(ed25519.PrivateKey)
	if !ok {
		return nil, errors.New("AUTH_PRIVATE_KEY_FILE is not an Ed25519 key")
	}
	return key, nil
}

// TTL returns the lifetime of issued access tokens.
func (s *Signer) TTL() time.Duration { return s.ttl }

// JWKS returns the public key set, or false for symmetric algorithms.
func (s *Signer) JWKS() ([]byte, bool) { return s.jwks, s.jwks != nil }

// Issue signs an access token for identity.
func (s *Signer) Issue(identity Identity, now time.Time) (string, error) {
	token := jwt.NewWithClaims(s.method, claims{
		Role: identity.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    s.issuer,
			Subject:   strconv.FormatInt(identity.UserID, 10),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.ttl)),
		},
	})
	if s.keyID != "" {
		token.Header["kid"] = s.keyID
	}

	signed, err := token.SignedString(s.signKey)
	if err != nil {
		return "", fmt.Errorf("sign access token: %w", err)
	}
	return signed, nil
}

// Verify checks the signature, issuer and expiry of an access token and
// returns the user it was issued to.
func (s *Signer) Verify(raw string) (int64, error) {
	var parsed claims
	_, err := jwt.ParseWithClaims(raw, &parsed,
		func(*jwt.Token) (any, error) { return s.verifyKey, nil },
		jwt.WithValidMethods([]string{s.method.Alg()}),
		jwt.WithIssuer(s.issuer),
		jwt.WithExpirationRequired(),
	)
	if err != nil {
		return 0, errs.Wrap(errs.CodeUnauthorized, "invalid or expired token", err)
	}

	userID, err := strconv.ParseInt(parsed.Subject, 10, 64)
	if err != nil || userID <= 0 {
		return 0, errs.Unauthorized("invalid or expired token")
	}
	return userID, nil
}
