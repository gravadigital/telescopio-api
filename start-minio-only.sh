#!/bin/bash

# Quick Start with MinIO - Alternative approach
# Starts MinIO and runs API locally with MinIO configuration

set -e

echo "🚀 Quick Start: Telescopio with MinIO Storage"
echo "=============================================="

# Detect docker compose command
if command -v docker &> /dev/null && docker compose version &> /dev/null; then
    DOCKER_COMPOSE="docker compose"
elif command -v docker-compose &> /dev/null; then
    DOCKER_COMPOSE="docker-compose"
else
    echo "❌ Error: docker compose is not installed"
    echo "   Install with: sudo apt install docker-compose-plugin"
    exit 1
fi

# Start only MinIO and PostgreSQL
echo "📦 Starting PostgreSQL and MinIO services..."
$DOCKER_COMPOSE --profile minio up -d postgres minio

echo "⏳ Waiting for services to be ready..."
sleep 3

# Check PostgreSQL
until docker exec telescopio-postgres pg_isready -U telescopio -d telescopio_db &> /dev/null 2>&1; do
    echo "   Waiting for PostgreSQL..."
    sleep 2
done
echo "✅ PostgreSQL is ready"

# Check MinIO
until curl -sf http://localhost:9000/minio/health/live &> /dev/null; do
    echo "   Waiting for MinIO..."
    sleep 2
done
echo "✅ MinIO is ready"

echo ""
echo "✅ Services started successfully!"
echo ""
echo "📍 Service URLs:"
echo "   - PostgreSQL:    localhost:5432"
echo "   - MinIO API:     http://localhost:9000"
echo "   - MinIO Console: http://localhost:9001"
echo ""
echo "🔑 MinIO Credentials:"
echo "   - Username: minioadmin"
echo "   - Password: minioadmin123"
echo ""
echo "🚀 Now start the API with MinIO configuration:"
echo ""
echo "   Option 1: Using environment variables"
echo "   ----------------------------------------"
echo "   export STORAGE_PROVIDER=minio"
echo "   export MINIO_ENDPOINT=localhost:9000"
echo "   export MINIO_ACCESS_KEY=minioadmin"
echo "   export MINIO_SECRET_KEY=minioadmin123"
echo "   export MINIO_BUCKET=telescopio"
echo "   go run ./cmd/api"
echo ""
echo "   Option 2: Using a .env file"
echo "   ---------------------------"
echo "   cat > .env << EOF"
echo "   STORAGE_PROVIDER=minio"
echo "   MINIO_ENDPOINT=localhost:9000"
echo "   MINIO_ACCESS_KEY=minioadmin"
echo "   MINIO_SECRET_KEY=minioadmin123"
echo "   MINIO_BUCKET=telescopio"
echo "   MINIO_USE_SSL=false"
echo "   DB_HOST=localhost"
echo "   DB_PORT=5432"
echo "   DB_USER=telescopio"
echo "   DB_PASSWORD=telescopio_password"
echo "   DB_NAME=telescopio_db"
echo "   EOF"
echo "   go run ./cmd/api"
echo ""
echo "📊 Manage services:"
echo "   - View logs:   $DOCKER_COMPOSE logs -f minio"
echo "   - Stop all:    $DOCKER_COMPOSE --profile minio down"
echo ""
