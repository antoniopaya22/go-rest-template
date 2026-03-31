package httpapi

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/antonioalfa22/go-rest-template/internal/app/api/auth"
	"github.com/antonioalfa22/go-rest-template/internal/app/api/domain/tasks"
	"github.com/antonioalfa22/go-rest-template/internal/app/api/domain/users"
	authmiddleware "github.com/antonioalfa22/go-rest-template/internal/app/api/httpapi/middleware"
	"github.com/antonioalfa22/go-rest-template/internal/pkg/errs"
	"github.com/go-chi/chi/v5"
)

type handler struct {
	logger         *slog.Logger
	users          *users.Service
	tasks          *tasks.Service
	tokens         *auth.TokenManager
	databaseHealth func(context.Context) error
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type loginResponse struct {
	Token auth.Token `json:"token"`
	User  users.User `json:"user"`
}

type createTaskRequest struct {
	Title       string       `json:"title"`
	Description string       `json:"description"`
	Status      tasks.Status `json:"status"`
	UserID      *int64       `json:"user_id,omitempty"`
}

type updateTaskRequest struct {
	Title       *string       `json:"title,omitempty"`
	Description *string       `json:"description,omitempty"`
	Status      *tasks.Status `json:"status,omitempty"`
}

func (h *handler) live(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *handler) ready(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	if h.databaseHealth == nil || h.databaseHealth(ctx) == nil {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
		return
	}

	writeJSON(w, http.StatusServiceUnavailable, errorResponse{Code: "unavailable", Error: "database not ready"})
}

func (h *handler) login(w http.ResponseWriter, r *http.Request) {
	var request loginRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, err)
		return
	}

	user, err := h.users.Authenticate(r.Context(), request.Username, request.Password)
	if err != nil {
		writeError(w, err)
		return
	}

	token, err := h.tokens.Issue(user.ID, user.Username, user.Role)
	if err != nil {
		h.logger.Error("issue token", "error", err)
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, loginResponse{Token: token, User: user})
}

func (h *handler) createUser(w http.ResponseWriter, r *http.Request) {
	var request users.CreateInput
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, err)
		return
	}

	if request.Role == users.RoleAdmin {
		writeError(w, errs.New(errs.CodeForbidden, "public registration cannot create admin users", nil))
		return
	}

	user, err := h.users.Create(r.Context(), request)
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, user)
}

func (h *handler) currentUser(w http.ResponseWriter, r *http.Request) {
	principal, ok := authmiddleware.PrincipalFromContext(r.Context())
	if !ok {
		writeError(w, errs.New(errs.CodeUnauthorized, "missing authenticated principal", nil))
		return
	}

	user, err := h.users.Get(r.Context(), principal.UserID)
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, user)
}

func (h *handler) listUsers(w http.ResponseWriter, r *http.Request) {
	principal, ok := authmiddleware.PrincipalFromContext(r.Context())
	if !ok {
		writeError(w, errs.New(errs.CodeUnauthorized, "missing authenticated principal", nil))
		return
	}

	if principal.Role != users.RoleAdmin {
		writeError(w, errs.New(errs.CodeForbidden, "admin role required", nil))
		return
	}

	list, err := h.users.List(r.Context(), users.Filter{
		Username: strings.TrimSpace(r.URL.Query().Get("username")),
		Role:     users.Role(strings.TrimSpace(r.URL.Query().Get("role"))),
	})
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, list)
}

func (h *handler) getUser(w http.ResponseWriter, r *http.Request) {
	principal, ok := authmiddleware.PrincipalFromContext(r.Context())
	if !ok {
		writeError(w, errs.New(errs.CodeUnauthorized, "missing authenticated principal", nil))
		return
	}

	id, err := parseID(chi.URLParam(r, "userID"))
	if err != nil {
		writeError(w, err)
		return
	}

	if principal.Role != users.RoleAdmin && principal.UserID != id {
		writeError(w, errs.New(errs.CodeForbidden, "insufficient permissions", nil))
		return
	}

	user, err := h.users.Get(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, user)
}

func (h *handler) updateUser(w http.ResponseWriter, r *http.Request) {
	principal, ok := authmiddleware.PrincipalFromContext(r.Context())
	if !ok {
		writeError(w, errs.New(errs.CodeUnauthorized, "missing authenticated principal", nil))
		return
	}

	id, err := parseID(chi.URLParam(r, "userID"))
	if err != nil {
		writeError(w, err)
		return
	}

	if principal.Role != users.RoleAdmin && principal.UserID != id {
		writeError(w, errs.New(errs.CodeForbidden, "insufficient permissions", nil))
		return
	}

	var request users.UpdateInput
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, err)
		return
	}

	if principal.Role != users.RoleAdmin {
		request.Role = nil
	}

	updated, err := h.users.Update(r.Context(), id, request)
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, updated)
}

func (h *handler) deleteUser(w http.ResponseWriter, r *http.Request) {
	principal, ok := authmiddleware.PrincipalFromContext(r.Context())
	if !ok {
		writeError(w, errs.New(errs.CodeUnauthorized, "missing authenticated principal", nil))
		return
	}

	id, err := parseID(chi.URLParam(r, "userID"))
	if err != nil {
		writeError(w, err)
		return
	}

	if principal.Role != users.RoleAdmin && principal.UserID != id {
		writeError(w, errs.New(errs.CodeForbidden, "insufficient permissions", nil))
		return
	}

	if err := h.users.Delete(r.Context(), id); err != nil {
		writeError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *handler) listTasks(w http.ResponseWriter, r *http.Request) {
	principal, ok := authmiddleware.PrincipalFromContext(r.Context())
	if !ok {
		writeError(w, errs.New(errs.CodeUnauthorized, "missing authenticated principal", nil))
		return
	}

	ownerID, err := parseOptionalID(r.URL.Query().Get("user_id"))
	if err != nil {
		writeError(w, err)
		return
	}

	if principal.Role != users.RoleAdmin {
		ownerID = &principal.UserID
	}

	list, err := h.tasks.List(r.Context(), tasks.Filter{
		OwnerID: ownerID,
		Status:  tasks.Status(strings.TrimSpace(r.URL.Query().Get("status"))),
	})
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, list)
}

func (h *handler) getTask(w http.ResponseWriter, r *http.Request) {
	principal, ok := authmiddleware.PrincipalFromContext(r.Context())
	if !ok {
		writeError(w, errs.New(errs.CodeUnauthorized, "missing authenticated principal", nil))
		return
	}

	id, err := parseID(chi.URLParam(r, "taskID"))
	if err != nil {
		writeError(w, err)
		return
	}

	task, err := h.tasks.Get(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}

	if principal.Role != users.RoleAdmin && principal.UserID != task.UserID {
		writeError(w, errs.New(errs.CodeForbidden, "insufficient permissions", nil))
		return
	}

	writeJSON(w, http.StatusOK, task)
}

func (h *handler) createTask(w http.ResponseWriter, r *http.Request) {
	principal, ok := authmiddleware.PrincipalFromContext(r.Context())
	if !ok {
		writeError(w, errs.New(errs.CodeUnauthorized, "missing authenticated principal", nil))
		return
	}

	var request createTaskRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, err)
		return
	}

	ownerID := principal.UserID
	if principal.Role == users.RoleAdmin && request.UserID != nil {
		ownerID = *request.UserID
	}

	created, err := h.tasks.Create(r.Context(), tasks.CreateInput{
		Title:       request.Title,
		Description: request.Description,
		Status:      request.Status,
		UserID:      ownerID,
	})
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, created)
}

func (h *handler) updateTask(w http.ResponseWriter, r *http.Request) {
	principal, ok := authmiddleware.PrincipalFromContext(r.Context())
	if !ok {
		writeError(w, errs.New(errs.CodeUnauthorized, "missing authenticated principal", nil))
		return
	}

	id, err := parseID(chi.URLParam(r, "taskID"))
	if err != nil {
		writeError(w, err)
		return
	}

	current, err := h.tasks.Get(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}

	if principal.Role != users.RoleAdmin && principal.UserID != current.UserID {
		writeError(w, errs.New(errs.CodeForbidden, "insufficient permissions", nil))
		return
	}

	var request updateTaskRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, err)
		return
	}

	updated, err := h.tasks.Update(r.Context(), id, tasks.UpdateInput{
		Title:       request.Title,
		Description: request.Description,
		Status:      request.Status,
	})
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, updated)
}

func (h *handler) deleteTask(w http.ResponseWriter, r *http.Request) {
	principal, ok := authmiddleware.PrincipalFromContext(r.Context())
	if !ok {
		writeError(w, errs.New(errs.CodeUnauthorized, "missing authenticated principal", nil))
		return
	}

	id, err := parseID(chi.URLParam(r, "taskID"))
	if err != nil {
		writeError(w, err)
		return
	}

	current, err := h.tasks.Get(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}

	if principal.Role != users.RoleAdmin && principal.UserID != current.UserID {
		writeError(w, errs.New(errs.CodeForbidden, "insufficient permissions", nil))
		return
	}

	if err := h.tasks.Delete(r.Context(), id); err != nil {
		writeError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
