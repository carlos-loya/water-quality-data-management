package storage

import (
	"context"
	"encoding/json"
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
