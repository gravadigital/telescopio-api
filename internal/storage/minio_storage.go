package storage

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/charmbracelet/log"
	"github.com/gravadigital/telescopio-api/internal/logger"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// MinIOStorage implements FileStorage interface using MinIO (S3-compatible)
type MinIOStorage struct {
	client     *minio.Client
	bucketName string
	useSSL     bool
	log        *log.Logger
}

// NewMinIOStorage creates a new MinIO storage client
func NewMinIOStorage(endpoint, accessKeyID, secretAccessKey, bucketName, region string, useSSL bool) (*MinIOStorage, error) {
	minioClient, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKeyID, secretAccessKey, ""),
		Secure: useSSL,
		Region: region,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create MinIO client: %w", err)
	}

	storage := &MinIOStorage{
		client:     minioClient,
		bucketName: bucketName,
		useSSL:     useSSL,
		log:        logger.Handler("minio-storage"),
	}

	ctx := context.Background()
	exists, err := minioClient.BucketExists(ctx, bucketName)
	if err != nil {
		return nil, fmt.Errorf("failed to check bucket existence: %w", err)
	}

	if !exists {
		storage.log.Info("creating bucket", "bucket", bucketName)
		err = minioClient.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{
			Region: region,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create bucket: %w", err)
		}
		storage.log.Info("bucket created successfully", "bucket", bucketName)
	}

	storage.log.Info("MinIO storage initialized", "endpoint", endpoint, "bucket", bucketName, "ssl", useSSL)
	return storage, nil
}

// Put stores a file in MinIO
func (s *MinIOStorage) Put(ctx context.Context, key string, reader io.Reader, size int64, contentType string) (string, error) {
	opts := minio.PutObjectOptions{
		ContentType: contentType,
	}

	info, err := s.client.PutObject(ctx, s.bucketName, key, reader, size, opts)
	if err != nil {
		return "", fmt.Errorf("failed to upload to MinIO: %w", err)
	}

	s.log.Debug("file stored successfully", 
		"key", key, 
		"size", info.Size, 
		"etag", info.ETag,
		"bucket", s.bucketName)
	
	return key, nil
}

// Get retrieves a file from MinIO
func (s *MinIOStorage) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	object, err := s.client.GetObject(ctx, s.bucketName, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get object from MinIO: %w", err)
	}

	_, err = object.Stat()
	if err != nil {
		object.Close()
		return nil, fmt.Errorf("object not found: %w", err)
	}

	return object, nil
}

// Delete removes a file from MinIO
func (s *MinIOStorage) Delete(ctx context.Context, key string) error {
	err := s.client.RemoveObject(ctx, s.bucketName, key, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete from MinIO: %w", err)
	}

	s.log.Debug("file deleted successfully", "key", key, "bucket", s.bucketName)
	return nil
}

func (s *MinIOStorage) GetURL(ctx context.Context, key string) (string, error) {
	expiry := 7 * 24 * time.Hour
	
	url, err := s.client.PresignedGetObject(ctx, s.bucketName, key, expiry, nil)
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned URL: %w", err)
	}

	return url.String(), nil
}

// Exists checks if a file exists in MinIO
func (s *MinIOStorage) Exists(ctx context.Context, key string) (bool, error) {
	_, err := s.client.StatObject(ctx, s.bucketName, key, minio.StatObjectOptions{})
	if err != nil {
		errResponse := minio.ToErrorResponse(err)
		if errResponse.Code == "NoSuchKey" {
			return false, nil
		}
		return false, fmt.Errorf("failed to check object: %w", err)
	}

	return true, nil
}

// GetInfo returns metadata about a stored file
func (s *MinIOStorage) GetInfo(ctx context.Context, key string) (*FileInfo, error) {
	objInfo, err := s.client.StatObject(ctx, s.bucketName, key, minio.StatObjectOptions{})
	if err != nil {
		errResponse := minio.ToErrorResponse(err)
		if errResponse.Code == "NoSuchKey" {
			return nil, fmt.Errorf("file not found: %s", key)
		}
		return nil, fmt.Errorf("failed to get object info: %w", err)
	}

	return &FileInfo{
		Key:          key,
		Size:         objInfo.Size,
		ContentType:  objInfo.ContentType,
		ETag:         objInfo.ETag,
		LastModified: objInfo.LastModified.Format(time.RFC3339),
	}, nil
}
