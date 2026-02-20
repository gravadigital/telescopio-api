# Storage System - Implementation Details

## Architecture

The storage system uses a **Strategy Pattern** to abstract file storage operations:

```
                    ┌─────────────────┐
                    │  FileStorage    │
                    │   (Interface)   │
                    └────────┬────────┘
                             │
                ┌────────────┴────────────┐
                │                         │
        ┌───────▼────────┐       ┌───────▼────────┐
        │ LocalStorage   │       │ MinIOStorage   │
        │                │       │                │
        │ - filesystem   │       │ - S3 API       │
        │ - direct I/O   │       │ - presigned    │
        └────────────────┘       └────────────────┘
```

## Interface Definition

```go
type FileStorage interface {
    Put(ctx, key, reader, size, contentType) (string, error)
    Get(ctx, key) (io.ReadCloser, error)
    Delete(ctx, key) error
    GetURL(ctx, key) (string, error)
    Exists(ctx, key) (bool, error)
    GetInfo(ctx, key) (*FileInfo, error)
}
```

## Implementation Details

### Local Storage

**File Structure:**
```
./uploads/
  ├── event-id_participant-id_timestamp.pdf
  ├── event-id_participant-id_timestamp.jpg
  └── ...
```

**Key Features:**
- Direct filesystem access
- Simple and fast for development
- No external dependencies
- Files stored in configurable directory

**Limitations:**
- Not scalable for high traffic
- No built-in redundancy
- Difficult to distribute across servers

### MinIO Storage

**File Structure:**
```
s3://telescopio/
  ├── event-id_participant-id_timestamp.pdf
  ├── event-id_participant-id_timestamp.jpg
  └── ...
```

**Key Features:**
- S3-compatible API
- Scalable and distributed
- Built-in redundancy (erasure coding)
- Presigned URLs for secure access
- Better performance for concurrent access

**Configuration:**
```yaml
services:
  minio:
    image: minio/minio:latest
    command: server /data --console-address ":9001"
    ports:
      - "9000:9000"  # API
      - "9001:9001"  # Console
```

## Handler Integration

The `AttachmentHandler` is storage-agnostic:

```go
type AttachmentHandler struct {
    fileStorage storage.FileStorage  // Interface, not concrete type
    // ... other fields
}

func (h *AttachmentHandler) UploadAttachment(c *gin.Context) {
    // ... validation ...
    
    // Use storage interface
    storageKey, err := h.fileStorage.Put(ctx, filename, file, size, contentType)
    
    // ... save to database ...
}

func (h *AttachmentHandler) DownloadAttachment(c *gin.Context) {
    // ... get attachment metadata ...
    
    // Use storage interface
    fileReader, err := h.fileStorage.Get(ctx, attachment.FilePath)
    
    // Stream to response
    io.Copy(c.Writer, fileReader)
}
```

## Factory Pattern

The factory creates the appropriate storage implementation based on configuration:

```go
func NewFileStorage(cfg *config.Config) (FileStorage, error) {
    switch cfg.Storage.Provider {
    case "local":
        return NewLocalStorage(cfg.Storage.LocalPath)
    case "minio":
        return NewMinIOStorage(
            cfg.Storage.MinIOEndpoint,
            cfg.Storage.MinIOAccessKey,
            cfg.Storage.MinIOSecretKey,
            cfg.Storage.MinIOBucket,
            cfg.Storage.MinIORegion,
            cfg.Storage.MinIOUseSSL,
        )
    default:
        return nil, fmt.Errorf("unsupported provider: %s", cfg.Storage.Provider)
    }
}
```

## Error Handling

All storage operations return standardized errors:

- **File not found**: `fmt.Errorf("file not found: %s", key)`
- **Storage error**: `fmt.Errorf("failed to store file: %w", err)`
- **Permission error**: `fmt.Errorf("access denied: %w", err)`

The handler translates these into appropriate HTTP status codes:
- `404 Not Found` - File doesn't exist
- `500 Internal Server Error` - Storage operation failed
- `403 Forbidden` - Permission denied

## Security Considerations

### Path Traversal Prevention

```go
// Sanitize filename to prevent path traversal
cleanKey := filepath.Clean(key)
if cleanKey != key {
    return "", errors.New("invalid file key")
}
```

### File Type Validation

```go
allowedTypes := map[string]string{
    "image/jpeg": "JPEG Image",
    "image/png":  "PNG Image",
    "application/pdf": "PDF Document",
    // ... more types
}

if _, isAllowed := allowedTypes[contentType]; !isAllowed {
    return error
}
```

### Size Limits

```go
if header.Size > h.config.Upload.MaxFileSize {
    return error // 413 Payload Too Large
}
```

## Testing

### Unit Tests

```go
func TestLocalStorage_Put(t *testing.T) {
    storage, _ := NewLocalStorage("/tmp/test")
    
    reader := strings.NewReader("test content")
    key, err := storage.Put(ctx, "test.txt", reader, 12, "text/plain")
    
    assert.NoError(t, err)
    assert.Equal(t, "test.txt", key)
}
```

### Integration Tests

```bash
# Start MinIO for testing
docker-compose --profile minio up -d minio

# Run integration tests
STORAGE_PROVIDER=minio go test ./internal/handlers/...
```

## Performance Considerations

### Local Storage
- **Sequential writes**: ~100-500 MB/s (SSD)
- **Concurrent reads**: Limited by disk I/O
- **Best for**: < 1000 files, single server

### MinIO Storage
- **Sequential writes**: ~200-1000 MB/s (depends on setup)
- **Concurrent reads**: Scales with number of nodes
- **Best for**: > 1000 files, distributed systems
- **Overhead**: ~5-10ms per request (network latency)

## Monitoring

### Local Storage
- Monitor disk space: `df -h`
- Monitor I/O: `iostat`

### MinIO Storage
- MinIO Console: http://localhost:9001
- Metrics endpoint: http://localhost:9000/minio/v2/metrics/cluster
- Prometheus integration available

## Backup Strategies

### Local Storage
```bash
# Simple backup
rsync -av ./uploads/ /backup/uploads/

# Scheduled backup (cron)
0 2 * * * rsync -av /app/uploads/ /backup/uploads/
```

### MinIO Storage
```bash
# Using mc (MinIO Client)
mc mirror myminio/telescopio s3/backup-bucket

# Scheduled replication
mc replicate add myminio/telescopio --remote-bucket backup-bucket
```

## Migration Tools

### Migrate from Local to MinIO

```bash
#!/bin/bash
# migrate-to-minio.sh

LOCAL_DIR="./uploads"
MINIO_ALIAS="myminio"
BUCKET="telescopio"

# Configure MinIO client
mc alias set $MINIO_ALIAS http://localhost:9000 minioadmin minioadmin123

# Create bucket if not exists
mc mb $MINIO_ALIAS/$BUCKET --ignore-existing

# Copy all files
mc cp --recursive $LOCAL_DIR/ $MINIO_ALIAS/$BUCKET/

echo "Migration complete!"
```

## Future Enhancements

Potential storage backends to add:

1. **AWS S3** - Production-grade cloud storage
2. **Google Cloud Storage** - GCP integration
3. **Azure Blob Storage** - Azure integration
4. **IPFS** - Decentralized storage
5. **FTP/SFTP** - Legacy system integration

Adding a new backend only requires:
1. Implement `FileStorage` interface
2. Add to factory
3. Add configuration options
4. Document usage
