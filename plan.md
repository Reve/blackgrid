1. **Database Schema & Queries**:
   - Verify `002_discovery_update.sql` migration is in place.
   - Verify `discovery.sql` queries are defined and generated via `sqlc`.
   - Ensure the database schema includes required properties: `scan_enabled`, `scan_interval_seconds` on `prefixes`, `last_seen_at` on `ip_addresses`, and fields on `discovery_scans` and `discovery_results`.
   - Update `sql/query.sql` to include new prefix and ip_address updates. Re-run sqlc.

2. **Backend Discovery Service**:
   - Create `backend/internal/service/discovery.go`.
   - Implement `DiscoveryService` with `StartManualScan`, `RunScan`, `RunScheduledScans`, `ListScans`, `ListResults`, `AcceptResult`, `IgnoreResult`.
   - Bounded concurrency with worker pools. Max prefix size /22.
   - Implement safe IP scanning logic (TCP connect on default ports: 22, 53, 80, 443, 5432, 6379, 8000, 8080, 9000, 9443).
   - Resolve reverse DNS, determine if known/new/changed/stale.
   - Scheduled scanning logic (`RunScheduledScans`) that runs automatically (every minute) looking for `scan_enabled` prefixes.

3. **Backend API Endpoints**:
   - Create `backend/internal/api/handlers/discovery.go`.
   - Implement handlers for:
     - `GET /api/v1/discovery/scans`
     - `POST /api/v1/discovery/scans`
     - `GET /api/v1/discovery/scans/:id`
     - `GET /api/v1/discovery/results`
     - `POST /api/v1/discovery/results/:id/accept`
     - `POST /api/v1/discovery/results/:id/ignore`
     - `POST /api/v1/prefixes/:id/scan`
   - Link these in `main.go`.

4. **Frontend API Client**:
   - Update `frontend/src/api/client.ts` with Discovery interfaces (`DiscoveryScan`, `DiscoveryResult`) and API functions.
   - Export correct types required by `verbatimModuleSyntax`.

5. **Frontend UI**:
   - Update `frontend/src/pages/IPAM.tsx` to include scan controls and last seen data for prefixes/IPs.
   - Create `frontend/src/pages/Discovery.tsx` (or update existing).
   - Display discovery scans, results. Handle accept/ignore actions.
   - Integrate "Create Monitor" prefill logic.
   - Preserve Terminal Frontier styling.
   - Update `App.tsx` navigation.

6. **Backend Tests**:
   - Write tests in `backend/internal/service/discovery_test.go` and `backend/internal/api/handlers/discovery_test.go`.
   - Test safety rules (reject arbitrary CIDR, large prefixes).
   - Test classifications (known, new, changed, stale).
   - Test reconciliation.

7. **Documentation**:
   - Update `README.md`.
   - Add `docs/api.md` and `docs/discovery.md`.

8. **Pre-commit Checks**:
   - Run verification checks to ensure testing, reviews, reflection.
