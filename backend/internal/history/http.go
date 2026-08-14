package history

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ndy-s/kasa/backend/internal/platform/apperr"
	"github.com/ndy-s/kasa/backend/internal/platform/postgres"
	"github.com/ndy-s/kasa/backend/internal/platform/web"
	"github.com/ndy-s/kasa/backend/internal/shared/money"
)

type Handler struct {
	q *postgres.Queries
}

func NewHandler(pool *pgxpool.Pool) *Handler { return &Handler{q: postgres.New(pool)} }

func (h *Handler) Mount(r chi.Router, guard func(http.Handler) http.Handler) {
	r.Group(func(r chi.Router) {
		r.Use(guard)
		r.Get("/accounts/{id}/transactions", h.transactions)
		r.Get("/transactions/{id}/journal", h.journal)
	})
}

const (
	defaultLimit = 20
	maxLimit     = 100
)

type entryView struct {
	EntryID     string `json:"entry_id"`
	Type        string `json:"type"`
	Description string `json:"description"`
	BookingDate string `json:"booking_date"`
}

// transactions lists an account's journal entries, most recent first.
func (h *Handler) transactions(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		web.Error(w, r, apperr.NotFound("account not found"))
		return
	}
	accID := pgtype.UUID{Bytes: id, Valid: true}

	acc, err := h.q.GetAccountByID(r.Context(), accID)
	if err != nil {
		web.Error(w, r, apperr.NotFound("account not found"))
		return
	}
	if customerID, _ := web.CustomerID(r.Context()); uuid.UUID(acc.CustomerID.Bytes).String() != customerID {
		web.Error(w, r, apperr.NotFound("account not found"))
		return
	}

	limit, offset := paginationParams(r)
	rows, err := h.q.ListEntriesByAccount(r.Context(), postgres.ListEntriesByAccountParams{
		LedgerAccountID: acc.LedgerAccountID,
		Limit:           limit,
		Offset:          offset,
	})
	if err != nil {
		web.Error(w, r, err)
		return
	}

	views := make([]entryView, 0, len(rows))
	for _, row := range rows {
		views = append(views, entryView{
			EntryID:     uuid.UUID(row.ID.Bytes).String(),
			Type:        row.TransactionType,
			Description: row.Description,
			BookingDate: row.BookingDate.Time.Format("2006-01-02"),
		})
	}
	web.JSON(w, http.StatusOK, views)
}

func paginationParams(r *http.Request) (limit, offset int32) {
	limit = defaultLimit
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= maxLimit {
			limit = int32(n)
		}
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			offset = int32(n)
		}
	}
	return limit, offset
}

type lineView struct {
	AccountID string `json:"account_id"`
	Direction string `json:"direction"`
	Amount    string `json:"amount"`
}

type journalView struct {
	EntryID  string     `json:"entry_id"`
	Type     string     `json:"type"`
	Lines    []lineView `json:"lines"`
	Balanced bool       `json:"balanced"`
}

func (h *Handler) journal(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		web.Error(w, r, apperr.NotFound("transaction not found"))
		return
	}
	entryID := pgtype.UUID{Bytes: id, Valid: true}

	entry, err := h.q.GetEntry(r.Context(), entryID)
	if err != nil {
		web.Error(w, r, apperr.NotFound("transaction not found"))
		return
	}

	customerID, _ := web.CustomerID(r.Context())
	custUUID, err := uuid.Parse(customerID)
	if err != nil {
		web.Error(w, r, apperr.NotFound("transaction not found"))
		return
	}
	touches, err := h.q.EntryTouchesCustomer(r.Context(), postgres.EntryTouchesCustomerParams{
		JournalEntryID: entryID,
		CustomerID:     pgtype.UUID{Bytes: custUUID, Valid: true},
	})
	if err != nil {
		web.Error(w, r, err)
		return
	}
	if !touches {
		web.Error(w, r, apperr.NotFound("transaction not found"))
		return
	}

	rows, err := h.q.ListLinesByEntry(r.Context(), entryID)
	if err != nil {
		web.Error(w, r, err)
		return
	}

	var debits, credits int64
	view := journalView{EntryID: uuid.UUID(entry.ID.Bytes).String(), Type: entry.TransactionType}
	for _, l := range rows {
		if l.Direction == "debit" {
			debits += l.AmountMinor
		} else {
			credits += l.AmountMinor
		}
		cur, _ := money.ForCode(l.Currency)
		view.Lines = append(view.Lines, lineView{
			AccountID: uuid.UUID(l.LedgerAccountID.Bytes).String(),
			Direction: l.Direction,
			Amount:    money.FromMinor(l.AmountMinor, cur).String(),
		})
	}
	view.Balanced = debits == credits
	web.JSON(w, http.StatusOK, view)
}
