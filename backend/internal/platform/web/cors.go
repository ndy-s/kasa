package web

import "net/http"

// CORS allows a browser-based client, like the web app, to call this API cross-origin.
// The API is stateless and bearer-token authenticated (no cookies), so a wildcard origin
// is safe here; it would need to be tightened to a specific origin if the API ever relied
// on cookie-based auth.
func CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, Idempotency-Key")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
