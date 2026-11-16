package tasks_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/antoniopaya22/go-rest-template/internal/app/apptest"
)

type taskJSON struct {
	ID     int64  `json:"id"`
	Title  string `json:"title"`
	Status string `json:"status"`
	UserID int64  `json:"user_id"`
}

type taskPage struct {
	Data       []taskJSON `json:"data"`
	NextCursor string     `json:"next_cursor"`
}

func TestTasksAPI(t *testing.T) {
	t.Parallel()

	apptest.Each(t, func(t *testing.T, env *apptest.Env) {
		admin := env.AdminToken()
		aliceID, alice := env.User("alice")
		_, bob := env.User("bob")

		create := func(token string, body map[string]any, want int) (taskJSON, apptest.Response) {
			var task taskJSON
			response := env.Do(want, apptest.Request{Method: http.MethodPost, Path: "/api/v1/tasks", Token: token, Body: body})
			if want == http.StatusCreated {
				response.Decode(t, &task)
			}
			return task, response
		}

		task, created := create(alice, map[string]any{"title": " Write docs ", "description": "All of them"}, http.StatusCreated)
		if task.Title != "Write docs" || task.Status != "pending" || task.UserID != aliceID {
			t.Fatalf("created task = %+v", task)
		}
		path := fmt.Sprintf("/api/v1/tasks/%d", task.ID)

		t.Run("validation", func(*testing.T) {
			create(alice, map[string]any{"title": "x"}, http.StatusBadRequest)
			create(alice, map[string]any{"title": "x", "description": "y", "status": "paused"}, http.StatusBadRequest)
			create(alice, map[string]any{"title": "   ", "description": "y"}, http.StatusBadRequest)
			create("", map[string]any{"title": "x", "description": "y"}, http.StatusUnauthorized)
		})

		t.Run("ownership", func(t *testing.T) {
			create(bob, map[string]any{"title": "x", "description": "y", "user_id": aliceID}, http.StatusForbidden)

			assigned, _ := create(admin, map[string]any{"title": "Assigned", "description": "By an admin", "user_id": aliceID}, http.StatusCreated)
			if assigned.UserID != aliceID {
				t.Errorf("owner = %d, want %d", assigned.UserID, aliceID)
			}

			_, missing := create(admin, map[string]any{"title": "x", "description": "y", "user_id": 999999}, http.StatusBadRequest)
			if code := missing.Code(t); code != "invalid_argument" {
				t.Errorf("code = %q, want invalid_argument", code)
			}

			// Another user's task looks like it does not exist.
			env.Do(http.StatusNotFound, apptest.Request{Method: http.MethodGet, Path: path, Token: bob})
			env.Do(http.StatusNotFound, apptest.Request{Method: http.MethodDelete, Path: path, Token: bob})
			env.Do(http.StatusOK, apptest.Request{Method: http.MethodGet, Path: path, Token: admin})
		})

		t.Run("update with optimistic concurrency", func(t *testing.T) {
			etag := created.Header.Get("ETag")

			var updated taskJSON
			env.Do(http.StatusOK, apptest.Request{
				Method: http.MethodPatch, Path: path, Token: alice,
				Body: map[string]string{"status": "in_progress"}, Header: map[string]string{"If-Match": etag},
			}).Decode(t, &updated)
			if updated.Status != "in_progress" {
				t.Errorf("status = %q, want in_progress", updated.Status)
			}

			env.Do(http.StatusPreconditionFailed, apptest.Request{
				Method: http.MethodPatch, Path: path, Token: alice,
				Body: map[string]string{"status": "done"}, Header: map[string]string{"If-Match": etag},
			})
			env.Do(http.StatusBadRequest, apptest.Request{
				Method: http.MethodPatch, Path: path, Token: alice, Body: map[string]string{"title": " "},
			})
		})

		t.Run("listing", func(t *testing.T) {
			create(bob, map[string]any{"title": "Bob's", "description": "task", "status": "done"}, http.StatusCreated)

			list := func(token string, query string) taskPage {
				var page taskPage
				env.Do(http.StatusOK, apptest.Request{Method: http.MethodGet, Path: "/api/v1/tasks?" + query, Token: token}).Decode(t, &page)
				return page
			}

			if page := list(alice, ""); len(page.Data) != 2 {
				t.Errorf("alice sees %d tasks, want her 2", len(page.Data))
			}
			// A regular user cannot widen the listing with user_id.
			if page := list(bob, fmt.Sprintf("user_id=%d", aliceID)); len(page.Data) != 1 {
				t.Errorf("bob sees %d tasks, want his 1", len(page.Data))
			}
			if page := list(admin, ""); len(page.Data) != 3 {
				t.Errorf("admin sees %d tasks, want 3", len(page.Data))
			}
			if page := list(admin, "status=done"); len(page.Data) != 1 {
				t.Errorf("status filter returned %d tasks, want 1", len(page.Data))
			}
			if page := list(admin, fmt.Sprintf("user_id=%d", aliceID)); len(page.Data) != 2 {
				t.Errorf("owner filter returned %d tasks, want 2", len(page.Data))
			}

			first := list(admin, "limit=2")
			if len(first.Data) != 2 || first.NextCursor == "" {
				t.Fatalf("first page = %+v, want 2 tasks and a cursor", first)
			}
			if second := list(admin, "limit=2&cursor="+first.NextCursor); len(second.Data) != 1 || second.NextCursor != "" {
				t.Fatalf("second page = %+v, want the last task and no cursor", second)
			}

			env.Do(http.StatusBadRequest, apptest.Request{Method: http.MethodGet, Path: "/api/v1/tasks?status=paused", Token: admin})
		})

		t.Run("deletion", func(*testing.T) {
			env.Do(http.StatusNoContent, apptest.Request{Method: http.MethodDelete, Path: path, Token: alice})
			env.Do(http.StatusNotFound, apptest.Request{Method: http.MethodGet, Path: path, Token: alice})
		})

		t.Run("deleting a user deletes their tasks", func(t *testing.T) {
			env.Do(http.StatusNoContent, apptest.Request{Method: http.MethodDelete, Path: fmt.Sprintf("/api/v1/users/%d", aliceID), Token: admin})

			var page taskPage
			env.Do(http.StatusOK, apptest.Request{Method: http.MethodGet, Path: fmt.Sprintf("/api/v1/tasks?user_id=%d", aliceID), Token: admin}).Decode(t, &page)
			if len(page.Data) != 0 {
				t.Errorf("%d tasks survived their owner", len(page.Data))
			}
		})
	})
}
