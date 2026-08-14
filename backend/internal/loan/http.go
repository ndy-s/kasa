package loan

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

func (h *Handler) Mount(r chi.Router, guard func(http.Handler) http.Handler) {
	r.Group(func(r chi.Router) {
		r.Use(guard)
		r.Post("/loans", h.originate)
		r.Get("/loans/{id}", h.get)
		r.Get("/loans/{id}/schedule", h.schedule)
		r.Post("/loans/{id}/repay", h.repay)
	})
}

type originateRequest struct {
	ProductCode      string `json:"product_code"`
	DepositAccountID string `json:"deposit_account_id"`
	Principal        string `json:"principal"`
	TermMonths       int    `json:"term_months"`
}

type loanResponse struct {
	ID               string `json:"id"`
	DepositAccountID string `json:"deposit_account_id"`
	Principal        string `json:"principal"`
	TermMonths       int    `json:"term_months"`
	InterestRateBps  int32  `json:"interest_rate_bps"`
	Status           string `json:"status"`
	DisbursedAt      string `json:"disbursed_at"`
}

func toResponse(l *Loan) loanResponse {
	cur, _ := money.ForCode(l.Currency)
	return loanResponse{
		ID:               l.ID,
		DepositAccountID: l.DepositAccountID,
		Principal:        money.FromMinor(l.PrincipalMinor, cur).String(),
		TermMonths:       l.TermMonths,
		InterestRateBps:  l.InterestRateBps,
		Status:           string(l.Status),
		DisbursedAt:      l.DisbursedAt.Format("2006-01-02"),
	}
}

func (h *Handler) originate(w http.ResponseWriter, r *http.Request) {
	customerID, _ := web.CustomerID(r.Context())
	var req originateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		web.Error(w, r, apperr.Invalid("invalid request body"))
		return
	}
	if req.TermMonths <= 0 {
		web.Error(w, r, apperr.Invalid("term_months must be positive"))
		return
	}
	principal, err := money.Parse(req.Principal, money.IDR)
	if err != nil || principal.IsZero() || principal.IsNegative() {
		web.Error(w, r, apperr.Invalid("invalid principal"))
		return
	}

	l, err := h.svc.Originate(r.Context(), customerID, req.ProductCode, req.DepositAccountID, principal, req.TermMonths)
	if err != nil {
		switch {
		case errors.Is(err, ErrDepositNotActive):
			web.Error(w, r, apperr.Invalid("deposit account is not active or not yours"))
		default:
			web.Error(w, r, apperr.Invalid("could not originate loan, check the product code and account"))
		}
		return
	}
	web.JSON(w, http.StatusCreated, toResponse(l))
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	customerID, _ := web.CustomerID(r.Context())
	l, err := h.svc.Get(r.Context(), chi.URLParam(r, "id"))
	if err != nil || l.CustomerID != customerID {
		web.Error(w, r, apperr.NotFound("loan not found"))
		return
	}
	web.JSON(w, http.StatusOK, toResponse(l))
}

type installmentResponse struct {
	Number    int    `json:"number"`
	DueDate   string `json:"due_date"`
	Principal string `json:"principal"`
	Interest  string `json:"interest"`
	Total     string `json:"total"`
	Balance   string `json:"balance"`
	Status    string `json:"status"`
}

func (h *Handler) schedule(w http.ResponseWriter, r *http.Request) {
	customerID, _ := web.CustomerID(r.Context())
	id := chi.URLParam(r, "id")

	l, err := h.svc.Get(r.Context(), id)
	if err != nil || l.CustomerID != customerID {
		web.Error(w, r, apperr.NotFound("loan not found"))
		return
	}

	installments, err := h.svc.Installments(r.Context(), id)
	if err != nil {
		web.Error(w, r, err)
		return
	}
	resp := make([]installmentResponse, len(installments))
	for i, inst := range installments {
		total, _ := inst.Principal.Add(inst.Interest)
		resp[i] = installmentResponse{
			Number:    inst.Number,
			DueDate:   inst.DueDate.Format("2006-01-02"),
			Principal: inst.Principal.String(),
			Interest:  inst.Interest.String(),
			Total:     total.String(),
			Balance:   inst.Balance.String(),
			Status:    inst.Status,
		}
	}
	web.JSON(w, http.StatusOK, resp)
}

func (h *Handler) repay(w http.ResponseWriter, r *http.Request) {
	customerID, _ := web.CustomerID(r.Context())
	entryID, err := h.svc.Repay(r.Context(), customerID, chi.URLParam(r, "id"))
	if err != nil {
		switch {
		case errors.Is(err, ErrLoanNotFound):
			web.Error(w, r, apperr.NotFound("loan not found"))
		case errors.Is(err, ErrNoInstallmentDue):
			web.Error(w, r, apperr.Invalid("loan has no installment due"))
		case errors.Is(err, ErrInsufficientFunds):
			web.Error(w, r, apperr.New("INSUFFICIENT_FUNDS", http.StatusUnprocessableEntity, "insufficient funds"))
		case errors.Is(err, ErrDepositNotActive):
			web.Error(w, r, apperr.Invalid("deposit account is not active"))
		default:
			web.Error(w, r, err)
		}
		return
	}
	web.JSON(w, http.StatusCreated, map[string]string{"entry_id": entryID})
}
