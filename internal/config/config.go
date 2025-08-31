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
		Port    string
		GinMode string
	}

	Upload struct {
		Dir         string
		MaxFileSize int64
	}

	CORS struct {
		AllowOrigins string
		AllowMethods string
		AllowHeaders string
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

	config.Upload.Dir = getEnv("UPLOADS_DIR", "./uploads")
	config.Upload.MaxFileSize = getEnvAsInt64("MAX_FILE_SIZE", 10485760)

	config.CORS.AllowOrigins = getEnv("CORS_ALLOW_ORIGINS", "*")
	config.CORS.AllowMethods = getEnv("CORS_ALLOW_METHODS", "GET,POST,PUT,PATCH,DELETE,HEAD,OPTIONS")
	config.CORS.AllowHeaders = getEnv("CORS_ALLOW_HEADERS", "Origin,Content-Length,Content-Type,Authorization")

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
