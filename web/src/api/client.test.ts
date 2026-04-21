import { describe, it, expect, beforeEach, vi } from "vitest";
import { api } from "./client";

// --- localStorage + fetch mocks ---
const store: Record<string, string> = {};

beforeEach(() => {
  for (const key of Object.keys(store)) delete store[key];

  vi.stubGlobal("localStorage", {
    getItem: (key: string) => store[key] ?? null,
    setItem: (key: string, value: string) => {
      store[key] = value;
    },
    removeItem: (key: string) => {
      delete store[key];
    },
  });

  // Stub window.location.reload to prevent errors in tests
  vi.stubGlobal("window", { location: { reload: vi.fn() } });
});

function mockFetch(body: unknown, status = 200) {
  vi.stubGlobal(
    "fetch",
    vi.fn().mockResolvedValue({
      ok: status >= 200 && status < 300,
      status,
      json: () => Promise.resolve(body),
    })
  );
}

function lastFetchCall(): [string, RequestInit] {
  return vi.mocked(fetch).mock.calls[0] as [string, RequestInit];
}

// =========================================================================
// Auth headers
// =========================================================================

describe("auth headers", () => {
  it("includes Bearer token when logged in", async () => {
    store["wqm_token"] = "my-jwt";
    mockFetch([]);
    await api.listFacilities("org-1");
    const [, init] = lastFetchCall();
    expect((init.headers as Record<string, string>)["Authorization"]).toBe("Bearer my-jwt");
  });

  it("omits Authorization when no token", async () => {
    mockFetch([]);
    await api.listFacilities("org-1");
    const [, init] = lastFetchCall();
    expect((init.headers as Record<string, string>)["Authorization"]).toBeUndefined();
  });
});

// =========================================================================
// handleResponse
// =========================================================================

describe("handleResponse", () => {
  it("clears auth and reloads on 401", async () => {
    store["wqm_token"] = "old-token";
    store["wqm_user"] = "{}";
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({
        ok: false,
        status: 401,
        json: () => Promise.resolve({ error: "session expired" }),
      })
    );

    await expect(api.listFacilities("org-1")).rejects.toThrow("session expired");
    expect(store["wqm_token"]).toBeUndefined();
  });

  it("throws with error message from body on non-OK response", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({
        ok: false,
        status: 400,
        json: () => Promise.resolve({ error: "invalid org_id" }),
      })
    );
    await expect(api.listFacilities("bad")).rejects.toThrow("invalid org_id");
  });

  it("throws with HTTP status when body has no error field", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({
        ok: false,
        status: 500,
        json: () => Promise.resolve({}),
      })
    );
    await expect(api.listFacilities("org-1")).rejects.toThrow("HTTP 500");
  });
});

// =========================================================================
// API methods — URL construction
// =========================================================================

describe("api.listFacilities", () => {
  it("constructs correct URL with orgId", async () => {
    mockFetch([]);
    await api.listFacilities("org-abc");
    expect(lastFetchCall()[0]).toBe("/api/v1/organizations/org-abc/facilities");
  });
});

describe("api.listMonitoringLocations", () => {
  it("constructs correct URL with facilityId", async () => {
    mockFetch([]);
    await api.listMonitoringLocations("fac-1");
    expect(lastFetchCall()[0]).toBe("/api/v1/facilities/fac-1/monitoring-locations");
  });
});

describe("api.listParameters", () => {
  it("constructs correct URL with orgId", async () => {
    mockFetch([]);
    await api.listParameters("org-1");
    expect(lastFetchCall()[0]).toBe("/api/v1/organizations/org-1/parameters");
  });
});

describe("api.listUnits", () => {
  it("constructs correct URL with orgId", async () => {
    mockFetch([]);
    await api.listUnits("org-1");
    expect(lastFetchCall()[0]).toBe("/api/v1/organizations/org-1/units");
  });
});

describe("api.listSampleResults", () => {
  it("constructs query string from params", async () => {
    mockFetch([]);
    await api.listSampleResults({ monitoring_location_id: "loc-1", status: "draft" });
    const url = lastFetchCall()[0];
    expect(url).toContain("/api/v1/sample-results?");
    expect(url).toContain("monitoring_location_id=loc-1");
    expect(url).toContain("status=draft");
  });
});

describe("api.createSampleResult", () => {
  it("sends POST with correct body", async () => {
    mockFetch({ id: "new-id" });
    const input: import("./types").CreateSampleResultInput = {
      monitoring_location_id: "loc-1",
      parameter_id: "param-1",
      unit_id: "unit-1",
      collected_at: "2025-06-01T00:00:00Z",
      entered_by: "user-1",
      result_value: 7.2,
      source: "manual",
    };
    await api.createSampleResult(input);
    const [url, init] = lastFetchCall();
    expect(url).toBe("/api/v1/sample-results");
    expect(init.method).toBe("POST");
    expect((init.headers as Record<string, string>)["Content-Type"]).toBe("application/json");
    const body = JSON.parse(init.body as string);
    expect(body.result_value).toBe(7.2);
  });
});

describe("api.evaluateCompliance", () => {
  it("constructs correct URL", async () => {
    mockFetch([]);
    await api.evaluateCompliance("fac-1");
    expect(lastFetchCall()[0]).toBe("/api/v1/facilities/fac-1/compliance");
  });
});

describe("api.getTrending", () => {
  it("includes days parameter", async () => {
    mockFetch([]);
    await api.getTrending("fac-1", 90);
    expect(lastFetchCall()[0]).toBe("/api/v1/facilities/fac-1/trending?days=90");
  });

  it("defaults to 30 days", async () => {
    mockFetch([]);
    await api.getTrending("fac-1");
    expect(lastFetchCall()[0]).toBe("/api/v1/facilities/fac-1/trending?days=30");
  });
});

describe("api.listInstrumentStatuses", () => {
  it("constructs correct URL", async () => {
    mockFetch([]);
    await api.listInstrumentStatuses("fac-1");
    expect(lastFetchCall()[0]).toBe("/api/v1/facilities/fac-1/instruments");
  });
});

describe("api.listCalibrationRecords", () => {
  it("constructs correct URL", async () => {
    mockFetch([]);
    await api.listCalibrationRecords("inst-1");
    expect(lastFetchCall()[0]).toBe("/api/v1/instruments/inst-1/calibrations");
  });
});

describe("api.getAuditLog", () => {
  it("constructs correct URL", async () => {
    mockFetch([]);
    await api.getAuditLog("rec-1");
    expect(lastFetchCall()[0]).toBe("/api/v1/audit-log/rec-1");
  });
});

describe("api.reviewSampleResult", () => {
  it("sends PATCH to correct URL", async () => {
    mockFetch({});
    await api.reviewSampleResult("sr-1");
    const [url, init] = lastFetchCall();
    expect(url).toBe("/api/v1/sample-results/sr-1/review");
    expect(init.method).toBe("PATCH");
  });
});

describe("api.approveSampleResult", () => {
  it("sends PATCH to correct URL", async () => {
    mockFetch({});
    await api.approveSampleResult("sr-1");
    const [url, init] = lastFetchCall();
    expect(url).toBe("/api/v1/sample-results/sr-1/approve");
    expect(init.method).toBe("PATCH");
  });
});

// =========================================================================
// Alerts
// =========================================================================

describe("api.listAlerts", () => {
  it("requests /alerts with no query when filter is empty", async () => {
    mockFetch([]);
    await api.listAlerts();
    expect(lastFetchCall()[0]).toBe("/api/v1/alerts");
  });

  it("includes facility_id filter in query", async () => {
    mockFetch([]);
    await api.listAlerts({ facility_id: "fac-1" });
    const url = lastFetchCall()[0];
    expect(url).toContain("/api/v1/alerts?");
    expect(url).toContain("facility_id=fac-1");
  });

  it("serializes type, dismissed, and limit", async () => {
    mockFetch([]);
    await api.listAlerts({ type: "exceedance", dismissed: false, limit: 25 });
    const url = lastFetchCall()[0];
    expect(url).toContain("type=exceedance");
    expect(url).toContain("dismissed=false");
    expect(url).toContain("limit=25");
  });

  it("includes dismissed=true explicitly", async () => {
    mockFetch([]);
    await api.listAlerts({ dismissed: true });
    expect(lastFetchCall()[0]).toContain("dismissed=true");
  });

  it("sends GET with auth header when token present", async () => {
    store["wqm_token"] = "tok-1";
    mockFetch([]);
    await api.listAlerts({ facility_id: "fac-1" });
    const [, init] = lastFetchCall();
    expect(init.method).toBeUndefined();
    expect((init.headers as Record<string, string>)["Authorization"]).toBe("Bearer tok-1");
  });
});

describe("api.dismissAlert", () => {
  it("sends POST to correct URL", async () => {
    mockFetch({ id: "a-1", dismissed_at: "2026-04-20T00:00:00Z" });
    await api.dismissAlert("a-1");
    const [url, init] = lastFetchCall();
    expect(url).toBe("/api/v1/alerts/a-1/dismiss");
    expect(init.method).toBe("POST");
    expect((init.headers as Record<string, string>)["Content-Type"]).toBe("application/json");
  });

  it("throws error from body on failure", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({
        ok: false,
        status: 409,
        json: () => Promise.resolve({ error: "already dismissed" }),
      })
    );
    await expect(api.dismissAlert("a-1")).rejects.toThrow("already dismissed");
  });
});
