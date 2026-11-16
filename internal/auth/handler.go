package auth

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/antoniopaya22/go-rest-template/internal/platform/httpx"
)

// Handler exposes the authentication endpoints.
type Handler struct {
	service *Service
	signer  *Signer
}

// NewHandler creates the authentication HTTP handler.
func NewHandler(service *Service, signer *Signer) *Handler {
	return &Handler{service: service, signer: signer}
}

// Routes registers the authentication endpoints. They are public; limiter, when
// not nil, applies a stricter rate limit to slow down credential guessing.
func (h *Handler) Routes(r chi.Router, limiter func(http.Handler) http.Handler) {
	r.Group(func(r chi.Router) {
		if limiter != nil {
			r.Use(limiter)
		}
		r.Post("/auth/login", h.login)
		r.Post("/auth/refresh", h.refresh)
		r.Post("/auth/logout", h.logout)
	})
}

// JWKS serves the public signing keys, when the algorithm is asymmetric.
func (h *Handler) JWKS() (http.HandlerFunc, bool) {
	keys, ok := h.signer.JWKS()
	if !ok {
		return nil, false
	}

	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/jwk-set+json")
		w.Header().Set("Cache-Control", "public, max-age=3600")
		_, _ = w.Write(keys)
	}, true
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	var request loginRequest
	if err := httpx.DecodeJSON(r, &request); err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	pair, err := h.service.Login(r.Context(), request.Username, request.Password)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	w.Header().Set("Cache-Control", "no-store")
	httpx.WriteJSON(w, http.StatusOK, pair)
}

func (h *Handler) refresh(w http.ResponseWriter, r *http.Request) {
	var request refreshRequest
	if err := httpx.DecodeJSON(r, &request); err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	pair, err := h.service.Refresh(r.Context(), request.RefreshToken)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	w.Header().Set("Cache-Control", "no-store")
	httpx.WriteJSON(w, http.StatusOK, pair)
}

func (h *Handler) logout(w http.ResponseWriter, r *http.Request) {
	var request refreshRequest
	if err := httpx.DecodeJSON(r, &request); err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	if err := h.service.Logout(r.Context(), request.RefreshToken); err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
