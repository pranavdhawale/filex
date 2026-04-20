package worker

import (
	"context"
	"log/slog"

	"github.com/pranavdhawale/filex/internal/repository"
	"github.com/pranavdhawale/filex/internal/storage"
)

func RunMultipartCleanup(ctx context.Context, sessionRepo *repository.MultipartRepository, st *storage.Storage) {
	sessions, err := sessionRepo.FindExpired(ctx, 100)
	if err != nil {
		slog.Error("MultipartCleanup: failed to find expired sessions", "error", err)
		return
	}
	for _, s := range sessions {
		objectKey := "uploads/" + s.FileID.Hex()
		if err := st.AbortMultipartUpload(ctx, objectKey, s.UploadID); err != nil {
			slog.Error("MultipartCleanup: failed to abort upload", "id", s.ID.Hex(), "error", err)
		}
		if err := sessionRepo.Delete(ctx, s.ID); err != nil {
			slog.Error("MultipartCleanup: failed to delete session", "id", s.ID.Hex(), "error", err)
		}
	}
	slog.Info("MultipartCleanup: completed", "sessions_cleaned", len(sessions))
}