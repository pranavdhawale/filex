package worker

import (
	"context"
	"log/slog"

	"github.com/pranavdhawale/filex/internal/repository"
	"github.com/pranavdhawale/filex/internal/storage"
)

func RunExpiry(ctx context.Context, fileRepo *repository.FileRepository, shareRepo *repository.ShareRepository, st *storage.Storage) {
	// Delete expired files
	files, err := fileRepo.FindExpired(ctx, 100)
	if err != nil {
		slog.Error("Expiry: failed to find expired files", "error", err)
		return
	}
	for _, f := range files {
		if err := st.RemoveObject(ctx, f.ObjectKey); err != nil {
			slog.Error("Expiry: failed to remove object", "key", f.ObjectKey, "error", err)
			continue
		}
		if err := fileRepo.Delete(ctx, f.ID); err != nil {
			slog.Error("Expiry: failed to delete file doc", "id", f.ID.Hex(), "error", err)
		}
	}

	// Delete expired shares
	shares, err := shareRepo.FindExpired(ctx, 100)
	if err != nil {
		slog.Error("Expiry: failed to find expired shares", "error", err)
		return
	}
	for _, s := range shares {
		if err := shareRepo.Delete(ctx, s.ID); err != nil {
			slog.Error("Expiry: failed to delete share doc", "id", s.ID.Hex(), "error", err)
		}
	}

	slog.Info("Expiry: completed", "files_deleted", len(files), "shares_deleted", len(shares))
}