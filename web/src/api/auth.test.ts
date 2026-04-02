import { describe, it, expect } from "vitest";
import {
  hasRole,
  hasAnyRole,
  canReview,
  canApprove,
  primaryRole,
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

describe("hasRole", () => {
  it("returns true when user has the role", () => {
    const user = makeUser([{ role: "admin" }]);
    expect(hasRole(user, "admin")).toBe(true);
  });

  it("returns false when user lacks the role", () => {
    const user = makeUser([{ role: "viewer" }]);
    expect(hasRole(user, "admin")).toBe(false);
  });

  it("returns false for empty roles", () => {
    const user = makeUser([]);
    expect(hasRole(user, "admin")).toBe(false);
  });

  it("handles null roles gracefully", () => {
    const user = { ...makeUser([]), roles: null as any };
    expect(hasRole(user, "admin")).toBe(false);
  });
});

describe("hasAnyRole", () => {
  it("returns true when user has one of the listed roles", () => {
    const user = makeUser([{ role: "operator" }]);
    expect(hasAnyRole(user, ["admin", "operator"])).toBe(true);
  });

  it("returns false when user has none of the listed roles", () => {
    const user = makeUser([{ role: "viewer" }]);
    expect(hasAnyRole(user, ["admin", "operator"])).toBe(false);
  });
});

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
