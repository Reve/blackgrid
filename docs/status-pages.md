# Status Pages

Status pages let an operator group selected monitors into a single internal or
public-facing health overview. They surface aggregate status, per-service uptime,
and recent incidents, while strictly hiding any private monitor, IPAM, or
notification configuration.

## Creating a status page

Status pages are managed in the admin UI under **Status Pages**, or via the
admin API:

```http
POST /api/v1/status-pages
```

A page has:

- `name` — human-friendly title (required)
- `slug` — URL slug, lowercase letters/digits/hyphens only. Auto-generated from
  the name if not provided.
- `description` — optional short description.
- `public` — whether the page is reachable at `/status/{slug}`.
- `show_uptime` — whether per-monitor uptime is included on the public view.
- `show_incidents` — whether recent incidents are included on the public view.

Once created, attach monitors via:

```http
POST /api/v1/status-pages/{id}/monitors
{ "monitor_id": "...", "display_name": "PostgreSQL", "display_order": 10 }
```

A monitor can only be attached to a given status page once. If `display_order`
is omitted, the next slot (max + 10) is used. Reorder with:

```http
POST /api/v1/status-pages/{id}/monitors/reorder
{ "monitor_ids": ["id1", "id2", "id3"] }
```

This sets `display_order` to 10, 20, 30, … in the supplied order. Any monitor
ID that is not currently attached causes the entire request to be rejected.

## Public vs private behaviour

- A status page with `public=false` is invisible at the public route. Requests
  to `/status/{slug}` return `404` (not `403`) so existence is not leaked.
- A status page with `public=true` is reachable to anyone who knows the slug —
  there is no authentication on the public endpoint.

## Safe data exposure

The public response intentionally excludes:

- monitor config JSON, target details (when potentially sensitive), and check
  internals
- IP address records, device records, and prefix metadata
- notification channel data, secrets, and delivery records
- raw check result rows, internal notes, debug output

It includes only:

- page name, slug, description
- aggregate status
- per-monitor display name, type, current status, last check time, and (if
  `show_uptime=true`) 24h / 30d uptime percentages
- if `show_incidents=true`, recent incidents (last 30 days, plus any still
  open) with display name, severity, status, started/resolved timestamps, and
  the human-readable summary

## Aggregate status

Computed from the current statuses of attached monitors:

- `down` — at least one attached monitor is `down`
- `degraded` — no monitor is `down`, but at least one is `degraded` or
  `unknown` (or paused/other-non-up)
- `up` — every attached monitor is `up`
- `empty` — no monitors are attached

## Uptime calculation

Per-monitor uptime over a window (24h, 30d) is computed from `monitor_results`:

```
uptime = up_count / total_count * 100
```

where:

- `up_count` is the number of results in the window with `status='up'`
- `total_count` is the total number of results in the window

If the window has no results at all, uptime is reported as `null` (the page
shows `—`) rather than implying 100% from missing data. `degraded` results
currently count as "not fully up" — they do not contribute to `up_count`.

## Phase scope

Phase 5 deliberately does **not** include:

- authentication or per-user access control
- realtime updates over SSE/WebSocket (the public view polls every 30s)
- Redis / multi-process coordination
- per-region status regions or scheduled maintenance windows
