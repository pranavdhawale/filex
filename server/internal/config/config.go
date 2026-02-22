package config

import (
	"os"
)

// Config holds all the environment-based configuration for the application.
type Config struct {
	Port                string
	Environment         string
	MongoURI            string
	RedisURI            string
	MinioEndpoint       string // internal (Docker service name) — used by Go server
	MinioPublicEndpoint string // external (browser-accessible) — used in presigned URLs
	MinioAccessKey      string
	MinioSecretKey      string
	ServerWrapKey       string
}

// Load reads from environment variables and provides defaults where appropriate.
func Load() *Config {
	minioEndpoint := getEnv("MINIO_ENDPOINT", "localhost:9000")
	return &Config{
		Port:                getEnv("PORT", "8080"),
		Environment:         getEnv("ENVIRONMENT", "development"),
		MongoURI:            getEnv("MONGO_URI", "mongodb://localhost:27017"),
		RedisURI:            getEnv("REDIS_URI", "redis://localhost:6379"),
		MinioEndpoint:       minioEndpoint,
		MinioPublicEndpoint: getEnv("MINIO_PUBLIC_ENDPOINT", minioEndpoint),
		MinioAccessKey:      getEnv("MINIO_ACCESS_KEY", "minioadmin"),
		MinioSecretKey:      getEnv("MINIO_SECRET_KEY", "minioadmin"),
		ServerWrapKey:       getEnv("SWK", "default-swk-for-dev-change-in-prod"),
	}
}

// getEnv retrieves the value of the environment variable named by the key.
// It returns the fallback value if the variable is not present.
func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}
