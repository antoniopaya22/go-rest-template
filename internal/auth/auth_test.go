package auth

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/antoniopaya22/go-rest-template/internal/config"
	"github.com/antoniopaya22/go-rest-template/internal/platform/errs"
)

func TestHasher(t *testing.T) {
	t.Parallel()

	hasher, err := NewHasher(4)
	if err != nil {
		t.Fatalf("NewHasher: %v", err)
	}

	hash, err := hasher.Hash(" spaces are significant ")
	if err != nil {
		t.Fatalf("Hash: %v", err)
	}
	if !hasher.Matches(hash, " spaces are significant ") {
		t.Error("the original password must match")
	}
	if hasher.Matches(hash, "spaces are significant") {
		t.Error("a trimmed password must not match")
	}
	hasher.Burn("anything") // must not panic

	if _, err := NewHasher(100); err == nil {
		t.Error("an out-of-range cost must be rejected")
	}
}

func TestValidatePassword(t *testing.T) {
	t.Parallel()

	tests := []struct {
		password string
		wantErr  bool
	}{
		{"short", true},
		{"12345678", false},
		{"contraseña", false},
		{strings.Repeat("a", 72), false},
		{strings.Repeat("a", 73), true},
	}
	for _, tt := range tests {
		if err := ValidatePassword(tt.password); (err != nil) != tt.wantErr {
			t.Errorf("ValidatePassword(%d bytes) = %v, wantErr %t", len(tt.password), err, tt.wantErr)
		}
	}
}

func hs256(t *testing.T, mutate func(*config.Auth)) *Signer {
	t.Helper()

	cfg := config.Auth{Algorithm: config.AlgorithmHS256, Secret: "test-secret", Issuer: "test", AccessTokenTTL: time.Minute}
	if mutate != nil {
		mutate(&cfg)
	}
	signer, err := NewSigner(cfg)
	if err != nil {
		t.Fatalf("NewSigner: %v", err)
	}
	return signer
}

func TestSignerHS256(t *testing.T) {
	t.Parallel()

	signer := hs256(t, nil)
	now := time.Now()

	token, err := signer.Issue(Identity{UserID: 42, Role: RoleAdmin}, now)
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}

	userID, err := signer.Verify(token)
	if err != nil || userID != 42 {
		t.Fatalf("Verify = %d, %v; want 42", userID, err)
	}
	if _, ok := signer.JWKS(); ok {
		t.Error("a symmetric signer must not publish keys")
	}

	expired, _ := signer.Issue(Identity{UserID: 42}, now.Add(-time.Hour))
	otherKey, _ := hs256(t, func(c *config.Auth) { c.Secret = "another-secret" }).Issue(Identity{UserID: 42}, now)
	otherIssuer, _ := hs256(t, func(c *config.Auth) { c.Issuer = "someone-else" }).Issue(Identity{UserID: 42}, now)
	// A token whose header claims "none" must never be accepted.
	unsigned := "eyJhbGciOiJub25lIiwidHlwIjoiSldUIn0." + strings.Split(token, ".")[1] + "."

	for name, raw := range map[string]string{
		"expired": expired, "wrong key": otherKey, "wrong issuer": otherIssuer,
		"unsigned": unsigned, "garbage": "not.a.token", "empty": "",
	} {
		if _, err := signer.Verify(raw); !errs.Is(err, errs.CodeUnauthorized) {
			t.Errorf("%s token: error = %v, want unauthorized", name, err)
		}
	}
}

func TestSignerEdDSA(t *testing.T) {
	t.Parallel()

	_, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	der, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		t.Fatalf("marshal key: %v", err)
	}
	path := filepath.Join(t.TempDir(), "key.pem")
	if err := os.WriteFile(path, pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der}), 0o600); err != nil {
		t.Fatalf("write key: %v", err)
	}

	cfg := config.Auth{Algorithm: config.AlgorithmEdDSA, PrivateKeyFile: path, Issuer: "test", AccessTokenTTL: time.Minute}
	signer, err := NewSigner(cfg)
	if err != nil {
		t.Fatalf("NewSigner: %v", err)
	}

	token, err := signer.Issue(Identity{UserID: 7, Role: RoleUser}, time.Now())
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}
	if userID, err := signer.Verify(token); err != nil || userID != 7 {
		t.Fatalf("Verify = %d, %v; want 7", userID, err)
	}

	// An HS256 token signed with the public key as the secret is the classic
	// algorithm confusion attack; it must be rejected.
	publicKey, _ := privateKey.Public().(ed25519.PublicKey)
	forged, _ := hs256(t, func(c *config.Auth) { c.Secret = string(publicKey) }).Issue(Identity{UserID: 7}, time.Now())
	if _, err := signer.Verify(forged); err == nil {
		t.Error("an HS256 token must not verify against an EdDSA signer")
	}

	raw, ok := signer.JWKS()
	if !ok {
		t.Fatal("an EdDSA signer must publish its key")
	}
	var set struct {
		Keys []map[string]string `json:"keys"`
	}
	if err := json.Unmarshal(raw, &set); err != nil || len(set.Keys) != 1 {
		t.Fatalf("JWKS = %s, %v", raw, err)
	}
	if key := set.Keys[0]; key["kty"] != "OKP" || key["crv"] != "Ed25519" || key["x"] == "" || key["kid"] == "" {
		t.Errorf("unexpected JWK: %v", key)
	}

	for name, file := range map[string]string{"missing file": filepath.Join(t.TempDir(), "nope.pem"), "not PEM": writeFile(t, "plain text")} {
		cfg.PrivateKeyFile = file
		if _, err := NewSigner(cfg); err == nil {
			t.Errorf("%s: NewSigner must fail", name)
		}
	}
}

func writeFile(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}
	return path
}

func TestIdentityAuthorize(t *testing.T) {
	t.Parallel()

	owner := Identity{UserID: 1, Role: RoleUser}
	if err := owner.Authorize(1); err != nil {
		t.Errorf("an owner must be authorized: %v", err)
	}
	if err := owner.Authorize(2); !errs.Is(err, errs.CodeForbidden) {
		t.Errorf("another user's resource: error = %v, want forbidden", err)
	}
	if err := (Identity{UserID: 9, Role: RoleAdmin}).Authorize(2); err != nil {
		t.Errorf("an administrator must be authorized: %v", err)
	}
}

type directory map[int64]Identity

func (d directory) Authenticate(context.Context, string, string) (Identity, error) {
	return Identity{}, errs.Unauthorized("invalid credentials")
}

func (d directory) Identity(_ context.Context, userID int64) (Identity, error) {
	identity, ok := d[userID]
	if !ok {
		return Identity{}, errs.NotFound("user not found")
	}
	return identity, nil
}

func TestMiddleware(t *testing.T) {
	t.Parallel()

	signer := hs256(t, nil)
	users := directory{1: {UserID: 1, Role: RoleUser}, 2: {UserID: 2, Role: RoleAdmin}}
	middleware := NewMiddleware(signer, users)

	token := func(userID int64) string {
		raw, err := signer.Issue(Identity{UserID: userID}, time.Now())
		if err != nil {
			t.Fatalf("Issue: %v", err)
		}
		return "Bearer " + raw
	}

	adminOnly := middleware.Authenticate(RequireRole(RoleAdmin)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(MustIdentity(r.Context()).Role))
	})))

	tests := []struct {
		name          string
		authorization string
		want          int
	}{
		{"no header", "", http.StatusUnauthorized},
		{"wrong scheme", "Basic abc", http.StatusUnauthorized},
		{"empty token", "Bearer ", http.StatusUnauthorized},
		{"invalid token", "Bearer nope", http.StatusUnauthorized},
		{"deleted user", token(99), http.StatusUnauthorized},
		{"wrong role", token(1), http.StatusForbidden},
		{"administrator", token(2), http.StatusOK},
		{"scheme is case-insensitive", strings.Replace(token(2), "Bearer", "bearer", 1), http.StatusOK},
	}

	for _, tt := range tests {
		request := httptest.NewRequest(http.MethodGet, "/", nil)
		if tt.authorization != "" {
			request.Header.Set("Authorization", tt.authorization)
		}
		recorder := httptest.NewRecorder()
		adminOnly.ServeHTTP(recorder, request)

		if recorder.Code != tt.want {
			t.Errorf("%s: status = %d, want %d", tt.name, recorder.Code, tt.want)
		}
	}
}
