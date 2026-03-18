# Database Migrations

SQL migration files for the water quality data management platform.

## Naming Convention

Files follow the pattern `NNN_description.up.sql` / `NNN_description.down.sql`:
- `.up.sql` — applies the migration
- `.down.sql` — reverts the migration

## Database Requirements

- **PostgreSQL 17+** with the following extensions:
  - `pgcrypto` — UUID generation fallback
  - `btree_gist` — required for the permit limits date-range exclusion constraint
  - `timescaledb` — time-series optimization for sample results and audit log

## Schema Overview

### Multi-tenancy

`organizations` is the tenant root. Most tables carry an `organization_id` either directly or through their parent (e.g., `monitoring_locations` inherit tenancy from `facilities`).

### Core Tables

| Table | Purpose |
|---|---|
| `organizations` | Multi-tenant root |
| `facilities` | Treatment plants (water, wastewater) |
| `monitoring_locations` | Sampling points within a facility |
| `parameters` | What's measured (pH, chlorine, turbidity, BOD, TSS...) |
| `units_of_measure` | mg/L, NTU, SU, CFU/100mL... |
| `analytical_methods` | EPA/Standard Methods references |
| `permit_limits` | Effective-dated compliance thresholds |
| `sample_results` | Operational measurements (TimescaleDB hypertable) |
| `instruments` | Lab and field equipment |
| `calibration_records` | Instrument verification and calibration history |
| `users` | System users |
| `roles` / `user_roles` | Role-based access control, optionally scoped to a facility |
| `audit_log` | Append-only change tracking (TimescaleDB hypertable) |

### Key Design Decisions

**Effective-dated permit limits** — The `permit_limits` table uses a PostgreSQL `EXCLUDE` constraint with `btree_gist` to prevent overlapping date ranges for the same location/parameter/limit type. This guarantees that compliance evaluations always reference exactly one limit for any given date.

**TimescaleDB hypertables** — `sample_results` and `audit_log` are partitioned by their timestamp columns for optimized time-series queries and future retention policies.

**Audit log isolation** — The `audit_log` table intentionally has no foreign keys. Audit records must never be lost due to cascading deletes or parent record changes.

**UUIDv7 primary keys** — All IDs are UUIDv7 (RFC 9562), generated in the application layer. UUIDv7 embeds a millisecond timestamp, giving natural sort order and excellent B-tree index performance. This also supports offline ID generation for the PowerSync mobile sync layer.

**user_roles NULL handling** — A NULL `facility_id` on `user_roles` means the role applies to all facilities. A `COALESCE`-based unique index prevents duplicate "all facilities" assignments since SQL UNIQUE constraints treat NULLs as distinct.
