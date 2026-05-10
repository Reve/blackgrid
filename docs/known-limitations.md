# Known Limitations

This document records the deliberate scope boundaries of the Blackgrid MVP.
None of these are bugs — they are decisions made to keep the first usable
release small, predictable, and operable as a homelab tool. Each entry notes
why the limitation exists and the workaround (if any).

## Device interface endpoints are not implemented

The MVP exposes device CRUD but **does not** include endpoints for managing
device interfaces (link state, port lists, MAC tables, LLDP neighbors).
The `devices` table holds only the basic identity fields needed to anchor
IP and monitor records.

Workaround: model interfaces via separate IP address records or external
systems for now. Interface modeling is a candidate for a post-MVP milestone.

## IPAM schema is simplified vs. NetBox

Blackgrid IPAM intentionally tracks only:

- prefixes (with optional discovery configuration)
- IP addresses (status, description, last_seen)
- VLANs (id, vid, name)
- sites
- devices (basic identity)

It does **not** model: VRFs, route targets, RIRs/ASNs, IP roles, services
attached to IPs, contacts, tenancy/groups, or tagged objects. Imports from
NetBox are out of scope.

Workaround: keep external authoritative IPAM where richer modelling is
required; use Blackgrid for monitoring + lightweight prefix tracking.

## In-process SSE has no durable replay

The realtime event bus (`/api/v1/events/stream`) is **in-process only**.
Events are not persisted; if a client disconnects, any events emitted while
it was offline are lost. There is no `Last-Event-ID` replay, no Redis
fan-out, and no multi-replica delivery guarantee.

Workaround: rely on REST polling for definitive state after a reconnect.
For multi-instance deployments, terminate SSE on a single backend or add an
external pub/sub layer (out of MVP scope).

## In-process rate limiting resets on restart

Login, setup, push, API-token-creation, monitor-test and notification-test
limits are enforced via in-memory token buckets. **Restarting the backend
clears all rate-limit state.**

Workaround: deploy behind a stable process; do not rely on rate limits as
a security control across restarts. A persistent / distributed limiter is
out of scope for MVP.

## Discovery is bounded and does not perform full IPv6 scanning

Discovery is intentionally constrained:

- IPv4 prefixes larger than `DISCOVERY_MAX_IPV4_PREFIX_SIZE` (default `/22`)
  are rejected for manual scan.
- IPv6 full-prefix scanning is unsupported (`ErrIPv6Unsupported`).
- Probing is TCP connect on a fixed port list; ICMP is not required.

Workaround: split very large prefixes into smaller scoped prefixes; track
IPv6 hosts manually or via separate discovery sources.

## MAC discovery is limited

Discovery records the IPs that respond to TCP probes. It does **not**
read the local ARP table or query upstream switches/routers, so MAC
addresses are typically unknown unless populated through another path.

Workaround: future work may add an opt-in safe-ARP source on the host
running Blackgrid; until then, populate MACs manually if needed.

## Notification secrets are masked, not encrypted

Notification channel configurations (SMTP password, webhook auth headers)
are stored as JSON in PostgreSQL. The API masks secrets on read and never
returns them to clients, but at rest they are **not** encrypted. Anyone
with database access can read them.

Workaround: protect database access with the usual controls (network
isolation, role-based DB users, encrypted backups). A pgcrypto- or
KMS-backed secret column is a candidate for a future release.
