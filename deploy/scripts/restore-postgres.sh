#!/bin/bash
set -e

BACKUP_FILE=$1

if [ -z "$BACKUP_FILE" ]; then
    echo "Usage: $0 <backup_file>"
    exit 1
fi

if [ ! -f "$BACKUP_FILE" ]; then
    echo "Error: Backup file $BACKUP_FILE not found."
    exit 1
fi

if [ -z "$DATABASE_URL" ]; then
    echo "Error: DATABASE_URL environment variable is not set."
    exit 1
fi

echo "WARNING: This will overwrite the current database!"
if [ "$FORCE" != "true" ]; then
    read -p "Are you sure you want to proceed? (y/N) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        echo "Restore cancelled."
        exit 1
    fi
fi

echo "Starting restore of Blackgrid database from $BACKUP_FILE..."
# Use pg_restore. -c drops objects before recreating. --if-exists avoids errors on drop.
pg_restore -d "$DATABASE_URL" --clean --if-exists --no-owner "$BACKUP_FILE"

echo "Restore completed successfully."
