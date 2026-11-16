package app_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/antoniopaya22/go-rest-template/internal/app/apptest"
)

func pathf(format string, args ...any) string { return fmt.Sprintf(format, args...) }

type userJSON struct {
	ID        int64  `json:"id"`
	Username  string `json:"username"`
	FirstName string `json:"first_name"`
	Role      string `json:"role"`
}

type userPage struct {
	Data       []userJSON `json:"data"`
	NextCursor string     `json:"next_cursor"`
}

func TestUsers(t *testing.T) {
	t.Parallel()

	apptest.Each(t, func(t *testing.T, env *apptest.Env) {
		admin := env.AdminToken()
		aliceID, alice := env.User("alice")
		bobID, _ := env.User("bob")
		alicePath := pathf("/api/v1/users/%d", aliceID)
		bobPath := pathf("/api/v1/users/%d", bobID)

		t.Run("registration", func(t *testing.T) {
			body := map[string]string{"username": "Alice", "first_name": "A", "last_name": "B", "password": "user-password-123"}
			if code := env.Do(http.StatusConflict, apptest.Request{Method: http.MethodPost, Path: "/api/v1/users", Body: body}).Code(t); code != "conflict" {
				t.Errorf("duplicate username code = %q, want conflict", code)
			}

			body["username"], body["role"] = "mallory", "admin"
			env.Do(http.StatusForbidden, apptest.Request{Method: http.MethodPost, Path: "/api/v1/users", Body: body})

			body["role"], body["password"] = "user", "short"
			env.Do(http.StatusBadRequest, apptest.Request{Method: http.MethodPost, Path: "/api/v1/users", Body: body})
		})

		t.Run("access control", func(*testing.T) {
			env.Do(http.StatusOK, apptest.Request{Method: http.MethodGet, Path: alicePath, Token: alice})
			env.Do(http.StatusForbidden, apptest.Request{Method: http.MethodGet, Path: bobPath, Token: alice})
			env.Do(http.StatusForbidden, apptest.Request{Method: http.MethodGet, Path: "/api/v1/users", Token: alice})
			env.Do(http.StatusOK, apptest.Request{Method: http.MethodGet, Path: bobPath, Token: admin})
			env.Do(http.StatusNotFound, apptest.Request{Method: http.MethodGet, Path: "/api/v1/users/999999", Token: admin})
			env.Do(http.StatusBadRequest, apptest.Request{Method: http.MethodGet, Path: "/api/v1/users/0", Token: admin})

			env.Do(http.StatusForbidden, apptest.Request{
				Method: http.MethodPatch, Path: alicePath, Token: alice, Body: map[string]string{"role": "admin"},
			})
		})

		t.Run("update with optimistic concurrency", func(t *testing.T) {
			current := env.Do(http.StatusOK, apptest.Request{Method: http.MethodGet, Path: alicePath, Token: alice})
			etag := current.Header.Get("ETag")
			if etag == "" {
				t.Fatal("GET did not return an ETag")
			}

			var updated userJSON
			response := env.Do(http.StatusOK, apptest.Request{
				Method: http.MethodPatch, Path: alicePath, Token: alice,
				Body: map[string]string{"first_name": "Alicia"}, Header: map[string]string{"If-Match": etag},
			})
			response.Decode(t, &updated)
			if updated.FirstName != "Alicia" {
				t.Errorf("first_name = %q, want Alicia", updated.FirstName)
			}
			if response.Header.Get("ETag") == etag {
				t.Error("update did not change the ETag")
			}

			stale := env.Do(http.StatusPreconditionFailed, apptest.Request{
				Method: http.MethodPatch, Path: alicePath, Token: alice,
				Body: map[string]string{"first_name": "Stale"}, Header: map[string]string{"If-Match": etag},
			})
			if code := stale.Code(t); code != "precondition_failed" {
				t.Errorf("code = %q, want precondition_failed", code)
			}

			env.Do(http.StatusBadRequest, apptest.Request{Method: http.MethodPatch, Path: alicePath, Token: alice, Body: map[string]string{}})
		})

		t.Run("password change", func(*testing.T) {
			env.Do(http.StatusOK, apptest.Request{
				Method: http.MethodPatch, Path: bobPath, Token: admin, Body: map[string]string{"password": "a-brand-new-password"},
			})
			env.Login("bob", "a-brand-new-password")
		})

		t.Run("listing and pagination", func(t *testing.T) {
			var first userPage
			env.Do(http.StatusOK, apptest.Request{Method: http.MethodGet, Path: "/api/v1/users?limit=2", Token: admin}).Decode(t, &first)
			if len(first.Data) != 2 || first.NextCursor == "" {
				t.Fatalf("first page = %+v, want 2 users and a cursor", first)
			}

			var second userPage
			env.Do(http.StatusOK, apptest.Request{Method: http.MethodGet, Path: "/api/v1/users?limit=2&cursor=" + first.NextCursor, Token: admin}).Decode(t, &second)
			if len(second.Data) != 1 || second.NextCursor != "" {
				t.Fatalf("second page = %+v, want the last user and no cursor", second)
			}

			var filtered userPage
			env.Do(http.StatusOK, apptest.Request{Method: http.MethodGet, Path: "/api/v1/users?username=ALI&role=user", Token: admin}).Decode(t, &filtered)
			if len(filtered.Data) != 1 || filtered.Data[0].Username != "alice" {
				t.Fatalf("filtered = %+v, want only alice", filtered)
			}

			for _, query := range []string{"limit=0", "limit=101", "limit=x", "cursor=%25%25", "role=root"} {
				env.Do(http.StatusBadRequest, apptest.Request{Method: http.MethodGet, Path: "/api/v1/users?" + query, Token: admin})
			}
		})

		t.Run("promotion and deletion", func(*testing.T) {
			env.Do(http.StatusOK, apptest.Request{Method: http.MethodPatch, Path: alicePath, Token: admin, Body: map[string]string{"role": "admin"}})
			// The role is read on every request, so the existing token is promoted too.
			env.Do(http.StatusOK, apptest.Request{Method: http.MethodGet, Path: "/api/v1/users", Token: alice})

			env.Do(http.StatusNoContent, apptest.Request{Method: http.MethodDelete, Path: bobPath, Token: admin})
			env.Do(http.StatusNotFound, apptest.Request{Method: http.MethodGet, Path: bobPath, Token: admin})
		})
	})
}

func TestRegistrationDisabled(t *testing.T) {
	t.Parallel()

	env := apptest.New(t, map[string]string{"AUTH_REGISTRATION_ENABLED": "false"})
	body := map[string]string{"username": "carol", "first_name": "C", "last_name": "D", "password": "user-password-123", "role": "admin"}

	env.Do(http.StatusUnauthorized, apptest.Request{Method: http.MethodPost, Path: "/api/v1/users", Body: body})

	var created userJSON
	env.Do(http.StatusCreated, apptest.Request{Method: http.MethodPost, Path: "/api/v1/users", Body: body, Token: env.AdminToken()}).Decode(t, &created)
	if created.Role != "admin" {
		t.Errorf("role = %q, want admin", created.Role)
	}
}
