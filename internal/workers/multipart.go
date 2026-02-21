package workers

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/pranavdhawale/bytefile/internal/repository"
	"github.com/pranavdhawale/bytefile/internal/storage"
)

// MultipartWorker periodically checks for abandoned multipart uploads,
// aborts them in MinIO, and removes the session from MongoDB.
type MultipartWorker struct {
	multipartRepo *repository.MultipartRepository
	st            *storage.Storage
	interval      time.Duration
	batch         int
}

func NewMultipartWorker(multipartRepo *repository.MultipartRepository, st *storage.Storage, interval time.Duration) *MultipartWorker {
	return &MultipartWorker{
		multipartRepo: multipartRepo,
		st:            st,
		interval:      interval,
		batch:         100,
	}
}

func (w *MultipartWorker) Run(ctx context.Context) {
	slog.Info("Starting Multipart Cleanup Worker", "interval", w.interval)
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("Context cancelled, Multipart Worker shutting down")
			return
		case <-ticker.C:
			w.processBatch(ctx)
		}
	}
}

func (w *MultipartWorker) processBatch(ctx context.Context) {
	sessions, err := w.multipartRepo.FindExpired(ctx, w.batch)
	if err != nil {
		slog.Error("MultipartWorker: failed to find expired sessions", "error", err)
		return
	}

	if len(sessions) == 0 {
		return
	}

	slog.Info("MultipartWorker: processing expired sessions", "count", len(sessions))

	for _, s := range sessions {
		objectKey := fmt.Sprintf("uploads/%s", s.FileID)

		// 1. Abort in MinIO
		err := w.st.AbortMultipartUpload(ctx, objectKey, s.UploadID)
		if err != nil {
			slog.Error("MultipartWorker: failed to abort upload in MinIO", "upload_id", s.UploadID, "error", err)
			continue
		}

		// 2. Delete from MongoDB
		err = w.multipartRepo.Delete(ctx, s.ID)
		if err != nil {
			slog.Error("MultipartWorker: failed to delete session from MongoDB", "session_id", s.ID, "error", err)
		} else {
			slog.Info("MultipartWorker: successfully aborted abandoned multipart upload", "file_id", s.FileID)
		}
	}
}
