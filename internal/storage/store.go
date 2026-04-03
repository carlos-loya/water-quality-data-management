package storage

import (
	"context"

	"github.com/google/uuid"
)

// Store defines the interface for all database operations.
// Implemented by *Queries; used by handlers and importers for testability.
type Store interface {
	GetUserByEmail(ctx context.Context, email string) (User, error)
	GetUserRoles(ctx context.Context, userID uuid.UUID) ([]UserRole, error)
	GetFacilityIDForLocation(ctx context.Context, locationID uuid.UUID) (uuid.UUID, error)
	ListFacilitiesForUser(ctx context.Context, orgID, userID uuid.UUID) ([]Facility, error)
	ListFacilities(ctx context.Context, orgID uuid.UUID) ([]Facility, error)
	ListMonitoringLocations(ctx context.Context, facilityID uuid.UUID) ([]MonitoringLocation, error)
	ListAllMonitoringLocations(ctx context.Context, orgID uuid.UUID) ([]MonitoringLocation, error)
	ListUnits(ctx context.Context, orgID uuid.UUID) ([]UnitOfMeasure, error)
	ListValidationRules(ctx context.Context, orgID uuid.UUID) ([]ValidationRule, error)
	GetValidationRule(ctx context.Context, parameterID uuid.UUID) (ValidationRule, error)
	ListParameters(ctx context.Context, orgID uuid.UUID) ([]Parameter, error)
	ListSampleResults(ctx context.Context, f SampleResultFilter) ([]SampleResult, error)
	CreateSampleResult(ctx context.Context, p CreateSampleResultParams) (SampleResult, error)
	GetSampleResult(ctx context.Context, id uuid.UUID) (SampleResult, error)
	ReviewSampleResult(ctx context.Context, id uuid.UUID, reviewerID uuid.UUID) (SampleResult, error)
	ApproveSampleResult(ctx context.Context, id uuid.UUID, approverID uuid.UUID) (SampleResult, error)
	EvaluateCompliance(ctx context.Context, facilityID uuid.UUID) ([]ComplianceResult, error)
	GetTrendingData(ctx context.Context, facilityID uuid.UUID, days int) ([]TrendingSeries, error)
	ListInstruments(ctx context.Context, facilityID uuid.UUID) ([]Instrument, error)
	ListCalibrationRecords(ctx context.Context, instrumentID uuid.UUID) ([]CalibrationRecord, error)
	ListInstrumentStatuses(ctx context.Context, facilityID uuid.UUID) ([]InstrumentStatus, error)
	GetOrganizationIDForResult(ctx context.Context, resultID uuid.UUID) (uuid.UUID, error)
	ListAuditLog(ctx context.Context, recordID uuid.UUID) ([]AuditEntry, error)
}
