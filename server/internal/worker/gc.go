package worker

import (
	"context"
	"log/slog"
	"time"

	"github.com/pranavdhawale/filex/internal/repository"
	"github.com/pranavdhawale/filex/internal/storage"
)

func RunGC(ctx context.Context, fileRepo *repository.FileRepository, st *storage.Storage) {
	objects := st.ListObjects(ctx, "uploads/")
	now := time.Now()
	deleted := 0

	for obj := range objects {
		if obj.Err != nil {
			slog.Error("GC: error listing object", "error", obj.Err)
			continue
		}
		if now.Sub(obj.LastModified) < 24*time.Hour {
			continue
		}
		exists, err := fileRepo.ExistsByObjectKey(ctx, obj.Key)
		if err != nil {
			slog.Error("GC: error checking object key", "key", obj.Key, "error", err)
			continue
		}
		if !exists {
			if err := st.RemoveObject(ctx, obj.Key); err != nil {
				slog.Error("GC: failed to delete orphan", "key", obj.Key, "error", err)
			} else {
				deleted++
			}
		}
	}
	slog.Info("GC: completed", "orphans_deleted", deleted)
}