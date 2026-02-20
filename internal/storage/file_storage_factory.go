package storage

import (
	"fmt"

	"github.com/gravadigital/telescopio-api/internal/config"
)

// NewFileStorage creates a new FileStorage instance based on configuration
func NewFileStorage(cfg *config.Config) (FileStorage, error) {
	switch cfg.Storage.Provider {
	case "local":
		return NewLocalStorage(cfg.Storage.LocalPath)
	
	case "minio":
		if cfg.Storage.MinIOAccessKey == "" || cfg.Storage.MinIOSecretKey == "" {
			return nil, fmt.Errorf("MinIO credentials not configured")
		}
		
		return NewMinIOStorage(
			cfg.Storage.MinIOEndpoint,
			cfg.Storage.MinIOAccessKey,
			cfg.Storage.MinIOSecretKey,
			cfg.Storage.MinIOBucket,
			cfg.Storage.MinIORegion,
			cfg.Storage.MinIOUseSSL,
		)
	
	default:
		return nil, fmt.Errorf("unsupported storage provider: %s (must be 'local' or 'minio')", cfg.Storage.Provider)
	}
}
