#!/bin/bash

# Setup environment for MinIO storage
# This script creates/updates .env file with MinIO configuration

echo "🔧 Configuring Telescopio API for MinIO Storage"
echo "================================================"

ENV_FILE=".env"

# Check if .env exists
if [ -f "$ENV_FILE" ]; then
    echo "⚠️  .env file already exists"
    read -p "Do you want to overwrite it? (y/N): " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        echo "❌ Configuration cancelled"
        exit 0
    fi
    echo "📝 Backing up existing .env to .env.backup"
    cp "$ENV_FILE" "$ENV_FILE.backup"
fi

# Create .env file
cat > "$ENV_FILE" << 'EOF'
# Database Configuration
DB_HOST=localhost
DB_PORT=5432
DB_USER=telescopio
DB_PASSWORD=telescopio_password
DB_NAME=telescopio_db
DB_SSLMODE=disable

# Server Configuration
PORT=8080
GIN_MODE=debug
FRONTEND_URL=http://localhost:3000

# JWT Configuration
JWT_SECRET=telescopio-dev-secret-change-in-production

# CORS Configuration
CORS_ALLOW_ORIGINS=http://localhost:3000
CORS_ALLOW_METHODS=GET,POST,PUT,PATCH,DELETE,HEAD,OPTIONS
CORS_ALLOW_HEADERS=Origin,Content-Length,Content-Type,Authorization

# Legacy Upload Configuration
UPLOADS_DIR=./uploads
MAX_FILE_SIZE=10485760

# ============================================
# MINIO STORAGE CONFIGURATION (ACTIVE)
# ============================================
STORAGE_PROVIDER=minio
MINIO_ENDPOINT=localhost:9000
MINIO_ACCESS_KEY=minioadmin
MINIO_SECRET_KEY=minioadmin123
MINIO_BUCKET=telescopio
MINIO_USE_SSL=false
MINIO_REGION=us-east-1

# Local storage (disabled when using MinIO)
STORAGE_LOCAL_PATH=./uploads
EOF

echo "✅ .env file created successfully!"
echo ""
echo "📋 Configuration:"
echo "   Storage Provider: MinIO"
echo "   MinIO Endpoint:   localhost:9000"
echo "   MinIO Bucket:     telescopio"
echo ""
echo "🚀 Next steps:"
echo "   1. Start MinIO:  ./start-minio-only.sh"
echo "   2. Run API:      go run ./cmd/api"
echo "   3. Test upload:  Open http://localhost:3000"
echo ""
echo "💡 To switch back to local storage, edit .env and set:"
echo "   STORAGE_PROVIDER=local"
echo ""
