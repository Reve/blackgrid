# Release Checklist

Run through this list before tagging a release or promoting a build to a
homelab/production environment. Every item should pass cleanly; if a step
fails, fix it before continuing — do not "tag through" failures.

## 1. Backend tests and build

```bash
cd backend
go test ./...
go build ./...
```

The integration tests under `internal/service` require a reachable
PostgreSQL (default `postgres://blackgrid:blackgrid@localhost:5432/blackgrid`);
override with `BLACKGRID_TEST_DATABASE_URL` if needed. Skipped tests are
acceptable when no DB is available, but failed tests are not.

## 2. Frontend install and build

```bash
cd frontend
npm ci
npm run build
```

`npm ci` (not `npm install`) ensures the lockfile is honoured. The build
must complete without TypeScript or ESLint errors.

## 3. Clean Docker Compose bring-up

From the repo root:

```bash
docker compose down -v
docker compose up --build
```

`down -v` deletes the `postgres_data` volume so you exercise the
first-run path. Wait for the backend to log a successful migration and
for the frontend to serve `index.html`.

## 4. First-run smoke test

In the freshly started stack, walk through the user-visible golden path:

- [ ] First admin setup at `/setup` (creates the bootstrap admin)
- [ ] Login at `/login` with the admin credentials
- [ ] Create at least one IPAM record (site → prefix → IP address)
- [ ] Create at least one monitor of each type you ship
- [ ] Trigger an incident (point an HTTP/TCP monitor at an unreachable
      target and wait for it to flip to `down`)
- [ ] Confirm notification delivery on the configured channel
- [ ] Run a discovery scan against a small lab prefix and accept one
      result
- [ ] Open the public status page (`/status/<slug>`) without auth
- [ ] Connect to `/api/v1/events/stream` (the UI does this) and watch
      for events on a status change

## 5. Operational endpoints

- [ ] `/api/v1/health` returns `{"status":"ok", "version":...}`
- [ ] `/api/v1/ready` returns `ok` while DB is reachable, `503` when down
- [ ] `/metrics` exposes Prometheus metrics (Go runtime + Blackgrid app
      metrics)

## 6. Backup and restore

- [ ] Run the backup script (`backups/`) and confirm a `.dump` file is
      produced
- [ ] If feasible, restore the dump into a clean database and verify
      schema/data via `psql` (`\dt`, sample selects). See
      [docs/backup-verification.md](backup-verification.md).

## 7. Tag and ship

When all of the above pass:

- [ ] Update version in `backend/internal/version` (or pass via
      `-ldflags`) and rebuild images with `VERSION`, `COMMIT_SHA`,
      `BUILD_DATE` build args
- [ ] Tag the git commit
- [ ] Note the release in changelog / release notes
