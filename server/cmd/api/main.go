package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/pranavdhawale/filex/internal/api"
	"github.com/pranavdhawale/filex/internal/config"
	"github.com/pranavdhawale/filex/internal/database"
	"github.com/pranavdhawale/filex/internal/logger"
	"github.com/pranavdhawale/filex/internal/ratelimit"
	"github.com/pranavdhawale/filex/internal/repository"
	"github.com/pranavdhawale/filex/internal/server"
	"github.com/pranavdhawale/filex/internal/storage"
	"github.com/pranavdhawale/filex/internal/workers"
	"go.mongodb.org/mongo-driver/v2/mongo"
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

// runWorker starts the specified background worker process.
// It blocks until the context is canceled.
func runWorker(ctx context.Context, workerType string, dbClient *mongo.Client, st *storage.Storage) {
	db := database.GetDatabase(dbClient)
	fileRepo := repository.NewFileRepository(db)
	multipartRepo := repository.NewMultipartRepository(db)

	switch workerType {
	case "expiry":
		worker := workers.NewExpiryWorker(fileRepo, st, 1*time.Minute)
		worker.Run(ctx)
	case "multipart":
		worker := workers.NewMultipartWorker(multipartRepo, st, 5*time.Minute)
		worker.Run(ctx)
	case "gc":
		worker := workers.NewGCWorker(fileRepo, st, 1*time.Hour)
		worker.Run(ctx)
	default:
		slog.Error("Unknown worker type", "type", workerType)
		os.Exit(1)
	}

	slog.Info("Worker stopped gracefully", "type", workerType)
}

// runAPI starts the HTTP server and manages graceful shutdown
func runAPI(ctx context.Context, cfg *config.Config, dbClient *mongo.Client, st *storage.Storage) {
	// Initialize Redis
	redisClient, err := database.InitRedis(ctx, cfg.RedisURI)
	if err != nil {
		slog.Error("Failed to connect to Redis", "error", err)
		os.Exit(1)
	}
	defer redisClient.Close()
	slog.Info("Successfully connected to Redis")

	// Initialize repositories
	db := database.GetDatabase(dbClient)
	fileRepo := repository.NewFileRepository(db)
	multipartRepo := repository.NewMultipartRepository(db)

	// Initialize Rate Limiter
	limiter := ratelimit.NewRateLimiter(redisClient)

	// Initialize handlers
	uploadHandler := api.NewUploadHandler(fileRepo, multipartRepo, st, cfg)
	downloadHandler := api.NewDownloadHandler(fileRepo, st, cfg)

	srv := server.New(cfg, uploadHandler, downloadHandler, limiter)

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
