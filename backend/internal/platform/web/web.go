package web

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5/middleware"

	"github.com/ndy-s/kasa/backend/internal/platform/apperr"
	"github.com/ndy-s/kasa/backend/internal/platform/auth"
)

type ctxKey int

const customerIDKey ctxKey = 0

func JSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Error("encode response failed", "err", err)
	}
}

type errorEnvelope struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func Error(w http.ResponseWriter, r *http.Request, err error) {
	var appErr *apperr.Error
	if !errors.As(err, &appErr) {
		appErr = apperr.Internal()
	}
	if appErr.HTTPStatus >= 500 {
		slog.Error("request failed",
			"request_id", middleware.GetReqID(r.Context()),
			"code", appErr.Code,
			"err", err,
		)
	}
	JSON(w, appErr.HTTPStatus, errorEnvelope{Code: appErr.Code, Message: appErr.Message})
}

func CustomerID(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(customerIDKey).(string)
	return id, ok
}

// Audit logs a completed money operation with its request id, for who/what/when traceability.
func Audit(ctx context.Context, event string, args ...any) {
	slog.Info(event, append([]any{"request_id", middleware.GetReqID(ctx)}, args...)...)
}

func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		next.ServeHTTP(ww, r)
		slog.Info("request",
			"request_id", middleware.GetReqID(r.Context()),
			"method", r.Method,
			"path", r.URL.Path,
			"status", ww.Status(),
			"duration", time.Since(start).String(),
		)
	})
}

func AuthGuard(issuer *auth.TokenIssuer) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if !strings.HasPrefix(header, "Bearer ") {
				Error(w, r, apperr.Unauthorized("missing bearer token"))
				return
			}
			customerID, err := issuer.Parse(strings.TrimPrefix(header, "Bearer "))
			if err != nil {
				Error(w, r, apperr.Unauthorized("invalid token"))
				return
			}
			ctx := context.WithValue(r.Context(), customerIDKey, customerID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
