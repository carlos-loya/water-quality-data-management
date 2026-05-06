package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/carlos-loya/water-quality-data-management/internal/auth"
	"github.com/carlos-loya/water-quality-data-management/internal/events"
	"github.com/carlos-loya/water-quality-data-management/internal/ingestion"
	"github.com/carlos-loya/water-quality-data-management/internal/reports"
	"github.com/carlos-loya/water-quality-data-management/internal/storage"
	"github.com/carlos-loya/water-quality-data-management/internal/storage/blob"
)

type handler struct {
	store     storage.Store
	bus       *events.Bus
	blobs     blob.Store
	jwtSecret string
}

const maxUploadBytes = 10 * 1024 * 1024 // 10 MiB

var allowedAttachmentTypes = map[string]bool{
	"application/pdf":  true,
	"image/png":        true,
	"image/jpeg":       true,
	"text/csv":         true,
	"application/vnd.ms-excel": true,
	"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet": true,
}

func (h *handler) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *handler) login(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if body.Email == "" || body.Password == "" {
		writeError(w, http.StatusBadRequest, "email and password are required")
		return
	}

	user, err := h.store.GetUserByEmail(r.Context(), body.Email)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}
	if !user.Active {
		writeError(w, http.StatusUnauthorized, "account is disabled")
		return
	}
	if !auth.CheckPassword(user.PasswordHash, body.Password) {
		writeError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	dbRoles, err := h.store.GetUserRoles(r.Context(), user.ID)
	if err != nil {
		slog.Error("get user roles", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	roleClaims := make([]auth.RoleClaim, len(dbRoles))
	for i, r := range dbRoles {
		roleClaims[i] = auth.RoleClaim{Role: r.RoleName}
		if r.FacilityID != nil {
			roleClaims[i].FacilityID = r.FacilityID.String()
		}
	}

	token, err := auth.GenerateToken(h.jwtSecret, user.ID, user.OrganizationID, user.Email, user.Name, roleClaims)
	if err != nil {
		slog.Error("generate token", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"token": token,
		"user": map[string]any{
			"id":              user.ID,
			"organization_id": user.OrganizationID,
			"email":           user.Email,
			"name":            user.Name,
			"roles":           roleClaims,
		},
	})
}

func (h *handler) me(w http.ResponseWriter, r *http.Request) {
	claims := auth.UserFrom(r.Context())
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"id":              claims.UserID,
		"organization_id": claims.OrganizationID,
		"email":           claims.Email,
		"name":            claims.Name,
		"roles":           claims.Roles,
	})
}

func (h *handler) listFacilities(w http.ResponseWriter, r *http.Request) {
	orgID, err := parseUUID(r.PathValue("org_id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid org_id")
		return
	}

	claims := auth.UserFrom(r.Context())
	facilities, err := h.store.ListFacilitiesForUser(r.Context(), orgID, claims.UserID)
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
	if !checkFacilityAccess(w, r, facilityID) {
		return
	}

	locations, err := h.store.ListMonitoringLocations(r.Context(), facilityID)
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

	params, err := h.store.ListParameters(r.Context(), orgID)
	if err != nil {
		slog.Error("list parameters", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, params)
}

func (h *handler) listUnits(w http.ResponseWriter, r *http.Request) {
	orgID, err := parseUUID(r.PathValue("org_id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid org_id")
		return
	}

	units, err := h.store.ListUnits(r.Context(), orgID)
	if err != nil {
		slog.Error("list units", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, units)
}

func (h *handler) listValidationRules(w http.ResponseWriter, r *http.Request) {
	orgID, err := parseUUID(r.PathValue("org_id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid org_id")
		return
	}

	rules, err := h.store.ListValidationRules(r.Context(), orgID)
	if err != nil {
		slog.Error("list validation rules", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, rules)
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

	results, err := h.store.ListSampleResults(r.Context(), filter)
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

	// Validate result_value against configurable parameter rules.
	rule, err := h.store.GetValidationRule(r.Context(), params.ParameterID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		slog.Error("get validation rule", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if err == nil {
		if rule.IsRequired && params.ResultValue == nil && params.ResultQualifier == nil {
			writeError(w, http.StatusBadRequest, "a numeric result_value is required for this parameter")
			return
		}
		// Out-of-range values are permitted only with a documented override reason.
		// This keeps the data defensible: we record the measurement AND why the
		// operator chose to save it despite the rule.
		if params.ResultValue != nil {
			v := *params.ResultValue
			outOfRange := (rule.MinValue != nil && v < *rule.MinValue) ||
				(rule.MaxValue != nil && v > *rule.MaxValue)
			if outOfRange && (params.OverrideReason == nil || strings.TrimSpace(*params.OverrideReason) == "") {
				writeError(w, http.StatusBadRequest,
					fmt.Sprintf("result_value %.4g is outside validation range; override_reason is required to save", v))
				return
			}
		}
	}

	result, err := h.store.CreateSampleResult(r.Context(), params)
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

	claims := auth.UserFrom(r.Context())
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	before, err := h.store.GetSampleResult(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "sample result not found")
			return
		}
		slog.Error("get sample result", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	after, err := h.store.ReviewSampleResult(r.Context(), id, claims.UserID)
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

	claims := auth.UserFrom(r.Context())
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	before, err := h.store.GetSampleResult(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "sample result not found")
			return
		}
		slog.Error("get sample result", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	after, err := h.store.ApproveSampleResult(r.Context(), id, claims.UserID)
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
	if !checkFacilityAccess(w, r, facilityID) {
		return
	}

	results, err := h.store.EvaluateCompliance(r.Context(), facilityID)
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
	if !checkFacilityAccess(w, r, facilityID) {
		return
	}

	results, err := h.store.EvaluateCompliance(r.Context(), facilityID)
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
	if !checkFacilityAccess(w, r, facilityID) {
		return
	}

	results, err := h.store.EvaluateCompliance(r.Context(), facilityID)
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

	entries, err := h.store.ListAuditLog(r.Context(), recordID)
	if err != nil {
		slog.Error("list audit log", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, entries)
}

func (h *handler) getFacilityOverview(w http.ResponseWriter, r *http.Request) {
	facilityID, err := parseUUID(r.PathValue("facility_id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid facility_id")
		return
	}
	if !checkFacilityAccess(w, r, facilityID) {
		return
	}

	overview, err := h.store.GetFacilityOverview(r.Context(), facilityID)
	if err != nil {
		slog.Error("get facility overview", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if overview.SamplesByDay == nil {
		overview.SamplesByDay = []storage.SampleDayBucket{}
	}
	if overview.RecentResults == nil {
		overview.RecentResults = []storage.RecentSampleResult{}
	}
	writeJSON(w, http.StatusOK, overview)
}

func (h *handler) listInstrumentStatuses(w http.ResponseWriter, r *http.Request) {
	facilityID, err := parseUUID(r.PathValue("facility_id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid facility_id")
		return
	}
	if !checkFacilityAccess(w, r, facilityID) {
		return
	}

	statuses, err := h.store.ListInstrumentStatuses(r.Context(), facilityID)
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

	records, err := h.store.ListCalibrationRecords(r.Context(), instrumentID)
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
	if !checkFacilityAccess(w, r, facilityID) {
		return
	}

	days := 30
	if v := r.URL.Query().Get("days"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 365 {
			days = n
		}
	}

	series, err := h.store.GetTrendingData(r.Context(), facilityID, days)
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

	importer := ingestion.NewCSVImporter(h.store)

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

// ============================================================================
// Alerts
// ============================================================================

func (h *handler) listAlerts(w http.ResponseWriter, r *http.Request) {
	claims := auth.UserFrom(r.Context())
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	q := r.URL.Query()
	filter := storage.AlertFilter{}

	if v := q.Get("facility_id"); v != "" {
		id, err := parseUUID(v)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid facility_id")
			return
		}
		if !claims.HasFacilityAccess(id.String()) {
			writeError(w, http.StatusForbidden, "no access to this facility")
			return
		}
		filter.FacilityID = &id
	}
	if v := q.Get("type"); v != "" {
		if v != "exceedance" && v != "overdue_calibration" {
			writeError(w, http.StatusBadRequest, "invalid type")
			return
		}
		filter.Type = &v
	}
	if v := q.Get("dismissed"); v != "" {
		b, err := strconv.ParseBool(v)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid dismissed, expected true/false")
			return
		}
		filter.Dismissed = &b
	}
	if v := q.Get("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 {
			writeError(w, http.StatusBadRequest, "invalid limit")
			return
		}
		filter.Limit = n
	}

	alerts, err := h.store.ListAlerts(r.Context(), filter)
	if err != nil {
		slog.Error("list alerts", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	// If the caller didn't scope by facility, filter the response to facilities
	// the user has access to. Admins with org-wide roles see everything.
	if filter.FacilityID == nil {
		filtered := make([]storage.Alert, 0, len(alerts))
		for _, a := range alerts {
			if claims.HasFacilityAccess(a.FacilityID.String()) {
				filtered = append(filtered, a)
			}
		}
		alerts = filtered
	}

	writeJSON(w, http.StatusOK, alerts)
}

func (h *handler) dismissAlert(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	claims := auth.UserFrom(r.Context())
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	existing, err := h.store.GetAlert(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "alert not found")
			return
		}
		slog.Error("get alert", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if !claims.HasFacilityAccess(existing.FacilityID.String()) {
		writeError(w, http.StatusForbidden, "no access to this facility")
		return
	}

	after, err := h.store.DismissAlert(r.Context(), id, claims.UserID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusConflict, "alert is already dismissed")
			return
		}
		slog.Error("dismiss alert", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	h.publishAlertEvent(events.SubjectAlertDismissed, "update", after, &existing)
	writeJSON(w, http.StatusOK, after)
}

// publishAlertEvent sends an alert change event to the bus for audit logging.
func (h *handler) publishAlertEvent(subject, action string, after storage.Alert, before *storage.Alert) {
	if h.bus == nil {
		return
	}
	newJSON, _ := json.Marshal(after)
	event := events.ChangeEvent{
		Subject:        subject,
		Timestamp:      time.Now(),
		OrganizationID: after.OrganizationID,
		TableName:      "alerts",
		RecordID:       after.ID,
		Action:         action,
		ChangedBy:      uuid.Nil,
		NewValues:      newJSON,
	}
	if before != nil {
		oldJSON, _ := json.Marshal(*before)
		event.OldValues = oldJSON
	}
	if after.DismissedBy != nil {
		event.ChangedBy = *after.DismissedBy
	}
	if err := h.bus.Publish(event); err != nil {
		slog.Error("publish alert event", "error", err, "subject", subject)
	}
}

// publishResultEvent sends a change event for a sample result to NATS.
// Failures are logged but do not block the HTTP response.
func (h *handler) publishResultEvent(ctx context.Context, subject, action string, result storage.SampleResult, before *storage.SampleResult) {
	orgID, err := h.store.GetOrganizationIDForResult(ctx, result.ID)
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

	if h.bus == nil {
		return
	}
	if err := h.bus.Publish(event); err != nil {
		slog.Error("publish event", "error", err, "subject", subject)
	}
}

// checkFacilityAccess verifies the authenticated user has access to the given facility.
// Returns true if access is allowed, false if a 403 was written.
func checkFacilityAccess(w http.ResponseWriter, r *http.Request, facilityID uuid.UUID) bool {
	claims := auth.UserFrom(r.Context())
	if claims == nil || !claims.HasFacilityAccess(facilityID.String()) {
		writeError(w, http.StatusForbidden, "no access to this facility")
		return false
	}
	return true
}

// =========================================================================
// Attachments
// =========================================================================

func (h *handler) uploadAttachment(w http.ResponseWriter, r *http.Request) {
	resultID, err := parseUUID(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	claims := auth.UserFrom(r.Context())
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	sr, err := h.store.GetSampleResult(r.Context(), resultID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "sample result not found")
			return
		}
		slog.Error("get sample result", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	facilityID, err := h.store.GetFacilityIDForLocation(r.Context(), sr.MonitoringLocationID)
	if err != nil {
		slog.Error("resolve facility", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if !claims.HasFacilityAccess(facilityID.String()) {
		writeError(w, http.StatusForbidden, "no access to this facility")
		return
	}

	// Cap the whole request body so a runaway upload can't blow up memory.
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadBytes+(1<<10))
	if err := r.ParseMultipartForm(maxUploadBytes + (1 << 10)); err != nil {
		writeError(w, http.StatusBadRequest, "upload too large or malformed")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "file field is required")
		return
	}
	defer file.Close()

	if header.Size <= 0 {
		writeError(w, http.StatusBadRequest, "empty file")
		return
	}
	if header.Size > maxUploadBytes {
		writeError(w, http.StatusRequestEntityTooLarge, "file exceeds 10MB limit")
		return
	}

	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	if !allowedAttachmentTypes[contentType] {
		writeError(w, http.StatusUnsupportedMediaType, fmt.Sprintf("content type %q not allowed", contentType))
		return
	}

	key := uuid.NewString()
	if err := h.blobs.Put(r.Context(), key, contentType, header.Size, file); err != nil {
		slog.Error("blob put", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	att, err := h.store.CreateAttachment(r.Context(), storage.CreateAttachmentParams{
		OrganizationID: claims.OrganizationID,
		SubjectType:    "sample_result",
		SubjectID:      resultID,
		Filename:       header.Filename,
		ContentType:    contentType,
		SizeBytes:      header.Size,
		StorageKey:     key,
		UploadedBy:     claims.UserID,
	})
	if err != nil {
		// Best-effort: the blob was written but metadata failed, so delete it.
		_ = h.blobs.Delete(r.Context(), key)
		slog.Error("create attachment", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	h.publishAttachmentEvent(events.SubjectAttachmentCreated, "insert", att, nil, claims.UserID)
	writeJSON(w, http.StatusCreated, att)
}

func (h *handler) listAttachments(w http.ResponseWriter, r *http.Request) {
	resultID, err := parseUUID(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	claims := auth.UserFrom(r.Context())
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	sr, err := h.store.GetSampleResult(r.Context(), resultID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "sample result not found")
			return
		}
		slog.Error("get sample result", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	facilityID, err := h.store.GetFacilityIDForLocation(r.Context(), sr.MonitoringLocationID)
	if err != nil {
		slog.Error("resolve facility", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if !claims.HasFacilityAccess(facilityID.String()) {
		writeError(w, http.StatusForbidden, "no access to this facility")
		return
	}

	atts, err := h.store.ListAttachments(r.Context(), "sample_result", resultID)
	if err != nil {
		slog.Error("list attachments", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if atts == nil {
		atts = []storage.Attachment{}
	}
	writeJSON(w, http.StatusOK, atts)
}

func (h *handler) downloadAttachment(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	claims := auth.UserFrom(r.Context())
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	att, err := h.store.GetAttachment(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "attachment not found")
			return
		}
		slog.Error("get attachment", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if att.DeletedAt != nil {
		writeError(w, http.StatusNotFound, "attachment not found")
		return
	}

	if err := h.checkAttachmentAccess(r.Context(), claims, att); err != nil {
		writeError(w, http.StatusForbidden, "no access to this attachment")
		return
	}

	rc, err := h.blobs.Open(r.Context(), att.StorageKey)
	if err != nil {
		if errors.Is(err, blob.ErrNotFound) {
			writeError(w, http.StatusNotFound, "attachment data missing")
			return
		}
		slog.Error("blob open", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	defer rc.Close()

	w.Header().Set("Content-Type", att.ContentType)
	w.Header().Set("Content-Length", strconv.FormatInt(att.SizeBytes, 10))
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename=%q`, att.Filename))
	if _, err := io.Copy(w, rc); err != nil {
		// Response has already started; just log.
		slog.Warn("attachment stream", "error", err)
	}
}

func (h *handler) deleteAttachment(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	claims := auth.UserFrom(r.Context())
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	existing, err := h.store.GetAttachment(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "attachment not found")
			return
		}
		slog.Error("get attachment", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if existing.DeletedAt != nil {
		writeError(w, http.StatusConflict, "attachment already deleted")
		return
	}

	if err := h.checkAttachmentAccess(r.Context(), claims, existing); err != nil {
		writeError(w, http.StatusForbidden, "no access to this attachment")
		return
	}

	after, err := h.store.SoftDeleteAttachment(r.Context(), id, claims.UserID)
	if err != nil {
		slog.Error("soft delete attachment", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	h.publishAttachmentEvent(events.SubjectAttachmentDeleted, "delete", after, &existing, claims.UserID)
	writeJSON(w, http.StatusOK, after)
}

// checkAttachmentAccess verifies that the current user's roles permit access
// to the facility owning the attachment's subject.
func (h *handler) checkAttachmentAccess(ctx context.Context, claims *auth.Claims, att storage.Attachment) error {
	if att.SubjectType != "sample_result" {
		return fmt.Errorf("unsupported subject type")
	}
	sr, err := h.store.GetSampleResult(ctx, att.SubjectID)
	if err != nil {
		return err
	}
	facilityID, err := h.store.GetFacilityIDForLocation(ctx, sr.MonitoringLocationID)
	if err != nil {
		return err
	}
	if !claims.HasFacilityAccess(facilityID.String()) {
		return fmt.Errorf("forbidden")
	}
	return nil
}

func (h *handler) publishAttachmentEvent(subject, action string, after storage.Attachment, before *storage.Attachment, changedBy uuid.UUID) {
	if h.bus == nil {
		return
	}
	newJSON, _ := json.Marshal(after)
	event := events.ChangeEvent{
		Subject:        subject,
		Timestamp:      time.Now(),
		OrganizationID: after.OrganizationID,
		TableName:      "attachments",
		RecordID:       after.ID,
		Action:         action,
		ChangedBy:      changedBy,
		NewValues:      newJSON,
	}
	if before != nil {
		oldJSON, _ := json.Marshal(*before)
		event.OldValues = oldJSON
	}
	if err := h.bus.Publish(event); err != nil {
		slog.Error("publish attachment event", "error", err, "subject", subject)
	}
}

// =========================================================================
// Comments
// =========================================================================

func (h *handler) createComment(w http.ResponseWriter, r *http.Request) {
	resultID, err := parseUUID(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	claims := auth.UserFrom(r.Context())
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	var body struct {
		Body string `json:"body"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	body.Body = strings.TrimSpace(body.Body)
	if body.Body == "" {
		writeError(w, http.StatusBadRequest, "body is required")
		return
	}
	if len(body.Body) > 4000 {
		writeError(w, http.StatusBadRequest, "body exceeds 4000 characters")
		return
	}

	sr, err := h.store.GetSampleResult(r.Context(), resultID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "sample result not found")
			return
		}
		slog.Error("get sample result", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	facilityID, err := h.store.GetFacilityIDForLocation(r.Context(), sr.MonitoringLocationID)
	if err != nil {
		slog.Error("resolve facility", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if !claims.HasFacilityAccess(facilityID.String()) {
		writeError(w, http.StatusForbidden, "no access to this facility")
		return
	}

	c, err := h.store.CreateComment(r.Context(), storage.CreateCommentParams{
		OrganizationID: claims.OrganizationID,
		SubjectType:    "sample_result",
		SubjectID:      resultID,
		AuthorID:       claims.UserID,
		Body:           body.Body,
	})
	if err != nil {
		slog.Error("create comment", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	h.publishCommentEvent(c, claims.UserID)
	writeJSON(w, http.StatusCreated, c)
}

func (h *handler) listComments(w http.ResponseWriter, r *http.Request) {
	resultID, err := parseUUID(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	claims := auth.UserFrom(r.Context())
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	sr, err := h.store.GetSampleResult(r.Context(), resultID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "sample result not found")
			return
		}
		slog.Error("get sample result", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	facilityID, err := h.store.GetFacilityIDForLocation(r.Context(), sr.MonitoringLocationID)
	if err != nil {
		slog.Error("resolve facility", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if !claims.HasFacilityAccess(facilityID.String()) {
		writeError(w, http.StatusForbidden, "no access to this facility")
		return
	}

	comments, err := h.store.ListComments(r.Context(), "sample_result", resultID)
	if err != nil {
		slog.Error("list comments", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if comments == nil {
		comments = []storage.Comment{}
	}
	writeJSON(w, http.StatusOK, comments)
}

func (h *handler) publishCommentEvent(c storage.Comment, changedBy uuid.UUID) {
	if h.bus == nil {
		return
	}
	newJSON, _ := json.Marshal(c)
	event := events.ChangeEvent{
		Subject:        events.SubjectCommentCreated,
		Timestamp:      time.Now(),
		OrganizationID: c.OrganizationID,
		TableName:      "comments",
		RecordID:       c.ID,
		Action:         "insert",
		ChangedBy:      changedBy,
		NewValues:      newJSON,
	}
	if err := h.bus.Publish(event); err != nil {
		slog.Error("publish comment event", "error", err)
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
