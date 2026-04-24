package handler

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/pranavdhawale/filex/internal/counter"
	"github.com/pranavdhawale/filex/internal/models"
	"github.com/pranavdhawale/filex/internal/repository"
	"github.com/pranavdhawale/filex/internal/slug"
	"github.com/pranavdhawale/filex/internal/storage"
)

type FileHandler struct {
	fileRepo     *repository.FileRepository
	shareRepo    *repository.ShareRepository
	sessionRepo  *repository.MultipartRepository
	storage      *storage.Storage
	counter      *counter.DownloadCounter
	maxChunkSize int64
}

func NewFileHandler(
	fileRepo *repository.FileRepository,
	shareRepo *repository.ShareRepository,
	sessionRepo *repository.MultipartRepository,
	storage *storage.Storage,
	counter *counter.DownloadCounter,
	maxChunkSize int64,
) *FileHandler {
	return &FileHandler{
		fileRepo:     fileRepo,
		shareRepo:    shareRepo,
		sessionRepo:  sessionRepo,
		storage:      storage,
		counter:      counter,
		maxChunkSize: maxChunkSize,
	}
}

type InitUploadRequest struct {
	Filename     string `json:"filename"`
	Size         int64  `json:"size"`
	TTLSeconds   int    `json:"ttl_seconds"`
	ContentType  string `json:"content_type"`
	EncryptedFEK string `json:"encrypted_fek"`
	Salt         string `json:"salt"`
	ChunkSize    int    `json:"chunk_size"`
	TotalChunks  int    `json:"total_chunks"`
}

type InitUploadResponse struct {
	FileID    string `json:"file_id"`
	UploadID  string `json:"upload_id"`
	UploadURL string `json:"upload_url"`
	Slug      string `json:"slug"`
}

func (h *FileHandler) HandleInit(w http.ResponseWriter, r *http.Request) {
	var req InitUploadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Size <= 0 || req.Size > 5*1024*1024*1024 {
		writeError(w, http.StatusBadRequest, "invalid file size (max 5GB)")
		return
	}
	validTTLs := map[int]bool{1800: true, 3600: true, 86400: true}
	if !validTTLs[req.TTLSeconds] {
		writeError(w, http.StatusBadRequest, "invalid TTL (must be 1800, 3600, or 86400)")
		return
	}
	if req.Filename == "" {
		writeError(w, http.StatusBadRequest, "filename required")
		return
	}
	if req.EncryptedFEK == "" || req.Salt == "" {
		writeError(w, http.StatusBadRequest, "encrypted_fek and salt required")
		return
	}

	filename := sanitizeFilename(req.Filename)
	fileID := bson.NewObjectID()
	objectKey := fmt.Sprintf("uploads/%s", fileID.Hex())

	slugStr, err := slug.GenerateFileSlug()
	if err != nil {
		slog.Error("Failed to generate slug", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	// Check slug uniqueness (extremely rare collision, try once more)
	exists, err := h.fileRepo.SlugExists(r.Context(), slugStr)
	if err != nil {
		slog.Error("Failed to check slug", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if exists {
		slugStr, err = slug.GenerateFileSlug()
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}
	}

	// Create multipart upload in MinIO
	uploadID, err := h.storage.CreateMultipartUpload(r.Context(), objectKey, "application/octet-stream")
	if err != nil {
		slog.Error("Failed to create multipart upload", "error", err)
		writeError(w, http.StatusInternalServerError, "storage error")
		return
	}

	expiresAt := time.Now().Add(time.Duration(req.TTLSeconds) * time.Second).UTC()
	chunkSize := req.ChunkSize
	if chunkSize == 0 {
		chunkSize = 10 * 1024 * 1024
	}

	session := &models.MultipartSession{
		ID:        bson.NewObjectID(),
		FileID:    fileID,
		UploadID:  uploadID,
		CreatedAt: time.Now().UTC(),
		ExpiresAt: expiresAt,
	}
	if err := h.sessionRepo.Insert(r.Context(), session); err != nil {
		_ = h.storage.AbortMultipartUpload(r.Context(), objectKey, uploadID)
		slog.Error("Failed to store session", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	file := &models.File{
		ID:            fileID,
		ObjectKey:     objectKey,
		Filename:      filename,
		Slug:          slugStr,
		Size:          req.Size,
		ContentType:   req.ContentType,
		TotalChunks:   req.TotalChunks,
		ChunkSize:     chunkSize,
		EncryptedFEK:  req.EncryptedFEK,
		Salt:          req.Salt,
		UploadID:      uploadID,
		Status:        "uploading",
		MaxDownloads:  0,
		DownloadCount: 0,
		CreatedAt:     time.Now().UTC(),
		LastAccessed:  time.Now().UTC(),
		ExpiresAt:     expiresAt,
	}
	if err := h.fileRepo.Insert(r.Context(), file); err != nil {
		_ = h.storage.AbortMultipartUpload(r.Context(), objectKey, uploadID)
		_ = h.sessionRepo.Delete(r.Context(), session.ID)
		slog.Error("Failed to insert file", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeJSON(w, http.StatusOK, InitUploadResponse{
		FileID:    fileID.Hex(),
		UploadID:  uploadID,
		UploadURL: fmt.Sprintf("/upload/%s", uploadID),
		Slug:      slugStr,
	})
}

type CompleteUploadRequest struct {
	FileID      string   `json:"file_id"`
	ChunkHashes []string `json:"chunk_hashes"`
}

type CompleteUploadResponse struct {
	Slug               string    `json:"slug"`
	ExpiresAt          time.Time `json:"expires_at"`
	PassphraseRequired bool      `json:"passphrase_required"`
}

func (h *FileHandler) HandleComplete(w http.ResponseWriter, r *http.Request) {
	var req CompleteUploadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	fileObjID, err := bson.ObjectIDFromHex(req.FileID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid file_id")
		return
	}

	file, err := h.fileRepo.GetByID(r.Context(), fileObjID)
	if err != nil || file == nil {
		writeError(w, http.StatusNotFound, "file not found")
		return
	}

	if file.Status == "available" {
		writeJSON(w, http.StatusOK, CompleteUploadResponse{
			Slug:               file.Slug,
			ExpiresAt:          file.ExpiresAt,
			PassphraseRequired: true,
		})
		return
	}

	// Compute Merkle root from chunk hashes
	chunkHashRoot := computeMerkleRoot(req.ChunkHashes)

	uploadID := file.UploadID

	// Complete multipart upload in MinIO
	realParts, err := h.storage.ListParts(r.Context(), file.ObjectKey, uploadID)
	if err != nil {
		slog.Error("Failed to list parts", "error", err)
		writeError(w, http.StatusInternalServerError, "storage error")
		return
	}
	if err := h.storage.CompleteMultipartUpload(r.Context(), file.ObjectKey, uploadID, realParts); err != nil {
		slog.Error("Failed to complete upload", "error", err)
		writeError(w, http.StatusInternalServerError, "storage error")
		return
	}

	// Update file status atomically
	update := bson.M{
		"$set": bson.M{
			"status":          "available",
			"chunk_hash_root": chunkHashRoot,
		},
	}
	if err := h.fileRepo.UpdateByID(r.Context(), file.ID, update); err != nil {
		slog.Error("Failed to update file status", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	// Clean up the multipart session
	_ = h.sessionRepo.Delete(r.Context(), fileObjID)

	writeJSON(w, http.StatusOK, CompleteUploadResponse{
		Slug:               file.Slug,
		ExpiresAt:          file.ExpiresAt,
		PassphraseRequired: true,
	})
}

type GetAccessResponse struct {
	Filename           string    `json:"filename"`
	Size               int64     `json:"size"`
	ContentType        string    `json:"contentType"`
	EncryptedFEK       string    `json:"encryptedFek"`
	Salt               string    `json:"salt"`
	ChunkSize          int       `json:"chunkSize"`
	TotalChunks        int       `json:"totalChunks"`
	DownloadURL        string    `json:"downloadUrl"`
	ExpiresAt          time.Time `json:"expiresAt"`
	DownloadsRemaining int       `json:"downloadsRemaining"`
}

func (h *FileHandler) HandleChunkUpload(w http.ResponseWriter, r *http.Request) {
	fileID := r.URL.Query().Get("file_id")
	uploadID := r.URL.Query().Get("upload_id")
	partNumberStr := r.URL.Query().Get("part_number")

	if fileID == "" || uploadID == "" || partNumberStr == "" {
		writeError(w, http.StatusBadRequest, "missing file_id, upload_id, or part_number")
		return
	}

	partNumber, err := strconv.Atoi(partNumberStr)
	if err != nil || partNumber < 1 {
		writeError(w, http.StatusBadRequest, "invalid part_number")
		return
	}

	fileObjID, err := bson.ObjectIDFromHex(fileID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid file_id")
		return
	}

	file, err := h.fileRepo.GetByID(r.Context(), fileObjID)
	if err != nil || file == nil {
		writeError(w, http.StatusNotFound, "file not found")
		return
	}

	if file.UploadID != uploadID {
		writeError(w, http.StatusBadRequest, "upload_id mismatch")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, h.maxChunkSize)

	body, err := io.ReadAll(r.Body)
	if err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			writeError(w, http.StatusRequestEntityTooLarge, "chunk body too large")
			return
		}
		writeError(w, http.StatusBadRequest, "failed to read chunk data")
		return
	}

	eTag, err := h.storage.PutPart(r.Context(), file.ObjectKey, uploadID, partNumber, body)
	if err != nil {
		slog.Error("Failed to upload part", "error", err, "file_id", fileID, "part", partNumber)
		writeError(w, http.StatusInternalServerError, "storage error")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"etag": eTag})
}

func (h *FileHandler) HandleGetAccess(w http.ResponseWriter, r *http.Request) {
	slugVal := r.PathValue("slug")
	if slugVal == "" {
		writeError(w, http.StatusBadRequest, "missing slug")
		return
	}

	file, err := h.fileRepo.GetBySlug(r.Context(), slugVal)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if file == nil {
		writeError(w, http.StatusNotFound, "file not found")
		return
	}

	// Check download limit
	if file.MaxDownloads > 0 && file.DownloadCount >= file.MaxDownloads {
		writeError(w, http.StatusGone, "download limit reached")
		return
	}

	// Determine salt: if file belongs to a share, use share's salt
	salt := file.Salt

	// Generate presigned URL (5 min TTL) — before incrementing counter
	presignedURL, err := h.storage.GetPresignedURL(r.Context(), file.ObjectKey, file.Filename, 5*time.Minute)
	if err != nil {
		slog.Error("Failed to generate presigned URL", "error", err)
		writeError(w, http.StatusInternalServerError, "storage error")
		return
	}

	// Increment in-memory counter only after successful presigned URL generation
	h.counter.Increment(slugVal)

	downloadsRemaining := -1
	if file.MaxDownloads > 0 {
		downloadsRemaining = file.MaxDownloads - file.DownloadCount - 1
	}

	writeJSON(w, http.StatusOK, GetAccessResponse{
		Filename:           file.Filename,
		Size:               file.Size,
		ContentType:        file.ContentType,
		EncryptedFEK:       file.EncryptedFEK,
		Salt:               salt,
		ChunkSize:          file.ChunkSize,
		TotalChunks:        file.TotalChunks,
		DownloadURL:        presignedURL,
		ExpiresAt:          file.ExpiresAt,
		DownloadsRemaining: downloadsRemaining,
	})
}

func sanitizeFilename(name string) string {
	name = filepath.Base(name)
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
	if utf8.RuneCountInString(name) > 200 {
		runes := []rune(name)
		name = string(runes[:200])
	}
	return name
}

func computeMerkleRoot(hashes []string) string {
	if len(hashes) == 0 {
		return ""
	}
	h := sha256.New()
	for _, hash := range hashes {
		h.Write([]byte(hash))
	}
	return hex.EncodeToString(h.Sum(nil))
}