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

	// Alerts
	ListAllFacilities(ctx context.Context) ([]Facility, error)
	ListFacilityExceedances(ctx context.Context, facilityID uuid.UUID) ([]Exceedance, error)
	CreateAlert(ctx context.Context, p CreateAlertParams) (Alert, bool, error)
	ListAlerts(ctx context.Context, f AlertFilter) ([]Alert, error)
	GetAlert(ctx context.Context, id uuid.UUID) (Alert, error)
	DismissAlert(ctx context.Context, id, userID uuid.UUID) (Alert, error)

	// Attachments & comments
	CreateAttachment(ctx context.Context, p CreateAttachmentParams) (Attachment, error)
	ListAttachments(ctx context.Context, subjectType string, subjectID uuid.UUID) ([]Attachment, error)
	GetAttachment(ctx context.Context, id uuid.UUID) (Attachment, error)
	SoftDeleteAttachment(ctx context.Context, id, userID uuid.UUID) (Attachment, error)
	CreateComment(ctx context.Context, p CreateCommentParams) (Comment, error)
	ListComments(ctx context.Context, subjectType string, subjectID uuid.UUID) ([]Comment, error)
}
