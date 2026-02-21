package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/pranavdhawale/bytefile/internal/api"
	"github.com/pranavdhawale/bytefile/internal/config"
	"github.com/pranavdhawale/bytefile/internal/database"
	"github.com/pranavdhawale/bytefile/internal/logger"
	"github.com/pranavdhawale/bytefile/internal/repository"
	"github.com/pranavdhawale/bytefile/internal/server"
	"github.com/pranavdhawale/bytefile/internal/storage"
	"go.mongodb.org/mongo-driver/mongo"
)

func main() {
	workerFlag := flag.String("worker", "", "Start as a specific worker (expiry, multipart, gc)")
	flag.Parse()

	cfg := config.Load()
	logger.Init(cfg.Environment)

	// Create root context that listens for termination signals
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Initialize Storage (MinIO)
	minioStorage, err := storage.NewStorage(cfg)
	if err != nil {
		slog.Error("Failed to initialize MinIO storage", "error", err)
		os.Exit(1)
	}

	// Verify bucket existence (Retry a few times as MinIO might be starting)
	var bucketExists bool
	for i := 0; i < 5; i++ {
		exists, err := minioStorage.BucketExists(ctx)
		if err == nil && exists {
			bucketExists = true
			break
		}
		slog.Warn("Waiting for MinIO bucket 'files' to be ready...", "attempt", i+1)
		time.Sleep(2 * time.Second)
	}
	if !bucketExists {
		slog.Error("MinIO bucket 'files' does not exist or is not reachable")
		os.Exit(1)
	}
	slog.Info("MinIO storage verified")

	// Initialize MongoDB Connection
	dbClient, err := database.Connect(ctx, cfg.MongoURI)
	if err != nil {
		slog.Error("Failed to connect to MongoDB", "error", err)
		os.Exit(1)
	}
	defer func() {
		// Ensure DB connection is closed on exit
		if err := database.Close(context.Background(), dbClient); err != nil {
			slog.Error("Failed to cleanly disconnect from MongoDB", "error", err)
		}
	}()

	// Initialize database layer and indexes
	db := database.GetDatabase(dbClient)
	fileRepo := repository.NewFileRepository(db)
	if err := fileRepo.InitializeIndexes(ctx); err != nil {
		slog.Error("Failed to initialize file indexes", "error", err)
		os.Exit(1)
	}

	multipartRepo := repository.NewMultipartRepository(db)
	if err := multipartRepo.InitializeIndexes(ctx); err != nil {
		slog.Error("Failed to initialize multipart indexes", "error", err)
		os.Exit(1)
	}

	switch *workerFlag {
	case "expiry":
		slog.Info("Starting worker", "type", "expiry")
		runWorker(ctx, "expiry", dbClient, minioStorage)
	case "multipart":
		slog.Info("Starting worker", "type", "multipart")
		runWorker(ctx, "multipart", dbClient, minioStorage)
	case "gc":
		slog.Info("Starting worker", "type", "gc")
		runWorker(ctx, "gc", dbClient, minioStorage)
	case "":
		slog.Info("Starting API server")
		runAPI(ctx, cfg, dbClient, minioStorage)
	default:
		slog.Error("Unknown worker type specified", "worker", *workerFlag)
		os.Exit(1)
	}
}

// runWorker is a placeholder for running background worker processes
// It blocks until the context is canceled.
func runWorker(ctx context.Context, workerType string, dbClient *mongo.Client, st *storage.Storage) {
	// TODO: Initialize specific worker implementations here

	// Block until signal is received
	<-ctx.Done()
	slog.Info("Shutting down worker gracefully", "type", workerType)
}

// runAPI starts the HTTP server and manages graceful shutdown
func runAPI(ctx context.Context, cfg *config.Config, dbClient *mongo.Client, st *storage.Storage) {
	// Initialize repositories
	db := database.GetDatabase(dbClient)
	fileRepo := repository.NewFileRepository(db)
	multipartRepo := repository.NewMultipartRepository(db)

	// Initialize handlers
	uploadHandler := api.NewUploadHandler(fileRepo, multipartRepo, st, cfg)
	downloadHandler := api.NewDownloadHandler(fileRepo, st, cfg)

	srv := server.New(cfg, uploadHandler, downloadHandler)

	// Start server in a separate goroutine
	errChan := make(chan error, 1)
	go func() {
		if err := srv.Start(); err != nil {
			errChan <- err
		}
	}()

	// Wait for an OS interrupt or a fatal server error
	select {
	case err := <-errChan:
		slog.Error("Server encountered a fatal error", "error", err)
		os.Exit(1)
	case <-ctx.Done():
		slog.Info("Received termination signal")

		// Create a timeout context for the shutdown
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := srv.Shutdown(shutdownCtx); err != nil {
			slog.Error("Server shutdown error", "error", err)
			os.Exit(1)
		}
		slog.Info("Server stopped cleanly")
	}
}
