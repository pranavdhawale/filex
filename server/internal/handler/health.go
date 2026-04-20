package handler

import (
	"net/http"

	"github.com/minio/minio-go/v7"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type HealthChecker struct {
	mongoClient *mongo.Client
	minioClient *minio.Client
	bucket      string
	ready       bool
}

func NewHealthChecker(mongoClient *mongo.Client, minioClient *minio.Client, bucket string) *HealthChecker {
	return &HealthChecker{mongoClient: mongoClient, minioClient: minioClient, bucket: bucket, ready: true}
}

func (h *HealthChecker) SetNotReady() { h.ready = false }

func (h *HealthChecker) HandleHealthz(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok"}`))
}

func (h *HealthChecker) HandleReadyz(w http.ResponseWriter, r *http.Request) {
	if !h.ready {
		writeError(w, http.StatusServiceUnavailable, "not ready")
		return
	}
	if err := h.mongoClient.Ping(r.Context(), nil); err != nil {
		writeError(w, http.StatusServiceUnavailable, "mongo unreachable")
		return
	}
	if _, err := h.minioClient.BucketExists(r.Context(), h.bucket); err != nil {
		writeError(w, http.StatusServiceUnavailable, "minio unreachable")
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ready"}`))
}