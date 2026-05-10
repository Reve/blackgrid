# Backup & Restore Verification

This document describes how to verify a Blackgrid backup is restorable. A
backup that has never been restored is just a file — treat regular restore
drills as part of operating Blackgrid.

The backup and restore scripts live in `deploy/scripts/`:

- `backup-postgres.sh` — produces a `pg_dump -Fc` (custom format) `.dump`
  file in `BACKUP_DIR` (default `./backups`).
- `restore-postgres.sh <file>` — invokes `pg_restore --clean --if-exists
  --no-owner` against `DATABASE_URL`. Set `FORCE=true` to skip the
  interactive confirmation.

Both scripts read `DATABASE_URL` from the environment and rely on
`pg_dump` / `pg_restore` from the system PATH (PostgreSQL client tools
package).

## Producing a backup against Docker Compose

Run `pg_dump` against the Compose-managed database via the `db` service:

```bash
# Dump from inside the running db container into ./backups on the host.
mkdir -p backups
docker compose exec -T db \
  pg_dump -U blackgrid -d blackgrid -Fc \
  > backups/blackgrid_$(date +%Y%m%d_%H%M%S).dump
```

Or use the script with `DATABASE_URL` pointed at the published port:

```bash
DATABASE_URL=postgres://blackgrid:blackgrid@localhost:5432/blackgrid?sslmode=disable \
  ./deploy/scripts/backup-postgres.sh
```

Both paths produce the same `.dump` artifact.

## Restoring into a clean database

The cleanest verification is to restore into a **separate** database (so
you do not stomp on live data). With Compose, that means a sidecar DB
container or a temporary local Postgres.

### Option A — restore into a throwaway container

```bash
# 1. Start a clean Postgres on a free port.
docker run -d --name bg-restore-test \
  -e POSTGRES_USER=blackgrid \
  -e POSTGRES_PASSWORD=blackgrid \
  -e POSTGRES_DB=blackgrid \
  -p 55432:5432 \
  postgres:15-alpine

# 2. Wait for it to come up.
until docker exec bg-restore-test pg_isready -U blackgrid >/dev/null; do sleep 1; done

# 3. Restore.
DATABASE_URL=postgres://blackgrid:blackgrid@localhost:55432/blackgrid?sslmode=disable \
FORCE=true \
  ./deploy/scripts/restore-postgres.sh backups/blackgrid_YYYYMMDD_HHMMSS.dump

# 4. Smoke-check.
docker exec bg-restore-test psql -U blackgrid -d blackgrid -c "\dt"
docker exec bg-restore-test psql -U blackgrid -d blackgrid \
  -c "SELECT count(*) FROM monitors;"

# 5. Tear down.
docker rm -f bg-restore-test
```

A successful run prints `Restore completed successfully.` and the
`\dt` listing should include at least: `users`, `sites`, `prefixes`,
`ip_addresses`, `devices`, `vlans`, `monitors`, `monitor_results`,
`incidents`, `notification_channels`, `notification_deliveries`,
`status_pages`, `status_page_monitors`, `discovery_scans`,
`discovery_results`, `audit_log`, `api_tokens`.

### Option B — restore into the running Compose `db` (DESTRUCTIVE)

Only do this in test environments. `--clean` drops every object first.

```bash
docker compose exec -T db \
  psql -U blackgrid -d blackgrid -c "SELECT 'restoring';"
docker compose exec -T db \
  pg_restore -U blackgrid -d blackgrid --clean --if-exists --no-owner \
  < backups/blackgrid_YYYYMMDD_HHMMSS.dump
```

## Optional: restore smoke-test command

For automation, the following one-liner returns non-zero if the dump
cannot be restored or schema verification fails:

```bash
DUMP=backups/blackgrid_YYYYMMDD_HHMMSS.dump \
bash -euo pipefail -c '
  docker rm -f bg-restore-test >/dev/null 2>&1 || true
  docker run -d --name bg-restore-test \
    -e POSTGRES_USER=blackgrid -e POSTGRES_PASSWORD=blackgrid \
    -e POSTGRES_DB=blackgrid -p 55432:5432 postgres:15-alpine >/dev/null
  until docker exec bg-restore-test pg_isready -U blackgrid >/dev/null; do sleep 1; done
  DATABASE_URL=postgres://blackgrid:blackgrid@localhost:55432/blackgrid?sslmode=disable \
    FORCE=true ./deploy/scripts/restore-postgres.sh "$DUMP"
  docker exec bg-restore-test psql -U blackgrid -d blackgrid -tAc \
    "SELECT count(*) FROM information_schema.tables WHERE table_schema=\"public\"" \
    | awk "{ if (\$1 < 10) { print \"too few tables: \" \$1; exit 1 } }"
  docker rm -f bg-restore-test >/dev/null
'
```

Treat any non-zero exit as a failed verification.

## Notes

- The application runs `db.RunMigrations` on startup. After a restore from
  an older dump, expect Blackgrid to apply any newer migrations on first
  boot — the dump should have `schema_migrations` populated, so only
  newer migrations execute.
- Notification channel secrets are stored as JSON in the dump (see
  [known-limitations.md](known-limitations.md)). Treat backup files as
  sensitive: keep them off public storage and encrypt them at rest.
