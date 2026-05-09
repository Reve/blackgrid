# Authentication and Security

Blackgrid implements a robust security framework to protect your homelab data and monitoring configuration.

## Authentication

### User Sessions
Standard session-based authentication for the web interface.
- **Login**: `POST /auth/login`
- **Logout**: `POST /auth/logout`
- **Me**: `GET /auth/me`

### API Tokens
Bearer tokens for automated access and external integrations.
- Tokens can be assigned `admin`, `operator`, or `viewer` roles.
- Plaintext tokens are shown only once upon creation.
- Tokens can have an optional expiration date.

## Roles and RBAC (Role-Based Access Control)

- **Admin**: Full access to the system, including user management and audit logs.
- **Operator**: Can manage monitors, status pages, and IPAM, but cannot manage users or view sensitive audit logs.
- **Viewer**: Read-only access to all dashboards and IPAM data. Cannot view secrets.

## Secret Masking

Sensitive fields are automatically masked in the API and UI to prevent accidental exposure.
- **DSN Passwords**: Masked in PostgreSQL monitor configurations.
- **SMTP/Webhook Secrets**: Masked in notification channel configurations.
- **Push Tokens**: Hashed in the database; plaintext shown only once during creation or rotation.

The masking logic uses pattern matching (e.g., `pass`, `secret`, `token`, `key`, `dsn`) to identify and replace sensitive values with `********`.

## Audit Logging

All destructive and sensitive actions are recorded in an immutable audit log.
- **Logged Actions**: Create/Update/Delete of monitors, users, status pages, tokens, and notification channels.
- **Context**: Records the actor (user or API token), timestamps, and before/after states.
- **Storage**: Stored in the `audit_logs` table.

## Push Heartbeats (Public Security)

Push heartbeat endpoints (`/push/{token}`) do not require a user session. They rely on a high-entropy UUID-based token. This allows simple `curl` or `wget` commands to signal service health from isolated environments without exposing full administrative access.
