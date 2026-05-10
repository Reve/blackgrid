# Blackgrid Deployment Guide

Blackgrid is designed to be deployed using Docker and Docker Compose. This guide covers the production deployment process, environment variables, and database configuration.

## Prerequisites

- Docker 20.10+
- Docker Compose 2.0+
- A PostgreSQL database (if not using the bundled one)

## Quick Start (Production)

1. **Clone the repository**:
   ```bash
   git clone https://github.com/your-org/blackgrid.git
   cd blackgrid
   ```

2. **Configure Environment Variables**:
   Create a `.env` file in the root directory (see [Environment Variables](#environment-variables) for details).

3. **Start the stack**:
   ```bash
   docker-compose -f docker-compose.yml up -d
   ```

## Environment Variables

### Runtime
- `GO_VERSION`: builds and Docker images use Go 1.25.7 (`golang:1.25-alpine`).
  `go.mod` declares `go 1.25.7`; newer 1.25.x patches and 1.26.x are
  forward-compatible. Anything older fails the build.

### Database
- `DATABASE_URL`: Connection string (e.g., `postgres://user:pass@db:5432/blackgrid?sslmode=disable`)
- `DB_MAX_OPEN_CONNS`: Maximum open connections in the pool (default: 25; mapped to `pgxpool.Config.MaxConns`).
- `DB_CONN_MAX_LIFETIME_MINUTES`: Max connection lifetime (default: 30; `MaxConnLifetime`).
- `DB_CONN_MAX_IDLE_TIME_MINUTES`: How long an idle connection survives before being closed (default: 10; `MaxConnIdleTime`).

> `DB_MAX_IDLE_CONNS` is **not** read by Blackgrid. `pgxpool` does not expose
> a max-idle-connections setting separate from `MaxConns`; idle handling is
> driven entirely by `MaxConnIdleTime`. Earlier docs listed it by mistake —
> remove it from your `.env` if present.

### Authentication & Sessions
- `AUTH_COOKIE_SECURE`: When `true`, the `blackgrid_session` cookie is
  flagged `Secure` (HTTPS only). Default: `false` for local development;
  set to `true` in any production deployment behind TLS.
- `AUTH_SESSION_TTL_HOURS`: Session lifetime in hours (default: 24).
  Resolved sessions older than this are rejected and require re-login.

### Security & Networking
- `CORS_ALLOWED_ORIGINS`: Comma-separated list of allowed origins for API access.
- `CORS_ALLOW_CREDENTIALS`: When `true` (default), the CORS middleware
  emits `Access-Control-Allow-Credentials: true` so the session cookie is
  accepted from configured origins. Set `false` only when serving a
  cookie-less, token-only frontend.
- `PORT`: Backend port (default: 8080)
- `SHUTDOWN_TIMEOUT_SECONDS`: Time to wait for graceful shutdown (default: 20)

### Monitor scheduler
- `MONITOR_WORKERS`: Number of concurrent monitor execution workers
  (default: 10). Increase for many fast monitors; values above ~50 mostly
  trade memory for diminishing returns.

### Data Retention
- `RETENTION_MONITOR_RESULTS_DAYS`: Days to keep monitor results (default: 90)
- `RETENTION_NOTIFICATION_DELIVERIES_DAYS`: Days to keep notification records (default: 30)
- `RETENTION_AUDIT_LOG_DAYS`: Days to keep audit logs (default: 365)
- `RETENTION_DISCOVERY_RESULTS_DAYS`: Days to keep discovery data (default: 90)
- `RETENTION_DISCOVERY_SCANS_DAYS`: Days to keep discovery scan rows (default: 90)
- `RETENTION_CLEANUP_INTERVAL_HOURS`: How often to run the cleanup job (default: 24)

### Discovery
- `DISCOVERY_WORKERS`: Parallel workers for scanning (default: 64)
- `DISCOVERY_TCP_TIMEOUT_MS`: Timeout for TCP port probes (default: 750)
- `DISCOVERY_PING_TIMEOUT_MS`: Timeout for ICMP pings (default: 750)
- `DISCOVERY_MAX_IPV4_PREFIX_SIZE`: Largest IPv4 prefix accepted for a
  scan, expressed as the prefix length. Default `22` — `/22` (1024
  addresses) is the largest allowed; smaller numeric values mean wider
  prefixes and are rejected with HTTP 422.
- `DISCOVERY_DEFAULT_PORTS`: Comma-separated TCP probe port list. Default
  `22,53,80,443,5432,6379,8000,8080,9000,9443`. Invalid entries are dropped
  and a warning is logged at startup. If parsing yields no valid ports the
  built-in defaults are used.
- `DISCOVERY_ENABLE_PING`: `true|false` (default `false`). When enabled and
  the runtime supports raw sockets (host networking or `CAP_NET_RAW`),
  Blackgrid may use ICMP to corroborate TCP probe results. The `/api/v1/discovery/diagnostics` endpoint reports whether ping is active.

### LAN discovery & Docker networking

Discovery uses TCP connect probes. A bridge-mode backend container can reach
LAN addresses only through the host's routing table; firewalled hosts with no
TCP ports in `DISCOVERY_DEFAULT_PORTS` will not appear.

For best LAN discovery on Linux, host networking sidesteps NAT:

```yaml
backend:
  network_mode: host
```

This is Linux-specific and changes how ports are exposed. ICMP/ARP discovery
typically requires host networking, `CAP_NET_RAW`, or a host-side agent. ARP
discovery cannot work reliably from a normal bridge container for arbitrary
LAN subnets. On Docker Desktop/macOS, containers run inside a VM, so LAN
visibility may differ from a Linux host.

## Resource Requirements

Blackgrid is lightweight. For a homelab environment (up to 1000 IPs and 100 monitors):
- **CPU**: 1 vCPU
- **RAM**: 512MB (Backend) + 128MB (PostgreSQL)
- **Storage**: 10GB+ (SSD recommended for PostgreSQL performance with high-frequency monitors)

## Build & test verification

A clean checkout must satisfy:

```bash
cd backend && go build ./... && go test ./...
cd frontend && npm ci && npm run build
docker compose up --build -d   # then walk docs/smoke-test.md
```

The integration test suite skips automatically if Postgres is unreachable.
To exercise it against a throwaway database:

```bash
docker compose up -d db
BLACKGRID_TEST_DATABASE_URL=postgres://blackgrid:blackgrid@localhost:5432/blackgrid_test?sslmode=disable \
  go test -p 1 ./...
```

`-p 1` keeps integration tests serial because they share the schema.

## High Availability

Blackgrid currently supports a single active backend instance. Multiple instances can point to the same database, but background jobs (scheduler, discovery) will run on all instances unless otherwise managed. HA support for jobs is planned for a future release.
