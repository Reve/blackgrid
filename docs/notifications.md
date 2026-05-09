# Notifications

Blackgrid delivers incident events to user-configured notification channels.
Two channel types are supported in Phase 3: **webhook** and **smtp**.

Delivery is synchronous (one HTTP/SMTP attempt per channel per event); a
background queue is intentionally out of scope for this phase. Failures of one
channel never affect the delivery of others, and a notification failure can
never crash monitor execution.

## Events

| Event | When |
| --- | --- |
| `incident.opened` | A new incident is opened (down or degraded transition). |
| `incident.resolved` | An open/acknowledged incident is resolved (auto or manual). |
| `test` | The user pressed “Test” in the UI / hit `POST /notification-channels/:id/test`. |

## Webhook

### Config

```json
{
  "url": "https://example.local/webhook",
  "method": "POST",
  "headers": { "Authorization": "Bearer example" },
  "timeout_seconds": 10
}
```

Rules:

- Only `http` and `https` URLs are accepted.
- `method` defaults to `POST`.
- Body is JSON with `Content-Type: application/json`.
- A delivery row is recorded for every attempt, with `status` of `sent` or
  `failed` and `last_error` populated on failure.
- Sensitive headers are not logged.
- Non-2xx (≥400) responses are treated as failures.

### Payload — `incident.opened`

```json
{
  "event": "incident.opened",
  "severity": "critical",
  "incident": {
    "id": "...",
    "status": "open",
    "started_at": "...",
    "acknowledged_at": null,
    "resolved_at": null
  },
  "monitor": {
    "id": "...",
    "name": "PostgreSQL",
    "type": "tcp",
    "target": "10.10.13.20:5432",
    "status": "down"
  },
  "message": "Monitor PostgreSQL is down"
}
```

### Payload — `incident.resolved`

Same shape; `event` is `incident.resolved`, `incident.status` is `resolved`,
`incident.resolved_at` is populated, and `monitor.status` reflects the
recovery (typically `up`).

## SMTP

### Config

```json
{
  "host": "smtp.example.local",
  "port": 587,
  "username": "blackgrid@example.local",
  "password": "secret",
  "from": "blackgrid@example.local",
  "to": ["admin@example.local"],
  "use_tls": true
}
```

Rules:

- `host`, `port`, `from` and at least one `to` recipient are required.
- `password` is **never** logged.
- API responses mask the password as `********`.
- When updating an SMTP channel, leaving `password` empty or sending the
  masked sentinel `********` preserves the existing stored password.
- `use_tls: true` opens an implicit TLS connection (typically port 465).
- `use_tls: false` uses plain SMTP (port 25/587 STARTTLS is not negotiated in
  this phase — operators should set `use_tls: true` whenever the server
  supports it).

The email body is a plain-text dump of the same structured payload used for
webhooks.

## Secret storage

Notification channel configs are stored as JSONB in PostgreSQL. There is **no
encryption-at-rest layer in this phase**: passwords and bearer tokens live in
plaintext inside the database. Operators must protect the database itself
(disk encryption, restricted user grants, network isolation). A future phase
will introduce a secrets layer.

## Implementation pointers

- Service: `backend/internal/service/notification.go`
- Handlers: `backend/internal/api/handlers/notification.go`
- Schema: `backend/migrations/003_incidents_notifications.sql`
