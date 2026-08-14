package web

import (
	"net"
	"net/http"
	"sync"

	"golang.org/x/time/rate"

	"github.com/ndy-s/kasa/backend/internal/platform/apperr"
)

// RateLimit allows at most rps requests per second per client IP, with a small burst. Intended for a
// single API instance; a multi-instance deployment would need a shared store (e.g. Redis) instead of
// this in-memory map.
func RateLimit(rps rate.Limit, burst int) func(http.Handler) http.Handler {
	var mu sync.Mutex
	limiters := map[string]*rate.Limiter{}

	limiterFor := func(ip string) *rate.Limiter {
		mu.Lock()
		defer mu.Unlock()
		l, ok := limiters[ip]
		if !ok {
			l = rate.NewLimiter(rps, burst)
			limiters[ip] = l
		}
		return l
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip, _, err := net.SplitHostPort(r.RemoteAddr)
			if err != nil {
				ip = r.RemoteAddr
			}
			if !limiterFor(ip).Allow() {
				Error(w, r, apperr.New("RATE_LIMITED", http.StatusTooManyRequests, "too many requests"))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// SecurityHeaders sets a conservative set of response headers.
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "no-referrer")
		h.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		next.ServeHTTP(w, r)
	})
}

// AdminGuard requires a matching X-Admin-Token header, in addition to the customer auth guard, on the
// learn-mode admin routes. It always rejects if no token is configured (fail closed).
func AdminGuard(token string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if token == "" || r.Header.Get("X-Admin-Token") != token {
				Error(w, r, apperr.New("FORBIDDEN", http.StatusForbidden, "admin access required"))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
