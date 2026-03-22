import type {
  Facility,
  MonitoringLocation,
  Parameter,
  SampleResult,
  ComplianceResult,
  TrendingSeries,
  InstrumentStatus,
  CalibrationRecord,
  AuditEntry,
} from "./types";

const BASE = "/api/v1";

async function get<T>(path: string): Promise<T> {
  const res = await fetch(`${BASE}${path}`);
  if (!res.ok) {
    const body = await res.json().catch(() => ({}));
    throw new Error(body.error || `HTTP ${res.status}`);
  }
  return res.json();
}

async function patch<T>(path: string, body: unknown): Promise<T> {
  const res = await fetch(`${BASE}${path}`, {
    method: "PATCH",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
  if (!res.ok) {
    const data = await res.json().catch(() => ({}));
    throw new Error(data.error || `HTTP ${res.status}`);
  }
  return res.json();
}

export const api = {
  listFacilities(orgId: string) {
    return get<Facility[]>(`/organizations/${orgId}/facilities`);
  },

  listMonitoringLocations(facilityId: string) {
    return get<MonitoringLocation[]>(
      `/facilities/${facilityId}/monitoring-locations`
    );
  },

  listParameters(orgId: string) {
    return get<Parameter[]>(`/organizations/${orgId}/parameters`);
  },

  listSampleResults(params: Record<string, string>) {
    const query = new URLSearchParams(params).toString();
    return get<SampleResult[]>(`/sample-results?${query}`);
  },

  evaluateCompliance(facilityId: string) {
    return get<ComplianceResult[]>(`/facilities/${facilityId}/compliance`);
  },

  getTrending(facilityId: string, days = 30) {
    return get<TrendingSeries[]>(
      `/facilities/${facilityId}/trending?days=${days}`
    );
  },

  listInstrumentStatuses(facilityId: string) {
    return get<InstrumentStatus[]>(`/facilities/${facilityId}/instruments`);
  },

  listCalibrationRecords(instrumentId: string) {
    return get<CalibrationRecord[]>(
      `/instruments/${instrumentId}/calibrations`
    );
  },

  getAuditLog(recordId: string) {
    return get<AuditEntry[]>(`/audit-log/${recordId}`);
  },

  reviewSampleResult(id: string, reviewerId: string) {
    return patch<SampleResult>(`/sample-results/${id}/review`, {
      reviewer_id: reviewerId,
    });
  },

  approveSampleResult(id: string, approverId: string) {
    return patch<SampleResult>(`/sample-results/${id}/approve`, {
      approver_id: approverId,
    });
  },
};
