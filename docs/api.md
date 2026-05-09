# Blackgrid API — Discovery endpoints

All endpoints are prefixed with `/api/v1`.

## Discovery scans

### `GET /discovery/scans`

Query parameters (all optional):

| Param | Type | Default |
| --- | --- | --- |
| `prefix_id` | UUID | — |
| `status` | `queued`/`running`/`completed`/`failed`/`cancelled` | — |
| `limit` | int | `100` |
| `offset` | int | `0` |

Returns an array of `DiscoveryScan` ordered by `created_at desc`.

### `POST /discovery/scans`

Body: `{ "prefix_id": "<uuid>" }`

Starts a manual scan against a stored prefix. Behaviour:

- `400` if `prefix_id` is missing or unknown.
- `422` if the prefix is too large for IPv4 or is IPv6.
- `409` if another scan for that prefix is already `queued`/`running`.
- `201` with the new `DiscoveryScan` otherwise.

### `GET /discovery/scans/{id}`

Returns a single `DiscoveryScan`. `404` if unknown.

### `POST /prefixes/{id}/scan`

Equivalent to `POST /discovery/scans` for the path-bound prefix.

## Discovery results

### `GET /discovery/results`

Query parameters (all optional):

| Param | Type | Default |
| --- | --- | --- |
| `scan_id` | UUID | — |
| `prefix_id` | UUID | — |
| `classification` | string | — |
| `ignored` | bool | `false` (effective default for the UI) |
| `limit` | int | `100` |
| `offset` | int | `0` |

Returns `DiscoveryResult[]` ordered by `seen_at desc`.

### `POST /discovery/results/{id}/accept`

Body (all optional): `{ "hostname": "...", "fqdn": "...", "status": "..." }`

- If the result is already accepted, returns the linked `IPAddress`.
- If the IP address already exists in the prefix, links to it without
  creating a duplicate.
- Otherwise creates a new `ip_addresses` row using:
  - `address` from the discovery result;
  - `description` = first non-empty of `fqdn`, `hostname`, `reverse_dns`,
    discovery hostname;
  - `status` = request `status` or `discovered`;
  - `last_seen_at` set to now.

### `POST /discovery/results/{id}/ignore`

Marks the result `ignored = true`, sets `classification` to `ignored`, and
returns the updated record.

## Prefix scan configuration

### `PUT /prefixes/{id}/scan-config`

Body: `{ "scan_enabled": bool, "scan_interval_seconds": int }`

- `scan_interval_seconds` must be `>= 60`.
- Returns the updated `Prefix`.

## Schemas

```ts
type DiscoveryScan = {
  id: string;
  prefix_id: string;
  status: 'queued' | 'running' | 'completed' | 'failed' | 'cancelled';
  started_at: string | null;
  completed_at: string | null;
  error: string | null;
  created_at: string;
  updated_at: string;
};

type DiscoveryClassification =
  | 'known' | 'new' | 'changed' | 'duplicate' | 'stale' | 'ignored';

type DiscoveryResult = {
  id: string;
  scan_id: string;
  prefix_id: string;
  address: string;            // bare IPv4 address
  mac_address: string | null;
  hostname: string | null;
  reverse_dns: string | null;
  open_ports: number[];
  latency_ms: number | null;
  classification: DiscoveryClassification;
  seen_at: string;
  ignored: boolean;
  accepted_at: string | null;
  created_ip_address_id: string | null;
  created_at: string;
  updated_at: string;
};
```
