package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime"
	"net/url"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/pranavdhawale/filex/internal/config"
)

type Storage struct {
	client            *minio.Client
	core              *minio.Core
	bucket            string
	downloadProxyPath string // e.g. "/minio" to rewrite presigned URLs through a reverse proxy
}

// NewStorage initializes a new MinIO storage client
func NewStorage(cfg *config.Config) (*Storage, error) {
	options := &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.MinioAccessKey, cfg.MinioSecretKey, ""),
		Secure: false,
	}
	client, err := minio.New(cfg.MinioEndpoint, options)
	if err != nil {
		return nil, fmt.Errorf("failed to create minio client: %w", err)
	}

	core, err := minio.NewCore(cfg.MinioEndpoint, options)
	if err != nil {
		return nil, fmt.Errorf("failed to create minio core client: %w", err)
	}

	s := &Storage{
		client:            client,
		core:              core,
		bucket:            cfg.MinioBucket,
		downloadProxyPath: cfg.MinioDownloadPrefix,
	}

	// Ensure bucket exists
	if err := s.ensureBucket(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to ensure bucket %s: %w", s.bucket, err)
	}

	return s, nil
}

// Client returns the underlying minio.Client for health checks
func (s *Storage) Client() *minio.Client {
	return s.client
}

// ensureBucket checks if the bucket exists, creates it if not.
func (s *Storage) ensureBucket(ctx context.Context) error {
	exists, err := s.client.BucketExists(ctx, s.bucket)
	if err != nil {
		return err
	}

	if !exists {
		err = s.client.MakeBucket(ctx, s.bucket, minio.MakeBucketOptions{})
		if err != nil {
			return err
		}
	}

	return nil
}

// CreateMultipartUpload initializes a new multipart upload in MinIO
func (s *Storage) CreateMultipartUpload(ctx context.Context, objectName string, contentType string) (string, error) {
	uploadID, err := s.core.NewMultipartUpload(ctx, s.bucket, objectName, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return "", fmt.Errorf("failed to create multipart upload: %w", err)
	}
	return uploadID, nil
}

// PutPart uploads a single part of a multipart upload
func (s *Storage) PutPart(ctx context.Context, objectName, uploadID string, partNumber int, data []byte) (string, error) {
	info, err := s.core.PutObjectPart(ctx, s.bucket, objectName, uploadID, partNumber, bytes.NewReader(data), int64(len(data)), minio.PutObjectPartOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to put part: %w", err)
	}
	return info.ETag, nil
}

// Part represents a completed upload part
type Part struct {
	ETag       string
	PartNumber int
}

// CompleteMultipartUpload finalizes the multipart upload
func (s *Storage) CompleteMultipartUpload(ctx context.Context, objectName, uploadID string, parts []Part) error {
	var completeParts []minio.CompletePart
	for _, p := range parts {
		completeParts = append(completeParts, minio.CompletePart{
			ETag:       p.ETag,
			PartNumber: p.PartNumber,
		})
	}

	_, err := s.core.CompleteMultipartUpload(ctx, s.bucket, objectName, uploadID, completeParts, minio.PutObjectOptions{})
	if err != nil {
		return fmt.Errorf("failed to complete multipart upload: %w", err)
	}
	return nil
}

// GetPresignedURL generates a presigned GET URL with the given expiry and content-disposition.
// When downloadProxyPath is set, the URL is rewritten to go through the reverse proxy
// (e.g. "/minio/filex/...?sig") so browsers can download without CORS issues.
func (s *Storage) GetPresignedURL(ctx context.Context, objectKey string, filename string, expiry time.Duration) (string, error) {
	reqParams := make(url.Values)
	encodedFilename := mime.QEncoding.Encode("utf-8", filename)
	reqParams.Set("response-content-disposition", fmt.Sprintf(`attachment; filename="%s"; filename*=%s`,
		escapeQuotedFilename(filename), encodedFilename))
	reqParams.Set("response-content-type", "application/octet-stream")

	presignedURL, err := s.client.PresignedGetObject(ctx, s.bucket, objectKey, expiry, reqParams)
	if err != nil {
		return "", fmt.Errorf("presign URL: %w", err)
	}

	raw := presignedURL.String()

	if s.downloadProxyPath != "" {
		// Rewrite http://minio:9000/filex/... → /minio/filex/...
		parsed, err := url.Parse(raw)
		if err != nil {
			return "", fmt.Errorf("parse presigned URL: %w", err)
		}
		parsed.Scheme = ""
		parsed.Host = ""
		rewritten := s.downloadProxyPath + parsed.String()
		return rewritten, nil
	}

	return raw, nil
}

// ExistsByObjectKey checks whether an object exists in the bucket by its object key
func (s *Storage) ExistsByObjectKey(ctx context.Context, objectKey string) (bool, error) {
	_, err := s.client.StatObject(ctx, s.bucket, objectKey, minio.StatObjectOptions{})
	if err != nil {
		errResponse := minio.ToErrorResponse(err)
		if errResponse.Code == "NoSuchKey" {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// GetObject returns a reader for the object
func (s *Storage) GetObject(ctx context.Context, objectName string) (io.ReadCloser, int64, string, error) {
	obj, err := s.client.GetObject(ctx, s.bucket, objectName, minio.GetObjectOptions{})
	if err != nil {
		return nil, 0, "", fmt.Errorf("failed to get object: %w", err)
	}

	stat, err := obj.Stat()
	if err != nil {
		obj.Close()
		return nil, 0, "", fmt.Errorf("failed to stat object: %w", err)
	}

	return obj, stat.Size, stat.ContentType, nil
}

// AbortMultipartUpload cancels a multipart upload and removes uploaded parts
func (s *Storage) AbortMultipartUpload(ctx context.Context, objectName, uploadID string) error {
	err := s.core.AbortMultipartUpload(ctx, s.bucket, objectName, uploadID)
	if err != nil {
		return fmt.Errorf("failed to abort multipart upload: %w", err)
	}
	return nil
}

// RemoveObject explicitly deletes an object from the bucket.
func (s *Storage) RemoveObject(ctx context.Context, objectName string) error {
	err := s.client.RemoveObject(ctx, s.bucket, objectName, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("failed to remove object %s: %w", objectName, err)
	}
	return nil
}

// ListObjects returns a channel of object info to iterate over all objects in the bucket.
func (s *Storage) ListObjects(ctx context.Context, prefix string) <-chan minio.ObjectInfo {
	return s.client.ListObjects(ctx, s.bucket, minio.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: true,
	})
}

// ListParts returns all uploaded parts for a multipart session
func (s *Storage) ListParts(ctx context.Context, objectName, uploadID string) ([]Part, error) {
	partsResult, err := s.core.ListObjectParts(ctx, s.bucket, objectName, uploadID, 0, 1000)
	if err != nil {
		return nil, fmt.Errorf("failed to list parts: %w", err)
	}

	var parts []Part
	for _, p := range partsResult.ObjectParts {
		parts = append(parts, Part{
			ETag:       p.ETag,
			PartNumber: p.PartNumber,
		})
	}
	return parts, nil
}

// escapeQuotedFilename replaces characters that would break the
// Content-Disposition header's quoted-string production (RFC 2616 §2.2).
func escapeQuotedFilename(name string) string {
	return strings.Map(func(r rune) rune {
		if r == '"' || r == '\\' || r < 0x20 {
			return '_'
		}
		return r
	}, name)
}

// BucketExists checks if the bucket exists
func (s *Storage) BucketExists(ctx context.Context) (bool, error) {
	return s.client.BucketExists(ctx, s.bucket)
}