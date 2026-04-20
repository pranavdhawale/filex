package models

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

type File struct {
	ID            bson.ObjectID     `bson:"_id"                   json:"id"`
	ObjectKey     string            `bson:"object_key"            json:"objectKey"`
	Filename      string            `bson:"filename"              json:"filename"`
	Slug          string            `bson:"slug"                  json:"slug"`
	Size          int64             `bson:"size"                  json:"size"`
	ContentType   string            `bson:"content_type"          json:"contentType"`
	TotalChunks   int               `bson:"total_chunks"          json:"totalChunks"`
	ChunkSize     int               `bson:"chunk_size"            json:"chunkSize"`
	EncryptedFEK  string            `bson:"encrypted_fek"         json:"encryptedFek"`
	Salt          string            `bson:"salt"                  json:"salt"`
	UploadID      string            `bson:"upload_id"             json:"uploadId"`
	ChunkHashRoot string            `bson:"chunk_hash_root"       json:"chunkHashRoot"`
	ShareID       *bson.ObjectID    `bson:"share_id,omitempty"    json:"shareId,omitempty"`
	MaxDownloads  int               `bson:"max_downloads"         json:"maxDownloads"`
	DownloadCount int               `bson:"download_count"        json:"downloadCount"`
	Status        string            `bson:"status"                json:"status"`
	CreatedAt     time.Time         `bson:"created_at"            json:"createdAt"`
	LastAccessed  time.Time         `bson:"last_accessed"         json:"lastAccessed"`
	ExpiresAt     time.Time         `bson:"expires_at"            json:"expiresAt"`
}

type Share struct {
	ID            bson.ObjectID    `bson:"_id"                   json:"id"`
	Slug          string           `bson:"slug"                  json:"slug"`
	FileIDs       []bson.ObjectID  `bson:"file_ids"              json:"fileIds"`
	Salt          string           `bson:"salt"                  json:"salt"`
	MaxDownloads  int              `bson:"max_downloads"         json:"maxDownloads"`
	DownloadCount int              `bson:"download_count"        json:"downloadCount"`
	ExpiresAt     time.Time        `bson:"expires_at"            json:"expiresAt"`
	CreatedAt     time.Time        `bson:"created_at"            json:"createdAt"`
}

type MultipartSession struct {
	ID        bson.ObjectID `bson:"_id"                   json:"id"`
	FileID    bson.ObjectID `bson:"file_id"               json:"fileId"`
	UploadID  string        `bson:"upload_id"              json:"uploadId"`
	CreatedAt time.Time     `bson:"created_at"             json:"createdAt"`
	ExpiresAt time.Time     `bson:"expires_at"             json:"expiresAt"`
}