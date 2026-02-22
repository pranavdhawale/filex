package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/pranavdhawale/bytefile/internal/config"
	"github.com/pranavdhawale/bytefile/internal/crypto"
	"github.com/pranavdhawale/bytefile/internal/repository"
	"github.com/pranavdhawale/bytefile/internal/storage"
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
	ExpiresAt      time.Time `json:"expires_at"`
}

func (h *DownloadHandler) HandleDownload(w http.ResponseWriter, r *http.Request) {
	// Extract the ID from the path. Route is registered as GET /f/{id}
	fileID := r.PathValue("id")
	if fileID == "" {
		http.Error(w, "missing file id", http.StatusBadRequest)
		return
	}

	// Validate UUID format
	if _, err := uuid.Parse(fileID); err != nil {
		slog.Warn("Invalid file ID format requested", "id", fileID, "ip", r.RemoteAddr)
		http.Error(w, "invalid file id format", http.StatusBadRequest)
		return
	}

	// Fetch file metadata
	file, err := h.fileRepo.GetByID(r.Context(), fileID)
	if err != nil {
		slog.Warn("File not found or expired", "id", fileID, "error", err)
		http.Error(w, "file not found", http.StatusNotFound)
		return
	}

	// Calculate new TTL (+1 day from whichever is later: current expiry or now)
	now := time.Now().UTC()
	baseExpiry := file.ExpiresAt
	if now.After(baseExpiry) { // Strictly speaking, Mongo TTL should delete this, but just in case
		baseExpiry = now
	}
	newExpiry := baseExpiry.Add(24 * time.Hour)

	// Atomically extend TTL in DB
	if err := h.fileRepo.ExtendTTL(r.Context(), fileID, newExpiry); err != nil {
		slog.Error("Failed to extend TTL", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	// Generate MinIO pre-signed GET URL (valid for 1 hour for download)
	downloadURL, err := h.storage.GenerateSignedGetURL(r.Context(), file.ObjectKey, 1*time.Hour)
	if err != nil {
		slog.Error("Failed to generate signed GET URL", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

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
		DownloadURL:    downloadURL,
		FEK:            responseFEK,
		EncryptionMode: file.EncryptionMode,
		ExpiresAt:      newExpiry,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
