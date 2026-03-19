package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"

	"github.com/carlos-loya/water-quality-data-management/internal/storage"
)

type handler struct {
	queries *storage.Queries
}

func (h *handler) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *handler) listFacilities(w http.ResponseWriter, r *http.Request) {
	orgID, err := parseUUID(r.PathValue("org_id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid org_id")
		return
	}

	facilities, err := h.queries.ListFacilities(r.Context(), orgID)
	if err != nil {
		slog.Error("list facilities", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, facilities)
}

func (h *handler) listMonitoringLocations(w http.ResponseWriter, r *http.Request) {
	facilityID, err := parseUUID(r.PathValue("facility_id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid facility_id")
		return
	}

	locations, err := h.queries.ListMonitoringLocations(r.Context(), facilityID)
	if err != nil {
		slog.Error("list monitoring locations", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, locations)
}

func (h *handler) listParameters(w http.ResponseWriter, r *http.Request) {
	orgID, err := parseUUID(r.PathValue("org_id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid org_id")
		return
	}

	params, err := h.queries.ListParameters(r.Context(), orgID)
	if err != nil {
		slog.Error("list parameters", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, params)
}

func (h *handler) listSampleResults(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	filter := storage.SampleResultFilter{}

	if v := q.Get("monitoring_location_id"); v != "" {
		id, err := parseUUID(v)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid monitoring_location_id")
			return
		}
		filter.MonitoringLocationID = &id
	}
	if v := q.Get("parameter_id"); v != "" {
		id, err := parseUUID(v)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid parameter_id")
			return
		}
		filter.ParameterID = &id
	}
	if v := q.Get("status"); v != "" {
		filter.Status = &v
	}
	if v := q.Get("start_date"); v != "" {
		t, err := time.Parse("2006-01-02", v)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid start_date, expected YYYY-MM-DD")
			return
		}
		filter.StartDate = &t
	}
	if v := q.Get("end_date"); v != "" {
		t, err := time.Parse("2006-01-02", v)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid end_date, expected YYYY-MM-DD")
			return
		}
		filter.EndDate = &t
	}
	if v := q.Get("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 {
			writeError(w, http.StatusBadRequest, "invalid limit")
			return
		}
		filter.Limit = n
	}

	results, err := h.queries.ListSampleResults(r.Context(), filter)
	if err != nil {
		slog.Error("list sample results", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, results)
}

func (h *handler) createSampleResult(w http.ResponseWriter, r *http.Request) {
	var params storage.CreateSampleResultParams
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	// Validation
	if params.MonitoringLocationID == uuid.Nil {
		writeError(w, http.StatusBadRequest, "monitoring_location_id is required")
		return
	}
	if params.ParameterID == uuid.Nil {
		writeError(w, http.StatusBadRequest, "parameter_id is required")
		return
	}
	if params.UnitID == uuid.Nil {
		writeError(w, http.StatusBadRequest, "unit_id is required")
		return
	}
	if params.CollectedAt.IsZero() {
		writeError(w, http.StatusBadRequest, "collected_at is required")
		return
	}
	if params.EnteredBy == uuid.Nil {
		writeError(w, http.StatusBadRequest, "entered_by is required")
		return
	}
	if params.Source == "" {
		params.Source = "manual"
	}
	if params.ResultValue == nil && params.ResultQualifier == nil {
		writeError(w, http.StatusBadRequest, "either result_value or result_qualifier is required")
		return
	}

	result, err := h.queries.CreateSampleResult(r.Context(), params)
	if err != nil {
		slog.Error("create sample result", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusCreated, result)
}

func (h *handler) evaluateCompliance(w http.ResponseWriter, r *http.Request) {
	facilityID, err := parseUUID(r.PathValue("facility_id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid facility_id")
		return
	}

	results, err := h.queries.EvaluateCompliance(r.Context(), facilityID)
	if err != nil {
		slog.Error("evaluate compliance", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, results)
}

// --- helpers ---

func parseUUID(s string) (uuid.UUID, error) {
	return uuid.Parse(s)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
