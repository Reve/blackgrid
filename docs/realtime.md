# Blackgrid Realtime (Server-Sent Events)

Blackgrid pushes domain events to authenticated clients over a single SSE
endpoint. The dashboard, monitor list, incident feed, and IPAM views all
consume the same stream.

## Endpoint

```
GET /api/v1/events/stream
```

The endpoint is served under the same `/api/v1` group as the rest of the
API and therefore requires the same authentication: a session cookie
(`blackgrid_session`) **or** a `Authorization: Bearer <api-token>` header.
Anonymous clients receive `401 unauthorized` with the standard error envelope.

## Query parameters

Both filters are optional. When both are supplied, an event must match
*every* filter to be delivered to that subscriber.

| Param | Format | Effect |
| --- | --- | --- |
| `types` | comma-separated event types | Only these `event.type` values are delivered. |
| `object_types` | comma-separated object types | Only events whose `object_type` is in this list. |

Example:

```
GET /api/v1/events/stream?types=monitor.status_changed,incident.opened&object_types=monitor,incident
```

## Frame format

Each frame follows the SSE wire format:

```
id: <event-uuid>
event: <event-type>
data: <json-encoded Event>

```

* `id` is the server-assigned UUID for this event. Browsers' `EventSource`
  preserves it as `MessageEvent.lastEventId`. **Note**: the server does not
  honour `Last-Event-ID` on reconnect — see "Limitations" below.
* `event` is the `Event.type` (e.g. `monitor.status_changed`). The browser
  uses this to dispatch `addEventListener("monitor.status_changed", …)`
  callbacks. Frames also fire the generic `message` event for code that
  uses a single `onmessage` handler.
* `data` is a JSON object with the fields described below.

## Event payload

```json
{
  "id": "uuid",
  "type": "monitor.status_changed",
  "created_at": "2025-05-10T18:30:00Z",
  "object_type": "monitor",
  "object_id": "uuid",
  "actor_type": "system",
  "actor_id": "",
  "payload": { "...": "type-specific" }
}
```

`created_at` is server time at publish, populated automatically by
`EventBus.Publish` if the caller didn't set it. `actor_type` is one of
`user`, `api_token`, or `system`.

## Event types

The full list lives in [`backend/internal/events/events.go`](../backend/internal/events/events.go).
Categories:

* **monitor** — `monitor.created`, `monitor.updated`, `monitor.deleted`,
  `monitor.paused`, `monitor.resumed`, `monitor.tested`,
  `monitor.result_created`, `monitor.status_changed`.
* **incident** — `incident.opened`, `incident.acknowledged`, `incident.resolved`.
* **notification** — `notification.delivery_sent`, `notification.delivery_failed`.
* **discovery** — `discovery.scan_started/completed/failed`,
  `discovery.result_created/accepted/ignored`, `discovery.new_host`,
  `discovery.conflict_detected`, `discovery.stale_detected`.
* **ipam** — `ipam.site_changed`, `ipam.vlan_changed`, `ipam.prefix_changed`,
  `ipam.ip_address_changed`, `ipam.device_changed`.
* **status_page** — `status_page.changed`, `status_page.monitor_changed`.
* **audit** — `audit.entry_created`.
* **auth** — `user.changed`, `api_token.changed`.

## Keepalives

The server sends a `: keepalive` comment every 20 seconds when there is no
real traffic. This keeps proxies and browsers from idling out the connection.
Comments are not delivered to `EventSource` listeners.

## Secret safety

Payloads pass through the same masking helpers as the REST API:

* Monitor `config` is run through `monitor.MaskConfig` before any payload is
  built — passwords, tokens, API keys, and Postgres DSNs are replaced with
  `"***"`.
* Notification channel config is filtered by `service.MaskConfig` (SMTP
  passwords, webhook authorization headers).
* Audit events never include raw before/after for password fields.

If you add a new event with type-specific config, mask it before publishing.

## Limitations

* **No durable replay.** The bus is in-memory only. If a client disconnects
  and reconnects, anything published in the gap is lost. `Last-Event-ID` is
  ignored. Clients that need a consistent view should fetch from REST after
  reconnecting and resume from the live stream.
* **No backpressure or per-subscriber buffering policy beyond drop.** Each
  subscriber has a buffered channel of 100 events; if it fills (typically
  because the client is slow), additional events are dropped silently for
  that subscriber. Other subscribers are unaffected.
* **Single process.** There is no fan-out across multiple Blackgrid
  instances; horizontal scaling would require an external pubsub.
* **Events are not persisted.** They are not stored in Postgres. The audit
  log captures intent but is a separate write path.

## Graceful shutdown

`EventBus.Shutdown()` is called during the server's SIGTERM/SIGINT handler
after Echo's HTTP server stops accepting new requests. It closes every
subscriber channel so the SSE handler returns cleanly instead of leaking
goroutines or leaving the client hung on a dead socket.
