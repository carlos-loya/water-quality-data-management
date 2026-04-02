# CLAUDE.md

## What This Project Is

Water quality data management platform for municipal utilities, built for the City of Waco RFI 2026-014. It manages water/wastewater compliance data: sample results, permit limits, instrument calibrations, trending, and regulatory reports.

The RFI PDF is in the repo root. Key sections: 6.1 (water/wastewater ops), 6.2 (lab integration), 6.3 (stormwater), 6.4 (industrial pretreatment). We're building a custom solution focused on 6.1 first.

## Tech Stack

- **Backend:** Go 1.26, net/http (stdlib router), pgx/v5
- **Database:** PostgreSQL 17 + TimescaleDB (hypertables for sample_results and audit_log)
- **Events:** NATS with JetStream (audit trail via pub/sub)
- **Frontend:** React 19, TypeScript 5.9, Vite 8, Tailwind CSS 4, TanStack Query 5, Recharts 3
- **Auth:** JWT (HS256, 24h expiry) + bcrypt passwords
- **Reports:** excelize (Excel), gofpdf (PDF)

## Project Structure

```
cmd/server/main.go          # Entry point — connects DB, NATS, runs migrations, starts HTTP
internal/
  api/router.go             # All route definitions in one place
  api/handlers.go           # HTTP handlers (~620 lines)
  api/middleware.go          # Logging, JWT auth, role authorization
  auth/auth.go              # JWT generation/validation, bcrypt, role checks
  storage/db.go             # pgxpool connection + migration runner
  storage/queries.go        # All SQL queries (~720 lines)
  events/bus.go             # NATS publish/subscribe wrapper
  events/audit.go           # Audit log consumer
  ingestion/csv.go          # CSV import with per-row validation
  reports/excel.go          # Compliance Excel export
  reports/pdf.go            # Compliance PDF export
migrations/
  001_initial_schema.up.sql # Full schema: orgs, users, roles, facilities, parameters, etc.
  002_validation_rules.up.sql # Configurable per-parameter range validation (PR #11)
  seed.sql                  # Demo data: Clearwater Utilities org, 5 users, 2 facilities
web/src/
  api/client.ts             # Typed fetch wrapper for all API endpoints
  api/auth.ts               # JWT token management, role helpers (hasRole, canReview, etc.)
  api/types.ts              # TypeScript interfaces matching Go types
  components/               # React components (table, form, charts, etc.)
  App.tsx                   # Main app: login → facility selector → tabbed dashboard
```

## Running Locally

```bash
docker compose up -d                  # PostgreSQL + NATS
go run ./cmd/server                   # Backend on :8080 (auto-runs migrations)
docker compose exec -T db psql -U wqm -d water_quality < migrations/seed.sql
cd web && npm install && npx vite     # Frontend on :5173 (proxies /api to :8080)
```

Demo logins (password: `demo1234`):
- `admin@clearwater.gov` — admin (all facilities)
- `jmartinez@clearwater.gov` — operator (North Water Treatment)
- `akim@clearwater.gov` — reviewer (all facilities)
- `rjohnson@clearwater.gov` — operator (South Wastewater Treatment)
- `tlee@clearwater.gov` — viewer (all facilities)

## Running Tests

```bash
go test ./...                  # Backend (auth, handlers, reports)
cd web && npm test             # Frontend (vitest — auth helpers)
```

## Key Design Decisions

- **UUIDv7 primary keys** — RFC 9562, naturally sortable by time
- **Effective-dated permit limits** — EXCLUDE constraint prevents overlapping date ranges per location/parameter/type
- **Sample result state machine** — draft → reviewed → approved, enforced in SQL (`UPDATE ... WHERE status = 'draft'` returns no rows if not in expected state)
- **Audit trail isolation** — audit_log has no foreign keys; records survive any deletion
- **TimescaleDB hypertables** — automatic time-based partitioning on sample_results and audit_log
- **Facility-scoped RBAC** — user_roles.facility_id is NULL for org-wide access, or a specific facility UUID for scoped access
- **Event-driven audit** — handlers publish to NATS, audit consumer writes to DB asynchronously; NATS failures don't block HTTP responses

## API Patterns

All routes are in `internal/api/router.go`. Pattern:
- Public: `GET /api/v1/health`, `POST /api/v1/auth/login`
- Protected: wrapped with `withAuth(jwtSecret, mux)` — requires `Authorization: Bearer <token>`
- Role-gated: `requireRole("admin", handler)` or `requireAnyRole([]string{"admin","operator"}, handler)`

Handlers validate inputs first (return 400), then call `storage.Queries` methods, then publish events. Error responses are always `{"error": "message"}`.

## Database

Schema is in `migrations/001_initial_schema.up.sql`. Core tables: organizations, users, roles, user_roles, facilities, monitoring_locations, parameters, units_of_measure, analytical_methods, permit_limits, sample_results, instruments, calibration_records, audit_log.

All queries are in `internal/storage/queries.go` as methods on `*Queries`. No ORM — handwritten SQL with pgx row scanning.

## Git Workflow

- Branch per feature: `feature/<name>`
- PR into `main` with summary + test plan
- Always run `go build ./...`, `go test ./...`, `cd web && npx tsc --noEmit`, and `cd web && npm test` before pushing
- Commit messages: imperative mood, explain "why" not just "what"

## What's Implemented

- Multi-tenant data model (orgs, facilities, locations, parameters, units, methods)
- JWT auth with bcrypt passwords and login flow
- Facility-scoped RBAC (admin, operator, reviewer, viewer)
- Sample result CRUD with draft → reviewed → approved workflow
- Operator data entry form with unit auto-selection
- Configurable validation rules per parameter (min/max range, precision) — PR #11
- CSV import with per-row validation and error reporting
- Compliance evaluation against effective-dated permit limits
- Time-series trending charts (Recharts) with permit limit reference lines
- Instrument/calibration tracking with overdue alerts
- Excel and PDF compliance report export
- NATS event bus with audit trail consumer
- Audit history panel per record

## What's Not Yet Built

Refer to the RFI sections. Major gaps:
- **6.1 remaining:** alerts/notifications, mobile-responsive layout, BTLIMS lab integration, Power BI data model, scheduled reports, qualifiers/comments/attachments on records
- **6.2:** Laboratory integration (BTLIMS mapping, reconciliation)
- **6.3:** Stormwater program (MS4 inventory, inspections, weather tracking, corrective actions)
- **6.4:** Industrial pretreatment & FOG (permits, inspections, hauler manifests, customer portal)
- **6.5/6.6:** Cross-program analytics and integration with city systems (KloudGin, SpryPoint, MaintStar)
- **Testing:** Comprehensive suite defined in issue #13. Current coverage is auth, handler validation, reports, and frontend auth helpers. Storage/integration tests still needed.

## Style Notes

- Don't over-scaffold; start minimal, let structure emerge
- Create docs alongside code, commit often on feature branches
- Always validate (build + test + typecheck) before pushing or creating PRs
- Backend tests don't require a database — test pure logic and input validation paths
- Frontend uses Vitest (not Jest)
