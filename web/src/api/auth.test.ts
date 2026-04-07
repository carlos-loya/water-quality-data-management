import { describe, it, expect, beforeEach, vi } from "vitest";
import {
  hasRole,
  hasAnyRole,
  canReview,
  canApprove,
  primaryRole,
  getToken,
  getUser,
  setAuth,
  clearAuth,
  login,
  type AuthUser,
} from "./auth";

function makeUser(roles: { role: string; facility_id?: string }[]): AuthUser {
  return {
    id: "user-1",
    organization_id: "org-1",
    email: "test@example.com",
    name: "Test User",
    roles,
  };
}

// --- localStorage mock ---
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
});

// =========================================================================
// hasRole
// =========================================================================

describe("hasRole", () => {
  it("returns true when user has the role", () => {
    expect(hasRole(makeUser([{ role: "admin" }]), "admin")).toBe(true);
  });

  it("returns false when user lacks the role", () => {
    expect(hasRole(makeUser([{ role: "viewer" }]), "admin")).toBe(false);
  });

  it("returns false for empty roles", () => {
    expect(hasRole(makeUser([]), "admin")).toBe(false);
  });

  it("handles null roles gracefully", () => {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const user = { ...makeUser([]), roles: null as any };
    expect(hasRole(user, "admin")).toBe(false);
  });
});

// =========================================================================
// hasAnyRole
// =========================================================================

describe("hasAnyRole", () => {
  it("returns true when user has one of the listed roles", () => {
    expect(hasAnyRole(makeUser([{ role: "operator" }]), ["admin", "operator"])).toBe(true);
  });

  it("returns false when user has none of the listed roles", () => {
    expect(hasAnyRole(makeUser([{ role: "viewer" }]), ["admin", "operator"])).toBe(false);
  });
});

// =========================================================================
// canReview / canApprove
// =========================================================================

describe("canReview", () => {
  it("returns true for admin", () => {
    expect(canReview(makeUser([{ role: "admin" }]))).toBe(true);
  });
  it("returns true for reviewer", () => {
    expect(canReview(makeUser([{ role: "reviewer" }]))).toBe(true);
  });
  it("returns false for operator", () => {
    expect(canReview(makeUser([{ role: "operator" }]))).toBe(false);
  });
  it("returns false for viewer", () => {
    expect(canReview(makeUser([{ role: "viewer" }]))).toBe(false);
  });
});

describe("canApprove", () => {
  it("returns true for admin", () => {
    expect(canApprove(makeUser([{ role: "admin" }]))).toBe(true);
  });
  it("returns false for reviewer", () => {
    expect(canApprove(makeUser([{ role: "reviewer" }]))).toBe(false);
  });
  it("returns false for operator", () => {
    expect(canApprove(makeUser([{ role: "operator" }]))).toBe(false);
  });
});

// =========================================================================
// primaryRole
// =========================================================================

describe("primaryRole", () => {
  it("returns admin when user is admin", () => {
    expect(primaryRole(makeUser([{ role: "admin" }]))).toBe("admin");
  });
  it("returns highest priority role when user has multiple", () => {
    const user = makeUser([{ role: "viewer" }, { role: "reviewer" }, { role: "operator" }]);
    expect(primaryRole(user)).toBe("reviewer");
  });
  it("returns operator for operator-only user", () => {
    expect(primaryRole(makeUser([{ role: "operator" }]))).toBe("operator");
  });
  it("returns viewer for viewer-only user", () => {
    expect(primaryRole(makeUser([{ role: "viewer" }]))).toBe("viewer");
  });
  it("returns 'user' when no recognized roles", () => {
    expect(primaryRole(makeUser([]))).toBe("user");
  });
});

// =========================================================================
// getToken / getUser / setAuth / clearAuth
// =========================================================================

describe("getToken", () => {
  it("returns token from localStorage", () => {
    store["wqm_token"] = "my-token";
    expect(getToken()).toBe("my-token");
  });
  it("returns null when no token stored", () => {
    expect(getToken()).toBeNull();
  });
});

describe("getUser", () => {
  it("returns parsed user from localStorage", () => {
    const user = makeUser([{ role: "admin" }]);
    store["wqm_user"] = JSON.stringify(user);
    const result = getUser();
    expect(result).not.toBeNull();
    expect(result!.email).toBe("test@example.com");
    expect(result!.roles).toHaveLength(1);
  });
  it("returns null when no user stored", () => {
    expect(getUser()).toBeNull();
  });
  it("returns null for corrupted JSON in localStorage", () => {
    store["wqm_user"] = "not-json{{{";
    expect(getUser()).toBeNull();
  });
});

describe("setAuth", () => {
  it("stores both token and user", () => {
    const user = makeUser([{ role: "operator" }]);
    setAuth("tok-123", user);
    expect(store["wqm_token"]).toBe("tok-123");
    expect(JSON.parse(store["wqm_user"]).email).toBe("test@example.com");
  });
});

describe("clearAuth", () => {
  it("removes both token and user", () => {
    store["wqm_token"] = "tok";
    store["wqm_user"] = "{}";
    clearAuth();
    expect(store["wqm_token"]).toBeUndefined();
    expect(store["wqm_user"]).toBeUndefined();
  });
});

// =========================================================================
// login
// =========================================================================

describe("login", () => {
  it("sends correct request body and stores auth on success", async () => {
    const mockUser = makeUser([{ role: "admin" }]);
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({
        ok: true,
        json: () => Promise.resolve({ token: "jwt-abc", user: mockUser }),
      })
    );

    const result = await login("test@example.com", "password123");
    expect(result.token).toBe("jwt-abc");
    expect(result.user.email).toBe("test@example.com");

    // Verify it was stored
    expect(store["wqm_token"]).toBe("jwt-abc");
    expect(JSON.parse(store["wqm_user"]).email).toBe("test@example.com");

    // Verify fetch was called correctly
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const fetchCall = (fetch as any).mock.calls[0];
    expect(fetchCall[0]).toBe("/api/v1/auth/login");
    expect(fetchCall[1].method).toBe("POST");
    const body = JSON.parse(fetchCall[1].body);
    expect(body.email).toBe("test@example.com");
    expect(body.password).toBe("password123");
  });

  it("throws on HTTP error with error message from body", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({
        ok: false,
        status: 401,
        json: () => Promise.resolve({ error: "invalid credentials" }),
      })
    );

    await expect(login("bad@test.com", "wrong")).rejects.toThrow("invalid credentials");
  });

  it("throws with HTTP status when body has no error field", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({
        ok: false,
        status: 500,
        json: () => Promise.reject(new Error("parse error")),
      })
    );

    await expect(login("a@b.com", "x")).rejects.toThrow("HTTP 500");
  });
});
