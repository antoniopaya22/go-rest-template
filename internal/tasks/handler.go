package tasks

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/antoniopaya22/go-rest-template/v3/internal/auth"
	"github.com/antoniopaya22/go-rest-template/v3/internal/platform/errs"
	"github.com/antoniopaya22/go-rest-template/v3/internal/platform/httpx"
)

// Handler exposes the task endpoints.
type Handler struct {
	service *Service
}

// NewHandler creates the task HTTP handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// Routes registers the task endpoints. All of them require authentication.
func (h *Handler) Routes(r chi.Router, authn *auth.Middleware) {
	r.Group(func(r chi.Router) {
		r.Use(authn.Authenticate)

		r.Get("/tasks", h.list)
		r.Post("/tasks", h.create)
		r.Get("/tasks/{taskID}", h.get)
		r.Patch("/tasks/{taskID}", h.update)
		r.Delete("/tasks/{taskID}", h.delete)
	})
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	page, err := httpx.ParsePage(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	userID, err := httpx.QueryID(r, "user_id")
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	// Regular users only ever see their own tasks.
	if caller := auth.MustIdentity(r.Context()); !caller.IsAdmin() {
		userID = &caller.UserID
	}

	tasks, err := h.service.List(r.Context(), Filter{
		UserID: userID,
		Status: Status(r.URL.Query().Get("status")),
	}, page)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, httpx.NewPage(tasks, page, func(t Task) int64 { return t.ID }))
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var input CreateInput
	if err := httpx.DecodeJSON(r, &input); err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	caller := auth.MustIdentity(r.Context())
	ownerID := caller.UserID
	if input.UserID != nil {
		if err := caller.Authorize(*input.UserID); err != nil {
			httpx.WriteError(w, r, errs.Forbidden("only administrators can create tasks for other users"))
			return
		}
		ownerID = *input.UserID
	}

	task, err := h.service.Create(r.Context(), ownerID, input)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	httpx.SetETag(w, task.Version)
	httpx.WriteJSON(w, http.StatusCreated, task)
}

// load resolves the task in the path and checks that the caller may act on it.
func (h *Handler) load(r *http.Request) (Task, error) {
	id, err := httpx.PathID(r, "taskID")
	if err != nil {
		return Task{}, err
	}

	task, err := h.service.Get(r.Context(), id)
	if err != nil {
		return Task{}, err
	}

	// Report a task the caller cannot access as missing, so identifiers of
	// other users' tasks cannot be probed.
	if err := auth.MustIdentity(r.Context()).Authorize(task.UserID); err != nil {
		return Task{}, errs.NotFound("task not found")
	}
	return task, nil
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	task, err := h.load(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	httpx.SetETag(w, task.Version)
	httpx.WriteJSON(w, http.StatusOK, task)
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	current, err := h.load(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	if err := httpx.CheckIfMatch(r, current.Version); err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	var input UpdateInput
	if err := httpx.DecodeJSON(r, &input); err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	updated, err := h.service.Update(r.Context(), current, input)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	httpx.SetETag(w, updated.Version)
	httpx.WriteJSON(w, http.StatusOK, updated)
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	current, err := h.load(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	if err := httpx.CheckIfMatch(r, current.Version); err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	if err := h.service.Delete(r.Context(), current.ID); err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
