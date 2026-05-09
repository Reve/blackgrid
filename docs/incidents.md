# Incidents

Blackgrid turns scheduled monitor failures into incidents that operators can
acknowledge and resolve. The lifecycle is intentionally simple and
status-driven.

## Lifecycle

```
[no incident]
     |
     | scheduled monitor transitions to `down` or `degraded`
     v
   open  ---- ack ---->  acknowledged
     \                       /
      \                     /
       monitor recovers (`up`) — both transition to:
                  v
              resolved
```

States:

- `open` — the monitor is currently failing (down/degraded) and nobody has
  acknowledged the page yet.
- `acknowledged` — a human has acknowledged the alert. The incident is still
  active; resolution is pending.
- `resolved` — the monitor recovered (or the operator manually resolved it).

## Severity

| Monitor status | Incident severity |
| -------------- | ----------------- |
| `down`         | `critical`        |
| `degraded`     | `warning`         |

`info` severity exists in the schema for future events but is not currently
emitted automatically.

## Creation rules

The monitor scheduler invokes the incident service after every scheduled
status transition. The rules are:

1. **Down transition** (`up` / `unknown` / `degraded` → `down`): open a
   `critical` incident if no `open` or `acknowledged` incident exists for that
   monitor.
2. **Degraded transition** (`up` / `unknown` / `down` → `degraded`): open a
   `warning` incident if no `open` or `acknowledged` incident exists.
3. **Recovery** (* → `up`): resolve the active (open or acknowledged)
   incident, if any.
4. **Manual `Test now`**: stores a `monitor_results` row but never opens or
   resolves incidents. Manual tests must not affect the incident table.
5. **Duplicate suppression**: while an `open` or `acknowledged` incident
   exists for a monitor, additional failures do not create new incidents.
6. **Re-occurrence**: once an incident is `resolved` and the monitor goes
   down again, a brand-new incident is opened.

## Acknowledgement

`POST /api/v1/incidents/{id}/acknowledge`

- `open` → `acknowledged`, sets `acknowledged_at`.
- `acknowledged` → no-op, returns the current incident.
- `resolved` → `409 Conflict`. Acknowledging a resolved incident is not
  meaningful and is rejected so the user notices the stale UI state.

Acknowledgement does **not** resolve. Recovery is what resolves.

## Resolution

`POST /api/v1/incidents/{id}/resolve`

- `open` / `acknowledged` → `resolved`, sets `resolved_at`.
- `resolved` → no-op, returns the current incident (idempotent).

Resolution can also happen automatically when the monitor recovers — the
scheduler invokes `IncidentService.ResolveForMonitor` which is idempotent.

## Notifications

When an incident is opened or resolved, the incident service hands off to the
notification service which delivers events to all enabled notification
channels. See [notifications.md](./notifications.md).

## Implementation pointers

- Schema: `backend/migrations/003_incidents_notifications.sql`
- Service: `backend/internal/service/incident.go`
- Bridge to scheduler: `backend/internal/service/incident_hook.go`
- HTTP handlers: `backend/internal/api/handlers/incident.go`
- Scheduler hook surface: `backend/internal/monitor/scheduler.go`
  (`SetIncidentHook`, `OnScheduledStatusChange`).
