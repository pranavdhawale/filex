package api

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/pranavdhawale/filex/internal/config"
	"github.com/pranavdhawale/filex/internal/crypto"
	"github.com/pranavdhawale/filex/internal/repository"
	"github.com/pranavdhawale/filex/internal/storage"
)

type DownloadHandler struct {
	fileRepo *repository.FileRepository
	storage  *storage.Storage
	cfg      *config.Config
}

func NewDownloadHandler(fileRepo *repository.FileRepository, storage *storage.Storage, cfg *config.Config) *DownloadHandler {
	return &DownloadHandler{
		fileRepo: fileRepo,
		storage:  storage,
		cfg:      cfg,
	}
}

type DownloadResponse struct {
	DownloadURL    string    `json:"download_url"`
	FEK            string    `json:"fek"` // Plaintext FEK for anonymous, encrypted for master
	EncryptionMode string    `json:"encryption_mode"`
	Filename       string    `json:"filename"` // Original filename for the save dialog
	ExpiresAt      time.Time `json:"expires_at"`
}

func (h *DownloadHandler) HandleDownload(w http.ResponseWriter, r *http.Request) {
	// Extract the slug from the path. Route is registered as GET /f/{id}
	slug := r.PathValue("id")
	if slug == "" {
		http.Error(w, "missing file slug", http.StatusBadRequest)
		return
	}

	// Look up by slug (filename-based, no UUID fallback)
	file, err := h.fileRepo.GetBySlug(r.Context(), slug)
	if err != nil {
		slog.Error("Database error fetching file by slug", "slug", slug, "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	if file == nil {
		slog.Warn("File not found for slug", "slug", slug, "ip", r.RemoteAddr)
		http.Error(w, "file not found", http.StatusNotFound)
		return
	}

	// Ensure we only use original expiry
	newExpiry := file.ExpiresAt


	// Handle FEK unwrapping
	var responseFEK string
	if file.EncryptionMode == "anonymous" {
		unwrapped, err := crypto.UnwrapKey(file.EncryptedFEK, h.cfg.ServerWrapKey)
		if err != nil {
			slog.Error("Failed to unwrap FEK", "error", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		responseFEK = string(unwrapped)
	} else {
		// Master mode: Return the ciphertext as client holds the key
		responseFEK = file.EncryptedFEK
	}

	resp := DownloadResponse{
		DownloadURL:    "/api/download/stream/" + file.ID,
		FEK:            responseFEK,
		EncryptionMode: file.EncryptionMode,
		Filename:       file.Filename,
		ExpiresAt:      newExpiry,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *DownloadHandler) HandleStreamDownload(w http.ResponseWriter, r *http.Request) {
	fileID := r.PathValue("id")
	if fileID == "" {
		http.Error(w, "missing file id", http.StatusBadRequest)
		return
	}

	file, err := h.fileRepo.GetByID(r.Context(), fileID)
	if err != nil {
		http.Error(w, "file not found", http.StatusNotFound)
		return
	}

	reader, size, contentType, err := h.storage.GetObject(r.Context(), file.ObjectKey)
	if err != nil {
		slog.Error("Failed to get object from storage", "error", err)
		http.Error(w, "storage error", http.StatusInternalServerError)
		return
	}
	defer reader.Close()

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Length", fmt.Sprintf("%d", size))
	w.WriteHeader(http.StatusOK)

	// Stream the data
	if _, err := io.Copy(w, reader); err != nil {
		slog.Error("Failed to stream object to client", "error", err)
	}
}
