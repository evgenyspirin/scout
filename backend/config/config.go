// Package config loads all runtime configuration from environment variables.
// No global variables are used: Load returns a fully populated Config value
// that is passed explicitly through the application (dependency injection).
package config

import (
	"os"
	"strconv"
	"time"
)

type (
	// App holds core service configuration.
	App struct {
		Name          string
		Host          string
		Port          string
		Env           string
		JWTSecret     string
		JWTTTL        time.Duration
		PublicBaseURL string // external base URL used to build absolute originalUrl values
		BodyLimit     int    // max JSON request body size in bytes
	}

	// SQLite holds the read-only dataset database location.
	SQLite struct {
		Path string
	}

	// MinIO holds object-storage configuration.
	MinIO struct {
		Endpoint   string
		AccessKey  string
		SecretKey  string
		Bucket     string
		UseSSL     bool
		PresignTTL time.Duration
	}

	// Redis holds cache configuration.
	Redis struct {
		Addr     string
		Password string
		DB       int
		TTL      time.Duration
	}

	// Thumbnail holds the thumbnail engine tuning knobs.
	Thumbnail struct {
		MaxConcurrency int   // semaphore size, bounded image processing
		LRUBytes       int64 // in-memory cache byte budget
		CacheMaxAge    int   // Cache-Control max-age seconds
	}

	// Seeder holds configuration used by the image seeder client.
	Seeder struct {
		APIBaseURL string
		ImagesDir  string
		AdminLogin string
		AdminPass  string
	}

	// Config is the aggregate configuration value.
	Config struct {
		App       App
		SQLite    SQLite
		MinIO     MinIO
		Redis     Redis
		Thumbnail Thumbnail
		Seeder    Seeder
	}
)

func getEnv(key, def string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return def
}

func getEnvInt(key string, def int) int {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return def
}

func getEnvInt64(key string, def int64) int64 {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		if i, err := strconv.ParseInt(v, 10, 64); err == nil {
			return i
		}
	}
	return def
}

func getEnvBool(key string, def bool) bool {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return def
}

func getEnvDuration(key string, def time.Duration) time.Duration {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return def
}

// Load reads configuration from the environment and returns a Config value.
func Load() Config {
	return Config{
		App: App{
			Name:          getEnv("SERVICE_NAME", "scout"),
			Host:          getEnv("SERVICE_HOST", "0.0.0.0"),
			Port:          getEnv("SERVICE_PORT", "8080"),
			Env:           getEnv("SERVICE_ENV", "development"),
			JWTSecret:     getEnv("JWT_SECRET", "scout-dev-secret-change-me"),
			JWTTTL:        getEnvDuration("JWT_TTL", 24*time.Hour),
			PublicBaseURL: getEnv("PUBLIC_BASE_URL", "http://localhost:8080"),
			BodyLimit:     getEnvInt("BODY_LIMIT_BYTES", 1<<20), // 1 MB
		},
		SQLite: SQLite{
			Path: getEnv("SQLITE_PATH", "../dataset/predictions.db"),
		},
		MinIO: MinIO{
			Endpoint:   getEnv("MINIO_ENDPOINT", "localhost:9000"),
			AccessKey:  getEnv("MINIO_ACCESS_KEY", "minioadmin"),
			SecretKey:  getEnv("MINIO_SECRET_KEY", "minioadmin"),
			Bucket:     getEnv("MINIO_BUCKET", "scout-originals"),
			UseSSL:     getEnvBool("MINIO_USE_SSL", false),
			PresignTTL: getEnvDuration("MINIO_PRESIGN_TTL", 15*time.Minute),
		},
		Redis: Redis{
			Addr:     getEnv("REDIS_ADDR", "localhost:6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvInt("REDIS_DB", 0),
			TTL:      getEnvDuration("REDIS_TTL", 24*time.Hour),
		},
		Thumbnail: Thumbnail{
			MaxConcurrency: getEnvInt("THUMBNAIL_MAX_CONCURRENCY", 2),
			LRUBytes:       getEnvInt64("THUMBNAIL_LRU_BYTES", 128*1024*1024),
			CacheMaxAge:    getEnvInt("THUMBNAIL_CACHE_MAX_AGE", 86400),
		},
		Seeder: Seeder{
			APIBaseURL: getEnv("SEEDER_API_BASE_URL", "http://localhost:8080"),
			ImagesDir:  getEnv("SEEDER_IMAGES_DIR", "../dataset/images"),
			AdminLogin: getEnv("ADMIN_LOGIN", "admin"),
			AdminPass:  getEnv("ADMIN_PASSWORD", "admin123"),
		},
	}
}
