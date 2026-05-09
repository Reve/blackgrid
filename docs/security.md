# Blackgrid Security Documentation

Blackgrid is built with security as a core principle. This document outlines the security features and configuration recommendations.

## Security Features

### Authentication & Authorization
- **JWT + Session Cookies**: Supports both for flexible browser and API access.
- **RBAC**: Three roles (Admin, Operator, Viewer) ensure the principle of least privilege.
- **Audit Logging**: All mutations and sensitive reads are logged with actor details and request IDs.

### Network Security
- **Security Headers**: All API responses include `X-Frame-Options: DENY`, `X-Content-Type-Options: nosniff`, and basic HSTS configurations.
- **CORS**: Origin validation is strictly enforced via the `CORS_ALLOWED_ORIGINS` setting.
- **Rate Limiting**:
    - Global IP-based limits on sensitive endpoints (Login, Setup).
    - Per-user/token limits on expensive operations (Manual Tests, API Token creation).
    - High-frequency limits on passive endpoints (Push Heartbeats).

### Data Protection
- **Password Hashing**: Bcrypt with adaptive cost is used for user passwords.
- **Secret Masking**: SMTP and Webhook secrets (passwords, tokens, authorization headers) are masked in API responses.
- **Request IDs**: Every request is assigned a unique UUID, propagated through logs and headers for traceability.

## Security Recommendations

### 1. Enable HTTPS
Blackgrid does not terminate TLS natively. It is **highly recommended** to run Blackgrid behind a reverse proxy (Nginx, Traefik, Caddy) that handles SSL/TLS termination.

### 2. Restrict CORS
Set `CORS_ALLOWED_ORIGINS` to only the domains where your frontend is hosted. Never use `*` in production.

### 3. Database Security
- Run PostgreSQL in a private network (default in Docker Compose).
- Use a strong, unique password for the `DATABASE_URL`.
- Regularly review `audit_log` entries for suspicious activity.

### 4. API Token Management
- Issue API tokens with the minimum required role.
- Use expiration dates for non-persistent integrations.
- Regularly rotate tokens used in CI/CD or scripts.

## Incident Reporting
If you discover a security vulnerability, please report it via the project's private vulnerability reporting process (if available) or via email to the maintainers.
