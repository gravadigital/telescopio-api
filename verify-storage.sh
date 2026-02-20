#!/bin/bash

# Verify Storage - Check where files are being stored
# Usage: ./verify-storage.sh

set -e

# Detect docker compose command
if command -v docker &> /dev/null && docker compose version &> /dev/null; then
    DOCKER_COMPOSE="docker compose"
elif command -v docker-compose &> /dev/null; then
    DOCKER_COMPOSE="docker-compose"
else
    DOCKER_COMPOSE=""
fi

echo "🔍 Telescopio Storage Verification"
echo "===================================="
echo ""

# Check configuration
echo "📋 Current Configuration:"
if [ -f .env ]; then
    PROVIDER=$(grep "^STORAGE_PROVIDER=" .env 2>/dev/null | cut -d'=' -f2 || echo "local")
else
    PROVIDER="local"
fi

echo "   Storage Provider: ${PROVIDER:-local (default)}"
echo ""

# Check based on provider
if [ "$PROVIDER" == "minio" ] || [ "$PROVIDER" == "MINIO" ]; then
    echo "☁️  MinIO Storage Status:"
    
    # Check if MinIO is running
    if curl -sf http://localhost:9000/minio/health/live &> /dev/null; then
        echo "   ✅ MinIO server is running"
        echo "   📍 MinIO API: http://localhost:9000"
        echo "   📍 MinIO Console: http://localhost:9001"
        echo ""
        
        # Try to list bucket contents (requires mc client)
        if command -v mc &> /dev/null; then
            echo "   📦 Checking bucket contents..."
            mc alias set verify http://localhost:9000 minioadmin minioadmin123 &> /dev/null || true
            mc ls verify/telescopio 2>/dev/null | tail -10 || echo "      (Install 'mc' client to see bucket contents)"
        else
            echo "   💡 Install MinIO client 'mc' to view bucket contents:"
            echo "      wget https://dl.min.io/client/mc/release/linux-amd64/mc"
            echo "      chmod +x mc && sudo mv mc /usr/local/bin/"
        fi
    else
        echo "   ❌ MinIO server is NOT running"
        echo "   💡 Start with: $DOCKER_COMPOSE --profile minio up -d minio"
    fi
else
    echo "📁 Local Storage Status:"
    
    UPLOAD_DIR="./uploads"
    if [ -d "$UPLOAD_DIR" ]; then
        FILE_COUNT=$(ls -1 "$UPLOAD_DIR" 2>/dev/null | wc -l)
        TOTAL_SIZE=$(du -sh "$UPLOAD_DIR" 2>/dev/null | cut -f1)
        
        echo "   ✅ Upload directory exists: $UPLOAD_DIR"
        echo "   📊 Total files: $FILE_COUNT"
        echo "   💾 Total size: $TOTAL_SIZE"
        echo ""
        
        if [ $FILE_COUNT -gt 0 ]; then
            echo "   📄 Recent uploads (last 10):"
            ls -lht "$UPLOAD_DIR" | head -11 | tail -10 | awk '{printf "      %s %s %s  %s\n", $6, $7, $8, $9}'
            echo ""
            
            echo "   📅 Latest file:"
            LATEST=$(ls -t "$UPLOAD_DIR" | head -1)
            ls -lh "$UPLOAD_DIR/$LATEST" | awk '{printf "      %s  %s %s %s  %s\n", $5, $6, $7, $8, $9}'
        else
            echo "   📭 No files uploaded yet"
        fi
    else
        echo "   ⚠️  Upload directory does not exist: $UPLOAD_DIR"
        echo "   💡 It will be created automatically on first upload"
    fi
fi

echo ""
echo "=================================="

# Check database for attachment records
echo ""
echo "🗄️  Database Records:"
if [ -n "$DOCKER_COMPOSE" ] && docker ps | grep -q telescopio-postgres; then
    ATTACHMENT_COUNT=$(docker exec telescopio-postgres-dev psql -U telescopio -d telescopio_db -tAc "SELECT COUNT(*) FROM attachments;" 2>/dev/null || echo "0")
    echo "   Total attachments in database: $ATTACHMENT_COUNT"
    
    if [ "$ATTACHMENT_COUNT" -gt 0 ]; then
        echo ""
        echo "   Recent attachments:"
        docker exec telescopio-postgres-dev psql -U telescopio -d telescopio_db -c "
            SELECT 
                LEFT(id::text, 8) as id,
                original_name,
                file_size,
                uploaded_at 
            FROM attachments 
            ORDER BY uploaded_at DESC 
            LIMIT 5;
        " 2>/dev/null | grep -v "row" || echo "      (Could not fetch records)"
    fi
else
    echo "   ⚠️  Cannot connect to database"
    echo "   💡 Start with: $DOCKER_COMPOSE up -d postgres"
fi

echo ""
echo "=================================="
echo ""

# Quick commands
echo "💡 Quick Commands:"
echo "   View all files:     ls -lh ./uploads/"
echo "   Count files:        ls -1 ./uploads/ | wc -l"
echo "   Check size:         du -sh ./uploads/"
echo "   Watch uploads:      watch -n2 'ls -lht ./uploads/ | head -15'"
echo "   Clear all files:    rm -rf ./uploads/*"
echo ""
echo "   Database query:     docker exec telescopio-postgres-dev psql -U telescopio -d telescopio_db -c 'SELECT * FROM attachments;'"
echo ""
