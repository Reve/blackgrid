# Discovery

Subnet discovery scans known IPAM prefixes, classifies the hosts that respond,
and reconciles findings back into IPAM.

## Safety rules

- **Stored prefixes only.** Manual scans take a `prefix_id`. Arbitrary CIDR
  values are never accepted from the API.
- **IPv4 only.** Full-range IPv6 scans are explicitly unsupported. The API
  rejects IPv6 prefixes with a `IPv6 full scanning is not supported` error.
- **Bounded prefix size.** IPv4 prefixes shorter than `/22` (more than ~1024
  hosts) are rejected. The limit is configurable via
  `DISCOVERY_MAX_IPV4_PREFIX_SIZE`.
- **Bounded concurrency.** A worker pool sized by `DISCOVERY_WORKERS`
  (default 64) probes hosts in parallel.
- **Skip network/broadcast.** For prefixes shorter than `/31` the network and
  broadcast addresses are excluded from the host list.
- **Failed scans never crash the API.** Errors are persisted in
  `discovery_scans.error`.

## Configuration

| Variable | Default | Description |
| --- | --- | --- |
| `DISCOVERY_WORKERS` | `64` | Worker pool size for probing |
| `DISCOVERY_MAX_IPV4_PREFIX_SIZE` | `22` | Reject manual scans of IPv4 prefixes shorter than this |
| `DISCOVERY_TCP_TIMEOUT_MS` | `750` | Per-port TCP connect timeout |
| `DISCOVERY_PING_TIMEOUT_MS` | `750` | ICMP ping timeout (when ping is enabled) |
| `DISCOVERY_DEFAULT_PORTS` | `22,53,80,443,5432,6379,8000,8080,9000,9443` | Comma-separated TCP probe ports. Invalid values are dropped; if no valid ports remain Blackgrid logs a warning and falls back to the default list. |
| `DISCOVERY_ENABLE_PING` | `false` | Enable ICMP-assisted discovery. Requires `CAP_NET_RAW` (Linux) or host networking. |

## Probe methods

1. **TCP connect** to a curated list of common service ports:
   `22, 53, 80, 443, 5432, 6379, 8000, 8080, 9000, 9443`.
   Any successful connect marks the host as seen and contributes its port to
   `open_ports`. Latency is the fastest successful probe time.
2. **Reverse DNS.** Looked up for hosts that responded. Failures are not fatal
   and never fail a scan.
3. **ICMP ping** (optional). Used only when the runtime supports raw sockets.
   Containerised deployments without `CAP_NET_RAW` fall back to TCP-only
   discovery.
4. **MAC / ARP.** Not collected in MVP; the column is present but typically
   `NULL`. We do not shell out to unsafe commands.

## Classifications

| Class | Meaning |
| --- | --- |
| `known` | IP is already in `ip_addresses` for this prefix and nothing material changed. |
| `new` | IP responded but is not present in `ip_addresses`. |
| `changed` | IP exists but observed properties (open ports, reverse DNS, MAC) changed since the previous scan. |
| `duplicate` | IP conflict: multiple active rows or a MAC collision when MAC is available. The corresponding `ip_addresses.status` is set to `conflict`. |
| `stale` | Existing IP (status `assigned`/`discovered`/`dhcp`/`active`) was not seen in this scan. Advisory only — no automatic status change in MVP. |
| `ignored` | Operator explicitly ignored this result; hidden from the active "new hosts" list. |

## Manual scans

`POST /api/v1/discovery/scans { "prefix_id": "..." }`
or `POST /api/v1/prefixes/{id}/scan`

- Rejects an unknown `prefix_id` (`400`).
- Rejects too-large IPv4 prefixes (`422`).
- Rejects IPv6 prefixes (`422`).
- Rejects if another scan is already `queued`/`running` for that prefix
  (`409`).
- Returns the queued/running scan record (`201`).

The scan executes asynchronously. Status transitions:
`queued → running → completed` or `→ failed` on error. `started_at` and
`completed_at` are populated when the corresponding transition occurs.

## Scheduled scans

Every minute the scheduler picks prefixes where:

- `scan_enabled = true`
- no scan is currently `queued` or `running`, AND
- there is no completed scan, OR the latest completed scan finished more than
  `scan_interval_seconds` ago.

`scan_interval_seconds` must be `>= 60`.

The scheduler stops cleanly on `SIGINT`/`SIGTERM` via a cancellable context.

## Reconciliation

- **Known IP seen** → `ip_addresses.last_seen_at` is bumped.
- **New result accepted** (`POST /api/v1/discovery/results/{id}/accept`) →
  creates an `ip_addresses` row in the prefix (or links an existing one when
  the address has been added since), defaults status to `discovered`, sets
  `last_seen_at` from the discovery, and stores `created_ip_address_id` on the
  result. Idempotent: re-accepting returns the existing IP.
- **Result ignored** (`POST /api/v1/discovery/results/{id}/ignore`) →
  `ignored = true`, classification rewritten to `ignored`, hidden from the
  default "new hosts" view.
- **Duplicate detected** → conflicting `ip_addresses.status` is set to
  `conflict`.

## Diagnostics & probing (operator/admin)

- `GET /api/v1/discovery/diagnostics` returns the configured worker count,
  default ports, TCP timeout, ping support flag, and runtime info (hostname,
  inside-container heuristic). It does not return secrets or raw routes.
- `POST /api/v1/discovery/probe { "address": "10.10.13.1", "ports": [22,80] }`
  runs a one-off TCP probe against an address that **must** belong to a stored
  prefix. Arbitrary internet probing is rejected. Ports are optional; when
  omitted the configured default port list is used. The Discovery page exposes
  both via the **Discovery Diagnostics** and **Probe Host** panels.
- If a scan completes with zero results the UI shows a hint explaining that
  TCP-only discovery cannot see firewalled hosts and pointing to the Probe Host
  form.

## Docker / runtime notes

- The default discovery scanner is **TCP-based**. It only sees hosts with
  open TCP ports listed in `DISCOVERY_DEFAULT_PORTS`.
- ICMP ping requires `CAP_NET_RAW` or `--privileged`. Without it Blackgrid
  silently falls back to TCP-only discovery; this is intentional. Set
  `DISCOVERY_ENABLE_PING=true` to opt in once the runtime allows raw sockets.
- ARP / MAC discovery is not performed in containers; the field stays `NULL`.
  ARP discovery cannot work reliably from a normal bridge container for
  arbitrary LAN subnets.
- If the backend runs inside Docker bridge mode it must have a route to the
  homelab subnet. On Linux Docker, bridge containers can usually reach LAN
  routes through the host if routing/firewall allows it. On Docker
  Desktop/macOS, LAN/subnet behaviour may differ because containers run inside
  a VM.

For best LAN discovery on Linux, one option is host networking:

```yaml
backend:
  network_mode: host
```

This is Linux-specific and changes how ports are exposed. An alternative if
you want ICMP without giving up bridge networking:

```yaml
backend:
  cap_add:
    - NET_RAW
```

Only useful when ping support is implemented and `DISCOVERY_ENABLE_PING=true`.
