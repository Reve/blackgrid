# Monitoring

Blackgrid supports several monitor types to ensure your services are running correctly.

## Supported Monitor Types

### HTTP
Standard HTTP(S) check. Verifies the status code and optionally the response body.

### TCP
Verifies that a TCP connection can be established on a specific port.

### Ping (ICMP)
Sends ICMP Echo requests to verify network connectivity.

### DNS
Verifies that a DNS query succeeds and optionally returns expected values.
- **Resolver**: Optional DNS server to use (e.g., `1.1.1.1:53`). Defaults to system resolver.
- **Record Type**: A, AAAA, CNAME, MX, TXT, NS, PTR.
- **Match Mode**: `any` (at least one expected value), `all` (all expected values), `exact` (exact set match).

### TLS Certificate
Checks TLS connectivity and certificate expiration.
- **Warning Days**: Triggers a 'degraded' status (default: 30 days).
- **Critical Days**: Triggers a 'down' status (default: 7 days).
- **Verify TLS**: Whether to verify the certificate chain. Even if false, expiry is still checked.

### Push Heartbeat (Passive)
Allows external jobs or scripts to signal success periodically via a secret URL.
- **Grace Period**: How long to wait before marking the monitor as 'down' if no heartbeat is received.
- **Endpoint**: `GET /push/{token}` or `POST /push/{token}`.

### PostgreSQL
Verifies connection and query execution on a PostgreSQL database.
- **DSN**: The connection string (masked in UI/API).
- **Query**: Optional SQL query to run (default: `SELECT 1`).

## Secret Masking
Blackgrid automatically masks sensitive information (passwords, tokens, DSNs) in all API responses, audit logs, and frontend forms. Secrets are never exposed to users with 'viewer' roles or on public status pages.

## Incident Lifecycle
When a monitor fails its retry threshold, an incident is opened. Incidents are automatically resolved when the monitor returns to an 'up' status. Notifications are sent via configured channels (SMTP, Webhook).
