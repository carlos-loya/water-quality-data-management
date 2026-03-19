package api

import (
	"net/http"

	"github.com/carlos-loya/water-quality-data-management/internal/storage"
)

// NewRouter creates the HTTP handler with all routes.
func NewRouter(queries *storage.Queries) http.Handler {
	mux := http.NewServeMux()
	h := &handler{queries: queries}

	mux.HandleFunc("GET /api/v1/health", h.health)
	mux.HandleFunc("GET /api/v1/organizations/{org_id}/facilities", h.listFacilities)
	mux.HandleFunc("GET /api/v1/facilities/{facility_id}/monitoring-locations", h.listMonitoringLocations)
	mux.HandleFunc("GET /api/v1/organizations/{org_id}/parameters", h.listParameters)
	mux.HandleFunc("GET /api/v1/sample-results", h.listSampleResults)
	mux.HandleFunc("POST /api/v1/sample-results", h.createSampleResult)
	mux.HandleFunc("GET /api/v1/facilities/{facility_id}/compliance", h.evaluateCompliance)

	return withLogging(mux)
}
