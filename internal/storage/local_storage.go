package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/charmbracelet/log"
	"github.com/gravadigital/telescopio-api/internal/logger"
)

// LocalStorage implements FileStorage interface using local filesystem
type LocalStorage struct {
	basePath string
	log      *log.Logger
}

// NewLocalStorage creates a new local filesystem storage
func NewLocalStorage(basePath string) (*LocalStorage, error) {
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create storage directory: %w", err)
	}

	return &LocalStorage{
		basePath: basePath,
		log:      logger.Handler("local-storage"),
	}, nil
}

// Put stores a file in the local filesystem
func (s *LocalStorage) Put(ctx context.Context, key string, reader io.Reader, size int64, contentType string) (string, error) {
	cleanKey := filepath.Clean(key)
	if cleanKey != key {
		return "", errors.New("invalid file key: potential path traversal detected")
	}

	fullPath := filepath.Join(s.basePath, key)

	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("failed to create directory: %w", err)
	}

	file, err := os.Create(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	written, err := io.Copy(file, reader)
	if err != nil {
		os.Remove(fullPath)
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	if size > 0 && written != size {
		os.Remove(fullPath)
		return "", fmt.Errorf("size mismatch: expected %d, wrote %d", size, written)
	}

	s.log.Debug("file stored successfully", "key", key, "size", written)
	return key, nil
}

// Get retrieves a file from the local filesystem
func (s *LocalStorage) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	fullPath := filepath.Join(s.basePath, key)

	file, err := os.Open(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("file not found: %s", key)
		}
		return nil, fmt.Errorf("failed to open file: %w", err)
	}

	return file, nil
}

// Delete removes a file from the local filesystem
func (s *LocalStorage) Delete(ctx context.Context, key string) error {
	fullPath := filepath.Join(s.basePath, key)

	if err := os.Remove(fullPath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("file not found: %s", key)
		}
		return fmt.Errorf("failed to delete file: %w", err)
	}

	s.log.Debug("file deleted successfully", "key", key)
	return nil
}

func (s *LocalStorage) GetURL(ctx context.Context, key string) (string, error) {
	fullPath := filepath.Join(s.basePath, key)

	if _, err := os.Stat(fullPath); err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("file not found: %s", key)
		}
		return "", fmt.Errorf("failed to check file: %w", err)
	}

	return key, nil
}

// Exists checks if a file exists in the local filesystem
func (s *LocalStorage) Exists(ctx context.Context, key string) (bool, error) {
	fullPath := filepath.Join(s.basePath, key)

	_, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("failed to check file: %w", err)
	}

	return true, nil
}

// GetInfo returns metadata about a stored file
func (s *LocalStorage) GetInfo(ctx context.Context, key string) (*FileInfo, error) {
	fullPath := filepath.Join(s.basePath, key)

	stat, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("file not found: %s", key)
		}
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}

	return &FileInfo{
		Key:          key,
		Size:         stat.Size(),
		ContentType:  "",
		ETag:         "",
		LastModified: stat.ModTime().Format(time.RFC3339),
	}, nil
}
