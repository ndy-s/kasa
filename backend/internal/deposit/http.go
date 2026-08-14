package deposit

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

func (h *Handler) Mount(r chi.Router, guard, rateLimit func(http.Handler) http.Handler) {
	r.Group(func(r chi.Router) {
		r.Use(guard)
		r.Use(rateLimit)
		r.Post("/accounts/{id}/deposit", h.deposit)
		r.Post("/accounts/{id}/withdraw", h.withdraw)
	})
}

type amountRequest struct {
	Amount string `json:"amount"` // e.g. "100.00"
}

func (h *Handler) deposit(w http.ResponseWriter, r *http.Request)  { h.handle(w, r, true) }
func (h *Handler) withdraw(w http.ResponseWriter, r *http.Request) { h.handle(w, r, false) }

func (h *Handler) handle(w http.ResponseWriter, r *http.Request, isDeposit bool) {
	op := "deposit"
	if !isDeposit {
		op = "withdraw"
	}

	var req amountRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		web.RecordMoneyOp(op, "failure")
		web.Error(w, r, apperr.Invalid("invalid request body"))
		return
	}
	amount, err := money.Parse(req.Amount, money.IDR)
	if err != nil || !amount.IsPositive() {
		web.RecordMoneyOp(op, "failure")
		web.Error(w, r, apperr.Invalid("amount must be a positive number"))
		return
	}

	actor, _ := web.CustomerID(r.Context())
	accountID := chi.URLParam(r, "id")
	var entryID string
	if isDeposit {
		entryID, err = h.svc.Deposit(r.Context(), actor, accountID, amount)
	} else {
		entryID, err = h.svc.Withdraw(r.Context(), actor, accountID, amount)
	}
	if err != nil {
		web.RecordMoneyOp(op, "failure")
		web.Error(w, r, toAppError(err))
		return
	}

	web.RecordMoneyOp(op, "success")
	web.Audit(r.Context(), "money."+op, "actor", actor, "account", accountID, "amount", amount.String(), "entry_id", entryID)

	web.JSON(w, http.StatusCreated, map[string]string{"entry_id": entryID})
}

func toAppError(err error) error {
	switch {
	case errors.Is(err, ErrInsufficientFunds):
		return apperr.New("INSUFFICIENT_FUNDS", http.StatusUnprocessableEntity, "insufficient funds")
	case errors.Is(err, ErrAccountNotActive):
		return apperr.New("ACCOUNT_NOT_ACTIVE", http.StatusUnprocessableEntity, "account is not active")
	case errors.Is(err, ErrInvalidAmount):
		return apperr.Invalid("amount must be a positive number")
	default:
		return err
	}
}
