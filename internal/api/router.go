package api

import (
	"net/http"

	"github.com/carlos-loya/water-quality-data-management/internal/events"
	"github.com/carlos-loya/water-quality-data-management/internal/storage"
)

// NewRouter creates the HTTP handler with all routes.
func NewRouter(queries *storage.Queries, bus *events.Bus, jwtSecret string) http.Handler {
	mux := http.NewServeMux()
	h := &handler{queries: queries, bus: bus, jwtSecret: jwtSecret}

	// Public routes
	mux.HandleFunc("GET /api/v1/health", h.health)
	mux.HandleFunc("POST /api/v1/auth/login", h.login)

	// Protected routes
	protected := http.NewServeMux()
	protected.HandleFunc("GET /api/v1/auth/me", h.me)
	protected.HandleFunc("GET /api/v1/organizations/{org_id}/facilities", h.listFacilities)
	protected.HandleFunc("GET /api/v1/facilities/{facility_id}/monitoring-locations", h.listMonitoringLocations)
	protected.HandleFunc("GET /api/v1/organizations/{org_id}/parameters", h.listParameters)
	protected.HandleFunc("GET /api/v1/organizations/{org_id}/units", h.listUnits)
	protected.HandleFunc("GET /api/v1/sample-results", h.listSampleResults)
	protected.HandleFunc("POST /api/v1/sample-results", requireAnyRole([]string{"admin", "operator"}, h.createSampleResult))
	protected.HandleFunc("PATCH /api/v1/sample-results/{id}/review", requireAnyRole([]string{"admin", "reviewer"}, h.reviewSampleResult))
	protected.HandleFunc("PATCH /api/v1/sample-results/{id}/approve", requireRole("admin", h.approveSampleResult))
	protected.HandleFunc("POST /api/v1/organizations/{org_id}/sample-results/import", requireAnyRole([]string{"admin", "operator"}, h.importSampleResults))
	protected.HandleFunc("GET /api/v1/facilities/{facility_id}/trending", h.getTrending)
	protected.HandleFunc("GET /api/v1/facilities/{facility_id}/instruments", h.listInstrumentStatuses)
	protected.HandleFunc("GET /api/v1/instruments/{instrument_id}/calibrations", h.listCalibrationRecords)
	protected.HandleFunc("GET /api/v1/facilities/{facility_id}/compliance", h.evaluateCompliance)
	protected.HandleFunc("GET /api/v1/facilities/{facility_id}/reports/compliance.xlsx", h.complianceExcel)
	protected.HandleFunc("GET /api/v1/facilities/{facility_id}/reports/compliance.pdf", h.compliancePDF)
	protected.HandleFunc("GET /api/v1/audit-log/{record_id}", h.listAuditLog)

	mux.Handle("/api/v1/", withAuth(jwtSecret, protected))

	return withLogging(mux)
}
