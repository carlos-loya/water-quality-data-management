// Package alerts provides a background evaluator that scans sample results
// and instruments for conditions that warrant operator attention, and records
// them as rows in the alerts table.
//
// The evaluator is idempotent: creating an alert for the same subject twice
// while the first is still active is a no-op (the storage layer deduplicates
// via a partial unique index). Creating an event is only emitted when a new
// alert is actually inserted.
package alerts

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"time"

	"github.com/google/uuid"

	"github.com/carlos-loya/water-quality-data-management/internal/events"
	"github.com/carlos-loya/water-quality-data-management/internal/storage"
)

// Store is the subset of storage.Store required by the evaluator.
type Store interface {
	ListAllFacilities(ctx context.Context) ([]storage.Facility, error)
	ListFacilityExceedances(ctx context.Context, facilityID uuid.UUID) ([]storage.Exceedance, error)
	ListInstrumentStatuses(ctx context.Context, facilityID uuid.UUID) ([]storage.InstrumentStatus, error)
	CreateAlert(ctx context.Context, p storage.CreateAlertParams) (storage.Alert, bool, error)
}

// Publisher publishes change events. *events.Bus satisfies this.
type Publisher interface {
	Publish(event events.ChangeEvent) error
}

// Evaluator periodically checks for exceedances and overdue calibrations.
type Evaluator struct {
	store    Store
	bus      Publisher
	interval time.Duration
}

// NewEvaluator creates an evaluator with the given tick interval.
// A zero or negative interval defaults to 5 minutes.
func NewEvaluator(store Store, bus Publisher, interval time.Duration) *Evaluator {
	if interval <= 0 {
		interval = 5 * time.Minute
	}
	return &Evaluator{store: store, bus: bus, interval: interval}
}

// Run evaluates once immediately, then ticks until the context is cancelled.
// Intended to be called in a goroutine. Errors are logged, not returned.
func (e *Evaluator) Run(ctx context.Context) {
	slog.Info("alerts evaluator started", "interval", e.interval)
	e.tick(ctx)

	ticker := time.NewTicker(e.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			slog.Info("alerts evaluator stopping")
			return
		case <-ticker.C:
			e.tick(ctx)
		}
	}
}

func (e *Evaluator) tick(ctx context.Context) {
	n, err := e.RunOnce(ctx)
	if err != nil {
		slog.Error("alerts evaluator tick failed", "error", err)
		return
	}
	if n > 0 {
		slog.Info("alerts evaluator created alerts", "count", n)
	}
}

// RunOnce evaluates across all facilities exactly once and returns the number
// of new alerts created. Duplicates (existing active alerts for the same
// subject) are skipped silently.
func (e *Evaluator) RunOnce(ctx context.Context) (int, error) {
	facilities, err := e.store.ListAllFacilities(ctx)
	if err != nil {
		return 0, fmt.Errorf("list facilities: %w", err)
	}

	created := 0
	for _, f := range facilities {
		c, err := e.evaluateFacility(ctx, f)
		if err != nil {
			slog.Error("evaluate facility", "facility_id", f.ID, "error", err)
			continue
		}
		created += c
	}
	return created, nil
}

func (e *Evaluator) evaluateFacility(ctx context.Context, f storage.Facility) (int, error) {
	created := 0

	// Exceedances
	exceedances, err := e.store.ListFacilityExceedances(ctx, f.ID)
	if err != nil {
		return created, fmt.Errorf("list exceedances: %w", err)
	}
	for _, ex := range exceedances {
		params := exceedanceAlertParams(ex)
		alert, inserted, err := e.store.CreateAlert(ctx, params)
		if err != nil {
			slog.Error("create exceedance alert", "sample_result_id", ex.SampleResultID, "error", err)
			continue
		}
		if inserted {
			created++
			e.publishAlertCreated(alert)
		}
	}

	// Overdue calibrations
	statuses, err := e.store.ListInstrumentStatuses(ctx, f.ID)
	if err != nil {
		return created, fmt.Errorf("list instrument statuses: %w", err)
	}
	for _, s := range statuses {
		if s.CalibrationStatus != "overdue" {
			continue
		}
		params := overdueCalibrationAlertParams(f, s)
		alert, inserted, err := e.store.CreateAlert(ctx, params)
		if err != nil {
			slog.Error("create overdue calibration alert", "instrument_id", s.ID, "error", err)
			continue
		}
		if inserted {
			created++
			e.publishAlertCreated(alert)
		}
	}

	return created, nil
}

func (e *Evaluator) publishAlertCreated(a storage.Alert) {
	if e.bus == nil {
		return
	}
	newJSON, _ := json.Marshal(a)
	event := events.ChangeEvent{
		Subject:        events.SubjectAlertCreated,
		Timestamp:      time.Now(),
		OrganizationID: a.OrganizationID,
		TableName:      "alerts",
		RecordID:       a.ID,
		Action:         "insert",
		ChangedBy:      uuid.Nil, // system
		NewValues:      newJSON,
	}
	if err := e.bus.Publish(event); err != nil {
		slog.Error("publish alert event", "error", err, "alert_id", a.ID)
	}
}

// exceedanceAlertParams derives the CreateAlertParams for a single exceedance.
func exceedanceAlertParams(ex storage.Exceedance) storage.CreateAlertParams {
	msg := fmt.Sprintf("%s at %s: %.4g %s exceeds %s of %.4g",
		ex.ParameterName, ex.LocationName,
		ex.ResultValue, ex.UnitCode,
		ex.LimitType, ex.LimitValue)

	details, _ := json.Marshal(map[string]any{
		"parameter_code":         ex.ParameterCode,
		"parameter_name":         ex.ParameterName,
		"location_name":          ex.LocationName,
		"monitoring_location_id": ex.MonitoringLocationID,
		"result_value":           ex.ResultValue,
		"unit_code":              ex.UnitCode,
		"limit_type":             ex.LimitType,
		"limit_value":            ex.LimitValue,
		"collected_at":           ex.CollectedAt,
	})

	return storage.CreateAlertParams{
		OrganizationID: ex.OrganizationID,
		FacilityID:     ex.FacilityID,
		Type:           "exceedance",
		Severity:       "critical",
		SubjectType:    "sample_result",
		SubjectID:      ex.SampleResultID,
		Message:        msg,
		Details:        details,
	}
}

// overdueCalibrationAlertParams derives the CreateAlertParams for an overdue instrument.
// Severity is "warning" if overdue less than 7 days, otherwise "critical".
func overdueCalibrationAlertParams(f storage.Facility, s storage.InstrumentStatus) storage.CreateAlertParams {
	severity := "warning"
	var daysOverdue int
	if s.DueAt != nil {
		elapsed := time.Since(*s.DueAt)
		daysOverdue = int(math.Floor(elapsed.Hours() / 24))
		if daysOverdue >= 7 {
			severity = "critical"
		}
	}

	msg := fmt.Sprintf("%s is %d day(s) overdue for calibration", s.Name, daysOverdue)

	details, _ := json.Marshal(map[string]any{
		"instrument_name":       s.Name,
		"instrument_type":       s.InstrumentType,
		"due_at":                s.DueAt,
		"last_performed_at":     s.LastPerformedAt,
		"last_calibration_type": s.LastCalibrationType,
		"days_overdue":          daysOverdue,
	})

	return storage.CreateAlertParams{
		OrganizationID: f.OrganizationID,
		FacilityID:     f.ID,
		Type:           "overdue_calibration",
		Severity:       severity,
		SubjectType:    "instrument",
		SubjectID:      s.ID,
		Message:        msg,
		Details:        details,
	}
}
