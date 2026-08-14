package web

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"net/http"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ndy-s/kasa/backend/internal/platform/apperr"
	"github.com/ndy-s/kasa/backend/internal/platform/postgres"
)

func Idempotency(pool *pgxpool.Pool) func(http.Handler) http.Handler {
	q := postgres.New(pool)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := r.Header.Get("Idempotency-Key")
			if key == "" {
				next.ServeHTTP(w, r)
				return
			}

			body, _ := io.ReadAll(r.Body)
			r.Body = io.NopCloser(bytes.NewReader(body))
			sum := sha256.Sum256(body)
			hash := hex.EncodeToString(sum[:])

			existing, err := q.GetIdempotencyKey(r.Context(), key)
			if err == nil {
				if existing.RequestHash != hash {
					Error(w, r, apperr.Conflict("IDEMPOTENCY_CONFLICT", "idempotency key reused with a different body"))
					return
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(int(existing.StatusCode))
				_, _ = w.Write(existing.ResponseBody)
				return
			}
			if !errors.Is(err, pgx.ErrNoRows) {
				Error(w, r, err)
				return
			}

			rec := &recorder{ResponseWriter: w, status: http.StatusOK, buf: &bytes.Buffer{}}
			next.ServeHTTP(rec, r)

			if rec.status >= 200 && rec.status < 300 {
				_ = q.CreateIdempotencyKey(r.Context(), postgres.CreateIdempotencyKeyParams{
					Key: key, RequestHash: hash, ResponseBody: rec.buf.Bytes(), StatusCode: int32(rec.status),
				})
			}
		})
	}
}

type recorder struct {
	http.ResponseWriter
	status int
	buf    *bytes.Buffer
}

func (rec *recorder) WriteHeader(code int) {
	rec.status = code
	rec.ResponseWriter.WriteHeader(code)
}

func (rec *recorder) Write(b []byte) (int, error) {
	rec.buf.Write(b)
	return rec.ResponseWriter.Write(b)
}
