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
