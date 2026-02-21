package storage

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/pranavdhawale/bytefile/internal/config"
)

type Storage struct {
	client *minio.Client
	core   *minio.Core
	bucket string
}

// NewStorage initializes a new MinIO storage client
func NewStorage(cfg *config.Config) (*Storage, error) {
	useSSL := false // MinIO in docker-compose is typically non-SSL

	options := &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.MinioAccessKey, cfg.MinioSecretKey, ""),
		Secure: useSSL,
	}
	client, err := minio.New(cfg.MinioEndpoint, options)
	if err != nil {
		return nil, fmt.Errorf("failed to create minio client: %w", err)
	}

	core, err := minio.NewCore(cfg.MinioEndpoint, options)
	if err != nil {
		return nil, fmt.Errorf("failed to create minio core client: %w", err)
	}

	return &Storage{
		client: client,
		core:   core,
		bucket: "files",
	}, nil
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

// GeneratePresignedPutURL generates a pre-signed URL for a specific part of a multipart upload
func (s *Storage) GeneratePresignedPutURL(ctx context.Context, objectName, uploadID string, partNumber int) (string, error) {
	reqParams := make(url.Values)
	reqParams.Set("uploadId", uploadID)
	reqParams.Set("partNumber", fmt.Sprint(partNumber))

	// Determine the pre-signed URL expiry
	expires := 10 * time.Minute

	// Use Presign to correctly calculate the signature inclusive of the multipart query parameters
	presignedURL, err := s.client.Presign(ctx, "PUT", s.bucket, objectName, expires, reqParams)
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned URL: %w", err)
	}

	return presignedURL.String(), nil
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

// GenerateSignedGetURL generates a pre-signed URL for downloading an object
func (s *Storage) GenerateSignedGetURL(ctx context.Context, objectName string, expires time.Duration) (string, error) {
	presignedURL, err := s.client.PresignedGetObject(ctx, s.bucket, objectName, expires, nil)
	if err != nil {
		return "", fmt.Errorf("failed to generate signed GET URL: %w", err)
	}
	return presignedURL.String(), nil
}

// AbortMultipartUpload cancels a multipart upload and removes uploaded parts
func (s *Storage) AbortMultipartUpload(ctx context.Context, objectName, uploadID string) error {
	err := s.core.AbortMultipartUpload(ctx, s.bucket, objectName, uploadID)
	if err != nil {
		return fmt.Errorf("failed to abort multipart upload: %w", err)
	}
	return nil
}

// BucketExists checks if the bucket exists
func (s *Storage) BucketExists(ctx context.Context) (bool, error) {
	return s.client.BucketExists(ctx, s.bucket)
}
