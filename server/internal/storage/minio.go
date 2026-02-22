package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"

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

	return &Storage{
		client: client,
		core:   core,
		bucket: "files",
	}, nil
}

// localhostRedirectTransport is no longer needed since we are not using publicClient

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

// BucketExists checks if the bucket exists
func (s *Storage) BucketExists(ctx context.Context) (bool, error) {
	return s.client.BucketExists(ctx, s.bucket)
}
