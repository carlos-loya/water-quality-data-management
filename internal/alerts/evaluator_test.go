package alerts

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/carlos-loya/water-quality-data-management/internal/events"
	"github.com/carlos-loya/water-quality-data-management/internal/storage"
)

// ============================================================================
// fakeStore — implements alerts.Store for evaluator tests.
// ============================================================================

type fakeStore struct {
	mu sync.Mutex

	facilities    []storage.Facility
	exceedances   map[uuid.UUID][]storage.Exceedance
	instruments   map[uuid.UUID][]storage.InstrumentStatus
	existingByKey map[string]bool // dedupe key -> exists

	createCalls []storage.CreateAlertParams
	listFacErr  error
	exErr       map[uuid.UUID]error
	instErr     map[uuid.UUID]error
	createErr   error
}

func newFakeStore() *fakeStore {
	return &fakeStore{
		exceedances:   make(map[uuid.UUID][]storage.Exceedance),
		instruments:   make(map[uuid.UUID][]storage.InstrumentStatus),
		existingByKey: make(map[string]bool),
		exErr:         make(map[uuid.UUID]error),
		instErr:       make(map[uuid.UUID]error),
	}
}

func (s *fakeStore) ListAllFacilities(_ context.Context) ([]storage.Facility, error) {
	return s.facilities, s.listFacErr
}

func (s *fakeStore) ListFacilityExceedances(_ context.Context, facilityID uuid.UUID) ([]storage.Exceedance, error) {
	if err, ok := s.exErr[facilityID]; ok {
		return nil, err
	}
	return s.exceedances[facilityID], nil
}

func (s *fakeStore) ListInstrumentStatuses(_ context.Context, facilityID uuid.UUID) ([]storage.InstrumentStatus, error) {
	if err, ok := s.instErr[facilityID]; ok {
		return nil, err
	}
	return s.instruments[facilityID], nil
}

func (s *fakeStore) CreateAlert(_ context.Context, p storage.CreateAlertParams) (storage.Alert, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.createErr != nil {
		return storage.Alert{}, false, s.createErr
	}

	s.createCalls = append(s.createCalls, p)

	key := dedupeKey(p)
	if s.existingByKey[key] {
		return storage.Alert{}, false, nil // duplicate; not inserted
	}
	s.existingByKey[key] = true

	return storage.Alert{
		ID:             uuid.New(),
		OrganizationID: p.OrganizationID,
		FacilityID:     p.FacilityID,
		Type:           p.Type,
		Severity:       p.Severity,
		SubjectType:    p.SubjectType,
		SubjectID:      p.SubjectID,
		Message:        p.Message,
		Details:        p.Details,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}, true, nil
}

func dedupeKey(p storage.CreateAlertParams) string {
	return p.FacilityID.String() + "|" + p.Type + "|" + p.SubjectType + "|" + p.SubjectID.String()
}

// ============================================================================
// fakePublisher
// ============================================================================

type fakePublisher struct {
	mu        sync.Mutex
	published []events.ChangeEvent
	err       error
}

func (p *fakePublisher) Publish(e events.ChangeEvent) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.published = append(p.published, e)
	return p.err
}

// ============================================================================
// Fixtures
// ============================================================================

func facility(id, orgID uuid.UUID, name string) storage.Facility {
	return storage.Facility{
		ID:             id,
		OrganizationID: orgID,
		Name:           name,
		FacilityType:   "water_treatment",
		Active:         true,
	}
}

func exceedance(facilityID, orgID uuid.UUID, resultValue, limitValue float64, limitType string) storage.Exceedance {
	return storage.Exceedance{
		SampleResultID:       uuid.New(),
		MonitoringLocationID: uuid.New(),
		FacilityID:           facilityID,
		OrganizationID:       orgID,
		LocationName:         "Effluent",
		ParameterCode:        "PH",
		ParameterName:        "pH",
		ResultValue:          resultValue,
		UnitCode:             "SU",
		LimitType:            limitType,
		LimitValue:           limitValue,
		CollectedAt:          time.Now().Add(-24 * time.Hour),
	}
}

func instrumentStatus(name, calStatus string, dueAt *time.Time) storage.InstrumentStatus {
	return storage.InstrumentStatus{
		ID:                uuid.New(),
		Name:              name,
		InstrumentType:    "benchtop",
		Active:            true,
		DueAt:             dueAt,
		CalibrationStatus: calStatus,
	}
}

func ptrTime(t time.Time) *time.Time { return &t }

// ============================================================================
// Tests
// ============================================================================

func TestRunOnce_NoFacilities_NoAlerts(t *testing.T) {
	s := newFakeStore()
	p := &fakePublisher{}
	e := NewEvaluator(s, p, time.Minute)

	n, err := e.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != 0 {
		t.Fatalf("expected 0 alerts, got %d", n)
	}
	if len(p.published) != 0 {
		t.Fatalf("expected 0 published events, got %d", len(p.published))
	}
}

func TestRunOnce_ExceedanceCreatesAlert(t *testing.T) {
	s := newFakeStore()
	facID := uuid.New()
	orgID := uuid.New()
	s.facilities = []storage.Facility{facility(facID, orgID, "North WTP")}
	s.exceedances[facID] = []storage.Exceedance{
		exceedance(facID, orgID, 9.2, 8.5, "daily_max"),
	}

	p := &fakePublisher{}
	e := NewEvaluator(s, p, time.Minute)

	n, err := e.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != 1 {
		t.Fatalf("expected 1 alert, got %d", n)
	}

	if len(s.createCalls) != 1 {
		t.Fatalf("expected 1 CreateAlert call, got %d", len(s.createCalls))
	}
	call := s.createCalls[0]
	if call.Type != "exceedance" {
		t.Errorf("expected type=exceedance, got %q", call.Type)
	}
	if call.Severity != "critical" {
		t.Errorf("expected severity=critical, got %q", call.Severity)
	}
	if call.SubjectType != "sample_result" {
		t.Errorf("expected subject_type=sample_result, got %q", call.SubjectType)
	}
	if call.FacilityID != facID {
		t.Errorf("facility_id mismatch")
	}
	if call.OrganizationID != orgID {
		t.Errorf("organization_id mismatch")
	}
	if call.Message == "" {
		t.Errorf("expected non-empty message")
	}

	// Event published
	if len(p.published) != 1 {
		t.Fatalf("expected 1 published event, got %d", len(p.published))
	}
	if p.published[0].Subject != events.SubjectAlertCreated {
		t.Errorf("expected subject=%s, got %s", events.SubjectAlertCreated, p.published[0].Subject)
	}
}

func TestRunOnce_Dedupe_SkipsActiveDuplicate(t *testing.T) {
	s := newFakeStore()
	facID := uuid.New()
	orgID := uuid.New()
	s.facilities = []storage.Facility{facility(facID, orgID, "North WTP")}

	// Same exceedance on two consecutive ticks.
	ex := exceedance(facID, orgID, 9.2, 8.5, "daily_max")
	s.exceedances[facID] = []storage.Exceedance{ex}

	p := &fakePublisher{}
	e := NewEvaluator(s, p, time.Minute)

	n1, _ := e.RunOnce(context.Background())
	n2, _ := e.RunOnce(context.Background())

	if n1 != 1 {
		t.Errorf("first run: expected 1 new alert, got %d", n1)
	}
	if n2 != 0 {
		t.Errorf("second run: expected 0 new alerts (dedupe), got %d", n2)
	}
	// CreateAlert was called twice (evaluator always tries; storage dedupes).
	if len(s.createCalls) != 2 {
		t.Errorf("expected 2 CreateAlert calls, got %d", len(s.createCalls))
	}
	// But only one event published (dedupe suppresses event on second try).
	if len(p.published) != 1 {
		t.Errorf("expected 1 event, got %d", len(p.published))
	}
}

func TestRunOnce_OverdueCalibration_SeverityByDaysOverdue(t *testing.T) {
	cases := []struct {
		name         string
		overdueDays  int
		wantSeverity string
	}{
		{"1 day overdue", 1, "warning"},
		{"6 days overdue", 6, "warning"},
		{"7 days overdue", 7, "critical"},
		{"30 days overdue", 30, "critical"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := newFakeStore()
			facID := uuid.New()
			orgID := uuid.New()
			s.facilities = []storage.Facility{facility(facID, orgID, "South WWTP")}
			dueAt := time.Now().Add(-time.Duration(tc.overdueDays) * 24 * time.Hour)
			s.instruments[facID] = []storage.InstrumentStatus{
				instrumentStatus("pH meter", "overdue", &dueAt),
			}

			e := NewEvaluator(s, &fakePublisher{}, time.Minute)
			_, err := e.RunOnce(context.Background())
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(s.createCalls) != 1 {
				t.Fatalf("expected 1 call, got %d", len(s.createCalls))
			}
			if got := s.createCalls[0].Severity; got != tc.wantSeverity {
				t.Errorf("severity: expected %q, got %q", tc.wantSeverity, got)
			}
			if s.createCalls[0].Type != "overdue_calibration" {
				t.Errorf("type: expected overdue_calibration, got %q", s.createCalls[0].Type)
			}
		})
	}
}

func TestRunOnce_NonOverdueStatuses_NoAlert(t *testing.T) {
	s := newFakeStore()
	facID := uuid.New()
	orgID := uuid.New()
	s.facilities = []storage.Facility{facility(facID, orgID, "Plant")}
	s.instruments[facID] = []storage.InstrumentStatus{
		instrumentStatus("meter-1", "current", ptrTime(time.Now().Add(30*24*time.Hour))),
		instrumentStatus("meter-2", "due_soon", ptrTime(time.Now().Add(24*time.Hour))),
		instrumentStatus("meter-3", "no_schedule", nil),
	}

	e := NewEvaluator(s, &fakePublisher{}, time.Minute)
	n, err := e.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != 0 {
		t.Fatalf("expected 0 alerts, got %d", n)
	}
	if len(s.createCalls) != 0 {
		t.Fatalf("expected 0 CreateAlert calls, got %d", len(s.createCalls))
	}
}

func TestRunOnce_MultipleFacilities_IndependentCounts(t *testing.T) {
	s := newFakeStore()
	f1, f2 := uuid.New(), uuid.New()
	orgID := uuid.New()
	s.facilities = []storage.Facility{
		facility(f1, orgID, "A"),
		facility(f2, orgID, "B"),
	}
	s.exceedances[f1] = []storage.Exceedance{exceedance(f1, orgID, 9, 8, "daily_max")}
	s.exceedances[f2] = []storage.Exceedance{
		exceedance(f2, orgID, 10, 8, "daily_max"),
		exceedance(f2, orgID, 11, 8, "daily_max"),
	}

	e := NewEvaluator(s, &fakePublisher{}, time.Minute)
	n, err := e.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != 3 {
		t.Fatalf("expected 3 alerts (1+2), got %d", n)
	}
}

func TestRunOnce_FacilityError_ContinuesToOthers(t *testing.T) {
	s := newFakeStore()
	f1, f2 := uuid.New(), uuid.New()
	orgID := uuid.New()
	s.facilities = []storage.Facility{
		facility(f1, orgID, "Failing"),
		facility(f2, orgID, "Working"),
	}
	s.exErr[f1] = errors.New("db unavailable")
	s.exceedances[f2] = []storage.Exceedance{exceedance(f2, orgID, 9, 8, "daily_max")}

	e := NewEvaluator(s, &fakePublisher{}, time.Minute)
	n, err := e.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("unexpected error at top level: %v", err)
	}
	if n != 1 {
		t.Fatalf("expected 1 alert from working facility, got %d", n)
	}
}

func TestRunOnce_ListAllFacilitiesError_Bubbles(t *testing.T) {
	s := newFakeStore()
	s.listFacErr = errors.New("boom")

	e := NewEvaluator(s, &fakePublisher{}, time.Minute)
	_, err := e.RunOnce(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestRunOnce_ExceedanceDetailsPayload(t *testing.T) {
	s := newFakeStore()
	facID := uuid.New()
	orgID := uuid.New()
	s.facilities = []storage.Facility{facility(facID, orgID, "WTP")}
	ex := exceedance(facID, orgID, 9.2, 8.5, "daily_max")
	s.exceedances[facID] = []storage.Exceedance{ex}

	e := NewEvaluator(s, &fakePublisher{}, time.Minute)
	_, _ = e.RunOnce(context.Background())

	if len(s.createCalls) != 1 {
		t.Fatalf("expected 1 call")
	}
	var details map[string]any
	if err := json.Unmarshal(s.createCalls[0].Details, &details); err != nil {
		t.Fatalf("unmarshal details: %v", err)
	}
	if details["parameter_code"] != "PH" {
		t.Errorf("parameter_code: got %v", details["parameter_code"])
	}
	if details["limit_type"] != "daily_max" {
		t.Errorf("limit_type: got %v", details["limit_type"])
	}
}

func TestNewEvaluator_DefaultInterval(t *testing.T) {
	e := NewEvaluator(newFakeStore(), &fakePublisher{}, 0)
	if e.interval != 5*time.Minute {
		t.Errorf("expected default 5m, got %v", e.interval)
	}
	e2 := NewEvaluator(newFakeStore(), &fakePublisher{}, -1*time.Second)
	if e2.interval != 5*time.Minute {
		t.Errorf("expected default 5m for negative, got %v", e2.interval)
	}
}

func TestRunOnce_NilBus_NoPanic(t *testing.T) {
	s := newFakeStore()
	facID := uuid.New()
	orgID := uuid.New()
	s.facilities = []storage.Facility{facility(facID, orgID, "WTP")}
	s.exceedances[facID] = []storage.Exceedance{exceedance(facID, orgID, 9, 8, "daily_max")}

	e := NewEvaluator(s, nil, time.Minute)
	n, err := e.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != 1 {
		t.Fatalf("expected 1, got %d", n)
	}
}
