# Blackgrid Operations Guide

This guide covers maintenance, monitoring, and database management for Blackgrid.

## Monitoring

### Health Checks
- **Liveness**: `GET /api/v1/health` - Returns 200 if the server is running.
- **Readiness**: `GET /api/v1/ready` - Returns 200 if the server can connect to the database.

### Prometheus Metrics
Metrics are exposed at `/metrics` in Prometheus format. Key metrics include:
- `blackgrid_http_requests_total`: API request counts.
- `blackgrid_monitor_checks_total`: Monitor execution results.
- `blackgrid_incidents_open`: Current number of active incidents.
- `blackgrid_discovery_scans_total`: Status of discovery jobs.
- `blackgrid_sse_clients_current`: Active dashboard connections.

## Database Maintenance

### Backup
A backup script is provided in `deploy/scripts/backup-postgres.sh`. It writes a timestamped file in pg_dump's **custom format** (suffix `.dump`), produced by `pg_dump -Fc`. The custom format is compressed by default and is the input format `pg_restore` expects.
```bash
./deploy/scripts/backup-postgres.sh
# → ./backups/blackgrid_<timestamp>.dump
```
It is recommended to run this via cron:
```cron
0 2 * * * /path/to/blackgrid/deploy/scripts/backup-postgres.sh >> /var/log/blackgrid-backup.log 2>&1
```

### Restore
To restore a backup, use `deploy/scripts/restore-postgres.sh`. **WARNING: This will drop and recreate objects in the target database.** The argument must be the `.dump` file produced by `backup-postgres.sh` (custom format) — not a `.sql` or `.sql.gz` file.
```bash
./deploy/scripts/restore-postgres.sh ./backups/blackgrid_20250510_020000.dump
```
Under the hood the script runs:
```bash
pg_restore -d "$DATABASE_URL" --clean --if-exists --no-owner "$BACKUP_FILE"
```
If you need a plain-text dump for grep/inspection, run `pg_restore --file=- <backup>.dump` to convert; do not feed plain SQL into `restore-postgres.sh`.

### Data Retention
Blackgrid automatically cleans up historical data based on environment variable settings. The cleanup job runs periodically in the background. You can manually trigger a cleanup via the API (admin only):
```bash
curl -X POST -H "Authorization: Bearer <TOKEN>" http://localhost:8080/api/v1/admin/retention/run
```

## Logs
Logs are emitted in JSON format to standard output. In Docker environments, view them with:
```bash
docker-compose logs -f backend
```

## Troubleshooting

### Monitor Failures
If monitors are failing unexpectedly, check the "Latest Check Details" in the UI or look for `level:error` and `msg:"monitor check failed"` in the logs.

### Discovery Not Finding Hosts
- Ensure the backend container has appropriate network permissions (ICMP requires specific capabilities if running as non-root).
- Check `DISCOVERY_TCP_TIMEOUT_MS` if your network is slow.
- Verify the Prefix is configured with the correct CIDR.
