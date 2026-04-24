package server

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/pranavdhawale/filex/internal/config"
	"github.com/pranavdhawale/filex/internal/handler"
	"github.com/pranavdhawale/filex/internal/ratelimit"
)

type Server struct {
	httpServer *http.Server
	health     *handler.HealthChecker
}

func New(
	cfg *config.Config,
	fileHandler *handler.FileHandler,
	shareHandler *handler.ShareHandler,
	health *handler.HealthChecker,
	apiLimiter *ratelimit.RateLimiter,
	uploadLimiter *ratelimit.RateLimiter,
) *Server {
	mux := http.NewServeMux()

	// Health probes
	mux.HandleFunc("GET /healthz", health.HandleHealthz)
	mux.HandleFunc("GET /readyz", health.HandleReadyz)

	// API endpoints — rate limited per IP per endpoint
	mux.HandleFunc("POST /api/v1/files/init",
		handler.RateLimitMiddleware(apiLimiter, "POST /api/v1/files/init", 10, time.Minute, fileHandler.HandleInit))
	mux.HandleFunc("POST /api/v1/files/complete",
		handler.RateLimitMiddleware(apiLimiter, "POST /api/v1/files/complete", 10, time.Minute, fileHandler.HandleComplete))
	mux.HandleFunc("POST /api/v1/files/{slug}/access",
		handler.RateLimitMiddleware(apiLimiter, "POST /api/v1/files/{slug}/access", 60, time.Minute, fileHandler.HandleGetAccess))
	mux.HandleFunc("POST /upload/{uploadID}",
		handler.RateLimitMiddleware(uploadLimiter, "POST /upload/", 300, time.Minute, fileHandler.HandleChunkUpload))
	mux.HandleFunc("POST /api/v1/shares",
		handler.RateLimitMiddleware(apiLimiter, "POST /api/v1/shares", 10, time.Minute, shareHandler.HandleCreate))
	mux.HandleFunc("GET /api/v1/shares/{shareSlug}",
		handler.RateLimitMiddleware(apiLimiter, "GET /api/v1/shares/{shareSlug}", 60, time.Minute, shareHandler.HandleGet))

	// Middleware stack
	var h http.Handler = mux
	h = handler.RequestLogger(h)
	h = handler.SecurityHeadersMiddleware(h)
	// Parse allowed origins from config
	allowedOrigins := parseAllowedOrigins(cfg.AllowedOrigins)
	h = handler.CORSMiddleware(allowedOrigins, h)

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      h,
		IdleTimeout:  time.Minute,
		ReadTimeout:  1 * time.Hour,
		WriteTimeout: 1 * time.Hour,
	}

	return &Server{httpServer: srv, health: health}
}

func parseAllowedOrigins(origins string) map[string]bool {
	result := make(map[string]bool)
	for _, origin := range strings.Split(origins, ",") {
		origin = strings.TrimSpace(origin)
		if origin != "" {
			result[origin] = true
		}
	}
	return result
}

func (s *Server) Start() error {
	slog.Info("Starting server", "addr", s.httpServer.Addr)
	if err := s.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	slog.Info("Shutting down server gracefully...")
	return s.httpServer.Shutdown(ctx)
}