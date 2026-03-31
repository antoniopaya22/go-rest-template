package middleware_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/antonioalfa22/go-rest-template/internal/app/api/auth"
	"github.com/antonioalfa22/go-rest-template/internal/app/api/domain/users"
	"github.com/antonioalfa22/go-rest-template/internal/app/api/httpapi/middleware"
)

func TestAuthenticatorRequire(t *testing.T) {
	t.Parallel()

	tokenManager, err := auth.NewTokenManager("test-secret", "go-rest-template", 0+24*60*60*1e9)
	if err != nil {
		t.Fatalf("create token manager: %v", err)
	}

	token, err := tokenManager.Issue(7, "anton", users.RoleAdmin)
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}

	tests := []struct {
		name       string
		header     string
		wantStatus int
	}{
		{name: "valid bearer token", header: "Bearer " + token.AccessToken, wantStatus: http.StatusNoContent},
		{name: "missing bearer token", header: "", wantStatus: http.StatusUnauthorized},
		{name: "invalid token", header: "Bearer invalid-token", wantStatus: http.StatusUnauthorized},
	}

	for _, testCase := range tests {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			authenticator := middleware.NewAuthenticator(tokenManager)
			handler := authenticator.Require(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				principal, ok := middleware.PrincipalFromContext(r.Context())
				if !ok {
					t.Fatal("expected principal in request context")
				}

				if principal.UserID != 7 || principal.Role != users.RoleAdmin {
					t.Fatalf("unexpected principal: %+v", principal)
				}

				w.WriteHeader(http.StatusNoContent)
			}))

			request := httptest.NewRequest(http.MethodGet, "/protected", nil)
			if testCase.header != "" {
				request.Header.Set("Authorization", testCase.header)
			}

			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, request)

			if recorder.Code != testCase.wantStatus {
				t.Fatalf("expected status %d, got %d", testCase.wantStatus, recorder.Code)
			}

			if recorder.Code == http.StatusUnauthorized {
				var payload map[string]string
				if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
					t.Fatalf("decode error response: %v", err)
				}

				if payload["code"] != "unauthorized" {
					t.Fatalf("expected unauthorized code, got %q", payload["code"])
				}
			}
		})
	}
}
