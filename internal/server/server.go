package server

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/pranavdhawale/bytefile/internal/api"
	"github.com/pranavdhawale/bytefile/internal/config"
)

type Server struct {
	httpServer *http.Server
}

// New creates a new web server configured with standard routes.
func New(cfg *config.Config, uploadHandler *api.UploadHandler, downloadHandler *api.DownloadHandler) *Server {
	mux := http.NewServeMux()

	// Health and Readiness endpoints
	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("/ready", readyHandler)

	// API endpoints
	mux.HandleFunc("POST /upload/init", uploadHandler.HandleInit)
	mux.HandleFunc("POST /upload/complete", uploadHandler.HandleComplete)
	mux.HandleFunc("GET /f/{id}", downloadHandler.HandleDownload)

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
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
