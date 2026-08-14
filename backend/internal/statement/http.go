package statement

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/ndy-s/kasa/backend/internal/platform/apperr"
	"github.com/ndy-s/kasa/backend/internal/platform/postgres"
	"github.com/ndy-s/kasa/backend/internal/platform/web"
)

type Handler struct {
	svc *Service
	q   *postgres.Queries
}

func NewHandler(svc *Service, q *postgres.Queries) *Handler { return &Handler{svc: svc, q: q} }

func (h *Handler) Mount(r chi.Router, guard func(http.Handler) http.Handler) {
	r.Group(func(r chi.Router) {
		r.Use(guard)
		r.Get("/accounts/{id}/statement", h.statement)
	})
}

type lineResponse struct {
	BookingDate string `json:"booking_date"`
	Type        string `json:"type"`
	Direction   string `json:"direction"`
	Amount      string `json:"amount"`
}

type statementResponse struct {
	Opening string         `json:"opening"`
	Lines   []lineResponse `json:"lines"`
	Closing string         `json:"closing"`
}

const dateLayout = "2006-01-02"

func (h *Handler) statement(w http.ResponseWriter, r *http.Request) {
	accountID := chi.URLParam(r, "id")
	id, err := uuid.Parse(accountID)
	if err != nil {
		web.Error(w, r, apperr.NotFound("account not found"))
		return
	}
	acc, err := h.q.GetAccountByID(r.Context(), pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		web.Error(w, r, apperr.NotFound("account not found"))
		return
	}
	if customerID, _ := web.CustomerID(r.Context()); uuid.UUID(acc.CustomerID.Bytes).String() != customerID {
		web.Error(w, r, apperr.NotFound("account not found"))
		return
	}

	from, err := time.Parse(dateLayout, r.URL.Query().Get("from"))
	if err != nil {
		web.Error(w, r, apperr.Invalid("invalid from date, want YYYY-MM-DD"))
		return
	}
	to, err := time.Parse(dateLayout, r.URL.Query().Get("to"))
	if err != nil {
		web.Error(w, r, apperr.Invalid("invalid to date, want YYYY-MM-DD"))
		return
	}

	st, err := h.svc.Generate(r.Context(), accountID, from, to)
	if err != nil {
		web.Error(w, r, err)
		return
	}

	resp := statementResponse{Opening: st.Opening.String(), Closing: st.Closing.String()}
	for _, l := range st.Lines {
		resp.Lines = append(resp.Lines, lineResponse{
			BookingDate: l.BookingDate.Format(dateLayout),
			Type:        l.Type,
			Direction:   l.Direction,
			Amount:      l.Amount.String(),
		})
	}
	web.JSON(w, http.StatusOK, resp)
}
