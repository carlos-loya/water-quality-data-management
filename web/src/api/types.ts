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
  created_at: string;
  updated_at: string;
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
