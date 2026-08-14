package education

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ndy-s/kasa/backend/internal/account"
	"github.com/ndy-s/kasa/backend/internal/deposit"
	"github.com/ndy-s/kasa/backend/internal/ledger"
	"github.com/ndy-s/kasa/backend/internal/platform/apperr"
	"github.com/ndy-s/kasa/backend/internal/platform/web"
	"github.com/ndy-s/kasa/backend/internal/shared/money"
)

type Handler struct {
	pool     *pgxpool.Pool
	accounts *account.Service
	deposits *deposit.Service
	ledger   *ledger.Service
}

func NewHandler(pool *pgxpool.Pool, accounts *account.Service, deposits *deposit.Service, ledgerSvc *ledger.Service) *Handler {
	return &Handler{pool: pool, accounts: accounts, deposits: deposits, ledger: ledgerSvc}
}

func (h *Handler) Mount(r chi.Router, guard func(http.Handler) http.Handler) {
	r.Group(func(r chi.Router) {
		r.Use(guard)
		r.Post("/education/scenarios/{name}/run", h.run)
	})
}

type step struct {
	Action string `json:"action"`
	Result string `json:"result"`
	Detail string `json:"detail"`
}

func (h *Handler) run(w http.ResponseWriter, r *http.Request) {
	customerID, _ := web.CustomerID(r.Context())
	switch chi.URLParam(r, "name") {
	case "insufficient-funds":
		h.insufficientFunds(w, r, customerID)
	case "reversal":
		h.reversal(w, r, customerID)
	default:
		web.Error(w, r, apperr.NotFound("unknown scenario"))
	}
}

func (h *Handler) insufficientFunds(w http.ResponseWriter, r *http.Request, customerID string) {
	steps := []step{}

	acc, err := h.accounts.OpenAccount(r.Context(), customerID, "SAV")
	if err != nil {
		web.Error(w, r, err)
		return
	}
	steps = append(steps, step{"open account", "ok", "a fresh savings account is opened"})

	if _, err := h.deposits.Deposit(r.Context(), customerID, acc.ID, money.FromMinor(5000, money.IDR)); err != nil {
		web.Error(w, r, err)
		return
	}
	steps = append(steps, step{"deposit 50.00", "ok", "the account now holds 50.00"})

	_, err = h.deposits.Withdraw(r.Context(), customerID, acc.ID, money.FromMinor(10000, money.IDR))
	if errors.Is(err, deposit.ErrInsufficientFunds) {
		steps = append(steps, step{"withdraw 100.00", "rejected",
			"the withdrawal is refused because available funds are only 50.00; the ledger is untouched"})
	} else {
		steps = append(steps, step{"withdraw 100.00", "unexpected", "expected a rejection"})
	}

	web.JSON(w, http.StatusOK, map[string]any{"scenario": "insufficient-funds", "steps": steps})
}

func (h *Handler) reversal(w http.ResponseWriter, r *http.Request, customerID string) {
	steps := []step{}

	acc, err := h.accounts.OpenAccount(r.Context(), customerID, "SAV")
	if err != nil {
		web.Error(w, r, err)
		return
	}
	steps = append(steps, step{"open account", "ok", "a fresh savings account is opened"})

	entryID, err := h.deposits.Deposit(r.Context(), customerID, acc.ID, money.FromMinor(5000, money.IDR))
	if err != nil {
		web.Error(w, r, err)
		return
	}
	steps = append(steps, step{"deposit 50.00", "ok", "entry " + entryID + " posted; balance is 50.00"})

	tx, err := h.pool.Begin(r.Context())
	if err != nil {
		web.Error(w, r, err)
		return
	}
	defer func() { _ = tx.Rollback(r.Context()) }() // no-op after a successful commit

	reversalID, err := h.ledger.Reverse(r.Context(), tx, entryID)
	if err != nil {
		web.Error(w, r, err)
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		web.Error(w, r, err)
		return
	}
	steps = append(steps, step{"reverse the deposit", "ok",
		"entry " + reversalID + " compensates entry " + entryID + "; balance returns to 0.00, and neither entry was deleted"})

	web.JSON(w, http.StatusOK, map[string]any{"scenario": "reversal", "steps": steps})
}
