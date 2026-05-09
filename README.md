# Blackgrid

Blackgrid is a homelab IPAM (IP Address Management) and endpoint monitoring
application. Phase 3 adds **incident management** and **notification
channels** (webhook, SMTP) so monitor failures turn into actionable alerts —
see [docs/incidents.md](docs/incidents.md) and
[docs/notifications.md](docs/notifications.md).

## Prerequisites

- Docker
- Docker Compose

## Startup

Run the application using Docker Compose:

```bash
docker-compose up --build
```

The services will be accessible at:
- **Frontend:** http://localhost:3000
- **Backend API:** http://localhost:8080
- **PostgreSQL Database:** localhost:5432 (default credentials: `blackgrid` / `blackgrid`)

## Development

### Backend (Go)

```bash
cd backend
go run cmd/server/main.go
```

To run tests:
```bash
cd backend
go test ./...
```

### Frontend (React + Vite)

```bash
cd frontend
npm install
npm run dev
```

## Discovery

Blackgrid scans **stored prefixes only** — arbitrary CIDR values are never
accepted from the API. See [docs/discovery.md](docs/discovery.md) for full
details and [docs/api.md](docs/api.md) for the endpoint reference.

Default TCP probe ports: `22, 53, 80, 443, 5432, 6379, 8000, 8080, 9000, 9443`.

### Environment variables

| Variable | Default | Description |
| --- | --- | --- |
| `DISCOVERY_WORKERS` | `64` | Worker pool size for parallel probing |
| `DISCOVERY_MAX_IPV4_PREFIX_SIZE` | `22` | Reject manual scans of IPv4 prefixes shorter than this (e.g. `/16` is rejected) |
| `DISCOVERY_TCP_TIMEOUT_MS` | `750` | Per-port TCP connect timeout |
| `DISCOVERY_PING_TIMEOUT_MS` | `750` | ICMP timeout (only used when ping is available) |

IPv6 full-range scanning is unsupported. Inside Docker, ICMP and ARP/MAC
discovery require additional capabilities; Blackgrid falls back to TCP-only
discovery without them.

## Incidents and notifications

When a scheduled monitor transitions to `down` or `degraded`, Blackgrid opens
an incident automatically and (if any notification channels are enabled)
delivers an `incident.opened` event. Recovery (`up`) auto-resolves the
incident and emits `incident.resolved`.

- API reference: [docs/api.md](docs/api.md) (Incidents and Notification
  channel sections).
- Lifecycle rules: [docs/incidents.md](docs/incidents.md).
- Channel config and event payloads: [docs/notifications.md](docs/notifications.md).

Notification secrets (SMTP passwords, webhook bearer tokens) are stored as
JSONB in PostgreSQL **without an additional encryption layer** in this phase
— protect the database accordingly.
