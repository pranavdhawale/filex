package api

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/pranavdhawale/filex/internal/config"
	"github.com/pranavdhawale/filex/internal/crypto"
	"github.com/pranavdhawale/filex/internal/models"
	"github.com/pranavdhawale/filex/internal/repository"
	"github.com/pranavdhawale/filex/internal/storage"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type UploadHandler struct {
	fileRepo      *repository.FileRepository
	multipartRepo *repository.MultipartRepository
	storage       *storage.Storage
	cfg           *config.Config
}

func NewUploadHandler(
	fileRepo *repository.FileRepository,
	multipartRepo *repository.MultipartRepository,
	storage *storage.Storage,
	cfg *config.Config,
) *UploadHandler {
	return &UploadHandler{
		fileRepo:      fileRepo,
		multipartRepo: multipartRepo,
		storage:       storage,
		cfg:           cfg,
	}
}

type InitUploadRequest struct {
	Size           int64  `json:"size"`
	TTLDays        int    `json:"ttl_days"`
	EncryptionMode string `json:"encryption_mode"` // "anonymous" | "master"
	Filename       string `json:"filename"`
}

type InitUploadResponse struct {
	FileID      string `json:"file_id"`
	UploadID    string `json:"upload_id"`
	ChunkSize   int64  `json:"chunk_size"`
	TotalChunks int    `json:"total_chunks"`
}

// sanitizeFilename removes path separators and trims whitespace,
// keeping the original name + extension intact.
func sanitizeFilename(name string) string {
	// Strip any directory traversal
	name = filepath.Base(name)
	// Replace null bytes and control characters
	name = strings.Map(func(r rune) rune {
		if r < 32 || r == 127 {
			return '_'
		}
		return r
	}, name)
	name = strings.TrimSpace(name)
	if name == "" || name == "." {
		return "file"
	}
	// Truncate to 200 runes (leave room for ~xxxx suffix)
	if utf8.RuneCountInString(name) > 200 {
		runes := []rune(name)
		name = string(runes[:200])
	}
	return name
}

// randomHex returns n random hex characters.
func randomHex(n int) string {
	b := make([]byte, (n+1)/2)
	rand.Read(b)
	return hex.EncodeToString(b)[:n]
}

// resolveSlug generates a unique slug for the given filename.
// If the filename is available, it's returned as-is.
// On collision, appends a "~xxxx" suffix (4 random hex chars).
func (h *UploadHandler) resolveSlug(r *http.Request, filename string) (string, error) {
	slug := filename
	exists, err := h.fileRepo.SlugExists(r.Context(), slug)
	if err != nil {
		return "", err
	}
	if !exists {
		return slug, nil
	}
	// Collision — try up to 5 times with a random suffix
	for i := 0; i < 5; i++ {
		candidate := filename + "~" + randomHex(4)
		exists, err := h.fileRepo.SlugExists(r.Context(), candidate)
		if err != nil {
			return "", err
		}
		if !exists {
			return candidate, nil
		}
	}
	// Extremely unlikely — fall back to full uuid suffix
	return filename + "~" + randomHex(8), nil
}

func (h *UploadHandler) HandleInit(w http.ResponseWriter, r *http.Request) {
	var req InitUploadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Validation
	if req.Size <= 0 || req.Size > 5*1024*1024*1024 { // 5GB max
		http.Error(w, "invalid file size (max 5GB)", http.StatusBadRequest)
		return
	}

	validTTLs := map[int]bool{1: true, 7: true, 15: true}
	if !validTTLs[req.TTLDays] {
		http.Error(w, "invalid TTL days (must be 1, 7, or 15)", http.StatusBadRequest)
		return
	}

	if req.EncryptionMode != "anonymous" && req.EncryptionMode != "master" {
		http.Error(w, "invalid encryption mode", http.StatusBadRequest)
		return
	}

	filename := sanitizeFilename(req.Filename)

	fileID := uuid.New().String()
	objectName := fmt.Sprintf("uploads/%s", fileID)

	// Default chunk size 10MB (matches client/lib/upload.ts)
	const chunkSize = 10 * 1024 * 1024
	totalChunks := int((req.Size + chunkSize - 1) / chunkSize)

	// Create multipart upload in storage
	uploadID, err := h.storage.CreateMultipartUpload(r.Context(), objectName, "application/octet-stream")
	if err != nil {
		slog.Error("Failed to create multipart upload", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	// Store session in DB (including filename)
	expiresAt := time.Now().Add(time.Duration(req.TTLDays) * 24 * time.Hour)
	session := &models.MultipartSession{
		ID:           fileID,
		FileID:       fileID,
		Filename:     filename,
		UploadID:     uploadID,
		OriginalSize: req.Size,
		ChunkSize:    chunkSize,
		TotalChunks:  totalChunks,
		CreatedAt:    time.Now().UTC(),
		ExpiresAt:    expiresAt.UTC(),
	}

	if err := h.multipartRepo.Insert(r.Context(), session); err != nil {
		slog.Error("Failed to store multipart session", "error", err)
		// Try to abort minio upload as cleanup
		_ = h.storage.AbortMultipartUpload(r.Context(), objectName, uploadID)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	resp := InitUploadResponse{
		FileID:      fileID,
		UploadID:    uploadID,
		ChunkSize:   chunkSize,
		TotalChunks: totalChunks,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *UploadHandler) HandleChunkUpload(w http.ResponseWriter, r *http.Request) {
	fileID := r.URL.Query().Get("file_id")
	uploadID := r.URL.Query().Get("upload_id")
	partNumberStr := r.URL.Query().Get("part_number")

	if fileID == "" || uploadID == "" || partNumberStr == "" {
		http.Error(w, "missing parameters", http.StatusBadRequest)
		return
	}

	var partNumber int
	if _, err := fmt.Sscanf(partNumberStr, "%d", &partNumber); err != nil {
		http.Error(w, "invalid part number", http.StatusBadRequest)
		return
	}

	// Read body (limited to chunk size + overhead)
	const maxChunkSize = 15 * 1024 * 1024 // 15MB max for a 10MB chunk + overhead
	data, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxChunkSize))
	if err != nil {
		slog.Error("Failed to read chunk body", "error", err)
		http.Error(w, "failed to read body or body too large", http.StatusBadRequest)
		return
	}

	objectName := fmt.Sprintf("uploads/%s", fileID)
	etag, err := h.storage.PutPart(r.Context(), objectName, uploadID, partNumber, data)
	if err != nil {
		slog.Error("Failed to upload part to storage", "error", err)
		http.Error(w, "storage upload failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("ETag", etag)
	w.WriteHeader(http.StatusOK)
}

type CompleteUploadRequest struct {
	FileID         string         `json:"file_id"`
	Parts          []storage.Part `json:"parts"`
	EncryptedFEK   string         `json:"encrypted_fek,omitempty"` // For master mode
	PlainFEK       string         `json:"plain_fek,omitempty"`     // For anonymous mode
	EncryptionMode string         `json:"encryption_mode"`
}

func (h *UploadHandler) HandleComplete(w http.ResponseWriter, r *http.Request) {
	var req CompleteUploadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Fetch session
	session, err := h.multipartRepo.GetByID(r.Context(), req.FileID)
	if err != nil || session == nil {
		// It could be nil if ErrNoDocuments was swallowed, or an actual error.
		// Check if file already exists (idempotency)
		if existing, _ := h.fileRepo.GetByID(r.Context(), req.FileID); existing != nil {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"status": "already_completed", "file_id": req.FileID, "slug": existing.Slug})
			return
		}

		if err != nil {
			slog.Error("Database error checking session", "error", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		http.Error(w, "upload session not found or expired", http.StatusNotFound)
		return
	}

	objectName := fmt.Sprintf("uploads/%s", session.FileID)

	// Retrieve real ETags from MinIO to bypass CORS browser restrictions.
	// Browsers often fail to see the ETag header during PUT requests.
	realParts, err := h.storage.ListParts(r.Context(), objectName, session.UploadID)
	if err != nil {
		slog.Error("Failed to list multipart parts from storage", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	// Complete in MinIO using the real parts retrieved directly from S3
	if err := h.storage.CompleteMultipartUpload(r.Context(), objectName, session.UploadID, realParts); err != nil {
		slog.Error("Failed to complete multipart upload in storage", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	// Resolve unique slug for this file
	slug, err := h.resolveSlug(r, session.Filename)
	if err != nil {
		slog.Error("Failed to resolve slug", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	// Handle FEK wrapping
	var finalEncryptedFEK string
	if req.EncryptionMode == "anonymous" {
		if req.PlainFEK == "" {
			http.Error(w, "plain_fek required for anonymous mode", http.StatusBadRequest)
			return
		}
		wrapped, err := crypto.WrapKey([]byte(req.PlainFEK), h.cfg.ServerWrapKey)
		if err != nil {
			slog.Error("Failed to wrap FEK", "error", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		finalEncryptedFEK = wrapped
	} else {
		finalEncryptedFEK = req.EncryptedFEK
	}

	// Insert file record
	file := &models.File{
		ID:             session.FileID,
		ObjectKey:      objectName,
		Filename:       session.Filename,
		Slug:           slug,
		Size:           session.OriginalSize,
		TotalChunks:    session.TotalChunks,
		EncryptionMode: req.EncryptionMode,
		EncryptedFEK:   finalEncryptedFEK,
		CreatedAt:      time.Now().UTC(),
		LastAccessed:   time.Now().UTC(),
		ExpiresAt:      session.ExpiresAt,
	}

	if err := h.fileRepo.Insert(r.Context(), file); err != nil {
		if mongo.IsDuplicateKeyError(err) {
			// This happens if two concurrent requests both tried to complete the same file.
			// One won the race, the other hit the unique _id constraint on the files collection.
			// Treat as idempotently successful.
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"status": "already_completed", "file_id": req.FileID, "slug": slug})
			return
		}
		slog.Error("Failed to insert file record", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	// Delete session
	_ = h.multipartRepo.Delete(r.Context(), req.FileID)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "completed",
		"file_id": file.ID,
		"slug":    file.Slug,
	})
}
