package transfer

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/ndy-s/kasa/backend/internal/platform/apperr"
	"github.com/ndy-s/kasa/backend/internal/platform/web"
	"github.com/ndy-s/kasa/backend/internal/shared/money"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) Mount(r chi.Router, guard, idempotency, rateLimit func(http.Handler) http.Handler) {
	r.Group(func(r chi.Router) {
		r.Use(guard)
		r.Use(rateLimit)
		r.Use(idempotency)
		r.Post("/transfers", h.transfer)
	})
}

type transferRequest struct {
	FromAccountID string `json:"from_account_id"`
	ToAccountID   string `json:"to_account_id"`
	Amount        string `json:"amount"`
}

func (h *Handler) transfer(w http.ResponseWriter, r *http.Request) {
	var req transferRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		web.Error(w, r, apperr.Invalid("invalid request body"))
		return
	}
	amount, err := money.Parse(req.Amount, money.IDR)
	if err != nil || !amount.IsPositive() {
		web.Error(w, r, apperr.Invalid("amount must be a positive number"))
		return
	}

	actor, _ := web.CustomerID(r.Context())
	entryID, err := h.svc.Transfer(r.Context(), actor, req.FromAccountID, req.ToAccountID, amount)
	if err != nil {
		switch {
		case errors.Is(err, ErrInsufficientFunds):
			web.Error(w, r, apperr.New("INSUFFICIENT_FUNDS", http.StatusUnprocessableEntity, "insufficient funds"))
		case errors.Is(err, ErrSameAccount):
			web.Error(w, r, apperr.Invalid("cannot transfer to the same account"))
		case errors.Is(err, ErrInvalidAmount):
			web.Error(w, r, apperr.Invalid("amount must be a positive number"))
		default:
			web.Error(w, r, err)
		}
		return
	}

	web.Audit(r.Context(), "money.transfer",
		"actor", actor, "from", req.FromAccountID, "to", req.ToAccountID, "amount", amount.String(), "entry_id", entryID)

	web.JSON(w, http.StatusCreated, map[string]string{"entry_id": entryID})
}
