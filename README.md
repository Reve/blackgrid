# Blackgrid

Blackgrid is a homelab IPAM (IP Address Management) and endpoint monitoring application.

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
