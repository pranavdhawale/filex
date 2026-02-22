package workers

import (
	"context"
	"log/slog"
	"time"

	"github.com/pranavdhawale/bytefile/internal/repository"
	"github.com/pranavdhawale/bytefile/internal/storage"
)

// ExpiryWorker periodically checks for expired files, deletes them from MinIO,
// and then explicitly removes their document from MongoDB.
type ExpiryWorker struct {
	fileRepo *repository.FileRepository
	st       *storage.Storage
	interval time.Duration
	batch    int
}

func NewExpiryWorker(fileRepo *repository.FileRepository, st *storage.Storage, interval time.Duration) *ExpiryWorker {
	return &ExpiryWorker{
		fileRepo: fileRepo,
		st:       st,
		interval: interval,
		batch:    100, // retrieve up to 100 expired files per tick
	}
}

func (w *ExpiryWorker) Run(ctx context.Context) {
	slog.Info("Starting Expiry Worker", "interval", w.interval)
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("Context cancelled, Expiry Worker shutting down")
			return
		case <-ticker.C:
			w.processBatch(ctx)
		}
	}
}

func (w *ExpiryWorker) processBatch(ctx context.Context) {
	files, err := w.fileRepo.FindExpired(ctx, w.batch)
	if err != nil {
		slog.Error("ExpiryWorker: failed to find expired files", "error", err)
		return
	}

	if len(files) == 0 {
		return
	}

	slog.Info("ExpiryWorker: processing expired files", "count", len(files))

	for _, f := range files {
		// 1. Delete from MinIO
		err := w.st.RemoveObject(ctx, f.ObjectKey)
		if err != nil {
			slog.Error("ExpiryWorker: failed to remove object from MinIO", "object_key", f.ObjectKey, "error", err)
			continue // skip deleting from mongo so we retry later
		}

		// 2. Delete from MongoDB
		err = w.fileRepo.Delete(ctx, f.ID)
		if err != nil {
			slog.Error("ExpiryWorker: failed to delete document from MongoDB", "id", f.ID, "error", err)
		} else {
			slog.Info("ExpiryWorker: successfully purged expired file", "id", f.ID, "object_key", f.ObjectKey)
		}
	}
}
