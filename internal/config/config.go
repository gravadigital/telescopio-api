package config

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// Config holds all configuration for the application
type Config struct {
	DB struct {
		Host     string
		Port     string
		User     string
		Password string
		Name     string
		SSLMode  string
	}

	Server struct {
		Port        string
		GinMode     string
		FrontendURL string
	}

	Upload struct {
		Dir         string
		MaxFileSize int64
	}

	Storage struct {
		Provider       string // "local" or "minio"
		LocalPath      string
		MinIOEndpoint  string
		MinIOAccessKey string
		MinIOSecretKey string
		MinIOBucket    string
		MinIOUseSSL    bool
		MinIORegion    string
	}

	CORS struct {
		AllowOrigins string
		AllowMethods string
		AllowHeaders string
	}

	Email struct {
		Enabled      bool
		SMTPHost     string
		SMTPPort     string
		SMTPUser     string
		SMTPPassword string
		FromAddress  string
		FromName     string
		Secure       bool // true = SSL/TLS directo (puerto 465), false = STARTTLS (puerto 587)
	}

	Google struct {
		ClientID string
	}
}

// Load loads configuration from environment variables
func Load() *Config {
	_ = godotenv.Load()

	config := &Config{}

	config.DB.Host = getEnv("DB_HOST", "localhost")
	config.DB.Port = getEnv("DB_PORT", "5432")
	config.DB.User = getEnv("DB_USER", "telescopio")
	config.DB.Password = getEnv("DB_PASSWORD", "telescopio_password")
	config.DB.Name = getEnv("DB_NAME", "telescopio_db")
	config.DB.SSLMode = getEnv("DB_SSLMODE", "disable")

	config.Server.Port = getEnv("PORT", "8080")
	config.Server.GinMode = getEnv("GIN_MODE", "debug")
	config.Server.FrontendURL = getEnv("FRONTEND_URL", "http://localhost:3000")

	config.Upload.Dir = getEnv("UPLOADS_DIR", "./uploads")
	config.Upload.MaxFileSize = getEnvAsInt64("MAX_FILE_SIZE", 10485760)

	// Storage configuration
	config.Storage.Provider = getEnv("STORAGE_PROVIDER", "local") // "local" or "minio"
	config.Storage.LocalPath = getEnv("STORAGE_LOCAL_PATH", "./uploads")
	config.Storage.MinIOEndpoint = getEnv("MINIO_ENDPOINT", "localhost:9000")
	config.Storage.MinIOAccessKey = getEnv("MINIO_ACCESS_KEY", "")
	config.Storage.MinIOSecretKey = getEnv("MINIO_SECRET_KEY", "")
	config.Storage.MinIOBucket = getEnv("MINIO_BUCKET", "telescopio")
	config.Storage.MinIOUseSSL = getEnvAsBool("MINIO_USE_SSL", false)
	config.Storage.MinIORegion = getEnv("MINIO_REGION", "us-east-1")

	config.CORS.AllowOrigins = getEnv("CORS_ALLOW_ORIGINS", "*")
	config.CORS.AllowMethods = getEnv("CORS_ALLOW_METHODS", "GET,POST,PUT,PATCH,DELETE,HEAD,OPTIONS")
	config.CORS.AllowHeaders = getEnv("CORS_ALLOW_HEADERS", "Origin,Content-Length,Content-Type,Authorization")

	config.Email.Enabled = getEnvAsBool("EMAIL_ENABLED", false)
	config.Email.SMTPHost = getEnv("SMTP_HOST", "")
	config.Email.SMTPPort = getEnv("SMTP_PORT", "587")
	config.Email.SMTPUser = getEnv("SMTP_USER", "")
	config.Email.SMTPPassword = getEnv("SMTP_PASSWORD", "")
	config.Email.FromAddress = getEnv("EMAIL_FROM", "")
	config.Email.FromName = getEnv("EMAIL_FROM_NAME", "Telescopio")
	config.Email.Secure = getEnvAsBool("SMTP_SECURE", false)
	config.Google.ClientID = getEnv("GOOGLE_CLIENT_ID", "")

	return config
}

// GetDatabaseURL returns the database connection URL
func (c *Config) GetDatabaseURL() string {
	return "postgres://" + c.DB.User + ":" + c.DB.Password + "@" + c.DB.Host + ":" + c.DB.Port + "/" + c.DB.Name + "?sslmode=" + c.DB.SSLMode
}

// getEnv gets an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvAsInt64 gets an environment variable as int64 or returns a default value
func getEnvAsInt64(key string, defaultValue int64) int64 {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.ParseInt(value, 10, 64); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// getEnvAsBool gets an environment variable as bool or returns a default value
func getEnvAsBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}
