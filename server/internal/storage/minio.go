package storage

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/pranavdhawale/bytefile/internal/config"
)

type Storage struct {
	client         *minio.Client
	core           *minio.Core
	publicClient   *minio.Client // used exclusively for generating presigned URLs
	bucket         string
	publicEndpoint string
}

// NewStorage initializes a new MinIO storage client
func NewStorage(cfg *config.Config) (*Storage, error) {
	// Internal client always uses HTTP inside Docker
	internalOptions := &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.MinioAccessKey, cfg.MinioSecretKey, ""),
		Secure: false,
	}

	client, err := minio.New(cfg.MinioEndpoint, internalOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to create minio client: %w", err)
	}

	core, err := minio.NewCore(cfg.MinioEndpoint, internalOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to create minio core client: %w", err)
	}

	// publicClient signs URLs using the browser-facing host.
	// In production, we MUST use HTTPS for presigned URLs (Mixed Content protection).
	usePublicSSL := cfg.Environment == "production"

	publicOptions := &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.MinioAccessKey, cfg.MinioSecretKey, ""),
		Secure: usePublicSSL,
		Transport: &localhostRedirectTransport{
			Base:     http.DefaultTransport,
			Internal: cfg.MinioEndpoint,
			Public:   cfg.MinioPublicEndpoint,
		},
	}

	publicClient, err := minio.New(cfg.MinioPublicEndpoint, publicOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to create minio public client: %w", err)
	}

	return &Storage{
		client:         client,
		core:           core,
		publicClient:   publicClient,
		bucket:         "files",
		publicEndpoint: cfg.MinioPublicEndpoint,
	}, nil
}

type localhostRedirectTransport struct {
	Base     http.RoundTripper
	Internal string
	Public   string
}

func (t *localhostRedirectTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// If the request is going to the public endpoint, redirect it to the internal one
	if req.URL.Host == t.Public {
		req.URL.Host = t.Internal
		req.URL.Scheme = "http" // Internal Docker traffic is always HTTP
	}
	return t.Base.RoundTrip(req)
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

	expires := 10 * time.Minute

	presignedURL, err := s.publicClient.Presign(ctx, "PUT", s.bucket, objectName, expires, reqParams)
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
	presignedURL, err := s.publicClient.PresignedGetObject(ctx, s.bucket, objectName, expires, nil)
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
