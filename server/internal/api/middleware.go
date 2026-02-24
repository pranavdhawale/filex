package api

import (
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/pranavdhawale/filex/internal/ratelimit"
)

// extractIP attempts to find the true client IP from common proxy headers,
// falling back to RemoteAddr.
func extractIP(r *http.Request) string {
	if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
		// X-Forwarded-For can be a comma-separated list of IPs. The first one is the original client.
		parts := strings.Split(ip, ",")
		return strings.TrimSpace(parts[0])
	}
	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		return strings.TrimSpace(ip)
	}

	// RemoteAddr includes the port (e.g., "192.168.1.1:54321"). Strip it.
	ip := r.RemoteAddr
	if colonIdx := strings.LastIndex(ip, ":"); colonIdx != -1 {
		ip = ip[:colonIdx]
	}
	return ip
}

// RateLimitMiddleware wraps an http.HandlerFunc with IP-based sliding window rate limiting.
func RateLimitMiddleware(limiter *ratelimit.RateLimiter, action string, limit int, window time.Duration, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ip := extractIP(r)

		allowed, err := limiter.Allow(r.Context(), ip, action, limit, window)
		if err != nil {
			slog.Error("Rate limiter failed", "ip", ip, "action", action, "error", err)
			// Fail open or closed? For a secure file sharer, fail closed is usually safer,
			// but we'll fail open for availability if Redis blips, though this is a design choice.
			// System prompt says "Redis for rate limiting", let's fail closed to strictly enforce limits.
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		if !allowed {
			slog.Warn("Rate limit exceeded", "ip", ip, "action", action)
			http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	}
}
