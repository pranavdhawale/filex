package worker

import (
	"context"
	"log/slog"
	"time"

	"github.com/pranavdhawale/filex/internal/counter"
	"github.com/pranavdhawale/filex/internal/repository"
	"github.com/pranavdhawale/filex/internal/storage"
	"github.com/robfig/cron/v3"
)

type Dependencies struct {
	FileRepo    *repository.FileRepository
	ShareRepo   *repository.ShareRepository
	SessionRepo *repository.MultipartRepository
	Storage     *storage.Storage
	Counter     *counter.DownloadCounter
}

func StartScheduler(ctx context.Context, deps Dependencies) {
	ist, err := time.LoadLocation("Asia/Kolkata")
	if err != nil {
		slog.Error("Failed to load IST timezone", "error", err)
		ist = time.UTC
	}

	c := cron.New(cron.WithLocation(ist), cron.WithSeconds())

	// Every day at 00:00 IST — expiry, multipart cleanup, GC
	c.AddFunc("0 0 0 * * *", func() {
		slog.Info("Running daily cleanup workers")
		RunExpiry(ctx, deps.FileRepo, deps.ShareRepo, deps.Storage)
		RunMultipartCleanup(ctx, deps.SessionRepo, deps.Storage)
		RunGC(ctx, deps.FileRepo, deps.Storage)
	})

	// Every hour at :00 — flush download counter
	c.AddFunc("0 0 * * * *", func() {
		slog.Info("Flushing download counter")
		snapshot := deps.Counter.Snapshot()
		if len(snapshot) == 0 {
			return
		}
		if err := deps.FileRepo.BulkIncrementDownloadCounts(ctx, snapshot); err != nil {
			slog.Error("Failed to flush file download counts", "error", err)
		}
		if err := deps.ShareRepo.BulkIncrementDownloadCounts(ctx, snapshot); err != nil {
			slog.Error("Failed to flush share download counts", "error", err)
		}
		deps.Counter.Clear()
		slog.Info("Download counter flushed", "entries", len(snapshot))
	})

	c.Start()
	slog.Info("Scheduler started (IST timezone)")

	<-ctx.Done()
	slog.Info("Scheduler stopping...")
	c.Stop()

	// Final flush
	snapshot := deps.Counter.Snapshot()
	if len(snapshot) > 0 {
		deps.FileRepo.BulkIncrementDownloadCounts(context.Background(), snapshot)
		deps.Counter.Clear()
		slog.Info("Final counter flush completed", "entries", len(snapshot))
	}
}