#!/bin/bash

# Start Telescopio API with MinIO Storage
# Usage: ./start-with-minio.sh

set -e

echo "🚀 Starting Telescopio with MinIO Storage"
echo "=========================================="

# Check if docker compose or docker-compose is available
if command -v docker &> /dev/null && docker compose version &> /dev/null; then
    DOCKER_COMPOSE="docker compose"
elif command -v docker-compose &> /dev/null; then
    DOCKER_COMPOSE="docker-compose"
else
    echo "❌ Error: docker compose is not installed"
    echo "   Install with: sudo apt install docker-compose-plugin"
    exit 1
fi

echo "📦 Using: $DOCKER_COMPOSE"

# Start services with MinIO profile
echo "📦 Starting PostgreSQL and MinIO..."
$DOCKER_COMPOSE --profile minio up -d postgres minio

# Wait for services to be healthy
echo "⏳ Waiting for services to be ready..."
sleep 5

# Check PostgreSQL
until $DOCKER_COMPOSE exec -T postgres pg_isready -U telescopio -d telescopio_db &> /dev/null; do
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

# Start API with MinIO configuration
echo "🔧 Starting API with MinIO storage..."
$DOCKER_COMPOSE --profile minio up -d api

echo ""
echo "✅ All services started successfully!"
echo ""
echo "📍 Service URLs:"
echo "   - API:           http://localhost:8080"
echo "   - MinIO Console: http://localhost:9001"
echo "   - PostgreSQL:    localhost:5432"
echo ""
echo "🔑 MinIO Credentials:"
echo "   - Username: minioadmin"
echo "   - Password: minioadmin123"
echo ""
echo "📊 Check status with: $DOCKER_COMPOSE ps"
echo "📝 View logs with:    $DOCKER_COMPOSE logs -f api"
echo "🛑 Stop all with:     $DOCKER_COMPOSE --profile minio down"
