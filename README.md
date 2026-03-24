# Water Quality Data Management

An open-source platform for managing water and wastewater utility compliance data, built with Go, React, PostgreSQL, and NATS.

Inspired by real municipal requirements for an integrated system covering water quality operations, compliance tracking, instrument calibration, trending analytics, and automated regulatory reporting.

## Features

**Operations & Compliance Data**
- Multi-tenant data model with facilities, monitoring locations, parameters, and effective-dated permit limits
- Sample results with draft/reviewed/approved workflow enforced at the database level
- Compliance evaluation engine that checks results against effective permit limits
- CSV import pipeline with per-row validation and error reporting

**Instrument Calibration Tracking**
- Instrument inventory with calibration schedules and overdue alerts
- Calibration history with pass/fail status, pre/post values, and method references
- Instruments sorted by urgency (overdue first)

**Trending & Analytics**
- Time-series trending charts (Recharts) with permit limit reference lines
- Configurable lookback periods (7d, 30d, 90d, 1y)
- Data grouped by parameter and monitoring location

**Reporting**
- Excel export with styled headers, auto-filters, and exceedance highlighting
- PDF export with summary counts, formatted tables, and alternating row colors

**Audit Trail**
- Every data change published to NATS event bus
- Audit consumer writes to append-only audit log with before/after snapshots
- Full change history viewable per record

**Dashboard**
- Facility selector with tabbed views (Sample Results, Trending, Compliance, Instruments)
- Inline review/approve workflow actions
- Status filtering and audit history modal

## Tech Stack

| Layer | Technology |
|---|---|
| Backend | Go 1.22+ (net/http, pgx/v5) |
| Database | PostgreSQL 17 + TimescaleDB |
| Event Bus | NATS with JetStream |
| Frontend | React + TypeScript + Vite |
| Styling | Tailwind CSS |
| Charts | Recharts |
| Data Fetching | TanStack Query |
| Reports | excelize (Excel), gofpdf (PDF) |
| Infrastructure | Docker Compose |

## Getting Started

### Prerequisites

- [Go 1.22+](https://go.dev/dl/)
- [Node.js 18+](https://nodejs.org/)
- [Docker](https://docs.docker.com/get-docker/) and Docker Compose

### Run

```bash
# Start PostgreSQL + TimescaleDB and NATS
docker compose up -d

# Start the Go API server (runs migrations automatically)
go run ./cmd/server

# Load seed data (fictional "City of Clearwater" with two treatment plants)
docker compose exec -T db psql -U wqm -d water_quality < migrations/seed.sql

# In a separate terminal, start the frontend
cd web && npm install && npx vite
```

Open [http://localhost:5173](http://localhost:5173) to access the dashboard.

### Environment Variables

| Variable | Default | Description |
|---|---|---|
| `DATABASE_URL` | `postgres://wqm:wqm_dev@localhost:5432/water_quality?sslmode=disable` | PostgreSQL connection string |
| `NATS_URL` | `nats://localhost:4222` | NATS server URL |
| `MIGRATIONS_PATH` | `file://migrations` | Path to SQL migration files |
| `ADDR` | `:8080` | HTTP server listen address |

## API Endpoints

| Method | Path | Description |
|---|---|---|
| `GET` | `/api/v1/health` | Health check |
| `GET` | `/api/v1/organizations/{org_id}/facilities` | List facilities |
| `GET` | `/api/v1/facilities/{id}/monitoring-locations` | List monitoring locations |
| `GET` | `/api/v1/organizations/{org_id}/parameters` | List parameters |
| `GET` | `/api/v1/sample-results` | List sample results (filterable) |
| `POST` | `/api/v1/sample-results` | Create sample result |
| `PATCH` | `/api/v1/sample-results/{id}/review` | Transition draft to reviewed |
| `PATCH` | `/api/v1/sample-results/{id}/approve` | Transition reviewed to approved |
| `POST` | `/api/v1/organizations/{org_id}/sample-results/import` | CSV file import |
| `GET` | `/api/v1/facilities/{id}/trending` | Time-series trending data |
| `GET` | `/api/v1/facilities/{id}/instruments` | Instrument calibration statuses |
| `GET` | `/api/v1/instruments/{id}/calibrations` | Calibration history |
| `GET` | `/api/v1/facilities/{id}/compliance` | Compliance evaluation |
| `GET` | `/api/v1/facilities/{id}/reports/compliance.xlsx` | Excel compliance report |
| `GET` | `/api/v1/facilities/{id}/reports/compliance.pdf` | PDF compliance report |
| `GET` | `/api/v1/audit-log/{record_id}` | Audit history for a record |

## Database Schema

The schema is designed around water and wastewater compliance workflows:

- **Multi-tenant** via `organizations` table
- **Effective-dated permit limits** with a PostgreSQL `EXCLUDE` constraint preventing overlapping date ranges
- **TimescaleDB hypertables** on `sample_results` and `audit_log` for optimized time-series queries
- **UUIDv7 primary keys** for natural sort order and offline-safe ID generation
- **Audit log** with no foreign keys (records survive any deletion)

See [`migrations/README.md`](migrations/README.md) for the full schema documentation.

## Project Structure

```
cmd/server/          Go entrypoint — connects DB, runs migrations, starts HTTP server
internal/
  api/               HTTP handlers, routing, middleware
  storage/           Database queries (pgx/v5)
  events/            NATS event bus and audit consumer
  ingestion/         CSV import pipeline
  reports/           Excel and PDF report generation
migrations/          SQL schema and seed data
web/                 React frontend (Vite + TypeScript + Tailwind)
  src/api/           API client and TypeScript types
  src/components/    React components
```

## License

MIT
