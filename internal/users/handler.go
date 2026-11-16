package users

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/antoniopaya22/go-rest-template/internal/auth"
	"github.com/antoniopaya22/go-rest-template/internal/platform/errs"
	"github.com/antoniopaya22/go-rest-template/internal/platform/httpx"
)

// Handler exposes the user endpoints.
type Handler struct {
	service             *Service
	registrationEnabled bool
}

// NewHandler creates the user HTTP handler. When registration is enabled,
// anonymous callers can create regular accounts; otherwise only administrators can.
func NewHandler(service *Service, registrationEnabled bool) *Handler {
	return &Handler{service: service, registrationEnabled: registrationEnabled}
}

// Routes registers the user endpoints.
func (h *Handler) Routes(r chi.Router, authn *auth.Middleware) {
	if h.registrationEnabled {
		r.Post("/users", h.register)
	}

	r.Group(func(r chi.Router) {
		r.Use(authn.Authenticate)

		r.Get("/users/me", h.me)
		r.Get("/users/{userID}", h.get)
		r.Patch("/users/{userID}", h.update)
		r.Delete("/users/{userID}", h.delete)

		r.Group(func(r chi.Router) {
			r.Use(auth.RequireRole(auth.RoleAdmin))
			r.Get("/users", h.list)
			if !h.registrationEnabled {
				r.Post("/users", h.create)
			}
		})
	})
}

// register creates a regular account for an anonymous caller.
func (h *Handler) register(w http.ResponseWriter, r *http.Request) {
	var input CreateInput
	if err := httpx.DecodeJSON(r, &input); err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	if input.Role != "" && input.Role != auth.RoleUser {
		httpx.WriteError(w, r, errs.Forbidden("registration cannot assign a privileged role"))
		return
	}
	input.Role = auth.RoleUser

	h.respondCreated(w, r, input)
}

// create lets an administrator create an account with any role.
func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var input CreateInput
	if err := httpx.DecodeJSON(r, &input); err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	h.respondCreated(w, r, input)
}

func (h *Handler) respondCreated(w http.ResponseWriter, r *http.Request, input CreateInput) {
	user, err := h.service.Create(r.Context(), input)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	httpx.SetETag(w, user.Version)
	httpx.WriteJSON(w, http.StatusCreated, user)
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	page, err := httpx.ParsePage(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	query := r.URL.Query()
	users, err := h.service.List(r.Context(), Filter{
		Username: query.Get("username"),
		Role:     auth.Role(query.Get("role")),
	}, page)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, httpx.NewPage(users, page, func(u User) int64 { return u.ID }))
}

func (h *Handler) me(w http.ResponseWriter, r *http.Request) {
	user, err := h.service.Get(r.Context(), auth.MustIdentity(r.Context()).UserID)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	httpx.SetETag(w, user.Version)
	httpx.WriteJSON(w, http.StatusOK, user)
}

// load resolves the user in the path and checks that the caller may act on it.
func (h *Handler) load(r *http.Request) (User, error) {
	id, err := httpx.PathID(r, "userID")
	if err != nil {
		return User{}, err
	}

	if err := auth.MustIdentity(r.Context()).Authorize(id); err != nil {
		return User{}, err
	}

	return h.service.Get(r.Context(), id)
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	user, err := h.load(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	httpx.SetETag(w, user.Version)
	httpx.WriteJSON(w, http.StatusOK, user)
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

	if input.Role != nil && !auth.MustIdentity(r.Context()).IsAdmin() {
		httpx.WriteError(w, r, errs.Forbidden("only administrators can change roles"))
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
