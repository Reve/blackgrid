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

### Database
- `DATABASE_URL`: Connection string (e.g., `postgres://user:pass@db:5432/blackgrid?sslmode=disable`)
- `DB_MAX_OPEN_CONNS`: Maximum open connections in the pool (default: 25)
- `DB_MAX_IDLE_CONNS`: Maximum idle connections (default: 10)
- `DB_CONN_MAX_LIFETIME_MINUTES`: Max connection lifetime (default: 30)

### Security & Networking
- `CORS_ALLOWED_ORIGINS`: Comma-separated list of allowed origins for API access.
- `PORT`: Backend port (default: 8080)
- `SHUTDOWN_TIMEOUT_SECONDS`: Time to wait for graceful shutdown (default: 20)

### Data Retention
- `RETENTION_MONITOR_RESULTS_DAYS`: Days to keep monitor results (default: 90)
- `RETENTION_NOTIFICATION_DELIVERIES_DAYS`: Days to keep notification records (default: 30)
- `RETENTION_AUDIT_LOG_DAYS`: Days to keep audit logs (default: 365)
- `RETENTION_DISCOVERY_RESULTS_DAYS`: Days to keep discovery data (default: 90)
- `RETENTION_CLEANUP_INTERVAL_HOURS`: How often to run the cleanup job (default: 24)

### Discovery
- `DISCOVERY_WORKERS`: Parallel workers for scanning (default: 64)
- `DISCOVERY_TCP_TIMEOUT_MS`: Timeout for TCP port probes (default: 750)
- `DISCOVERY_PING_TIMEOUT_MS`: Timeout for ICMP pings (default: 750)

## Resource Requirements

Blackgrid is lightweight. For a homelab environment (up to 1000 IPs and 100 monitors):
- **CPU**: 1 vCPU
- **RAM**: 512MB (Backend) + 128MB (PostgreSQL)
- **Storage**: 10GB+ (SSD recommended for PostgreSQL performance with high-frequency monitors)

## High Availability

Blackgrid currently supports a single active backend instance. Multiple instances can point to the same database, but background jobs (scheduler, discovery) will run on all instances unless otherwise managed. HA support for jobs is planned for a future release.
