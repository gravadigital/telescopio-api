package storage

import (
	"fmt"

	"github.com/gravadigital/telescopio-api/internal/config"
	"github.com/gravadigital/telescopio-api/internal/storage/postgres"
)

// StorageType represents the type of storage backend
type StorageType string

const (
	// StorageTypePostgres represents PostgreSQL storage
	StorageTypePostgres StorageType = "postgres"
)

// Factory provides a factory pattern for creating storage containers
type Factory struct {
	storageType StorageType
}

// NewFactory creates a new storage factory
func NewFactory(storageType StorageType) *Factory {
	return &Factory{
		storageType: storageType,
	}
}

// CreateContainer creates a storage container based on the configured type
func (f *Factory) CreateContainer(cfg *config.Config) (postgres.RepositoryContainer, error) {
	switch f.storageType {
	case StorageTypePostgres:
		return postgres.NewContainer(cfg)
	default:
		return nil, fmt.Errorf("unsupported storage type: %s", f.storageType)
	}
}

// GetSupportedTypes returns a list of supported storage types
func GetSupportedTypes() []StorageType {
	return []StorageType{
		StorageTypePostgres,
	}
}

// ValidateStorageType validates if a storage type is supported
func ValidateStorageType(storageType string) (StorageType, error) {
	st := StorageType(storageType)

	for _, supported := range GetSupportedTypes() {
		if st == supported {
			return st, nil
		}
	}

	return "", fmt.Errorf("unsupported storage type: %s. Supported types: %v", storageType, GetSupportedTypes())
}

// DefaultFactory returns a factory configured with the default storage type
func DefaultFactory() *Factory {
	return NewFactory(StorageTypePostgres)
}
