package config

import (
	"os"
	"testing"
)

func TestLoadDefaults(t *testing.T) {
	cfg := Load()
	if cfg.Port != "8080" {
		t.Errorf("expected Port 8080, got %s", cfg.Port)
	}
	if cfg.MongoURI != "mongodb://localhost:27017" {
		t.Errorf("expected default MongoURI, got %s", cfg.MongoURI)
	}
	if cfg.MinioPublicEndpoint != "localhost:9000" {
		t.Errorf("expected MinioPublicEndpoint localhost:9000, got %s", cfg.MinioPublicEndpoint)
	}
}

func TestLoadAllowedOrigins(t *testing.T) {
	cfg := Load()
	if cfg.AllowedOrigins != "http://localhost:5173" {
		t.Errorf("expected default AllowedOrigins, got %s", cfg.AllowedOrigins)
	}
}

func TestLoadFromEnv(t *testing.T) {
	os.Setenv("PORT", "9090")
	os.Setenv("MONGO_URI", "mongodb://mongo:27017")
	os.Setenv("MINIO_ENDPOINT", "minio:9000")
	os.Setenv("MINIO_PUBLIC_ENDPOINT", "cdn.example.com")
	defer func() {
		os.Unsetenv("PORT")
		os.Unsetenv("MONGO_URI")
		os.Unsetenv("MINIO_ENDPOINT")
		os.Unsetenv("MINIO_PUBLIC_ENDPOINT")
	}()

	cfg := Load()
	if cfg.Port != "9090" {
		t.Errorf("expected Port 9090, got %s", cfg.Port)
	}
	if cfg.MongoURI != "mongodb://mongo:27017" {
		t.Errorf("expected overridden MongoURI, got %s", cfg.MongoURI)
	}
	if cfg.MinioPublicEndpoint != "cdn.example.com" {
		t.Errorf("expected MinioPublicEndpoint cdn.example.com, got %s", cfg.MinioPublicEndpoint)
	}
}