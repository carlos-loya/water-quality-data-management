package storage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Queries provides database access methods.
type Queries struct {
	pool *pgxpool.Pool
}

// New creates a new Queries instance.
func New(pool *pgxpool.Pool) *Queries {
	return &Queries{pool: pool}
}

// User represents a system user.
type User struct {
	ID             uuid.UUID `json:"id"`
	OrganizationID uuid.UUID `json:"organization_id"`
	Email          string    `json:"email"`
	Name           string    `json:"name"`
	PasswordHash   string    `json:"-"`
	Active         bool      `json:"active"`
}

// GetUserByEmail returns a user by email within an organization.
func (q *Queries) GetUserByEmail(ctx context.Context, email string) (User, error) {
	var u User
	err := q.pool.QueryRow(ctx, `
		SELECT id, organization_id, email, name, password_hash, active
		FROM users
		WHERE email = $1`, email).Scan(&u.ID, &u.OrganizationID, &u.Email, &u.Name, &u.PasswordHash, &u.Active)
	return u, err
}

// UserRole represents a role assignment for a user.
type UserRole struct {
	RoleName   string     `json:"role"`
	FacilityID *uuid.UUID `json:"facility_id,omitempty"`
}

// GetUserRoles returns all role assignments for a user.
func (q *Queries) GetUserRoles(ctx context.Context, userID uuid.UUID) ([]UserRole, error) {
	rows, err := q.pool.Query(ctx, `
		SELECT r.name AS role_name, ur.facility_id
		FROM user_roles ur
		JOIN roles r ON ur.role_id = r.id
		WHERE ur.user_id = $1`, userID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByName[UserRole])
}

// GetFacilityIDForLocation resolves the facility_id for a monitoring location.
func (q *Queries) GetFacilityIDForLocation(ctx context.Context, locationID uuid.UUID) (uuid.UUID, error) {
	var facilityID uuid.UUID
	err := q.pool.QueryRow(ctx, `
		SELECT facility_id FROM monitoring_locations WHERE id = $1`, locationID).Scan(&facilityID)
	return facilityID, err
}

// ListFacilitiesForUser returns facilities the user has access to based on role assignments.
// Users with org-wide roles (facility_id IS NULL) get all facilities.
// Users with facility-scoped roles get only those facilities.
func (q *Queries) ListFacilitiesForUser(ctx context.Context, orgID, userID uuid.UUID) ([]Facility, error) {
	rows, err := q.pool.Query(ctx, `
		SELECT DISTINCT f.id, f.organization_id, f.name, f.facility_type, f.address,
		       f.latitude, f.longitude, f.active, f.created_at, f.updated_at
		FROM facilities f
		JOIN user_roles ur ON ur.user_id = $2
		JOIN roles r ON ur.role_id = r.id
		WHERE f.organization_id = $1
		  AND (ur.facility_id IS NULL OR ur.facility_id = f.id)
		ORDER BY f.name`, orgID, userID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByName[Facility])
}

// Facility represents a treatment plant.
type Facility struct {
	ID             uuid.UUID  `json:"id"`
	OrganizationID uuid.UUID `json:"organization_id"`
	Name           string    `json:"name"`
	FacilityType   string    `json:"facility_type"`
	Address        *string   `json:"address,omitempty"`
	Latitude       *float64  `json:"latitude,omitempty"`
	Longitude      *float64  `json:"longitude,omitempty"`
	Active         bool      `json:"active"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// MonitoringLocation represents a sampling point within a facility.
type MonitoringLocation struct {
	ID           uuid.UUID `json:"id"`
	FacilityID   uuid.UUID `json:"facility_id"`
	Name         string    `json:"name"`
	Description  *string   `json:"description,omitempty"`
	LocationType *string   `json:"location_type,omitempty"`
	Latitude     *float64  `json:"latitude,omitempty"`
	Longitude    *float64  `json:"longitude,omitempty"`
	Active       bool      `json:"active"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Parameter represents a measured analyte.
type Parameter struct {
	ID             uuid.UUID  `json:"id"`
	OrganizationID uuid.UUID `json:"organization_id"`
	Code           string    `json:"code"`
	Name           string    `json:"name"`
	Description    *string   `json:"description,omitempty"`
	DefaultUnitID  *uuid.UUID `json:"default_unit_id,omitempty"`
	Category       *string   `json:"category,omitempty"`
	Active         bool      `json:"active"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// SampleResult represents a water quality measurement.
type SampleResult struct {
	ID                   uuid.UUID  `json:"id"`
	MonitoringLocationID uuid.UUID  `json:"monitoring_location_id"`
	ParameterID          uuid.UUID  `json:"parameter_id"`
	MethodID             *uuid.UUID `json:"method_id,omitempty"`
	UnitID               uuid.UUID  `json:"unit_id"`
	CollectedAt          time.Time  `json:"collected_at"`
	AnalyzedAt           *time.Time `json:"analyzed_at,omitempty"`
	ResultValue          *float64   `json:"result_value"`
	ResultQualifier      *string    `json:"result_qualifier,omitempty"`
	DetectionLimit       *float64   `json:"detection_limit,omitempty"`
	Status               string     `json:"status"`
	EnteredBy            uuid.UUID  `json:"entered_by"`
	EnteredAt            time.Time  `json:"entered_at"`
	ReviewedBy           *uuid.UUID `json:"reviewed_by,omitempty"`
	ReviewedAt           *time.Time `json:"reviewed_at,omitempty"`
	ApprovedBy           *uuid.UUID `json:"approved_by,omitempty"`
	ApprovedAt           *time.Time `json:"approved_at,omitempty"`
	Source               string     `json:"source"`
	SourceReference      *string    `json:"source_reference,omitempty"`
	Notes                *string    `json:"notes,omitempty"`
	CreatedAt            time.Time  `json:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at"`
}

// ComplianceResult joins a sample result with its applicable permit limit.
type ComplianceResult struct {
	FacilityName   string   `json:"facility_name"`
	LocationName   string   `json:"location_name"`
	ParameterCode  string   `json:"parameter_code"`
	ParameterName  string   `json:"parameter_name"`
	ResultValue    *float64 `json:"result_value"`
	Qualifier      *string  `json:"result_qualifier,omitempty"`
	UnitCode       string   `json:"unit_code"`
	CollectedAt    time.Time `json:"collected_at"`
	Status         string   `json:"status"`
	LimitType      string   `json:"limit_type"`
	LimitValue     float64  `json:"limit_value"`
	Compliance     string   `json:"compliance"`
}

// ListFacilities returns all facilities for an organization.
func (q *Queries) ListFacilities(ctx context.Context, orgID uuid.UUID) ([]Facility, error) {
	rows, err := q.pool.Query(ctx, `
		SELECT id, organization_id, name, facility_type, address, latitude, longitude, active, created_at, updated_at
		FROM facilities
		WHERE organization_id = $1
		ORDER BY name`, orgID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByName[Facility])
}

// ListMonitoringLocations returns all monitoring locations for a facility.
func (q *Queries) ListMonitoringLocations(ctx context.Context, facilityID uuid.UUID) ([]MonitoringLocation, error) {
	rows, err := q.pool.Query(ctx, `
		SELECT id, facility_id, name, description, location_type, latitude, longitude, active, created_at, updated_at
		FROM monitoring_locations
		WHERE facility_id = $1
		ORDER BY name`, facilityID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByName[MonitoringLocation])
}

// ListAllMonitoringLocations returns all monitoring locations across all facilities for an organization.
func (q *Queries) ListAllMonitoringLocations(ctx context.Context, orgID uuid.UUID) ([]MonitoringLocation, error) {
	rows, err := q.pool.Query(ctx, `
		SELECT ml.id, ml.facility_id, ml.name, ml.description, ml.location_type, ml.latitude, ml.longitude, ml.active, ml.created_at, ml.updated_at
		FROM monitoring_locations ml
		JOIN facilities f ON ml.facility_id = f.id
		WHERE f.organization_id = $1
		ORDER BY ml.name`, orgID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByName[MonitoringLocation])
}

// UnitOfMeasure represents a unit entry.
type UnitOfMeasure struct {
	ID   uuid.UUID `json:"id"`
	Code string    `json:"code"`
	Name string    `json:"name"`
}

// ListUnits returns all units of measure for an organization.
func (q *Queries) ListUnits(ctx context.Context, orgID uuid.UUID) ([]UnitOfMeasure, error) {
	rows, err := q.pool.Query(ctx, `
		SELECT id, code, name
		FROM units_of_measure
		WHERE organization_id = $1
		ORDER BY code`, orgID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByName[UnitOfMeasure])
}

// ValidationRule defines configurable validation constraints for a parameter.
type ValidationRule struct {
	ID              uuid.UUID `json:"id" db:"id"`
	ParameterID     uuid.UUID `json:"parameter_id" db:"parameter_id"`
	MinValue        *float64  `json:"min_value" db:"min_value"`
	MaxValue        *float64  `json:"max_value" db:"max_value"`
	PrecisionDigits *int16    `json:"precision_digits" db:"precision_digits"`
	IsRequired      bool      `json:"is_required" db:"is_required"`
	Active          bool      `json:"active" db:"active"`
}

// ListValidationRules returns all active validation rules for an organization's parameters.
func (q *Queries) ListValidationRules(ctx context.Context, orgID uuid.UUID) ([]ValidationRule, error) {
	rows, err := q.pool.Query(ctx, `
		SELECT vr.id, vr.parameter_id, vr.min_value, vr.max_value,
		       vr.precision_digits, vr.is_required, vr.active
		FROM validation_rules vr
		JOIN parameters p ON p.id = vr.parameter_id
		WHERE p.organization_id = $1 AND vr.active = true
		ORDER BY p.code`, orgID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByName[ValidationRule])
}

// GetValidationRule returns the active validation rule for a specific parameter.
func (q *Queries) GetValidationRule(ctx context.Context, parameterID uuid.UUID) (ValidationRule, error) {
	rows, err := q.pool.Query(ctx, `
		SELECT id, parameter_id, min_value, max_value,
		       precision_digits, is_required, active
		FROM validation_rules
		WHERE parameter_id = $1 AND active = true`, parameterID)
	if err != nil {
		return ValidationRule{}, err
	}
	return pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[ValidationRule])
}

// ListParameters returns all parameters for an organization.
func (q *Queries) ListParameters(ctx context.Context, orgID uuid.UUID) ([]Parameter, error) {
	rows, err := q.pool.Query(ctx, `
		SELECT id, organization_id, code, name, description, default_unit_id, category, active, created_at, updated_at
		FROM parameters
		WHERE organization_id = $1
		ORDER BY code`, orgID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByName[Parameter])
}

// SampleResultFilter controls which sample results are returned.
type SampleResultFilter struct {
	MonitoringLocationID *uuid.UUID
	ParameterID          *uuid.UUID
	Status               *string
	StartDate            *time.Time
	EndDate              *time.Time
	Limit                int
}

// ListSampleResults returns sample results matching the given filter.
func (q *Queries) ListSampleResults(ctx context.Context, f SampleResultFilter) ([]SampleResult, error) {
	query := `
		SELECT id, monitoring_location_id, parameter_id, method_id, unit_id,
		       collected_at, analyzed_at, result_value, result_qualifier, detection_limit,
		       status, entered_by, entered_at, reviewed_by, reviewed_at,
		       approved_by, approved_at, source, source_reference, notes,
		       created_at, updated_at
		FROM sample_results
		WHERE 1=1`
	args := []any{}
	argN := 1

	if f.MonitoringLocationID != nil {
		query += fmt.Sprintf(" AND monitoring_location_id = $%d", argN)
		args = append(args, *f.MonitoringLocationID)
		argN++
	}
	if f.ParameterID != nil {
		query += fmt.Sprintf(" AND parameter_id = $%d", argN)
		args = append(args, *f.ParameterID)
		argN++
	}
	if f.Status != nil {
		query += fmt.Sprintf(" AND status = $%d", argN)
		args = append(args, *f.Status)
		argN++
	}
	if f.StartDate != nil {
		query += fmt.Sprintf(" AND collected_at >= $%d", argN)
		args = append(args, *f.StartDate)
		argN++
	}
	if f.EndDate != nil {
		query += fmt.Sprintf(" AND collected_at <= $%d", argN)
		args = append(args, *f.EndDate)
		argN++
	}

	query += " ORDER BY collected_at DESC"

	limit := f.Limit
	if limit <= 0 || limit > 1000 {
		limit = 100
	}
	query += fmt.Sprintf(" LIMIT $%d", argN)
	args = append(args, limit)

	rows, err := q.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByName[SampleResult])
}

// CreateSampleResultParams contains the fields needed to create a new sample result.
type CreateSampleResultParams struct {
	MonitoringLocationID uuid.UUID  `json:"monitoring_location_id"`
	ParameterID          uuid.UUID  `json:"parameter_id"`
	MethodID             *uuid.UUID `json:"method_id,omitempty"`
	UnitID               uuid.UUID  `json:"unit_id"`
	CollectedAt          time.Time  `json:"collected_at"`
	AnalyzedAt           *time.Time `json:"analyzed_at,omitempty"`
	ResultValue          *float64   `json:"result_value"`
	ResultQualifier      *string    `json:"result_qualifier,omitempty"`
	DetectionLimit       *float64   `json:"detection_limit,omitempty"`
	EnteredBy            uuid.UUID  `json:"entered_by"`
	Source               string     `json:"source"`
	SourceReference      *string    `json:"source_reference,omitempty"`
	Notes                *string    `json:"notes,omitempty"`
}

// CreateSampleResult inserts a new sample result and returns it.
func (q *Queries) CreateSampleResult(ctx context.Context, p CreateSampleResultParams) (SampleResult, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return SampleResult{}, fmt.Errorf("generate uuid: %w", err)
	}

	var result SampleResult
	err = q.pool.QueryRow(ctx, `
		INSERT INTO sample_results (
			id, monitoring_location_id, parameter_id, method_id, unit_id,
			collected_at, analyzed_at, result_value, result_qualifier, detection_limit,
			status, entered_by, source, source_reference, notes
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8, $9, $10,
			'draft', $11, $12, $13, $14
		)
		RETURNING id, monitoring_location_id, parameter_id, method_id, unit_id,
		          collected_at, analyzed_at, result_value, result_qualifier, detection_limit,
		          status, entered_by, entered_at, reviewed_by, reviewed_at,
		          approved_by, approved_at, source, source_reference, notes,
		          created_at, updated_at`,
		id, p.MonitoringLocationID, p.ParameterID, p.MethodID, p.UnitID,
		p.CollectedAt, p.AnalyzedAt, p.ResultValue, p.ResultQualifier, p.DetectionLimit,
		p.EnteredBy, p.Source, p.SourceReference, p.Notes,
	).Scan(
		&result.ID, &result.MonitoringLocationID, &result.ParameterID, &result.MethodID, &result.UnitID,
		&result.CollectedAt, &result.AnalyzedAt, &result.ResultValue, &result.ResultQualifier, &result.DetectionLimit,
		&result.Status, &result.EnteredBy, &result.EnteredAt, &result.ReviewedBy, &result.ReviewedAt,
		&result.ApprovedBy, &result.ApprovedAt, &result.Source, &result.SourceReference, &result.Notes,
		&result.CreatedAt, &result.UpdatedAt,
	)
	return result, err
}

// EvaluateCompliance checks sample results against effective permit limits.
func (q *Queries) EvaluateCompliance(ctx context.Context, facilityID uuid.UUID) ([]ComplianceResult, error) {
	rows, err := q.pool.Query(ctx, `
		SELECT f.name AS facility_name, ml.name AS location_name,
		       p.code AS parameter_code, p.name AS parameter_name,
		       sr.result_value, sr.result_qualifier AS qualifier, u.code AS unit_code,
		       sr.collected_at, sr.status,
		       pl.limit_type, pl.limit_value,
		       CASE
		           WHEN sr.result_value IS NULL THEN 'N/A'
		           WHEN pl.limit_type LIKE '%max%' AND sr.result_value > pl.limit_value THEN 'EXCEEDANCE'
		           WHEN pl.limit_type LIKE '%min%' AND sr.result_value < pl.limit_value THEN 'EXCEEDANCE'
		           ELSE 'OK'
		       END AS compliance
		FROM sample_results sr
		JOIN monitoring_locations ml ON sr.monitoring_location_id = ml.id
		JOIN facilities f ON ml.facility_id = f.id
		JOIN parameters p ON sr.parameter_id = p.id
		JOIN units_of_measure u ON sr.unit_id = u.id
		JOIN permit_limits pl ON pl.monitoring_location_id = sr.monitoring_location_id
		    AND pl.parameter_id = sr.parameter_id
		    AND sr.collected_at::date >= pl.effective_start
		    AND (pl.effective_end IS NULL OR sr.collected_at::date <= pl.effective_end)
		WHERE f.id = $1
		ORDER BY p.code, sr.collected_at`, facilityID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByName[ComplianceResult])
}

// GetSampleResult retrieves a single sample result by ID.
func (q *Queries) GetSampleResult(ctx context.Context, id uuid.UUID) (SampleResult, error) {
	rows, err := q.pool.Query(ctx, `
		SELECT id, monitoring_location_id, parameter_id, method_id, unit_id,
		       collected_at, analyzed_at, result_value, result_qualifier, detection_limit,
		       status, entered_by, entered_at, reviewed_by, reviewed_at,
		       approved_by, approved_at, source, source_reference, notes,
		       created_at, updated_at
		FROM sample_results
		WHERE id = $1`, id)
	if err != nil {
		return SampleResult{}, err
	}
	return pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[SampleResult])
}

// ReviewSampleResult transitions a sample result from 'draft' to 'reviewed'.
func (q *Queries) ReviewSampleResult(ctx context.Context, id uuid.UUID, reviewerID uuid.UUID) (SampleResult, error) {
	rows, err := q.pool.Query(ctx, `
		UPDATE sample_results
		SET status = 'reviewed', reviewed_by = $2, reviewed_at = now(), updated_at = now()
		WHERE id = $1 AND status = 'draft'
		RETURNING id, monitoring_location_id, parameter_id, method_id, unit_id,
		          collected_at, analyzed_at, result_value, result_qualifier, detection_limit,
		          status, entered_by, entered_at, reviewed_by, reviewed_at,
		          approved_by, approved_at, source, source_reference, notes,
		          created_at, updated_at`, id, reviewerID)
	if err != nil {
		return SampleResult{}, err
	}
	return pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[SampleResult])
}

// ApproveSampleResult transitions a sample result from 'reviewed' to 'approved'.
func (q *Queries) ApproveSampleResult(ctx context.Context, id uuid.UUID, approverID uuid.UUID) (SampleResult, error) {
	rows, err := q.pool.Query(ctx, `
		UPDATE sample_results
		SET status = 'approved', approved_by = $2, approved_at = now(), updated_at = now()
		WHERE id = $1 AND status = 'reviewed'
		RETURNING id, monitoring_location_id, parameter_id, method_id, unit_id,
		          collected_at, analyzed_at, result_value, result_qualifier, detection_limit,
		          status, entered_by, entered_at, reviewed_by, reviewed_at,
		          approved_by, approved_at, source, source_reference, notes,
		          created_at, updated_at`, id, approverID)
	if err != nil {
		return SampleResult{}, err
	}
	return pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[SampleResult])
}

// AuditEntry represents a row from the audit_log table.
type AuditEntry struct {
	ID             uuid.UUID        `json:"id"`
	OrganizationID uuid.UUID       `json:"organization_id"`
	TableName      string           `json:"table_name"`
	RecordID       uuid.UUID        `json:"record_id"`
	Action         string           `json:"action"`
	OldValues      *json.RawMessage `json:"old_values,omitempty"`
	NewValues      *json.RawMessage `json:"new_values,omitempty"`
	ChangedBy      uuid.UUID        `json:"changed_by"`
	ChangedAt      time.Time        `json:"changed_at"`
	Reason         *string          `json:"reason,omitempty"`
}

// ListAuditLog returns audit entries for a given record.
func (q *Queries) ListAuditLog(ctx context.Context, recordID uuid.UUID) ([]AuditEntry, error) {
	rows, err := q.pool.Query(ctx, `
		SELECT id, organization_id, table_name, record_id, action,
		       old_values, new_values, changed_by, changed_at, reason
		FROM audit_log
		WHERE record_id = $1
		ORDER BY changed_at DESC`, recordID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByName[AuditEntry])
}

// TrendingPoint represents a single data point for time-series charting.
type TrendingPoint struct {
	CollectedAt   time.Time `json:"collected_at"`
	ResultValue   *float64  `json:"result_value"`
	Qualifier     *string   `json:"result_qualifier,omitempty"`
	LocationName  string    `json:"location_name"`
	ParameterCode string   `json:"parameter_code"`
	ParameterName string   `json:"parameter_name"`
	UnitCode      string   `json:"unit_code"`
}

// TrendingLimit represents a permit limit line for a chart.
type TrendingLimit struct {
	LimitType  string  `json:"limit_type"`
	LimitValue float64 `json:"limit_value"`
}

// TrendingSeries groups data points and limits for one parameter at one location.
type TrendingSeries struct {
	ParameterCode string          `json:"parameter_code"`
	ParameterName string          `json:"parameter_name"`
	LocationName  string          `json:"location_name"`
	UnitCode      string          `json:"unit_code"`
	Points        []TrendingPoint `json:"points"`
	Limits        []TrendingLimit `json:"limits"`
}

// GetTrendingData returns time-series data for a facility, grouped by parameter+location.
func (q *Queries) GetTrendingData(ctx context.Context, facilityID uuid.UUID, days int) ([]TrendingSeries, error) {
	if days <= 0 {
		days = 30
	}

	rows, err := q.pool.Query(ctx, `
		SELECT sr.collected_at, sr.result_value, sr.result_qualifier AS qualifier,
		       ml.name AS location_name, p.code AS parameter_code, p.name AS parameter_name,
		       u.code AS unit_code
		FROM sample_results sr
		JOIN monitoring_locations ml ON sr.monitoring_location_id = ml.id
		JOIN facilities f ON ml.facility_id = f.id
		JOIN parameters p ON sr.parameter_id = p.id
		JOIN units_of_measure u ON sr.unit_id = u.id
		WHERE f.id = $1 AND sr.collected_at >= now() - make_interval(days => $2)
		ORDER BY p.code, ml.name, sr.collected_at`, facilityID, days)
	if err != nil {
		return nil, err
	}
	points, err := pgx.CollectRows(rows, pgx.RowToStructByName[TrendingPoint])
	if err != nil {
		return nil, err
	}

	// Get current limits for this facility
	limitRows, err := q.pool.Query(ctx, `
		SELECT DISTINCT p.code AS parameter_code, ml.name AS location_name,
		       pl.limit_type, pl.limit_value
		FROM permit_limits pl
		JOIN monitoring_locations ml ON pl.monitoring_location_id = ml.id
		JOIN parameters p ON pl.parameter_id = p.id
		WHERE ml.facility_id = $1
		  AND pl.effective_start <= CURRENT_DATE
		  AND (pl.effective_end IS NULL OR pl.effective_end >= CURRENT_DATE)
		ORDER BY p.code, ml.name, pl.limit_type`, facilityID)
	if err != nil {
		return nil, err
	}

	type limitKey struct{ param, loc string }
	limitMap := make(map[limitKey][]TrendingLimit)
	for limitRows.Next() {
		var paramCode, locName, limitType string
		var limitValue float64
		if err := limitRows.Scan(&paramCode, &locName, &limitType, &limitValue); err != nil {
			return nil, err
		}
		k := limitKey{paramCode, locName}
		limitMap[k] = append(limitMap[k], TrendingLimit{limitType, limitValue})
	}
	limitRows.Close()

	// Group points into series by parameter+location
	type seriesKey struct{ param, loc string }
	seriesMap := make(map[seriesKey]*TrendingSeries)
	var seriesOrder []seriesKey

	for _, pt := range points {
		k := seriesKey{pt.ParameterCode, pt.LocationName}
		s, ok := seriesMap[k]
		if !ok {
			limits := limitMap[limitKey{pt.ParameterCode, pt.LocationName}]
			if limits == nil {
				limits = []TrendingLimit{}
			}
			s = &TrendingSeries{
				ParameterCode: pt.ParameterCode,
				ParameterName: pt.ParameterName,
				LocationName:  pt.LocationName,
				UnitCode:      pt.UnitCode,
				Limits:        limits,
			}
			seriesMap[k] = s
			seriesOrder = append(seriesOrder, k)
		}
		s.Points = append(s.Points, pt)
	}

	result := make([]TrendingSeries, 0, len(seriesOrder))
	for _, k := range seriesOrder {
		result = append(result, *seriesMap[k])
	}
	return result, nil
}

// Instrument represents a lab or field instrument.
type Instrument struct {
	ID                  uuid.UUID `json:"id"`
	FacilityID          uuid.UUID `json:"facility_id"`
	Name                string    `json:"name"`
	SerialNumber        *string   `json:"serial_number,omitempty"`
	InstrumentType      string    `json:"instrument_type"`
	Manufacturer        *string   `json:"manufacturer,omitempty"`
	Model               *string   `json:"model,omitempty"`
	LocationDescription *string   `json:"location_description,omitempty"`
	Active              bool      `json:"active"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

// CalibrationRecord represents a verification or calibration event.
type CalibrationRecord struct {
	ID               uuid.UUID  `json:"id"`
	InstrumentID     uuid.UUID  `json:"instrument_id"`
	CalibrationType  string     `json:"calibration_type"`
	PerformedAt      time.Time  `json:"performed_at"`
	PerformedBy      uuid.UUID  `json:"performed_by"`
	DueAt            *time.Time `json:"due_at,omitempty"`
	Status           string     `json:"status"`
	PreValue         *float64   `json:"pre_value,omitempty"`
	PostValue        *float64   `json:"post_value,omitempty"`
	MethodReference  *string    `json:"method_reference,omitempty"`
	CorrectiveAction *string    `json:"corrective_action,omitempty"`
	Notes            *string    `json:"notes,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

// InstrumentStatus combines an instrument with its latest calibration info.
type InstrumentStatus struct {
	ID                  uuid.UUID  `json:"id"`
	Name                string     `json:"name"`
	SerialNumber        *string    `json:"serial_number,omitempty"`
	InstrumentType      string     `json:"instrument_type"`
	Manufacturer        *string    `json:"manufacturer,omitempty"`
	Model               *string    `json:"model,omitempty"`
	Active              bool       `json:"active"`
	LastCalibrationType *string    `json:"last_calibration_type,omitempty"`
	LastPerformedAt     *time.Time `json:"last_performed_at,omitempty"`
	LastStatus          *string    `json:"last_status,omitempty"`
	DueAt               *time.Time `json:"due_at,omitempty"`
	CalibrationStatus   string     `json:"calibration_status"` // "current", "due_soon", "overdue", "no_schedule"
}

// ListInstruments returns all instruments for a facility.
func (q *Queries) ListInstruments(ctx context.Context, facilityID uuid.UUID) ([]Instrument, error) {
	rows, err := q.pool.Query(ctx, `
		SELECT id, facility_id, name, serial_number, instrument_type,
		       manufacturer, model, location_description, active, created_at, updated_at
		FROM instruments
		WHERE facility_id = $1
		ORDER BY name`, facilityID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByName[Instrument])
}

// ListCalibrationRecords returns calibration history for an instrument.
func (q *Queries) ListCalibrationRecords(ctx context.Context, instrumentID uuid.UUID) ([]CalibrationRecord, error) {
	rows, err := q.pool.Query(ctx, `
		SELECT id, instrument_id, calibration_type, performed_at, performed_by,
		       due_at, status, pre_value, post_value, method_reference,
		       corrective_action, notes, created_at, updated_at
		FROM calibration_records
		WHERE instrument_id = $1
		ORDER BY performed_at DESC`, instrumentID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByName[CalibrationRecord])
}

// ListInstrumentStatuses returns instruments with their current calibration status.
// Flags instruments as overdue (past due_at), due_soon (within 2 days), current, or no_schedule.
func (q *Queries) ListInstrumentStatuses(ctx context.Context, facilityID uuid.UUID) ([]InstrumentStatus, error) {
	rows, err := q.pool.Query(ctx, `
		WITH latest_cal AS (
			SELECT DISTINCT ON (instrument_id)
				instrument_id,
				calibration_type AS last_calibration_type,
				performed_at AS last_performed_at,
				status AS last_status,
				due_at
			FROM calibration_records
			ORDER BY instrument_id, performed_at DESC
		)
		SELECT
			i.id, i.name, i.serial_number, i.instrument_type,
			i.manufacturer, i.model, i.active,
			lc.last_calibration_type, lc.last_performed_at, lc.last_status, lc.due_at,
			CASE
				WHEN lc.due_at IS NULL THEN 'no_schedule'
				WHEN lc.due_at < now() THEN 'overdue'
				WHEN lc.due_at < now() + interval '2 days' THEN 'due_soon'
				ELSE 'current'
			END AS calibration_status
		FROM instruments i
		LEFT JOIN latest_cal lc ON lc.instrument_id = i.id
		WHERE i.facility_id = $1
		ORDER BY
			CASE
				WHEN lc.due_at IS NULL THEN 3
				WHEN lc.due_at < now() THEN 0
				WHEN lc.due_at < now() + interval '2 days' THEN 1
				ELSE 2
			END,
			i.name`, facilityID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByName[InstrumentStatus])
}

// GetOrganizationIDForResult resolves the organization_id for a sample result
// by traversing the facility hierarchy.
func (q *Queries) GetOrganizationIDForResult(ctx context.Context, resultID uuid.UUID) (uuid.UUID, error) {
	var orgID uuid.UUID
	err := q.pool.QueryRow(ctx, `
		SELECT f.organization_id
		FROM sample_results sr
		JOIN monitoring_locations ml ON sr.monitoring_location_id = ml.id
		JOIN facilities f ON ml.facility_id = f.id
		WHERE sr.id = $1`, resultID).Scan(&orgID)
	return orgID, err
}

// ListAllFacilities returns every active facility across all organizations.
// Used by the alerts evaluator, which has no user context.
func (q *Queries) ListAllFacilities(ctx context.Context) ([]Facility, error) {
	rows, err := q.pool.Query(ctx, `
		SELECT id, organization_id, name, facility_type, address, latitude, longitude, active, created_at, updated_at
		FROM facilities
		WHERE active = true
		ORDER BY organization_id, name`)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByName[Facility])
}

// Exceedance describes a sample result that violates a currently effective permit limit.
// Purpose-built for the alerts evaluator — carries the IDs needed to create a targeted alert.
type Exceedance struct {
	SampleResultID       uuid.UUID `json:"sample_result_id" db:"sample_result_id"`
	MonitoringLocationID uuid.UUID `json:"monitoring_location_id" db:"monitoring_location_id"`
	FacilityID           uuid.UUID `json:"facility_id" db:"facility_id"`
	OrganizationID       uuid.UUID `json:"organization_id" db:"organization_id"`
	LocationName         string    `json:"location_name" db:"location_name"`
	ParameterCode        string    `json:"parameter_code" db:"parameter_code"`
	ParameterName        string    `json:"parameter_name" db:"parameter_name"`
	ResultValue          float64   `json:"result_value" db:"result_value"`
	UnitCode             string    `json:"unit_code" db:"unit_code"`
	LimitType            string    `json:"limit_type" db:"limit_type"`
	LimitValue           float64   `json:"limit_value" db:"limit_value"`
	CollectedAt          time.Time `json:"collected_at" db:"collected_at"`
}

// ListFacilityExceedances returns sample results that currently exceed an effective permit limit.
// Rows with NULL result_value are excluded (non-detects can't exceed a numeric limit).
func (q *Queries) ListFacilityExceedances(ctx context.Context, facilityID uuid.UUID) ([]Exceedance, error) {
	rows, err := q.pool.Query(ctx, `
		SELECT sr.id AS sample_result_id,
		       ml.id AS monitoring_location_id,
		       f.id  AS facility_id,
		       f.organization_id,
		       ml.name AS location_name,
		       p.code  AS parameter_code,
		       p.name  AS parameter_name,
		       sr.result_value,
		       u.code  AS unit_code,
		       pl.limit_type, pl.limit_value,
		       sr.collected_at
		FROM sample_results sr
		JOIN monitoring_locations ml ON sr.monitoring_location_id = ml.id
		JOIN facilities f             ON ml.facility_id = f.id
		JOIN parameters p             ON sr.parameter_id = p.id
		JOIN units_of_measure u       ON sr.unit_id = u.id
		JOIN permit_limits pl         ON pl.monitoring_location_id = sr.monitoring_location_id
		    AND pl.parameter_id = sr.parameter_id
		    AND sr.collected_at::date >= pl.effective_start
		    AND (pl.effective_end IS NULL OR sr.collected_at::date <= pl.effective_end)
		WHERE f.id = $1
		  AND sr.result_value IS NOT NULL
		  AND (
		      (pl.limit_type LIKE '%max%' AND sr.result_value > pl.limit_value) OR
		      (pl.limit_type LIKE '%min%' AND sr.result_value < pl.limit_value)
		  )`, facilityID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByName[Exceedance])
}

// ============================================================================
// Alerts
// ============================================================================

// Alert represents a notification about a compliance or operational condition.
type Alert struct {
	ID             uuid.UUID       `json:"id"`
	OrganizationID uuid.UUID       `json:"organization_id"`
	FacilityID     uuid.UUID       `json:"facility_id"`
	Type           string          `json:"type"`
	Severity       string          `json:"severity"`
	SubjectType    string          `json:"subject_type"`
	SubjectID      uuid.UUID       `json:"subject_id"`
	Message        string          `json:"message"`
	Details        json.RawMessage `json:"details,omitempty"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
	DismissedAt    *time.Time      `json:"dismissed_at,omitempty"`
	DismissedBy    *uuid.UUID      `json:"dismissed_by,omitempty"`
}

// AlertFilter controls which alerts are returned by ListAlerts.
type AlertFilter struct {
	FacilityID *uuid.UUID
	Type       *string
	Dismissed  *bool // nil = any state
	Limit      int
}

// CreateAlertParams is the input for CreateAlert.
type CreateAlertParams struct {
	OrganizationID uuid.UUID
	FacilityID     uuid.UUID
	Type           string
	Severity       string
	SubjectType    string
	SubjectID      uuid.UUID
	Message        string
	Details        json.RawMessage
}

// CreateAlert inserts a new alert. If an active (not dismissed) alert already exists
// for the same (facility, type, subject), the insert is skipped and `created` is false.
// The partial unique index `alerts_active_subject_idx` enforces this at the DB level.
func (q *Queries) CreateAlert(ctx context.Context, p CreateAlertParams) (Alert, bool, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return Alert{}, false, fmt.Errorf("generate uuid: %w", err)
	}

	var a Alert
	err = q.pool.QueryRow(ctx, `
		INSERT INTO alerts (id, organization_id, facility_id, type, severity, subject_type, subject_id, message, details)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (facility_id, type, subject_type, subject_id) WHERE dismissed_at IS NULL
		DO NOTHING
		RETURNING id, organization_id, facility_id, type, severity, subject_type, subject_id,
		          message, details, created_at, updated_at, dismissed_at, dismissed_by`,
		id, p.OrganizationID, p.FacilityID, p.Type, p.Severity, p.SubjectType, p.SubjectID, p.Message, p.Details,
	).Scan(
		&a.ID, &a.OrganizationID, &a.FacilityID, &a.Type, &a.Severity, &a.SubjectType, &a.SubjectID,
		&a.Message, &a.Details, &a.CreatedAt, &a.UpdatedAt, &a.DismissedAt, &a.DismissedBy,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		// Active duplicate — not an error. Caller decides what to do.
		return Alert{}, false, nil
	}
	if err != nil {
		return Alert{}, false, err
	}
	return a, true, nil
}

// ListAlerts returns alerts matching the filter, newest first.
func (q *Queries) ListAlerts(ctx context.Context, f AlertFilter) ([]Alert, error) {
	query := `
		SELECT id, organization_id, facility_id, type, severity, subject_type, subject_id,
		       message, details, created_at, updated_at, dismissed_at, dismissed_by
		FROM alerts
		WHERE 1=1`
	args := []any{}
	argN := 1

	if f.FacilityID != nil {
		query += fmt.Sprintf(" AND facility_id = $%d", argN)
		args = append(args, *f.FacilityID)
		argN++
	}
	if f.Type != nil {
		query += fmt.Sprintf(" AND type = $%d", argN)
		args = append(args, *f.Type)
		argN++
	}
	if f.Dismissed != nil {
		if *f.Dismissed {
			query += " AND dismissed_at IS NOT NULL"
		} else {
			query += " AND dismissed_at IS NULL"
		}
	}

	query += " ORDER BY created_at DESC"

	limit := f.Limit
	if limit <= 0 || limit > 1000 {
		limit = 200
	}
	query += fmt.Sprintf(" LIMIT $%d", argN)
	args = append(args, limit)

	rows, err := q.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	alerts := make([]Alert, 0)
	for rows.Next() {
		var a Alert
		if err := rows.Scan(
			&a.ID, &a.OrganizationID, &a.FacilityID, &a.Type, &a.Severity, &a.SubjectType, &a.SubjectID,
			&a.Message, &a.Details, &a.CreatedAt, &a.UpdatedAt, &a.DismissedAt, &a.DismissedBy,
		); err != nil {
			return nil, err
		}
		alerts = append(alerts, a)
	}
	return alerts, rows.Err()
}

// GetAlert retrieves a single alert by ID.
func (q *Queries) GetAlert(ctx context.Context, id uuid.UUID) (Alert, error) {
	var a Alert
	err := q.pool.QueryRow(ctx, `
		SELECT id, organization_id, facility_id, type, severity, subject_type, subject_id,
		       message, details, created_at, updated_at, dismissed_at, dismissed_by
		FROM alerts
		WHERE id = $1`, id,
	).Scan(
		&a.ID, &a.OrganizationID, &a.FacilityID, &a.Type, &a.Severity, &a.SubjectType, &a.SubjectID,
		&a.Message, &a.Details, &a.CreatedAt, &a.UpdatedAt, &a.DismissedAt, &a.DismissedBy,
	)
	return a, err
}

// DismissAlert marks an alert as dismissed by the given user.
// Returns the updated alert. If the alert is already dismissed, returns pgx.ErrNoRows.
func (q *Queries) DismissAlert(ctx context.Context, id, userID uuid.UUID) (Alert, error) {
	var a Alert
	err := q.pool.QueryRow(ctx, `
		UPDATE alerts
		SET dismissed_at = now(), dismissed_by = $2, updated_at = now()
		WHERE id = $1 AND dismissed_at IS NULL
		RETURNING id, organization_id, facility_id, type, severity, subject_type, subject_id,
		          message, details, created_at, updated_at, dismissed_at, dismissed_by`,
		id, userID,
	).Scan(
		&a.ID, &a.OrganizationID, &a.FacilityID, &a.Type, &a.Severity, &a.SubjectType, &a.SubjectID,
		&a.Message, &a.Details, &a.CreatedAt, &a.UpdatedAt, &a.DismissedAt, &a.DismissedBy,
	)
	return a, err
}
