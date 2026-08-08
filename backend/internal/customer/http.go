package customer

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/ndy-s/kasa/backend/internal/platform/apperr"
	"github.com/ndy-s/kasa/backend/internal/platform/web"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Mount(r chi.Router, guard func(http.Handler) http.Handler) {
	r.Post("/register", h.register)
	r.Post("/login", h.login)
	r.Group(func(r chi.Router) {
		r.Use(guard)
		r.Get("/me", h.me)
	})
}

type registerRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type customerResponse struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Email  string `json:"email"`
	Status string `json:"status"`
}

func toResponse(c *Customer) customerResponse {
	return customerResponse{ID: c.ID, Name: c.Name, Email: c.Email, Status: string(c.Status)}
}

func (h *Handler) register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		web.Error(w, r, apperr.Invalid("invalid request body"))
		return
	}
	c, err := h.svc.Register(r.Context(), req.Name, req.Email, req.Password)
	if err != nil {
		web.Error(w, r, toAppError(err))
		return
	}
	web.JSON(w, http.StatusCreated, toResponse(c))
}

func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		web.Error(w, r, apperr.Invalid("invalid request body"))
		return
	}
	token, err := h.svc.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		web.Error(w, r, toAppError(err))
		return
	}
	web.JSON(w, http.StatusOK, map[string]string{"token": token})
}

func (h *Handler) me(w http.ResponseWriter, r *http.Request) {
	id, ok := web.CustomerID(r.Context())
	if !ok {
		web.Error(w, r, apperr.Unauthorized("not authenticated"))
		return
	}
	c, err := h.svc.Get(r.Context(), id)
	if err != nil {
		web.Error(w, r, toAppError(err))
		return
	}
	web.JSON(w, http.StatusOK, toResponse(c))
}

func toAppError(err error) error {
	switch {
	case errors.Is(err, ErrInvalidEmail):
		return apperr.Invalid("invalid email")
	case errors.Is(err, ErrEmailTaken):
		return apperr.Conflict("EMAIL_TAKEN", "email already registered")
	case errors.Is(err, ErrInvalidCredentials):
		return apperr.Unauthorized("invalid email or password")
	case errors.Is(err, ErrCustomerNotFound):
		return apperr.NotFound("customer not found")
	default:
		return err
	}
}
