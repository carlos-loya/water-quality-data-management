package ingestion

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/carlos-loya/water-quality-data-management/internal/storage"
)

// mockStore implements storage.Store for testing. Only the methods used by
// CSVImporter are implemented; the rest panic if called.
type mockStore struct {
	locations  []storage.MonitoringLocation
	parameters []storage.Parameter
	units      []storage.UnitOfMeasure

	// createFunc is called for each row insert. If nil, a default result is returned.
	createFunc func(ctx context.Context, p storage.CreateSampleResultParams) (storage.SampleResult, error)

	created []storage.CreateSampleResultParams // records every call
}

func (m *mockStore) ListAllMonitoringLocations(_ context.Context, _ uuid.UUID) ([]storage.MonitoringLocation, error) {
	return m.locations, nil
}
func (m *mockStore) ListParameters(_ context.Context, _ uuid.UUID) ([]storage.Parameter, error) {
	return m.parameters, nil
}
func (m *mockStore) ListUnits(_ context.Context, _ uuid.UUID) ([]storage.UnitOfMeasure, error) {
	return m.units, nil
}
func (m *mockStore) CreateSampleResult(ctx context.Context, p storage.CreateSampleResultParams) (storage.SampleResult, error) {
	m.created = append(m.created, p)
	if m.createFunc != nil {
		return m.createFunc(ctx, p)
	}
	return storage.SampleResult{
		ID:                   uuid.New(),
		MonitoringLocationID: p.MonitoringLocationID,
		ParameterID:          p.ParameterID,
		UnitID:               p.UnitID,
		CollectedAt:          p.CollectedAt,
		ResultValue:          p.ResultValue,
		ResultQualifier:      p.ResultQualifier,
		DetectionLimit:       p.DetectionLimit,
		Status:               "draft",
		EnteredBy:            p.EnteredBy,
		Source:               p.Source,
		Notes:                p.Notes,
	}, nil
}

// Unused methods — satisfy the interface.
func (m *mockStore) GetUserByEmail(context.Context, string) (storage.User, error) {
	panic("not implemented")
}
func (m *mockStore) GetUserRoles(context.Context, uuid.UUID) ([]storage.UserRole, error) {
	panic("not implemented")
}
func (m *mockStore) GetFacilityIDForLocation(context.Context, uuid.UUID) (uuid.UUID, error) {
	panic("not implemented")
}
func (m *mockStore) ListFacilitiesForUser(context.Context, uuid.UUID, uuid.UUID) ([]storage.Facility, error) {
	panic("not implemented")
}
func (m *mockStore) ListFacilities(context.Context, uuid.UUID) ([]storage.Facility, error) {
	panic("not implemented")
}
func (m *mockStore) ListMonitoringLocations(context.Context, uuid.UUID) ([]storage.MonitoringLocation, error) {
	panic("not implemented")
}
func (m *mockStore) ListValidationRules(context.Context, uuid.UUID) ([]storage.ValidationRule, error) {
	panic("not implemented")
}
func (m *mockStore) GetValidationRule(context.Context, uuid.UUID) (storage.ValidationRule, error) {
	panic("not implemented")
}
func (m *mockStore) ListSampleResults(context.Context, storage.SampleResultFilter) ([]storage.SampleResult, error) {
	panic("not implemented")
}
func (m *mockStore) GetSampleResult(context.Context, uuid.UUID) (storage.SampleResult, error) {
	panic("not implemented")
}
func (m *mockStore) ReviewSampleResult(context.Context, uuid.UUID, uuid.UUID) (storage.SampleResult, error) {
	panic("not implemented")
}
func (m *mockStore) ApproveSampleResult(context.Context, uuid.UUID, uuid.UUID) (storage.SampleResult, error) {
	panic("not implemented")
}
func (m *mockStore) EvaluateCompliance(context.Context, uuid.UUID) ([]storage.ComplianceResult, error) {
	panic("not implemented")
}
func (m *mockStore) GetTrendingData(context.Context, uuid.UUID, int) ([]storage.TrendingSeries, error) {
	panic("not implemented")
}
func (m *mockStore) ListInstruments(context.Context, uuid.UUID) ([]storage.Instrument, error) {
	panic("not implemented")
}
func (m *mockStore) ListCalibrationRecords(context.Context, uuid.UUID) ([]storage.CalibrationRecord, error) {
	panic("not implemented")
}
func (m *mockStore) ListInstrumentStatuses(context.Context, uuid.UUID) ([]storage.InstrumentStatus, error) {
	panic("not implemented")
}
func (m *mockStore) GetOrganizationIDForResult(context.Context, uuid.UUID) (uuid.UUID, error) {
	panic("not implemented")
}
func (m *mockStore) ListAuditLog(context.Context, uuid.UUID) ([]storage.AuditEntry, error) {
	panic("not implemented")
}
func (m *mockStore) ListAllFacilities(context.Context) ([]storage.Facility, error) {
	panic("not implemented")
}
func (m *mockStore) ListFacilityExceedances(context.Context, uuid.UUID) ([]storage.Exceedance, error) {
	panic("not implemented")
}
func (m *mockStore) CreateAlert(context.Context, storage.CreateAlertParams) (storage.Alert, bool, error) {
	panic("not implemented")
}
func (m *mockStore) ListAlerts(context.Context, storage.AlertFilter) ([]storage.Alert, error) {
	panic("not implemented")
}
func (m *mockStore) GetAlert(context.Context, uuid.UUID) (storage.Alert, error) {
	panic("not implemented")
}
func (m *mockStore) DismissAlert(context.Context, uuid.UUID, uuid.UUID) (storage.Alert, error) {
	panic("not implemented")
}
func (m *mockStore) CreateAttachment(context.Context, storage.CreateAttachmentParams) (storage.Attachment, error) {
	panic("not implemented")
}
func (m *mockStore) ListAttachments(context.Context, string, uuid.UUID) ([]storage.Attachment, error) {
	panic("not implemented")
}
func (m *mockStore) GetAttachment(context.Context, uuid.UUID) (storage.Attachment, error) {
	panic("not implemented")
}
func (m *mockStore) SoftDeleteAttachment(context.Context, uuid.UUID, uuid.UUID) (storage.Attachment, error) {
	panic("not implemented")
}
func (m *mockStore) CreateComment(context.Context, storage.CreateCommentParams) (storage.Comment, error) {
	panic("not implemented")
}
func (m *mockStore) ListComments(context.Context, string, uuid.UUID) ([]storage.Comment, error) {
	panic("not implemented")
}

// --- test fixtures ---

var (
	locID   = uuid.MustParse("00000000-0000-0000-0000-000000000001")
	paramID = uuid.MustParse("00000000-0000-0000-0000-000000000002")
	unitID  = uuid.MustParse("00000000-0000-0000-0000-000000000003")
	orgID   = uuid.MustParse("00000000-0000-0000-0000-000000000010")
	userID  = uuid.MustParse("00000000-0000-0000-0000-000000000020")
)

func newMock() *mockStore {
	return &mockStore{
		locations: []storage.MonitoringLocation{
			{ID: locID, Name: "Effluent"},
		},
		parameters: []storage.Parameter{
			{ID: paramID, Code: "PH"},
		},
		units: []storage.UnitOfMeasure{
			{ID: unitID, Code: "SU"},
		},
	}
}

func csvInput(lines ...string) string {
	return strings.Join(lines, "\n") + "\n"
}

// =========================================================================
// 1. Header Validation
// =========================================================================

func TestImport_AllRequiredColumnsPresent(t *testing.T) {
	m := newMock()
	imp := NewCSVImporter(m)
	input := csvInput(
		"monitoring_location,parameter_code,collected_at,result_value,unit_code",
		"Effluent,PH,2025-06-01,7.2,SU",
	)
	result, err := imp.Import(context.Background(), strings.NewReader(input), orgID, userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Imported != 1 {
		t.Errorf("expected 1 imported, got %d", result.Imported)
	}
}

func TestImport_MissingMonitoringLocationColumn(t *testing.T) {
	m := newMock()
	imp := NewCSVImporter(m)
	input := csvInput("parameter_code,collected_at,result_value,unit_code")
	_, err := imp.Import(context.Background(), strings.NewReader(input), orgID, userID)
	if err == nil || !strings.Contains(err.Error(), "monitoring_location") {
		t.Fatalf("expected error about missing monitoring_location, got: %v", err)
	}
}

func TestImport_MissingParameterCodeColumn(t *testing.T) {
	m := newMock()
	imp := NewCSVImporter(m)
	input := csvInput("monitoring_location,collected_at,result_value,unit_code")
	_, err := imp.Import(context.Background(), strings.NewReader(input), orgID, userID)
	if err == nil || !strings.Contains(err.Error(), "parameter_code") {
		t.Fatalf("expected error about missing parameter_code, got: %v", err)
	}
}

func TestImport_MissingCollectedAtColumn(t *testing.T) {
	m := newMock()
	imp := NewCSVImporter(m)
	input := csvInput("monitoring_location,parameter_code,result_value,unit_code")
	_, err := imp.Import(context.Background(), strings.NewReader(input), orgID, userID)
	if err == nil || !strings.Contains(err.Error(), "collected_at") {
		t.Fatalf("expected error about missing collected_at, got: %v", err)
	}
}

func TestImport_MissingResultValueColumn(t *testing.T) {
	m := newMock()
	imp := NewCSVImporter(m)
	input := csvInput("monitoring_location,parameter_code,collected_at,unit_code")
	_, err := imp.Import(context.Background(), strings.NewReader(input), orgID, userID)
	if err == nil || !strings.Contains(err.Error(), "result_value") {
		t.Fatalf("expected error about missing result_value, got: %v", err)
	}
}

func TestImport_MissingUnitCodeColumn(t *testing.T) {
	m := newMock()
	imp := NewCSVImporter(m)
	input := csvInput("monitoring_location,parameter_code,collected_at,result_value")
	_, err := imp.Import(context.Background(), strings.NewReader(input), orgID, userID)
	if err == nil || !strings.Contains(err.Error(), "unit_code") {
		t.Fatalf("expected error about missing unit_code, got: %v", err)
	}
}

func TestImport_ExtraColumnsIgnored(t *testing.T) {
	m := newMock()
	imp := NewCSVImporter(m)
	input := csvInput(
		"monitoring_location,parameter_code,collected_at,result_value,unit_code,extra_col,another",
		"Effluent,PH,2025-06-01,7.2,SU,foo,bar",
	)
	result, err := imp.Import(context.Background(), strings.NewReader(input), orgID, userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Imported != 1 {
		t.Errorf("expected 1 imported, got %d", result.Imported)
	}
}

func TestImport_ColumnNamesCaseInsensitive(t *testing.T) {
	m := newMock()
	imp := NewCSVImporter(m)
	input := csvInput(
		"Monitoring_Location,PARAMETER_CODE,Collected_At,Result_Value,Unit_Code",
		"Effluent,PH,2025-06-01,7.2,SU",
	)
	result, err := imp.Import(context.Background(), strings.NewReader(input), orgID, userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Imported != 1 {
		t.Errorf("expected 1 imported, got %d", result.Imported)
	}
}

func TestImport_ColumnNamesWithWhitespace(t *testing.T) {
	m := newMock()
	imp := NewCSVImporter(m)
	input := csvInput(
		" monitoring_location , parameter_code , collected_at , result_value , unit_code ",
		"Effluent,PH,2025-06-01,7.2,SU",
	)
	result, err := imp.Import(context.Background(), strings.NewReader(input), orgID, userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Imported != 1 {
		t.Errorf("expected 1 imported, got %d", result.Imported)
	}
}

// =========================================================================
// 2. Row Parsing — Monitoring Location
// =========================================================================

func TestImport_ValidLocationName(t *testing.T) {
	m := newMock()
	imp := NewCSVImporter(m)
	input := csvInput(
		"monitoring_location,parameter_code,collected_at,result_value,unit_code",
		"Effluent,PH,2025-06-01,7.2,SU",
	)
	result, err := imp.Import(context.Background(), strings.NewReader(input), orgID, userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(m.created) != 1 || m.created[0].MonitoringLocationID != locID {
		t.Error("location ID not resolved correctly")
	}
	if result.Imported != 1 {
		t.Errorf("expected 1 imported, got %d", result.Imported)
	}
}

func TestImport_UnknownLocationName(t *testing.T) {
	m := newMock()
	imp := NewCSVImporter(m)
	input := csvInput(
		"monitoring_location,parameter_code,collected_at,result_value,unit_code",
		"Nonexistent,PH,2025-06-01,7.2,SU",
	)
	result, err := imp.Import(context.Background(), strings.NewReader(input), orgID, userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Rejected != 1 {
		t.Errorf("expected 1 rejected, got %d", result.Rejected)
	}
	if !hasErrorField(result.Errors, "monitoring_location") {
		t.Error("expected error on monitoring_location field")
	}
}

func TestImport_EmptyLocationName(t *testing.T) {
	m := newMock()
	imp := NewCSVImporter(m)
	input := csvInput(
		"monitoring_location,parameter_code,collected_at,result_value,unit_code",
		",PH,2025-06-01,7.2,SU",
	)
	result, err := imp.Import(context.Background(), strings.NewReader(input), orgID, userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Rejected != 1 {
		t.Errorf("expected 1 rejected, got %d", result.Rejected)
	}
	if !hasErrorField(result.Errors, "monitoring_location") {
		t.Error("expected error on monitoring_location field")
	}
}

func TestImport_LocationNameCaseInsensitive(t *testing.T) {
	m := newMock()
	imp := NewCSVImporter(m)
	input := csvInput(
		"monitoring_location,parameter_code,collected_at,result_value,unit_code",
		"effluent,PH,2025-06-01,7.2,SU",
	)
	result, err := imp.Import(context.Background(), strings.NewReader(input), orgID, userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Imported != 1 {
		t.Errorf("expected 1 imported, got %d", result.Imported)
	}
}

// =========================================================================
// 3. Row Parsing — Parameter
// =========================================================================

func TestImport_ValidParameterCode(t *testing.T) {
	m := newMock()
	imp := NewCSVImporter(m)
	input := csvInput(
		"monitoring_location,parameter_code,collected_at,result_value,unit_code",
		"Effluent,PH,2025-06-01,7.2,SU",
	)
	result, err := imp.Import(context.Background(), strings.NewReader(input), orgID, userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(m.created) != 1 || m.created[0].ParameterID != paramID {
		t.Error("parameter ID not resolved correctly")
	}
	if result.Imported != 1 {
		t.Errorf("expected 1 imported, got %d", result.Imported)
	}
}

func TestImport_UnknownParameterCode(t *testing.T) {
	m := newMock()
	imp := NewCSVImporter(m)
	input := csvInput(
		"monitoring_location,parameter_code,collected_at,result_value,unit_code",
		"Effluent,UNKNOWN,2025-06-01,7.2,SU",
	)
	result, err := imp.Import(context.Background(), strings.NewReader(input), orgID, userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Rejected != 1 {
		t.Errorf("expected 1 rejected, got %d", result.Rejected)
	}
	if !hasErrorField(result.Errors, "parameter_code") {
		t.Error("expected error on parameter_code field")
	}
}

func TestImport_EmptyParameterCode(t *testing.T) {
	m := newMock()
	imp := NewCSVImporter(m)
	input := csvInput(
		"monitoring_location,parameter_code,collected_at,result_value,unit_code",
		"Effluent,,2025-06-01,7.2,SU",
	)
	result, err := imp.Import(context.Background(), strings.NewReader(input), orgID, userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Rejected != 1 {
		t.Errorf("expected 1 rejected, got %d", result.Rejected)
	}
	if !hasErrorField(result.Errors, "parameter_code") {
		t.Error("expected error on parameter_code field")
	}
}

func TestImport_ParameterCodeCaseInsensitive(t *testing.T) {
	m := newMock()
	imp := NewCSVImporter(m)
	input := csvInput(
		"monitoring_location,parameter_code,collected_at,result_value,unit_code",
		"Effluent,ph,2025-06-01,7.2,SU",
	)
	result, err := imp.Import(context.Background(), strings.NewReader(input), orgID, userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Imported != 1 {
		t.Errorf("expected 1 imported, got %d", result.Imported)
	}
}

// =========================================================================
// 4. Row Parsing — Unit
// =========================================================================

func TestImport_ValidUnitCode(t *testing.T) {
	m := newMock()
	imp := NewCSVImporter(m)
	input := csvInput(
		"monitoring_location,parameter_code,collected_at,result_value,unit_code",
		"Effluent,PH,2025-06-01,7.2,SU",
	)
	result, err := imp.Import(context.Background(), strings.NewReader(input), orgID, userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(m.created) != 1 || m.created[0].UnitID != unitID {
		t.Error("unit ID not resolved correctly")
	}
	if result.Imported != 1 {
		t.Errorf("expected 1 imported, got %d", result.Imported)
	}
}

func TestImport_UnknownUnitCode(t *testing.T) {
	m := newMock()
	imp := NewCSVImporter(m)
	input := csvInput(
		"monitoring_location,parameter_code,collected_at,result_value,unit_code",
		"Effluent,PH,2025-06-01,7.2,UNKNOWN",
	)
	result, err := imp.Import(context.Background(), strings.NewReader(input), orgID, userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Rejected != 1 {
		t.Errorf("expected 1 rejected, got %d", result.Rejected)
	}
	if !hasErrorField(result.Errors, "unit_code") {
		t.Error("expected error on unit_code field")
	}
}

func TestImport_EmptyUnitCode(t *testing.T) {
	m := newMock()
	imp := NewCSVImporter(m)
	input := csvInput(
		"monitoring_location,parameter_code,collected_at,result_value,unit_code",
		"Effluent,PH,2025-06-01,7.2,",
	)
	result, err := imp.Import(context.Background(), strings.NewReader(input), orgID, userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Rejected != 1 {
		t.Errorf("expected 1 rejected, got %d", result.Rejected)
	}
	if !hasErrorField(result.Errors, "unit_code") {
		t.Error("expected error on unit_code field")
	}
}

func TestImport_UnitCodeCaseInsensitive(t *testing.T) {
	m := newMock()
	imp := NewCSVImporter(m)
	input := csvInput(
		"monitoring_location,parameter_code,collected_at,result_value,unit_code",
		"Effluent,PH,2025-06-01,7.2,su",
	)
	result, err := imp.Import(context.Background(), strings.NewReader(input), orgID, userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Imported != 1 {
		t.Errorf("expected 1 imported, got %d", result.Imported)
	}
}

// =========================================================================
// 5. Row Parsing — Timestamp
// =========================================================================

func TestImport_TimestampFormats(t *testing.T) {
	tests := []struct {
		name  string
		value string
	}{
		{"RFC3339", "2025-06-01T10:30:00Z"},
		{"RFC3339 with offset", "2025-06-01T10:30:00-05:00"},
		{"datetime space", "2025-06-01 10:30:00"},
		{"datetime no seconds", "2025-06-01 10:30"},
		{"date only", "2025-06-01"},
		{"US datetime", "06/01/2025 10:30"},
		{"US date only", "06/01/2025"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := newMock()
			imp := NewCSVImporter(m)
			input := csvInput(
				"monitoring_location,parameter_code,collected_at,result_value,unit_code",
				fmt.Sprintf("Effluent,PH,%s,7.2,SU", tc.value),
			)
			result, err := imp.Import(context.Background(), strings.NewReader(input), orgID, userID)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result.Imported != 1 {
				t.Errorf("expected 1 imported, got %d (errors: %v)", result.Imported, result.Errors)
			}
		})
	}
}

func TestImport_InvalidTimestamp(t *testing.T) {
	m := newMock()
	imp := NewCSVImporter(m)
	input := csvInput(
		"monitoring_location,parameter_code,collected_at,result_value,unit_code",
		"Effluent,PH,not-a-date,7.2,SU",
	)
	result, err := imp.Import(context.Background(), strings.NewReader(input), orgID, userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Rejected != 1 {
		t.Errorf("expected 1 rejected, got %d", result.Rejected)
	}
	if !hasErrorField(result.Errors, "collected_at") {
		t.Error("expected error on collected_at field")
	}
}

func TestImport_EmptyTimestamp(t *testing.T) {
	m := newMock()
	imp := NewCSVImporter(m)
	input := csvInput(
		"monitoring_location,parameter_code,collected_at,result_value,unit_code",
		"Effluent,PH,,7.2,SU",
	)
	result, err := imp.Import(context.Background(), strings.NewReader(input), orgID, userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Rejected != 1 {
		t.Errorf("expected 1 rejected, got %d", result.Rejected)
	}
	if !hasErrorField(result.Errors, "collected_at") {
		t.Error("expected error on collected_at field")
	}
}

// =========================================================================
// 6. Row Parsing — Result Value
// =========================================================================

func TestImport_NumericValue(t *testing.T) {
	m := newMock()
	imp := NewCSVImporter(m)
	input := csvInput(
		"monitoring_location,parameter_code,collected_at,result_value,unit_code",
		"Effluent,PH,2025-06-01,7.2,SU",
	)
	result, err := imp.Import(context.Background(), strings.NewReader(input), orgID, userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Imported != 1 {
		t.Errorf("expected 1 imported, got %d", result.Imported)
	}
	if m.created[0].ResultValue == nil || *m.created[0].ResultValue != 7.2 {
		t.Errorf("expected result_value 7.2, got %v", m.created[0].ResultValue)
	}
	if m.created[0].ResultQualifier != nil {
		t.Errorf("expected nil qualifier, got %v", *m.created[0].ResultQualifier)
	}
}

func TestImport_NonDetectBelowDetectionLimit(t *testing.T) {
	m := newMock()
	imp := NewCSVImporter(m)
	input := csvInput(
		"monitoring_location,parameter_code,collected_at,result_value,unit_code",
		"Effluent,PH,2025-06-01,<0.1,SU",
	)
	result, err := imp.Import(context.Background(), strings.NewReader(input), orgID, userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Imported != 1 {
		t.Errorf("expected 1 imported, got %d", result.Imported)
	}
	p := m.created[0]
	if p.ResultQualifier == nil || *p.ResultQualifier != "<" {
		t.Errorf("expected qualifier '<', got %v", p.ResultQualifier)
	}
	if p.DetectionLimit == nil || *p.DetectionLimit != 0.1 {
		t.Errorf("expected detection_limit 0.1, got %v", p.DetectionLimit)
	}
	if p.ResultValue != nil {
		t.Errorf("expected nil result_value, got %v", *p.ResultValue)
	}
}

func TestImport_ND(t *testing.T) {
	m := newMock()
	imp := NewCSVImporter(m)
	input := csvInput(
		"monitoring_location,parameter_code,collected_at,result_value,unit_code",
		"Effluent,PH,2025-06-01,ND,SU",
	)
	result, err := imp.Import(context.Background(), strings.NewReader(input), orgID, userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Imported != 1 {
		t.Errorf("expected 1 imported, got %d", result.Imported)
	}
	p := m.created[0]
	if p.ResultQualifier == nil || *p.ResultQualifier != "ND" {
		t.Errorf("expected qualifier 'ND', got %v", p.ResultQualifier)
	}
	if p.DetectionLimit != nil {
		t.Errorf("expected nil detection_limit for ND, got %v", *p.DetectionLimit)
	}
}

func TestImport_LessThanWithNonNumericSuffix(t *testing.T) {
	m := newMock()
	imp := NewCSVImporter(m)
	input := csvInput(
		"monitoring_location,parameter_code,collected_at,result_value,unit_code",
		"Effluent,PH,2025-06-01,<LOD,SU",
	)
	result, err := imp.Import(context.Background(), strings.NewReader(input), orgID, userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Imported != 1 {
		t.Errorf("expected 1 imported, got %d", result.Imported)
	}
	p := m.created[0]
	if p.ResultQualifier == nil || *p.ResultQualifier != "<" {
		t.Errorf("expected qualifier '<', got %v", p.ResultQualifier)
	}
	if p.DetectionLimit != nil {
		t.Errorf("expected nil detection_limit for '<LOD', got %v", *p.DetectionLimit)
	}
}

func TestImport_EmptyResultValue(t *testing.T) {
	m := newMock()
	imp := NewCSVImporter(m)
	input := csvInput(
		"monitoring_location,parameter_code,collected_at,result_value,unit_code",
		"Effluent,PH,2025-06-01,,SU",
	)
	result, err := imp.Import(context.Background(), strings.NewReader(input), orgID, userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Rejected != 1 {
		t.Errorf("expected 1 rejected, got %d", result.Rejected)
	}
	if !hasErrorField(result.Errors, "result_value") {
		t.Error("expected error on result_value field")
	}
}

func TestImport_NonNumericNonQualifierText(t *testing.T) {
	m := newMock()
	imp := NewCSVImporter(m)
	input := csvInput(
		"monitoring_location,parameter_code,collected_at,result_value,unit_code",
		"Effluent,PH,2025-06-01,abc,SU",
	)
	result, err := imp.Import(context.Background(), strings.NewReader(input), orgID, userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Rejected != 1 {
		t.Errorf("expected 1 rejected, got %d", result.Rejected)
	}
	if !hasErrorField(result.Errors, "result_value") {
		t.Error("expected error on result_value field")
	}
}

func TestImport_NegativeValue(t *testing.T) {
	m := newMock()
	imp := NewCSVImporter(m)
	input := csvInput(
		"monitoring_location,parameter_code,collected_at,result_value,unit_code",
		"Effluent,PH,2025-06-01,-5.3,SU",
	)
	result, err := imp.Import(context.Background(), strings.NewReader(input), orgID, userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Imported != 1 {
		t.Errorf("expected 1 imported, got %d", result.Imported)
	}
	if m.created[0].ResultValue == nil || *m.created[0].ResultValue != -5.3 {
		t.Errorf("expected -5.3, got %v", m.created[0].ResultValue)
	}
}

func TestImport_ZeroValue(t *testing.T) {
	m := newMock()
	imp := NewCSVImporter(m)
	input := csvInput(
		"monitoring_location,parameter_code,collected_at,result_value,unit_code",
		"Effluent,PH,2025-06-01,0,SU",
	)
	result, err := imp.Import(context.Background(), strings.NewReader(input), orgID, userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Imported != 1 {
		t.Errorf("expected 1 imported, got %d", result.Imported)
	}
	if m.created[0].ResultValue == nil || *m.created[0].ResultValue != 0 {
		t.Errorf("expected 0, got %v", m.created[0].ResultValue)
	}
}

func TestImport_VeryLargeNumber(t *testing.T) {
	m := newMock()
	imp := NewCSVImporter(m)
	input := csvInput(
		"monitoring_location,parameter_code,collected_at,result_value,unit_code",
		"Effluent,PH,2025-06-01,9999999.999,SU",
	)
	result, err := imp.Import(context.Background(), strings.NewReader(input), orgID, userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Imported != 1 {
		t.Errorf("expected 1 imported, got %d", result.Imported)
	}
}

// =========================================================================
// 7. Row Parsing — Notes
// =========================================================================

func TestImport_NotesColumnPresent(t *testing.T) {
	m := newMock()
	imp := NewCSVImporter(m)
	input := csvInput(
		"monitoring_location,parameter_code,collected_at,result_value,unit_code,notes",
		"Effluent,PH,2025-06-01,7.2,SU,Sample looked clear",
	)
	result, err := imp.Import(context.Background(), strings.NewReader(input), orgID, userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Imported != 1 {
		t.Errorf("expected 1 imported, got %d", result.Imported)
	}
	if m.created[0].Notes == nil || *m.created[0].Notes != "Sample looked clear" {
		t.Errorf("expected notes 'Sample looked clear', got %v", m.created[0].Notes)
	}
}

func TestImport_NotesColumnEmpty(t *testing.T) {
	m := newMock()
	imp := NewCSVImporter(m)
	input := csvInput(
		"monitoring_location,parameter_code,collected_at,result_value,unit_code,notes",
		"Effluent,PH,2025-06-01,7.2,SU,",
	)
	result, err := imp.Import(context.Background(), strings.NewReader(input), orgID, userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Imported != 1 {
		t.Errorf("expected 1 imported, got %d", result.Imported)
	}
	if m.created[0].Notes != nil {
		t.Errorf("expected nil notes for empty string, got %v", *m.created[0].Notes)
	}
}

func TestImport_NotesColumnAbsent(t *testing.T) {
	m := newMock()
	imp := NewCSVImporter(m)
	input := csvInput(
		"monitoring_location,parameter_code,collected_at,result_value,unit_code",
		"Effluent,PH,2025-06-01,7.2,SU",
	)
	result, err := imp.Import(context.Background(), strings.NewReader(input), orgID, userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Imported != 1 {
		t.Errorf("expected 1 imported, got %d", result.Imported)
	}
	if m.created[0].Notes != nil {
		t.Errorf("expected nil notes when column absent, got %v", *m.created[0].Notes)
	}
}

// =========================================================================
// 8. End-to-End Import
// =========================================================================

func TestImport_AllRowsValid(t *testing.T) {
	m := newMock()
	imp := NewCSVImporter(m)
	input := csvInput(
		"monitoring_location,parameter_code,collected_at,result_value,unit_code",
		"Effluent,PH,2025-06-01,7.2,SU",
		"Effluent,PH,2025-06-02,7.4,SU",
		"Effluent,PH,2025-06-03,7.1,SU",
	)
	result, err := imp.Import(context.Background(), strings.NewReader(input), orgID, userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.TotalRows != 3 {
		t.Errorf("expected 3 total, got %d", result.TotalRows)
	}
	if result.Imported != 3 {
		t.Errorf("expected 3 imported, got %d", result.Imported)
	}
	if result.Rejected != 0 {
		t.Errorf("expected 0 rejected, got %d", result.Rejected)
	}
	if len(result.Errors) != 0 {
		t.Errorf("expected 0 errors, got %d", len(result.Errors))
	}
}

func TestImport_AllRowsInvalid(t *testing.T) {
	m := newMock()
	imp := NewCSVImporter(m)
	input := csvInput(
		"monitoring_location,parameter_code,collected_at,result_value,unit_code",
		"Bad,PH,2025-06-01,7.2,SU",
		"Effluent,BAD,2025-06-02,7.4,SU",
		"Effluent,PH,bad-date,7.1,SU",
	)
	result, err := imp.Import(context.Background(), strings.NewReader(input), orgID, userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Imported != 0 {
		t.Errorf("expected 0 imported, got %d", result.Imported)
	}
	if result.Rejected != 3 {
		t.Errorf("expected 3 rejected, got %d", result.Rejected)
	}
}

func TestImport_MixedValidAndInvalid(t *testing.T) {
	m := newMock()
	imp := NewCSVImporter(m)
	input := csvInput(
		"monitoring_location,parameter_code,collected_at,result_value,unit_code",
		"Effluent,PH,2025-06-01,7.2,SU",
		"Bad,PH,2025-06-02,7.4,SU",
		"Effluent,PH,2025-06-03,7.1,SU",
	)
	result, err := imp.Import(context.Background(), strings.NewReader(input), orgID, userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.TotalRows != 3 {
		t.Errorf("expected 3 total, got %d", result.TotalRows)
	}
	if result.Imported != 2 {
		t.Errorf("expected 2 imported, got %d", result.Imported)
	}
	if result.Rejected != 1 {
		t.Errorf("expected 1 rejected, got %d", result.Rejected)
	}
	if len(result.Errors) != 1 {
		t.Errorf("expected 1 error, got %d", len(result.Errors))
	}
}

func TestImport_SourceIsCSVImport(t *testing.T) {
	m := newMock()
	imp := NewCSVImporter(m)
	input := csvInput(
		"monitoring_location,parameter_code,collected_at,result_value,unit_code",
		"Effluent,PH,2025-06-01,7.2,SU",
	)
	_, err := imp.Import(context.Background(), strings.NewReader(input), orgID, userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(m.created) != 1 {
		t.Fatalf("expected 1 created, got %d", len(m.created))
	}
	if m.created[0].Source != "csv_import" {
		t.Errorf("expected source 'csv_import', got %q", m.created[0].Source)
	}
}

func TestImport_EmptyCSVHeaderOnly(t *testing.T) {
	m := newMock()
	imp := NewCSVImporter(m)
	input := csvInput("monitoring_location,parameter_code,collected_at,result_value,unit_code")
	result, err := imp.Import(context.Background(), strings.NewReader(input), orgID, userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.TotalRows != 0 {
		t.Errorf("expected 0 total, got %d", result.TotalRows)
	}
	if result.Imported != 0 {
		t.Errorf("expected 0 imported, got %d", result.Imported)
	}
}

func TestImport_DBInsertFailure(t *testing.T) {
	m := newMock()
	m.createFunc = func(_ context.Context, _ storage.CreateSampleResultParams) (storage.SampleResult, error) {
		return storage.SampleResult{}, fmt.Errorf("connection refused")
	}
	imp := NewCSVImporter(m)
	input := csvInput(
		"monitoring_location,parameter_code,collected_at,result_value,unit_code",
		"Effluent,PH,2025-06-01,7.2,SU",
		"Effluent,PH,2025-06-02,7.4,SU",
	)
	result, err := imp.Import(context.Background(), strings.NewReader(input), orgID, userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Imported != 0 {
		t.Errorf("expected 0 imported, got %d", result.Imported)
	}
	if result.Rejected != 2 {
		t.Errorf("expected 2 rejected, got %d", result.Rejected)
	}
	for _, e := range result.Errors {
		if !strings.Contains(e.Detail, "database insert") {
			t.Errorf("expected 'database insert' in error detail, got %q", e.Detail)
		}
	}
}

func TestImport_RowWithFewerColumnsThanHeader(t *testing.T) {
	m := newMock()
	imp := NewCSVImporter(m)
	// Row has only 3 fields, header has 5 → Go csv.Reader returns an error
	input := csvInput(
		"monitoring_location,parameter_code,collected_at,result_value,unit_code",
		"Effluent,PH,2025-06-01",
	)
	_, err := imp.Import(context.Background(), strings.NewReader(input), orgID, userID)
	if err == nil {
		t.Fatal("expected error for mismatched field count")
	}
	if !strings.Contains(err.Error(), "wrong number of fields") {
		t.Errorf("expected 'wrong number of fields' error, got: %v", err)
	}
}

func TestImport_EnteredByPreserved(t *testing.T) {
	m := newMock()
	imp := NewCSVImporter(m)
	input := csvInput(
		"monitoring_location,parameter_code,collected_at,result_value,unit_code",
		"Effluent,PH,2025-06-01,7.2,SU",
	)
	_, err := imp.Import(context.Background(), strings.NewReader(input), orgID, userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.created[0].EnteredBy != userID {
		t.Errorf("expected entered_by %s, got %s", userID, m.created[0].EnteredBy)
	}
}

func TestImport_MultipleErrorsPerRow(t *testing.T) {
	m := newMock()
	imp := NewCSVImporter(m)
	// Row has bad location, bad parameter, bad unit, bad date, and empty value — all fail
	input := csvInput(
		"monitoring_location,parameter_code,collected_at,result_value,unit_code",
		"Bad,BAD,bad-date,,BADUNIT",
	)
	result, err := imp.Import(context.Background(), strings.NewReader(input), orgID, userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Rejected != 1 {
		t.Errorf("expected 1 rejected, got %d", result.Rejected)
	}
	// Should have errors for all 5 fields
	if len(result.Errors) < 5 {
		t.Errorf("expected at least 5 errors, got %d: %v", len(result.Errors), result.Errors)
	}
}

// =========================================================================
// 9. Helper Functions
// =========================================================================

func TestBuildColumnIndex_NormalizesToLowercase(t *testing.T) {
	idx := buildColumnIndex([]string{"FOO", "Bar", "baz"})
	for _, key := range []string{"foo", "bar", "baz"} {
		if _, ok := idx[key]; !ok {
			t.Errorf("expected key %q in index", key)
		}
	}
}

func TestBuildColumnIndex_TrimsWhitespace(t *testing.T) {
	idx := buildColumnIndex([]string{" foo ", "\tbar\t"})
	if _, ok := idx["foo"]; !ok {
		t.Error("expected 'foo' after trimming")
	}
	if _, ok := idx["bar"]; !ok {
		t.Error("expected 'bar' after trimming")
	}
}

func TestGetCol_OutOfBoundsIndex(t *testing.T) {
	record := []string{"a", "b"}
	colIndex := map[string]int{"c": 5}
	result := getCol(record, colIndex, "c")
	if result != "" {
		t.Errorf("expected empty string for out-of-bounds, got %q", result)
	}
}

func TestGetCol_MissingColumnName(t *testing.T) {
	record := []string{"a", "b"}
	colIndex := map[string]int{"x": 0}
	result := getCol(record, colIndex, "missing")
	if result != "" {
		t.Errorf("expected empty string for missing column, got %q", result)
	}
}

func TestParseTimestamp_AllFormats(t *testing.T) {
	tests := []struct {
		input    string
		expected time.Time
	}{
		{"2025-06-01T10:30:00Z", time.Date(2025, 6, 1, 10, 30, 0, 0, time.UTC)},
		{"2025-06-01 10:30:00", time.Date(2025, 6, 1, 10, 30, 0, 0, time.UTC)},
		{"2025-06-01 10:30", time.Date(2025, 6, 1, 10, 30, 0, 0, time.UTC)},
		{"2025-06-01", time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)},
		{"06/01/2025 10:30", time.Date(2025, 6, 1, 10, 30, 0, 0, time.UTC)},
		{"06/01/2025", time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)},
	}
	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			got, err := parseTimestamp(tc.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !got.Equal(tc.expected) {
				t.Errorf("expected %v, got %v", tc.expected, got)
			}
		})
	}
}

func TestParseTimestamp_UnrecognizedFormat(t *testing.T) {
	_, err := parseTimestamp("June 1st, 2025")
	if err == nil {
		t.Error("expected error for unrecognized format")
	}
}

func TestParseTimestamp_EmptyString(t *testing.T) {
	_, err := parseTimestamp("")
	if err == nil {
		t.Error("expected error for empty string")
	}
}

// --- test helpers ---

func hasErrorField(errors []RowError, field string) bool {
	for _, e := range errors {
		if e.Field == field {
			return true
		}
	}
	return false
}
