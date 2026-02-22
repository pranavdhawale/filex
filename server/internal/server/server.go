package server

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/pranavdhawale/bytefile/internal/api"
	"github.com/pranavdhawale/bytefile/internal/config"
	"github.com/pranavdhawale/bytefile/internal/ratelimit"
)

type Server struct {
	httpServer *http.Server
}

// New creates a new web server configured with standard routes.
func New(cfg *config.Config, uploadHandler *api.UploadHandler, downloadHandler *api.DownloadHandler, limiter *ratelimit.RateLimiter) *Server {
	mux := http.NewServeMux()

	// Health and Readiness endpoints
	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("/ready", readyHandler)

	// API endpoints (wrapped with rate limits)
	// POST /upload/init: 10 reqs / 1 min
	mux.HandleFunc("POST /upload/init", api.RateLimitMiddleware(limiter, "init", 10, time.Minute, uploadHandler.HandleInit))

	// POST /upload/complete: 10 reqs / 1 min
	mux.HandleFunc("POST /upload/complete", api.RateLimitMiddleware(limiter, "complete", 10, time.Minute, uploadHandler.HandleComplete))

	// GET /f/{id}: 60 reqs / 1 min
	mux.HandleFunc("GET /f/{id}", api.RateLimitMiddleware(limiter, "download", 60, time.Minute, downloadHandler.HandleDownload))

	// Apply Middleware Stack:
	// 1. CORS (handle preflight quickly)
	// 2. RequestLogger (log all incoming requests)
	// 3. TimeoutMiddleware (15s global timeout to prevent hanging connections)
	var handler http.Handler = mux
	handler = api.TimeoutMiddleware(15*time.Second, handler)
	handler = api.RequestLogger(handler)
	handler = api.CORSMiddleware(handler)

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      handler,
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	return &Server{
		httpServer: srv,
	}
}

// Start runs the HTTP server. It will block until an error occurs or it is shut down.
func (s *Server) Start() error {
	slog.Info("Starting server", "addr", s.httpServer.Addr)
	if err := s.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

// Shutdown gracefully shuts down the server without interrupting any active connections.
func (s *Server) Shutdown(ctx context.Context) error {
	slog.Info("Shutting down server gracefully...")
	return s.httpServer.Shutdown(ctx)
}

// healthHandler indicates if the application is running
func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status": "ok"}`))
}

// readyHandler indicates if the application is ready to accept traffic
func readyHandler(w http.ResponseWriter, r *http.Request) {
	// In the future: Add checks for DB/Redis/MinIO connections here
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status": "ready"}`))
}
