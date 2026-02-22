package models

import (
	"time"
)

// File represents the metadata of an uploaded file in MongoDB
type File struct {
	ID             string    `bson:"_id" json:"id"`
	ObjectKey      string    `bson:"object_key" json:"object_key"`
	Size           int64     `bson:"size" json:"size"`
	TotalChunks    int       `bson:"total_chunks" json:"total_chunks"`
	EncryptionMode string    `bson:"encryption_mode" json:"encryption_mode"` // "anonymous" | "master"
	EncryptedFEK   string    `bson:"encrypted_fek" json:"encrypted_fek"`
	EmailHash      *string   `bson:"email_hash,omitempty" json:"email_hash,omitempty"`
	CreatedAt      time.Time `bson:"created_at" json:"created_at"`
	LastAccessed   time.Time `bson:"last_accessed" json:"last_accessed"`
	ExpiresAt      time.Time `bson:"expires_at" json:"expires_at"`
}

// MultipartSession tracks the state of an S3 multipart upload in MongoDB
type MultipartSession struct {
	ID           string    `bson:"_id" json:"id"`
	FileID       string    `bson:"file_id" json:"file_id"`
	UploadID     string    `bson:"upload_id" json:"upload_id"`
	OriginalSize int64     `bson:"original_size" json:"original_size"`
	ChunkSize    int64     `bson:"chunk_size" json:"chunk_size"`
	TotalChunks  int       `bson:"total_chunks" json:"total_chunks"`
	CreatedAt    time.Time `bson:"created_at" json:"created_at"`
	ExpiresAt    time.Time `bson:"expires_at" json:"expires_at"`
}
