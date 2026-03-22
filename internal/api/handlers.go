package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/carlos-loya/water-quality-data-management/internal/events"
	"github.com/carlos-loya/water-quality-data-management/internal/ingestion"
	"github.com/carlos-loya/water-quality-data-management/internal/reports"
	"github.com/carlos-loya/water-quality-data-management/internal/storage"
)

type handler struct {
	queries *storage.Queries
	bus     *events.Bus
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

	h.publishResultEvent(r.Context(), events.SubjectSampleResultCreated, "insert", result, nil)
	writeJSON(w, http.StatusCreated, result)
}

func (h *handler) reviewSampleResult(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	var body struct {
		ReviewerID uuid.UUID `json:"reviewer_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if body.ReviewerID == uuid.Nil {
		writeError(w, http.StatusBadRequest, "reviewer_id is required")
		return
	}

	before, err := h.queries.GetSampleResult(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "sample result not found")
			return
		}
		slog.Error("get sample result", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	after, err := h.queries.ReviewSampleResult(r.Context(), id, body.ReviewerID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusConflict, "result is not in 'draft' status")
			return
		}
		slog.Error("review sample result", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	h.publishResultEvent(r.Context(), events.SubjectSampleResultReviewed, "update", after, &before)
	writeJSON(w, http.StatusOK, after)
}

func (h *handler) approveSampleResult(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	var body struct {
		ApproverID uuid.UUID `json:"approver_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if body.ApproverID == uuid.Nil {
		writeError(w, http.StatusBadRequest, "approver_id is required")
		return
	}

	before, err := h.queries.GetSampleResult(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "sample result not found")
			return
		}
		slog.Error("get sample result", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	after, err := h.queries.ApproveSampleResult(r.Context(), id, body.ApproverID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusConflict, "result is not in 'reviewed' status")
			return
		}
		slog.Error("approve sample result", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	h.publishResultEvent(r.Context(), events.SubjectSampleResultApproved, "update", after, &before)
	writeJSON(w, http.StatusOK, after)
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

func (h *handler) complianceExcel(w http.ResponseWriter, r *http.Request) {
	facilityID, err := parseUUID(r.PathValue("facility_id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid facility_id")
		return
	}

	results, err := h.queries.EvaluateCompliance(r.Context(), facilityID)
	if err != nil {
		slog.Error("compliance excel", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	facilityName := "Facility"
	if len(results) > 0 {
		facilityName = results[0].FacilityName
	}

	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="compliance-report-%s.xlsx"`, time.Now().Format("2006-01-02")))

	if err := reports.WriteComplianceExcel(w, facilityName, results); err != nil {
		slog.Error("write excel", "error", err)
	}
}

func (h *handler) compliancePDF(w http.ResponseWriter, r *http.Request) {
	facilityID, err := parseUUID(r.PathValue("facility_id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid facility_id")
		return
	}

	results, err := h.queries.EvaluateCompliance(r.Context(), facilityID)
	if err != nil {
		slog.Error("compliance pdf", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	facilityName := "Facility"
	if len(results) > 0 {
		facilityName = results[0].FacilityName
	}

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="compliance-report-%s.pdf"`, time.Now().Format("2006-01-02")))

	if err := reports.WriteCompliancePDF(w, facilityName, results); err != nil {
		slog.Error("write pdf", "error", err)
	}
}

func (h *handler) listAuditLog(w http.ResponseWriter, r *http.Request) {
	recordID, err := parseUUID(r.PathValue("record_id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid record_id")
		return
	}

	entries, err := h.queries.ListAuditLog(r.Context(), recordID)
	if err != nil {
		slog.Error("list audit log", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, entries)
}

func (h *handler) listInstrumentStatuses(w http.ResponseWriter, r *http.Request) {
	facilityID, err := parseUUID(r.PathValue("facility_id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid facility_id")
		return
	}

	statuses, err := h.queries.ListInstrumentStatuses(r.Context(), facilityID)
	if err != nil {
		slog.Error("list instrument statuses", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, statuses)
}

func (h *handler) listCalibrationRecords(w http.ResponseWriter, r *http.Request) {
	instrumentID, err := parseUUID(r.PathValue("instrument_id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid instrument_id")
		return
	}

	records, err := h.queries.ListCalibrationRecords(r.Context(), instrumentID)
	if err != nil {
		slog.Error("list calibration records", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, records)
}

func (h *handler) getTrending(w http.ResponseWriter, r *http.Request) {
	facilityID, err := parseUUID(r.PathValue("facility_id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid facility_id")
		return
	}

	days := 30
	if v := r.URL.Query().Get("days"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 365 {
			days = n
		}
	}

	series, err := h.queries.GetTrendingData(r.Context(), facilityID, days)
	if err != nil {
		slog.Error("get trending", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, series)
}

func (h *handler) importSampleResults(w http.ResponseWriter, r *http.Request) {
	orgID, err := parseUUID(r.PathValue("org_id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid org_id")
		return
	}

	// Parse multipart form — 10 MB max
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		writeError(w, http.StatusBadRequest, "invalid multipart form")
		return
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "missing file field")
		return
	}
	defer file.Close()

	enteredByStr := r.FormValue("entered_by")
	enteredBy, err := parseUUID(enteredByStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid or missing entered_by")
		return
	}

	importer := ingestion.NewCSVImporter(h.queries)
	result, err := importer.Import(r.Context(), file, orgID, enteredBy)
	if err != nil {
		slog.Error("csv import", "error", err)
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Publish audit events for each imported result
	for _, sr := range result.Results {
		h.publishResultEvent(r.Context(), events.SubjectSampleResultCreated, "insert", sr, nil)
	}

	writeJSON(w, http.StatusOK, result)
}

// publishResultEvent sends a change event for a sample result to NATS.
// Failures are logged but do not block the HTTP response.
func (h *handler) publishResultEvent(ctx context.Context, subject, action string, result storage.SampleResult, before *storage.SampleResult) {
	orgID, err := h.queries.GetOrganizationIDForResult(ctx, result.ID)
	if err != nil {
		slog.Error("resolve org for audit", "error", err)
		return
	}

	newJSON, _ := json.Marshal(result)
	event := events.ChangeEvent{
		Subject:        subject,
		Timestamp:      time.Now(),
		OrganizationID: orgID,
		TableName:      "sample_results",
		RecordID:       result.ID,
		Action:         action,
		ChangedBy:      result.EnteredBy,
		NewValues:      newJSON,
	}

	if action == "update" && before != nil {
		oldJSON, _ := json.Marshal(before)
		event.OldValues = oldJSON
		// For reviews/approvals, the changer is the reviewer/approver, not the original enterer
		if result.ReviewedBy != nil && (before.ReviewedBy == nil) {
			event.ChangedBy = *result.ReviewedBy
		}
		if result.ApprovedBy != nil && (before.ApprovedBy == nil) {
			event.ChangedBy = *result.ApprovedBy
		}
	}

	if err := h.bus.Publish(event); err != nil {
		slog.Error("publish event", "error", err, "subject", subject)
	}
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
