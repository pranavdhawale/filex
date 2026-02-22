package api

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// responseInfoWriter wraps http.ResponseWriter to capture the status code.
type responseInfoWriter struct {
	http.ResponseWriter
	statusCode int
}

func (w *responseInfoWriter) WriteHeader(code int) {
	w.statusCode = code
	w.ResponseWriter.WriteHeader(code)
}

// RequestLogger is an HTTP middleware that logs incoming requests and injects
// a unique RequestID into the response headers.
func RequestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		reqID := uuid.New().String()

		// Inject Request-ID for tracing
		w.Header().Set("X-Request-ID", reqID)

		// Wrap the writer to capture the response status
		rw := &responseInfoWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK, // default if WriteHeader is never explicitly called
		}

		ip := extractIP(r)

		slog.Info("Request started",
			"req_id", reqID,
			"method", r.Method,
			"path", r.URL.Path,
			"ip", ip,
		)

		next.ServeHTTP(rw, r)

		duration := time.Since(start).Milliseconds()

		slog.Info("Request completed",
			"req_id", reqID,
			"method", r.Method,
			"path", r.URL.Path,
			"status", rw.statusCode,
			"duration_ms", duration,
			"ip", ip,
		)
	})
}
