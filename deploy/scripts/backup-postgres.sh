#!/bin/bash
set -e

# Configuration
BACKUP_DIR="${BACKUP_DIR:-./backups}"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
BACKUP_FILE="${BACKUP_DIR}/blackgrid_${TIMESTAMP}.dump"

mkdir -p "$BACKUP_DIR"

if [ -z "$DATABASE_URL" ]; then
    echo "Error: DATABASE_URL environment variable is not set."
    exit 1
fi

echo "Starting backup of Blackgrid database..."
# Use pg_dump with custom format
pg_dump "$DATABASE_URL" -Fc -f "$BACKUP_FILE"

echo "Backup completed: $BACKUP_FILE"
echo "To restore this backup, use: ./restore-postgres.sh $BACKUP_FILE"
