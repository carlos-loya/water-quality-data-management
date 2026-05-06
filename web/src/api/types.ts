export interface Facility {
  id: string;
  organization_id: string;
  name: string;
  facility_type: string;
  address?: string;
  latitude?: number;
  longitude?: number;
  active: boolean;
  created_at: string;
  updated_at: string;
}

export interface MonitoringLocation {
  id: string;
  facility_id: string;
  name: string;
  description?: string;
  location_type?: string;
  active: boolean;
  created_at: string;
  updated_at: string;
}

export interface Parameter {
  id: string;
  organization_id: string;
  code: string;
  name: string;
  description?: string;
  default_unit_id?: string;
  category?: string;
  active: boolean;
  created_at: string;
  updated_at: string;
}

export interface UnitOfMeasure {
  id: string;
  code: string;
  name: string;
}

export interface ValidationRule {
  id: string;
  parameter_id: string;
  min_value: number | null;
  max_value: number | null;
  precision_digits: number | null;
  is_required: boolean;
  active: boolean;
}

export interface CreateSampleResultInput {
  monitoring_location_id: string;
  parameter_id: string;
  unit_id: string;
  collected_at: string;
  result_value?: number | null;
  result_qualifier?: string;
  detection_limit?: number;
  entered_by: string;
  source: string;
  notes?: string;
  override_reason?: string;
}

export interface SampleResult {
  id: string;
  monitoring_location_id: string;
  parameter_id: string;
  method_id?: string;
  unit_id: string;
  collected_at: string;
  analyzed_at?: string;
  result_value: number | null;
  result_qualifier?: string;
  detection_limit?: number;
  status: "draft" | "reviewed" | "approved";
  entered_by: string;
  entered_at: string;
  reviewed_by?: string;
  reviewed_at?: string;
  approved_by?: string;
  approved_at?: string;
  source: string;
  source_reference?: string;
  notes?: string;
  override_reason?: string;
  created_at: string;
  updated_at: string;
}

export interface Attachment {
  id: string;
  subject_type: string;
  subject_id: string;
  filename: string;
  content_type: string;
  size_bytes: number;
  storage_key: string;
  uploaded_by: string;
  uploaded_at: string;
  deleted_at?: string;
  deleted_by?: string;
}

export interface Comment {
  id: string;
  subject_type: string;
  subject_id: string;
  author_id: string;
  body: string;
  created_at: string;
}

export interface ComplianceResult {
  facility_name: string;
  location_name: string;
  parameter_code: string;
  parameter_name: string;
  result_value: number | null;
  result_qualifier?: string;
  unit_code: string;
  collected_at: string;
  status: string;
  limit_type: string;
  limit_value: number;
  compliance: string;
}

export interface TrendingPoint {
  collected_at: string;
  result_value: number | null;
  result_qualifier?: string;
  location_name: string;
  parameter_code: string;
  parameter_name: string;
  unit_code: string;
}

export interface TrendingLimit {
  limit_type: string;
  limit_value: number;
}

export interface TrendingSeries {
  parameter_code: string;
  parameter_name: string;
  location_name: string;
  unit_code: string;
  points: TrendingPoint[];
  limits: TrendingLimit[];
}

export interface InstrumentStatus {
  id: string;
  name: string;
  serial_number?: string;
  instrument_type: string;
  manufacturer?: string;
  model?: string;
  active: boolean;
  last_calibration_type?: string;
  last_performed_at?: string;
  last_status?: string;
  due_at?: string;
  calibration_status: "current" | "due_soon" | "overdue" | "no_schedule";
}

export interface CalibrationRecord {
  id: string;
  instrument_id: string;
  calibration_type: string;
  performed_at: string;
  performed_by: string;
  due_at?: string;
  status: string;
  pre_value?: number;
  post_value?: number;
  method_reference?: string;
  corrective_action?: string;
  notes?: string;
  created_at: string;
  updated_at: string;
}

export interface AuditEntry {
  id: string;
  organization_id: string;
  table_name: string;
  record_id: string;
  action: string;
  old_values?: Record<string, unknown>;
  new_values?: Record<string, unknown>;
  changed_by: string;
  changed_at: string;
  reason?: string;
}

export type AlertType = "exceedance" | "overdue_calibration";
export type AlertSeverity = "info" | "warning" | "critical";
export type AlertSubjectType = "sample_result" | "instrument";

export interface Alert {
  id: string;
  organization_id: string;
  facility_id: string;
  type: AlertType;
  severity: AlertSeverity;
  subject_type: AlertSubjectType;
  subject_id: string;
  message: string;
  details?: Record<string, unknown>;
  created_at: string;
  updated_at: string;
  dismissed_at?: string;
  dismissed_by?: string;
}

export interface AlertFilter {
  facility_id?: string;
  type?: AlertType;
  dismissed?: boolean;
  limit?: number;
}

export interface SampleDayBucket {
  day: string;
  count: number;
}

export interface RecentSampleResult {
  id: string;
  collected_at: string;
  status: "draft" | "reviewed" | "approved";
  result_value: number | null;
  result_qualifier?: string;
  unit_code: string;
  parameter_name: string;
  parameter_code: string;
  location_name: string;
}

export interface FacilityOverview {
  samples_last_7d: number;
  samples_last_30d: number;
  pending_review: number;
  pending_approval: number;
  samples_by_day: SampleDayBucket[];
  recent_results: RecentSampleResult[];
}
