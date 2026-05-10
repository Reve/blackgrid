# 24-hour Soak Test Plan

A soak test runs Blackgrid under realistic homelab load for an extended
period (≥ 24 hours) to catch issues that unit and smoke tests miss:
goroutine leaks, unbounded table growth, slow notification handlers,
SSE connection churn, scheduler drift, and discovery memory creep.

This is a planning + observation guide, not an automated harness. Run it
before tagging a release that touches schedulers, discovery, the event
bus, or retention.

## Setup

Bring up Blackgrid via Docker Compose on a dedicated host (a Pi, a
small VM, or a workstation that can be left alone for a day):

```bash
docker compose down -v
docker compose up --build -d
```

Configure the following monitors. Pick targets that already exist in
the lab and that you are happy to probe every 30–60 s for a day.

| # | Type     | Target example                            | Interval |
|---|----------|-------------------------------------------|----------|
| 1 | HTTP     | `https://example.lab/health`              | 30 s     |
| 2 | TCP      | `192.0.2.10:22`                           | 30 s     |
| 3 | DNS      | `lab.example.com` against `192.0.2.53`    | 60 s     |
| 4 | TLS      | `mail.lab:443` cert expiry                | 300 s    |
| 5 | Push     | from a cron on another host (`curl /push/...`) | 60 s     |
| 6 | Postgres | the Blackgrid `db` itself, role-limited   | 60 s     |

Configure at least one scheduled discovery prefix (e.g. a `/24` lab
network) with a 1 hour scan interval. Configure one notification
channel (webhook to a test endpoint or a dedicated email address) so
incident open/resolve produces an artifact.

Pre-flight:

- [ ] `docker compose ps` shows `db`, `backend`, `frontend` healthy
- [ ] `/api/v1/health` returns the expected version
- [ ] Onboarding checklist on the dashboard is dismissed (or completed)
- [ ] Settings → Diagnostics shows both schedulers running
- [ ] Prometheus is scraping `/metrics` (or you are tailing it manually)

## What to track

Capture each metric **at start, every ~6 hours, and at end**. A simple
journal is fine; a spreadsheet is better.

| Signal                       | How to measure                                  |
|------------------------------|------------------------------------------------|
| Backend RSS / heap           | `docker stats blackgrid-backend-1`, `process_resident_memory_bytes` from `/metrics` |
| Goroutine count              | `go_goroutines` from `/metrics`                |
| Database size                | `SELECT pg_size_pretty(pg_database_size('blackgrid'));` |
| Largest tables               | `SELECT relname, pg_size_pretty(pg_total_relation_size(relid)) FROM pg_catalog.pg_statio_user_tables ORDER BY pg_total_relation_size(relid) DESC LIMIT 10;` |
| Monitor result rows          | `SELECT count(*) FROM monitor_results;`        |
| Notification deliveries      | `SELECT status, count(*) FROM notification_deliveries GROUP BY 1;` |
| Open incidents               | dashboard or `SELECT count(*) FROM incidents WHERE status<>'resolved';` |
| SSE clients                  | Settings → Diagnostics → Active SSE Clients    |
| Scheduler last tick freshness| Diagnostics → Monitor / Discovery Last Tick    |
| Discovery scan duration      | Discovery page; or `SELECT max(completed_at - started_at) FROM discovery_scans WHERE completed_at > now() - interval '6 hours';` |
| Backup success               | run `deploy/scripts/backup-postgres.sh` mid-soak; verify file size and `pg_restore -l` lists tables |

## Expected observations

- Memory should stabilise within the first hour. Linear growth across
  6-hour samples is a leak.
- `go_goroutines` should plateau in the low hundreds. Steady upward
  drift indicates a leaking subscriber, scan, or worker.
- `monitor_results` row count grows roughly linearly with checks/sec
  until the retention job (default daily) trims it. After 24 h the
  oldest rows should match `RETENTION_MONITOR_RESULTS_DAYS` policy.
- Notification deliveries match incident transitions only — not the
  raw failed-check rate (see `incident_test.go`
  `TestNotifierFiresOnceOnOpenAndOnceOnResolve`).
- Diagnostics last-tick timestamps should never be more than ~10 s
  (monitor) or ~70 s (discovery) old while the schedulers are alive.
- Discovery scan durations should be roughly constant across runs of
  the same prefix; large variance suggests blocked workers or DNS
  flakiness.
- `docker compose logs backend` contains no panics, no `level=error`
  bursts, and no `subscriber buffer full` warnings.

## Failure signs

Stop the soak and investigate if any of the following appear:

- Heap RSS more than doubles between two consecutive 6-hour samples.
- `go_goroutines` grows monotonically across all samples.
- A scheduler `last_tick_at` is older than 5 minutes in Diagnostics.
- The same monitor produces more than one `incident.opened` notification
  for the same incident lifecycle.
- `notification_deliveries.status='failed'` rate trends upward (esp.
  webhook 5xx bursts).
- Database size grows past expectations (>1 GB without bulk discovery).
- Backup script exits non-zero, or `pg_restore -l` cannot list tables
  from the produced dump.
- Backend OOM-killed or restarted by Compose during the run.

## Wrap-up

When the soak completes:

- [ ] Capture final metrics into a short report (paste into the release
      notes or PR description).
- [ ] Run `deploy/scripts/backup-postgres.sh` and verify per
      [docs/backup-verification.md](backup-verification.md).
- [ ] Re-run the smoke test (`docs/smoke-test.md`) to confirm no
      drift in user-visible behaviour.
- [ ] If anything in **Failure signs** triggered, file an issue before
      tagging the release.
