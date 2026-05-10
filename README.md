# Blackgrid

Blackgrid is a powerful, production-hardened homelab IPAM (IP Address Management) and endpoint monitoring application. It combines traditional IPAM features with a modern, real-time monitoring suite and an event-driven notification engine.

## Key Features

- **Robust IPAM**: Manage sites, VLANs, prefixes, and IP addresses with automatic discovery and reconciliation.
- **Advanced Monitoring**: Proactive checks for HTTP, TCP, Ping, DNS, TLS Certificates, and custom Push heartbeats.
- **Incident Management**: Automated incident lifecycle with severity levels, acknowledgement, and resolution workflows.
- **Event-Driven Architecture**: Real-time dashboard updates via SSE and a flexible notification engine (Webhooks, SMTP).
- **Security First**: RBAC, audit logging, rate limiting, and secure header management.
- **Production Ready**: Prometheus metrics, health/readiness probes, data retention policies, and graceful shutdown.

## Quick Start

### Docker Compose (Recommended)

Run the entire stack using Docker Compose:

```bash
docker-compose up --build -d
```

Access the application at:
- **Frontend**: [http://localhost:3000](http://localhost:3000)
- **Backend API**: [http://localhost:8080](http://localhost:8080)
- **Metrics**: `http://localhost:8080/metrics`

### First Time Setup

1. Navigate to `http://localhost:3000/setup`.
2. Create your initial Administrator account.
3. Configure your first Site and Prefix to begin scanning your network.

## Documentation

Detailed documentation is available in the `docs` directory:

- **[Deployment Guide](docs/deployment.md)**: Environment variables, Docker config, and scaling.
- **[Operations Guide](docs/operations.md)**: Monitoring, backups, and maintenance.
- **[Security Overview](docs/security.md)**: Auth, encryption, and protection mechanisms.
- **[IPAM & Discovery](docs/discovery.md)**: How network scanning works.
- **[Monitoring & Incidents](docs/incidents.md)**: Monitor types and incident lifecycle.
- **[Notification Engine](docs/notifications.md)**: Webhooks, SMTP, and event payloads.
- **[Status Pages](docs/status-pages.md)**: Public health pages and uptime calculation.
- **[API Reference](docs/api.md)**: Full REST API documentation.

## Development

### Backend (Go)
```bash
cd backend
go run cmd/server/main.go
```

### Frontend (React + Vite)
```bash
cd frontend
npm install
npm run dev
```

## Contributing

Contributions are welcome! Please see the contribution guidelines (coming soon) for more information.

## License

MIT

## Build prerequisites

- **Backend**: Go 1.25.7 or newer. The Docker image pins `golang:1.25-alpine`; `go.mod` declares `go 1.25.7` as the minimum toolchain that compiles the current `pgx/v5` and `goose` releases. Newer 1.25.x patches and 1.26.x work unchanged. See [docs/deployment.md](docs/deployment.md) for the supported runtime matrix.
- **Frontend**: Node 20 LTS or newer, npm 10+. The repository ships with `package-lock.json`; see [frontend/README.md](frontend/README.md) for the supported install command.

A clean checkout builds with:

```bash
# backend
cd backend && go build ./... && go test ./...

# frontend
cd frontend && npm ci && npm run build
```

The backend integration tests skip cleanly if Postgres is unreachable. To
run them against a throwaway DB:

```bash
docker compose up -d db
BLACKGRID_TEST_DATABASE_URL=postgres://blackgrid:blackgrid@localhost:5432/blackgrid_test?sslmode=disable \
  go test -p 1 ./...
```

`-p 1` disables parallel package execution so the integration tests share
the schema without stomping on each other. CI should run the same two
commands; no other setup is required.

For the manual end-to-end check after a deploy, follow [docs/smoke-test.md](docs/smoke-test.md).

## API conformance notes

The HTTP API uses a single error envelope and a documented IPAM surface. See [docs/api.md](docs/api.md) for the full reference and the list of intentional deviations from the original phased plan (notably: device interfaces are not modelled as standalone resources; `next-ip` is exposed alongside `next-available` as a compatibility alias).

Push monitors use a dedicated `last_heartbeat_at` column. The scheduler's overdue check compares against that field, not `last_checked_at`, so a scheduled evaluation never masks a missing heartbeat. Push status changes flow through the same incident lifecycle hook as scheduled status changes.
