package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/pranavdhawale/filex/internal/config"
	"github.com/pranavdhawale/filex/internal/counter"
	"github.com/pranavdhawale/filex/internal/database"
	"github.com/pranavdhawale/filex/internal/handler"
	"github.com/pranavdhawale/filex/internal/ratelimit"
	"github.com/pranavdhawale/filex/internal/repository"
	"github.com/pranavdhawale/filex/internal/server"
	"github.com/pranavdhawale/filex/internal/storage"
	"github.com/pranavdhawale/filex/internal/worker"
)

func main() {
	cfg := config.Load()
	initLogger(cfg.Environment)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Initialize MinIO
	minioStorage, err := storage.NewStorage(cfg)
	if err != nil {
		slog.Error("Failed to initialize MinIO", "error", err)
		os.Exit(1)
	}
	slog.Info("MinIO storage verified")

	// Initialize MongoDB
	dbClient, err := database.Connect(ctx, cfg.MongoURI)
	if err != nil {
		slog.Error("Failed to connect to MongoDB", "error", err)
		os.Exit(1)
	}
	defer database.Close(context.Background(), dbClient)

	db := database.GetDatabase(dbClient)
	fileRepo := repository.NewFileRepository(db)
	shareRepo := repository.NewShareRepository(db)
	sessionRepo := repository.NewMultipartRepository(db)

	// Initialize indexes
	if err := fileRepo.InitializeIndexes(ctx); err != nil {
		slog.Error("Failed to init file indexes", "error", err)
		os.Exit(1)
	}
	if err := shareRepo.InitializeIndexes(ctx); err != nil {
		slog.Error("Failed to init share indexes", "error", err)
		os.Exit(1)
	}
	if err := sessionRepo.InitializeIndexes(ctx); err != nil {
		slog.Error("Failed to init session indexes", "error", err)
		os.Exit(1)
	}

	// Initialize rate limiter (in-memory)
	limiter := ratelimit.NewRateLimiter(64)

	// Initialize download counter
	dlCounter := counter.NewDownloadCounter()

	// Initialize handlers
	fileHandler := handler.NewFileHandler(fileRepo, shareRepo, sessionRepo, minioStorage, dlCounter)
	shareHandler := handler.NewShareHandler(shareRepo, fileRepo, minioStorage, dlCounter)

	// Initialize health checker
	health := handler.NewHealthChecker(dbClient, minioStorage.Client(), cfg.MinioBucket)

	// Start server
	srv := server.New(cfg, fileHandler, shareHandler, health, limiter)

	// Start workers in goroutines
	workerDeps := worker.Dependencies{
		FileRepo:    fileRepo,
		ShareRepo:   shareRepo,
		SessionRepo: sessionRepo,
		Storage:     minioStorage,
		Counter:     dlCounter,
	}
	go worker.StartScheduler(ctx, workerDeps)

	// Start HTTP server
	errChan := make(chan error, 1)
	go func() {
		if err := srv.Start(); err != nil {
			errChan <- err
		}
	}()

	select {
	case err := <-errChan:
		slog.Error("Server fatal error", "error", err)
		os.Exit(1)
	case <-ctx.Done():
		slog.Info("Shutdown signal received")
		health.SetNotReady()
		time.Sleep(3 * time.Second) // Drain period
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		srv.Shutdown(shutdownCtx)
		slog.Info("Server stopped cleanly")
	}
}

func initLogger(env string) {
	level := slog.LevelInfo
	if env == "development" {
		level = slog.LevelDebug
	}
	h := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})
	slog.SetDefault(slog.New(h))
}