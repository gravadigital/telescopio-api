package storage

import (
	"context"
	"io"
)

type FileStorage interface {
	Put(ctx context.Context, key string, reader io.Reader, size int64, contentType string) (string, error)
	Get(ctx context.Context, key string) (io.ReadCloser, error)
	Delete(ctx context.Context, key string) error
	GetURL(ctx context.Context, key string) (string, error)
	Exists(ctx context.Context, key string) (bool, error)
	GetInfo(ctx context.Context, key string) (*FileInfo, error)
}

type FileInfo struct {
	Key         string
	Size        int64
	ContentType string
	ETag        string
	LastModified string
}

type StorageConfig struct {
	Provider string
	LocalPath string
	Endpoint        string
	AccessKeyID     string
	SecretAccessKey string
	BucketName      string
	UseSSL          bool
	Region          string
}
