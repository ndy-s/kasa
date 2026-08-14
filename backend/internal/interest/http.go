package interest

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/ndy-s/kasa/backend/internal/platform/apperr"
	"github.com/ndy-s/kasa/backend/internal/platform/clock"
	"github.com/ndy-s/kasa/backend/internal/platform/web"
)

type AdminHandler struct {
	svc   *Service
	clock *clock.Fake
}

func NewAdminHandler(svc *Service, fake *clock.Fake) *AdminHandler {
	return &AdminHandler{svc: svc, clock: fake}
}

func (h *AdminHandler) Mount(r chi.Router, guard func(http.Handler) http.Handler) {
	r.Group(func(r chi.Router) {
		r.Use(guard)
		r.Post("/admin/clock/advance", h.advance)
		r.Post("/admin/interest/capitalize", h.capitalize)
	})
}

type advanceRequest struct {
	Days int `json:"days"`
}

func (h *AdminHandler) advance(w http.ResponseWriter, r *http.Request) {
	var req advanceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Days <= 0 {
		web.Error(w, r, apperr.Invalid("days must be a positive integer"))
		return
	}
	for i := 0; i < req.Days; i++ {
		h.clock.Advance(24 * time.Hour)
		if err := h.svc.Accrue(r.Context(), h.clock.Now()); err != nil {
			web.Error(w, r, err)
			return
		}
	}
	web.JSON(w, http.StatusOK, map[string]string{"now": h.clock.Now().Format("2006-01-02")})
}

func (h *AdminHandler) capitalize(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.CapitalizeAll(r.Context(), h.clock.Now()); err != nil {
		web.Error(w, r, err)
		return
	}
	web.JSON(w, http.StatusOK, map[string]string{"now": h.clock.Now().Format("2006-01-02")})
}
