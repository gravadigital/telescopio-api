#!/bin/bash
# Script rápido para verificar MinIO local

echo "🔍 Verificación MinIO Local"
echo "================================"
echo ""

# 1. Servicios
echo "📦 Servicios:"
docker compose ps minio postgres api | tail -4
echo ""

# 2. Health
echo "💚 Health checks:"
curl -s http://localhost:8080/health | jq -r '"API: " + .status' 2>/dev/null || echo "API: ❌ No responde"
curl -s http://localhost:9000/minio/health/live >/dev/null && echo "MinIO: ✅ OK" || echo "MinIO: ❌ No responde"
echo ""

# 3. Archivos en MinIO
echo "📁 Archivos en bucket 'telescopio':"
FILES=$(docker compose exec minio mc ls /data/telescopio/ 2>/dev/null)
if [ -z "$FILES" ]; then
    echo "  (vacío - sin archivos aún)"
else
    echo "$FILES" | tail -5
    COUNT=$(echo "$FILES" | wc -l | tr -d ' ')
    echo "  Total: $COUNT archivos"
fi
echo ""

# 4. Consola MinIO
echo "🌐 Consola MinIO:"
echo "  URL: http://localhost:9001"
echo "  User: minioadmin"
echo "  Pass: minioadmin123"
echo ""

echo "✅ Para subir archivos: http://localhost:3000"
