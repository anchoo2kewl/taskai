#!/bin/bash
set -e

echo "🚀 Migrating Staging to Postgres"
echo "================================="

# Configuration
POSTGRES_PASSWORD="${POSTGRES_PASSWORD:-taskai-staging-secure-password}"
POSTGRES_USER="taskai"
POSTGRES_DB="taskai"
POSTGRES_HOST="localhost"
POSTGRES_PORT="5432"

echo "📋 Pre-migration checklist:"
echo "1. Backing up current SQLite database..."
docker compose exec -T api cp /data/taskai.db /data/taskai.db.backup
echo "✅ Backup created: /data/taskai.db.backup"

echo ""
echo "2. Building migration tool..."
cd ../api/cmd/migrate-to-postgres
go build -o migrate-to-postgres main.go
echo "✅ Migration tool built"

echo ""
echo "3. Copying migration tool to container..."
docker cp migrate-to-postgres taskai-staging-api-1:/tmp/
echo "✅ Tool copied"

echo ""
echo "4. Running migration..."
docker compose exec -T api /tmp/migrate-to-postgres \
  -sqlite /data/taskai.db \
  -postgres "postgres://${POSTGRES_USER}:${POSTGRES_PASSWORD}@taskai-staging-postgres-1:5432/${POSTGRES_DB}?sslmode=disable"

echo ""
echo "✅ Migration completed!"
echo ""
echo "📋 Next steps:"
echo "1. Update .env to set DB_DRIVER=postgres"
echo "2. Update .env to set DB_DSN with Postgres connection string"
echo "3. Restart API container: docker compose restart api"
echo "4. Verify: curl http://localhost:XXXX/api/health"
