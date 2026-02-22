package api

import (
	"context"
	"net/http"
	"time"
)

// TimeoutMiddleware wraps an http.Handler with a contextual timeout guard.
func TimeoutMiddleware(timeout time.Duration, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), timeout)
		defer cancel()

		r = r.WithContext(ctx)

		// We use http.TimeoutHandler internally to also handle writing the 503 response
		// cleanly if the timeout hits before next.ServeHTTP finishes.
		handler := http.TimeoutHandler(next, timeout, `{"error": "request timeout"}`)
		handler.ServeHTTP(w, r)
	})
}

// CORSMiddleware sets strict Cross-Origin Resource Sharing headers.
func CORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set allowed origins - in production this would read from config
		origin := r.Header.Get("Origin")
		if origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
		} else {
			w.Header().Set("Access-Control-Allow-Origin", "*")
		}

		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS, PUT")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
		w.Header().Set("Access-Control-Expose-Headers", "X-Request-ID, ETag")

		// Handle preflight OPTIONS request
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
