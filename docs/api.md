# Blackgrid API — Discovery endpoints

All endpoints are prefixed with `/api/v1`.

## Discovery scans

### `GET /discovery/scans`

Query parameters (all optional):

| Param | Type | Default |
| --- | --- | --- |
| `prefix_id` | UUID | — |
| `status` | `queued`/`running`/`completed`/`failed`/`cancelled` | — |
| `limit` | int | `100` |
| `offset` | int | `0` |

Returns an array of `DiscoveryScan` ordered by `created_at desc`.

### `POST /discovery/scans`

Body: `{ "prefix_id": "<uuid>" }`

Starts a manual scan against a stored prefix. Behaviour:

- `400` if `prefix_id` is missing or unknown.
- `422` if the prefix is too large for IPv4 or is IPv6.
- `409` if another scan for that prefix is already `queued`/`running`.
- `201` with the new `DiscoveryScan` otherwise.

### `GET /discovery/scans/{id}`

Returns a single `DiscoveryScan`. `404` if unknown.

### `POST /prefixes/{id}/scan`

Equivalent to `POST /discovery/scans` for the path-bound prefix.

## Discovery results

### `GET /discovery/results`

Query parameters (all optional):

| Param | Type | Default |
| --- | --- | --- |
| `scan_id` | UUID | — |
| `prefix_id` | UUID | — |
| `classification` | string | — |
| `ignored` | bool | `false` (effective default for the UI) |
| `limit` | int | `100` |
| `offset` | int | `0` |

Returns `DiscoveryResult[]` ordered by `seen_at desc`.

### `POST /discovery/results/{id}/accept`

Body (all optional): `{ "hostname": "...", "fqdn": "...", "status": "..." }`

- If the result is already accepted, returns the linked `IPAddress`.
- If the IP address already exists in the prefix, links to it without
  creating a duplicate.
- Otherwise creates a new `ip_addresses` row using:
  - `address` from the discovery result;
  - `description` = first non-empty of `fqdn`, `hostname`, `reverse_dns`,
    discovery hostname;
  - `status` = request `status` or `discovered`;
  - `last_seen_at` set to now.

### `POST /discovery/results/{id}/ignore`

Marks the result `ignored = true`, sets `classification` to `ignored`, and
returns the updated record.

## Prefix scan configuration

### `PUT /prefixes/{id}/scan-config`

Body: `{ "scan_enabled": bool, "scan_interval_seconds": int }`

- `scan_interval_seconds` must be `>= 60`.
- Returns the updated `Prefix`.

## Schemas

```ts
type MonitorType = 'http' | 'tcp' | 'ping' | 'dns' | 'tls' | 'push' | 'postgres';

type DiscoveryScan = {
  id: string;
  prefix_id: string;
  status: 'queued' | 'running' | 'completed' | 'failed' | 'cancelled';
  started_at: string | null;
  completed_at: string | null;
  error: string | null;
  created_at: string;
  updated_at: string;
};

type DiscoveryClassification =
  | 'known' | 'new' | 'changed' | 'duplicate' | 'stale' | 'ignored';

type DiscoveryResult = {
  id: string;
  scan_id: string;
  prefix_id: string;
  address: string;            // bare IPv4 address
  mac_address: string | null;
  hostname: string | null;
  reverse_dns: string | null;
  open_ports: number[];
  latency_ms: number | null;
  classification: DiscoveryClassification;
  seen_at: string;
  ignored: boolean;
  accepted_at: string | null;
  created_ip_address_id: string | null;
  created_at: string;
};
```

---

# Incidents

All endpoints are prefixed with `/api/v1`.

## `GET /incidents`

Query parameters (all optional):

| Param | Type | Default | Notes |
| --- | --- | --- | --- |
| `status` | `open`/`acknowledged`/`resolved` | — | |
| `severity` | `info`/`warning`/`critical` | — | |
| `monitor_id` | UUID | — | |
| `limit` | int | `100` | |
| `offset` | int | `0` | |

Returns an array of `Incident` ordered by `started_at desc`.

## `GET /incidents/{id}`

Returns a single `Incident` or `404`.

## `GET /incidents/counts`

Returns dashboard tallies:
```json
{ "open_count": 0, "acknowledged_count": 0, "critical_count": 0, "resolved_24h_count": 0 }
```

## `POST /incidents/{id}/acknowledge`

Body: `{ "note": "optional string" }`

Behaviour:
- If `open`, transitions to `acknowledged` and stamps `acknowledged_at`.
- If already `acknowledged`, returns the current incident (idempotent).
- If `resolved`, returns `409 Conflict`.

## `POST /incidents/{id}/resolve`

Body: `{ "note": "optional string" }`

Behaviour:
- If `open` or `acknowledged`, transitions to `resolved` and stamps `resolved_at`.
- If already `resolved`, returns the current incident (idempotent).

### `Incident` shape

```ts
type Incident = {
  id: string;
  monitor_id: string;
  status: 'open' | 'acknowledged' | 'resolved';
  severity: 'info' | 'warning' | 'critical';
  started_at: string | null;
  acknowledged_at: string | null;
  resolved_at: string | null;
  summary: string;
  details: string | null;
  created_at: string | null;
  updated_at: string | null;
};
```

---

# Notification channels

## `GET /notification-channels`
Returns all channels with secrets masked.

## `POST /notification-channels`
Body:
```json
{
  "name": "Ops",
  "channel_type": "webhook",
  "enabled": true,
  "config": { "url": "https://example.local/hook", "method": "POST", "headers": {} }
}
```
SMTP config:
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

## `GET /notification-channels/{id}`
Returns a single channel (password masked as `********`).

## `PATCH /notification-channels/{id}`
Same shape as `POST`. For SMTP, omitting `password` (or sending the masked sentinel `********`) preserves the stored password.

## `DELETE /notification-channels/{id}`
Returns `204`.

## `POST /notification-channels/{id}/test`
Sends a test payload immediately and stores a delivery row. Returns:
```json
{ "status": "sent" | "failed", "event_type": "test", "error": "...", "sent_at": "..." }
```

# Status pages

## `GET /api/v1/status-pages`
Returns all configured status pages.

## `POST /api/v1/status-pages`
Creates a status page. Slug is generated from name when omitted.
```json
{
  "name": "Homelab Core Services",
  "slug": "homelab-core",
  "description": "Core internal services",
  "public": false,
  "show_uptime": true,
  "show_incidents": true
}
```
- `400` — invalid slug or missing name
- `409` — slug already in use

## `GET /api/v1/status-pages/{id}`
Returns admin metadata plus attached monitors with their full monitor objects.

## `PATCH /api/v1/status-pages/{id}`
Partial update. Any combination of `name`, `slug`, `description`, `public`, `show_uptime`, `show_incidents`.

## `DELETE /api/v1/status-pages/{id}`
Cascade-removes attached monitor links. Returns `204`.

## `POST /api/v1/status-pages/{id}/monitors`
Attaches a monitor to a status page.
```json
{ "monitor_id": "...", "display_name": "PostgreSQL", "display_order": 10 }
```
If `display_order` is omitted, the next available order (max + 10) is used.
- `409` — monitor already attached

## `PATCH /api/v1/status-pages/{id}/monitors/{monitor_id}`
Updates display name and/or order for an attached monitor.

## `DELETE /api/v1/status-pages/{id}/monitors/{monitor_id}`
Detaches a monitor from a status page.

## `POST /api/v1/status-pages/{id}/monitors/reorder`
Reorders attached monitors. Each ID must currently be attached or the entire request is rejected.
```json
{ "monitor_ids": ["id1", "id2", "id3"] }
```
Sets `display_order` to 10, 20, 30, ... in the order given.

# Public status endpoint

## `GET /status/{slug}`
Public-safe page representation. Returns `404` if the page does not exist OR is private (existence is not leaked).

```json
{
  "name": "...",
  "slug": "...",
  "description": "...",
  "aggregate_status": "up" | "degraded" | "down" | "empty",
  "monitors": [
    {
      "display_name": "...",
      "monitor_type": "http",
      "status": "up",
      "last_checked_at": "...",
      "uptime_24h": 99.84,
      "uptime_30d": 99.51
    }
  ],
  "incidents": [
    {
      "monitor_display_name": "...",
      "severity": "critical",
      "status": "resolved",
      "started_at": "...",
      "resolved_at": "...",
      "summary": "..."
    }
  ]
}
```

The public response **never** includes monitor config, IP/device metadata, notification channels, raw check details, internal notes, or other private information.

---

# Push Heartbeats

Passive monitoring endpoints.

## `GET /push/{token}`
## `POST /push/{token}`

Accepts heartbeats from external scripts. Does not require user session auth; the token itself is the credential.
Optional query/body parameters:
- `status`: `up`, `down`, `degraded` (default: `up`)
- `message`: human-readable status message
- `latency_ms`: numerical latency in milliseconds

## `POST /api/v1/monitors/{id}/rotate-push-token`
Generates a new token for a push monitor. Returns the plaintext token once.
```json
{
  "token": "...",
  "message": "Push token rotated. Store this token securely; it will not be shown again.",
  "push_url": "/push/..."
}
```

## IPAM endpoints

The currently implemented IPAM surface is:

### Prefixes

- `GET /api/v1/prefixes`
- `GET /api/v1/prefixes/{id}`
- `GET /api/v1/prefixes/{id}/addresses` — all `ip_addresses` recorded under this prefix.
- `GET /api/v1/prefixes/{id}/utilization` — returns `{prefix_id, prefix, total_hosts, allocated, free, percent_used}`. `total_hosts` is the addressable host count (network and broadcast excluded for IPv4 prefixes shorter than `/31`). `allocated` counts rows with `status != "available"`.
- `GET /api/v1/prefixes/{id}/next-ip`
- `GET /api/v1/prefixes/{id}/next-available` — alias for `next-ip`, matching the original phased spec name.
- `POST /api/v1/prefixes` (operator)
- `PUT /api/v1/prefixes/{id}` (operator)
- `PUT /api/v1/prefixes/{id}/scan-config` (operator)
- `DELETE /api/v1/prefixes/{id}` (operator)
- `POST /api/v1/prefixes/{id}/scan` (operator) — kick a discovery scan for this prefix.

### IP addresses

- `GET /api/v1/ip-addresses`
- `GET /api/v1/ip-addresses/{id}`
- `POST /api/v1/ip-addresses` (operator)
- `PUT /api/v1/ip-addresses/{id}` (operator)
- `POST /api/v1/ip-addresses/{id}/reserve` (operator) — sets `status` to `reserved`.
- `POST /api/v1/ip-addresses/{id}/assign` (operator) — sets `status` to `assigned`.
- `POST /api/v1/ip-addresses/{id}/release` (operator) — sets `status` back to `available`.
- `DELETE /api/v1/ip-addresses/{id}` (operator)

### Devices

- `GET /api/v1/devices`
- `GET /api/v1/devices/{id}`
- `POST /api/v1/devices` (operator)
- `PUT /api/v1/devices/{id}` (operator)
- `DELETE /api/v1/devices/{id}` (operator)

### Intentional deviations from the original phased plan

- `GET /api/v1/devices/{id}/interfaces` and `POST /api/v1/devices/{id}/interfaces` are **not implemented**. The current schema does not model a standalone `device_interfaces` table; interface metadata, where present, is referenced only as a `interface_id` column on `ip_addresses`. Adding first-class interface management would require a new table and is deferred until a use-case demands it.
- The original spec listed `GET /prefixes/{id}/next-available`; the implementation also accepts the legacy name `GET /prefixes/{id}/next-ip` so older clients keep working. Both routes return the same body.
- `reserve` / `assign` / `release` are thin wrappers around `UpdateIPAddressStatus`. They accept no body. They are idempotent — calling `release` on an already-available address is a no-op.

## Error envelope

All non-2xx API responses use the common envelope:

```json
{
  "error": {
    "code": "validation_error",
    "message": "human-readable message",
    "request_id": "...",
    "details": {}
  }
}
```

Error codes used:

| HTTP | `code` |
| --- | --- |
| 400 | `validation_error` |
| 401 | `unauthorized` |
| 403 | `forbidden` |
| 404 | `not_found` |
| 409 | `conflict` |
| 429 | `rate_limited` |
| 500 | `internal_error` |
| 503 | `service_unavailable` |

500 responses never leak raw database error strings; the message is always a generic `"internal error"` and the underlying error is logged server-side.

## Pagination on `/monitors/{id}/results`

`GET /api/v1/monitors/{id}/results` accepts `limit` (default `100`, max `1000`) and `offset` (default `0`) as query parameters. Results are returned ordered by `checked_at` descending.

## Push monitor heartbeat semantics

Push monitors use a dedicated `last_heartbeat_at` column on `monitors` (added in migration `007_push_heartbeat.sql`).

- `POST /push/{token}` (or `GET /push/{token}`) updates `last_heartbeat_at` to "now". An optional `status=up|down|degraded` query param or JSON body field overrides the default `up`.
- The scheduled overdue check compares `now() - last_heartbeat_at` against `grace_seconds`. It does **not** read `last_checked_at`, so the scheduler running an overdue check never makes a missing-heartbeat monitor look healthy.
- Pushed `down`/`degraded` and scheduled overdue both flow through the same `IncidentHook`, so they create incidents identically. Pushed `up` after a `down` resolves the open incident and triggers the same notification path as a scheduled recovery.
