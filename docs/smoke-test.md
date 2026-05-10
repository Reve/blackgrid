# Manual smoke test

Run this checklist after a clean deploy or a substantial change to verify
the application end-to-end. It assumes a clean database (no users yet).

Each step lists the action, the expected result, and where to look if it
fails.

## 0. Bring the stack up

```bash
docker compose up --build -d
docker compose ps
```

* All three containers (`db`, `backend`, `frontend`) report `Up` and the DB
  reports `(healthy)`.
* Backend logs (`docker compose logs -f backend`) show migrations running
  through `008_session_hash_at_rest.sql` without error.

## 1. First-admin setup

* Open `http://localhost:3000`. The frontend redirects to `/setup`.
* Submit name + email + password (≥12 chars).
* Expected: redirect to dashboard; no further setup prompt on refresh.
* On failure: check `audit_log` for a `user.create` row.

## 2. Login and session

* Log out via the user menu.
* Log back in with the same credentials.
* Expected: session cookie `blackgrid_session` is set with `HttpOnly`,
  `SameSite=Lax`, and (in production) `Secure`.
* Open DevTools → Application → Cookies. The cookie value is a base64
  random token, **not** a SHA-256 hex string. The hash is what the DB
  stores; the cookie value is the plaintext.

```sql
SELECT length(session_hash), session_hash FROM sessions LIMIT 1;
-- length should be 64 (SHA-256 hex), value must NOT equal the cookie.
```

## 3. Core IPAM flow

* Settings → Sites → create a site.
* IPAM → Prefixes → create `192.0.2.0/24` linked to the site.
* `GET /api/v1/prefixes/{id}/utilization` returns `total_hosts: 254`,
  `allocated: 0`, `free: 254`, `percent_used: 0`.
* `GET /api/v1/prefixes/{id}/next-available` returns the first usable host.
* IP Addresses → create `192.0.2.10` under the prefix → `POST .../reserve`
  → `POST .../assign` → `POST .../release`. Status field follows
  `available` → `reserved` → `assigned` → `available`.
* Each step writes an `ipam.*` row to `audit_log`.

## 4. Monitor creation and testing

* Monitors → Add → HTTP monitor against `https://example.com`.
* Click "Test" → response body shows status `up` with a latency.
* Expected: `audit_log` has a `monitor.create` row and a `monitor.test` row.
* Live dashboard shows the monitor flip to `up` without manual refresh
  (SSE delivered `monitor.tested` + `monitor.result_created`).
* `GET /api/v1/monitors/{id}/results?limit=5&offset=0` returns the latest
  five results.

## 5. Push monitor overdue behaviour

* Monitors → Add → Push monitor with `grace_seconds: 30`.
* Copy the one-time token from the response. Construct
  `POST /push/<token>?status=up`.
* Send one heartbeat → monitor flips to `up` (live dashboard updates).
* Stop sending. After 30 seconds + one scheduler tick (~5s) the monitor
  flips to `down` and an incident is opened — see Incidents page.
* Resend `POST /push/<token>?status=up` → incident resolves automatically;
  notification fires through the configured channels.
* Sanity check: `last_heartbeat_at` advances on each push;
  `last_checked_at` advances on every scheduler tick. The two values are
  different — the scheduler does **not** mask a missing heartbeat.

## 6. Push down → incident

* With a push monitor in `up`, send `POST /push/<token>?status=down`.
* Expected: incident opens immediately (without waiting for grace), and
  notification delivery rows appear with `status='sent'`.
* Send `?status=up` → incident resolves; recovery notification fires.

## 7. Discovery scan

* IPAM → Prefixes → enable scanning on a small home prefix or a known
  loopback range; click "scan now".
* Discovery page shows the scan progress to `completed`.
* Discovery results list populates. Accept one and ignore one — both
  actions write `discovery.result_accept`/`discovery.result_ignore` audit
  rows.

## 8. Status pages

* Status Pages → Create → public, one monitor attached.
* Visit `http://localhost:8080/status/<slug>` (no auth) → renders.
* Toggle the page to private → public URL now returns 404.
* Each change appears as `status_page.create`/`update` in `audit_log`.

## 9. SSE live dashboard

* With the dashboard open in tab A, edit a monitor in tab B.
* Tab A reflects the change without refresh.
* Curl the stream directly:

  ```bash
  curl -N --cookie "blackgrid_session=…" http://localhost:8080/api/v1/events/stream
  ```

  Frames must contain all three lines: `id:`, `event:`, `data:` followed
  by a blank line. Keepalives appear as `: keepalive` every 20s.

## 10. Metrics endpoint

```bash
curl -s http://localhost:8080/metrics | grep blackgrid_
```

Expected counters present and incrementing under load:
`blackgrid_http_requests_total`, `blackgrid_event_bus_events_total`,
`blackgrid_sse_clients_current`, `blackgrid_incidents_open`.

## 11. Error envelope sanity

```bash
curl -s -H "Content-Type: application/json" \
  http://localhost:8080/api/v1/monitors/not-a-uuid | jq
```

```json
{
  "error": {
    "code": "validation_error",
    "message": "invalid UUID",
    "request_id": "..."
  }
}
```

Force a 500 by hitting any handler with a poisoned payload — the body is
the same shape and `message` is the generic `"internal error"` (no raw
SQL or stack trace).

## 12. Role-aware UI

* Create a viewer-role user via `/api/v1/users` (admin only).
* Log in as that viewer.
* Monitors page: no "Add Monitor", "Edit", "Delete", "Pause", or "Test"
  buttons. The detail panel is read-only.
* Settings → Notifications: the table is visible, but no `+ WEBHOOK` /
  `+ SMTP` buttons; the action column shows `read-only`.
* Settings: the `USERS`, `API TOKENS`, and `AUDIT LOG` tabs are hidden.

## 13. Backup + restore round-trip

Follow [docs/operations.md](operations.md) — `pg_dump … --format=custom -f
backup.dump`, drop the database, restore with `pg_restore … backup.dump`.
After restore, log in as the existing admin and re-run steps 4–6.

## Pass criteria

A run is "green" when every step completes as described **and** the
backend log shows no `ERROR` lines that were not introduced by deliberate
fault injection (e.g. step 11). Capture screenshots of dashboard, status
page, and metrics output for the change record.
