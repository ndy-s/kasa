package account

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/ndy-s/kasa/backend/internal/platform/apperr"
	"github.com/ndy-s/kasa/backend/internal/platform/web"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) Mount(r chi.Router, guard func(http.Handler) http.Handler) {
	r.Group(func(r chi.Router) {
		r.Use(guard)
		r.Post("/accounts", h.open)
		r.Get("/accounts/{id}", h.get)
		r.Get("/accounts/{id}/balances", h.balances)
	})
}

type openRequest struct {
	ProductCode string `json:"product_code"`
}

type accountResponse struct {
	ID       string `json:"id"`
	Status   string `json:"status"`
	Currency string `json:"currency"`
}

func toResponse(a *Account) accountResponse {
	return accountResponse{ID: a.ID, Status: string(a.Status), Currency: a.Currency}
}

func (h *Handler) open(w http.ResponseWriter, r *http.Request) {
	customerID, _ := web.CustomerID(r.Context())
	var req openRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		web.Error(w, r, apperr.Invalid("invalid request body"))
		return
	}
	acc, err := h.svc.OpenAccount(r.Context(), customerID, req.ProductCode)
	if err != nil {
		web.Error(w, r, err)
		return
	}
	web.JSON(w, http.StatusCreated, toResponse(acc))
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	acc, err := h.svc.Get(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		web.Error(w, r, apperr.NotFound("account not found"))
		return
	}
	// authorization: a customer may only see their own account
	if customerID, _ := web.CustomerID(r.Context()); acc.CustomerID != customerID {
		web.Error(w, r, apperr.NotFound("account not found"))
		return
	}
	web.JSON(w, http.StatusOK, toResponse(acc))
}

type balancesResponse struct {
	Ledger string `json:"ledger"`
}

func (h *Handler) balances(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	acc, err := h.svc.Get(r.Context(), id)
	if err != nil {
		web.Error(w, r, apperr.NotFound("account not found"))
		return
	}
	if customerID, _ := web.CustomerID(r.Context()); acc.CustomerID != customerID {
		web.Error(w, r, apperr.NotFound("account not found"))
		return
	}

	bal, err := h.svc.Balance(r.Context(), id)
	if err != nil {
		web.Error(w, r, err)
		return
	}
	web.JSON(w, http.StatusOK, balancesResponse{Ledger: bal.String()})
}
