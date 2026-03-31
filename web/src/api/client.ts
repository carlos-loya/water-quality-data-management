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

import { getToken, clearAuth } from "./auth";

const BASE = "/api/v1";

function authHeaders(): Record<string, string> {
  const token = getToken();
  return token ? { Authorization: `Bearer ${token}` } : {};
}

async function handleResponse<T>(res: Response): Promise<T> {
  if (res.status === 401) {
    clearAuth();
    window.location.reload();
    throw new Error("session expired");
  }
  if (!res.ok) {
    const body = await res.json().catch(() => ({}));
    throw new Error(body.error || `HTTP ${res.status}`);
  }
  return res.json();
}

async function get<T>(path: string): Promise<T> {
  const res = await fetch(`${BASE}${path}`, { headers: authHeaders() });
  return handleResponse<T>(res);
}

async function patch<T>(path: string, body: unknown): Promise<T> {
  const res = await fetch(`${BASE}${path}`, {
    method: "PATCH",
    headers: { "Content-Type": "application/json", ...authHeaders() },
    body: JSON.stringify(body),
  });
  return handleResponse<T>(res);
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

  reviewSampleResult(id: string) {
    return patch<SampleResult>(`/sample-results/${id}/review`, {});
  },

  approveSampleResult(id: string) {
    return patch<SampleResult>(`/sample-results/${id}/approve`, {});
  },
};
