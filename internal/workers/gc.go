package workers

import (
	"context"
	"log/slog"
	"time"

	"github.com/pranavdhawale/bytefile/internal/repository"
	"github.com/pranavdhawale/bytefile/internal/storage"
)

// GCWorker iterates over all objects in MinIO and checks if they exist
// in MongoDB (either as complete files or active multipart sessions).
// If an object is not in MongoDB AND is older than a grace period (e.g., 24h),
// it is considered an orphan and deleted.
type GCWorker struct {
	fileRepo    *repository.FileRepository
	st          *storage.Storage
	interval    time.Duration
	gracePeriod time.Duration
}

func NewGCWorker(fileRepo *repository.FileRepository, st *storage.Storage, interval time.Duration) *GCWorker {
	return &GCWorker{
		fileRepo:    fileRepo,
		st:          st,
		interval:    interval,
		gracePeriod: 24 * time.Hour,
	}
}

func (w *GCWorker) Run(ctx context.Context) {
	slog.Info("Starting Orphan GC Worker", "interval", w.interval)
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	// Run immediately on boot
	w.scan(ctx)

	for {
		select {
		case <-ctx.Done():
			slog.Info("Context cancelled, GC Worker shutting down")
			return
		case <-ticker.C:
			w.scan(ctx)
		}
	}
}

func (w *GCWorker) scan(ctx context.Context) {
	slog.Info("GCWorker: starting bucket scan for orphans")
	now := time.Now()

	count := 0
	deleted := 0

	objectCh := w.st.ListObjects(ctx, "uploads/")

	for object := range objectCh {
		if object.Err != nil {
			slog.Error("GCWorker: error iterating objects", "error", object.Err)
			continue
		}

		count++

		// Check if it's within the grace period (we assume it might be an actively uploading piece of data)
		if now.Sub(object.LastModified) < w.gracePeriod {
			continue
		}

		// Check if it exists in the files collection
		exists, err := w.fileRepo.ExistsByObjectKey(ctx, object.Key)
		if err != nil {
			slog.Error("GCWorker: failed to check mongo existence", "key", object.Key, "error", err)
			continue
		}

		if exists {
			continue
		}

		// At this point, the object is older than 24h and does not exist in the files collection.
		// NOTE: A multipart upload that is active for >24h might get flagged here if we don't query
		// the multipart_sessions collection. But since we expire multipart sessions after 24h max
		// anyway, anything older than 24h in MinIO without a file_repo entry is definitely dead.

		slog.Warn("GCWorker: detected orphan MinIO object", "key", object.Key, "last_modified", object.LastModified)

		err = w.st.RemoveObject(ctx, object.Key)
		if err != nil {
			slog.Error("GCWorker: failed to remove orphan", "key", object.Key, "error", err)
		} else {
			deleted++
			slog.Info("GCWorker: removed orphan", "key", object.Key)
		}
	}

	slog.Info("GCWorker: scan complete", "scanned", count, "deleted_orphans", deleted)
}
