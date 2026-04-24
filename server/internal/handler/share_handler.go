package handler

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/pranavdhawale/filex/internal/counter"
	"github.com/pranavdhawale/filex/internal/models"
	"github.com/pranavdhawale/filex/internal/repository"
	"github.com/pranavdhawale/filex/internal/slug"
	"github.com/pranavdhawale/filex/internal/storage"
)

type ShareHandler struct {
	shareRepo *repository.ShareRepository
	fileRepo  *repository.FileRepository
	storage   *storage.Storage
	counter   *counter.DownloadCounter
}

func NewShareHandler(
	shareRepo *repository.ShareRepository,
	fileRepo *repository.FileRepository,
	storage *storage.Storage,
	counter *counter.DownloadCounter,
) *ShareHandler {
	return &ShareHandler{
		shareRepo: shareRepo,
		fileRepo:  fileRepo,
		storage:   storage,
		counter:   counter,
	}
}

type CreateShareRequest struct {
	FileIDs      []string `json:"file_ids"`
	TTLSeconds   int      `json:"ttl_seconds"`
	MaxDownloads int      `json:"max_downloads"`
}

type CreateShareResponse struct {
	ShareSlug string    `json:"shareSlug"`
	ExpiresAt time.Time `json:"expiresAt"`
}

func (h *ShareHandler) HandleCreate(w http.ResponseWriter, r *http.Request) {
	var req CreateShareRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if len(req.FileIDs) == 0 {
		writeError(w, http.StatusBadRequest, "at least one file_id required")
		return
	}
	if len(req.FileIDs) > 20 {
		writeError(w, http.StatusBadRequest, "max 20 files per share")
		return
	}

	validTTLs := map[int]bool{1800: true, 3600: true, 86400: true}
	if !validTTLs[req.TTLSeconds] {
		writeError(w, http.StatusBadRequest, "invalid TTL")
		return
	}

	fileObjIDs := make([]bson.ObjectID, 0, len(req.FileIDs))
	for _, fid := range req.FileIDs {
		objID, err := bson.ObjectIDFromHex(fid)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid file_id: "+fid)
			return
		}
		fileObjIDs = append(fileObjIDs, objID)
	}

	shareSlug, err := slug.GenerateShareSlug()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	expiresAt := time.Now().Add(time.Duration(req.TTLSeconds) * time.Second).UTC()
	salt := generateSalt()

	share := &models.Share{
		ID:            bson.NewObjectID(),
		Slug:          shareSlug,
		FileIDs:       fileObjIDs,
		Salt:          salt,
		MaxDownloads:  req.MaxDownloads,
		DownloadCount: 0,
		ExpiresAt:     expiresAt,
		CreatedAt:     time.Now().UTC(),
	}

	if err := h.shareRepo.Insert(r.Context(), share); err != nil {
		slog.Error("Failed to create share", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	// Link files to this share
	for _, fid := range fileObjIDs {
		if err := h.fileRepo.SetShareID(r.Context(), fid, share.ID); err != nil {
			slog.Error("Failed to link file to share", "fileId", fid.Hex(), "error", err)
		}
	}

	writeJSON(w, http.StatusOK, CreateShareResponse{
		ShareSlug: shareSlug,
		ExpiresAt: expiresAt,
	})
}

type ShareFileEntry struct {
	Slug        string `json:"slug"`
	Filename    string `json:"filename"`
	Size        int64  `json:"size"`
	ContentType string `json:"contentType"`
}

type GetShareResponse struct {
	Files              []ShareFileEntry `json:"files"`
	Salt               string           `json:"salt"`
	ExpiresAt          time.Time        `json:"expiresAt"`
	DownloadsRemaining int              `json:"downloadsRemaining"`
}

func (h *ShareHandler) HandleGet(w http.ResponseWriter, r *http.Request) {
	shareSlug := r.PathValue("shareSlug")
	if shareSlug == "" {
		writeError(w, http.StatusBadRequest, "missing share slug")
		return
	}

	share, err := h.shareRepo.GetBySlug(r.Context(), shareSlug)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if share == nil {
		writeError(w, http.StatusNotFound, "share not found")
		return
	}

	// Check download limit
	if share.MaxDownloads > 0 && share.DownloadCount >= share.MaxDownloads {
		writeError(w, http.StatusGone, "download limit reached")
		return
	}

	fileEntries := make([]ShareFileEntry, 0, len(share.FileIDs))
	for _, fid := range share.FileIDs {
		file, err := h.fileRepo.GetByID(r.Context(), fid)
		if err != nil || file == nil {
			continue
		}
		fileEntries = append(fileEntries, ShareFileEntry{
			Slug:        file.Slug,
			Filename:    file.Filename,
			Size:        file.Size,
			ContentType: file.ContentType,
		})
	}

	// Increment counter after all data is successfully fetched
	h.counter.Increment(shareSlug)

	downloadsRemaining := -1
	if share.MaxDownloads > 0 {
		downloadsRemaining = share.MaxDownloads - share.DownloadCount - 1
	}

	writeJSON(w, http.StatusOK, GetShareResponse{
		Files:              fileEntries,
		Salt:               share.Salt,
		ExpiresAt:          share.ExpiresAt,
		DownloadsRemaining: downloadsRemaining,
	})
}

func generateSalt() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}