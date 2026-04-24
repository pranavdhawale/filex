package config

import "os"

type Config struct {
	Port                string
	Environment         string
	MongoURI            string
	MinioEndpoint       string
	MinioPublicEndpoint string
	MinioBucket         string
	MinioAccessKey      string
	MinioSecretKey      string
	MinioDownloadPrefix string // e.g. "/minio" — rewrites presigned URLs through a reverse proxy
	AllowedOrigins      string // comma-separated, e.g. "http://localhost:5173,https://filex.pranavdhawale.in"
	MaxChunkSize        int64  // Max encrypted chunk body size in bytes
}

func Load() *Config {
	return &Config{
		Port:                getEnv("PORT", "8080"),
		Environment:         getEnv("ENVIRONMENT", "development"),
		MongoURI:            getEnv("MONGO_URI", "mongodb://localhost:27017"),
		MinioEndpoint:       getEnv("MINIO_ENDPOINT", "localhost:9000"),
		MinioPublicEndpoint: getEnv("MINIO_PUBLIC_ENDPOINT", "localhost:9000"),
		MinioBucket:         getEnv("MINIO_BUCKET", "filex"),
		MinioAccessKey:      getEnv("MINIO_ACCESS_KEY", "minioadmin"),
		MinioSecretKey:      getEnv("MINIO_SECRET_KEY", "minioadmin"),
		MinioDownloadPrefix: getEnv("MINIO_DOWNLOAD_PREFIX", ""),
		AllowedOrigins:      getEnv("ALLOWED_ORIGINS", "http://localhost:5173"),
		MaxChunkSize:        11 * 1024 * 1024,
	}
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}