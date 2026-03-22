package api

import (
	"net/http"

	"github.com/carlos-loya/water-quality-data-management/internal/events"
	"github.com/carlos-loya/water-quality-data-management/internal/storage"
)

// NewRouter creates the HTTP handler with all routes.
func NewRouter(queries *storage.Queries, bus *events.Bus) http.Handler {
	mux := http.NewServeMux()
	h := &handler{queries: queries, bus: bus}

	mux.HandleFunc("GET /api/v1/health", h.health)
	mux.HandleFunc("GET /api/v1/organizations/{org_id}/facilities", h.listFacilities)
	mux.HandleFunc("GET /api/v1/facilities/{facility_id}/monitoring-locations", h.listMonitoringLocations)
	mux.HandleFunc("GET /api/v1/organizations/{org_id}/parameters", h.listParameters)
	mux.HandleFunc("GET /api/v1/sample-results", h.listSampleResults)
	mux.HandleFunc("POST /api/v1/sample-results", h.createSampleResult)
	mux.HandleFunc("PATCH /api/v1/sample-results/{id}/review", h.reviewSampleResult)
	mux.HandleFunc("PATCH /api/v1/sample-results/{id}/approve", h.approveSampleResult)
	mux.HandleFunc("POST /api/v1/organizations/{org_id}/sample-results/import", h.importSampleResults)
	mux.HandleFunc("GET /api/v1/facilities/{facility_id}/trending", h.getTrending)
	mux.HandleFunc("GET /api/v1/facilities/{facility_id}/compliance", h.evaluateCompliance)
	mux.HandleFunc("GET /api/v1/facilities/{facility_id}/reports/compliance.xlsx", h.complianceExcel)
	mux.HandleFunc("GET /api/v1/facilities/{facility_id}/reports/compliance.pdf", h.compliancePDF)
	mux.HandleFunc("GET /api/v1/audit-log/{record_id}", h.listAuditLog)

	return withLogging(mux)
}
