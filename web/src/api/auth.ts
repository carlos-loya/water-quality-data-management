const TOKEN_KEY = "wqm_token";
const USER_KEY = "wqm_user";

export interface RoleClaim {
  role: string;
  facility_id?: string;
}

export interface AuthUser {
  id: string;
  organization_id: string;
  email: string;
  name: string;
  roles: RoleClaim[];
}

export function hasRole(user: AuthUser, role: string): boolean {
  return user.roles?.some((r) => r.role === role) ?? false;
}

export function hasAnyRole(user: AuthUser, roles: string[]): boolean {
  return roles.some((role) => hasRole(user, role));
}

export function canReview(user: AuthUser): boolean {
  return hasAnyRole(user, ["admin", "reviewer"]);
}

export function canApprove(user: AuthUser): boolean {
  return hasRole(user, "admin");
}

export function primaryRole(user: AuthUser): string {
  const order = ["admin", "reviewer", "operator", "viewer"];
  for (const role of order) {
    if (hasRole(user, role)) return role;
  }
  return "user";
}

export function getToken(): string | null {
  return localStorage.getItem(TOKEN_KEY);
}

export function getUser(): AuthUser | null {
  const raw = localStorage.getItem(USER_KEY);
  if (!raw) return null;
  try {
    return JSON.parse(raw);
  } catch {
    return null;
  }
}

export function setAuth(token: string, user: AuthUser) {
  localStorage.setItem(TOKEN_KEY, token);
  localStorage.setItem(USER_KEY, JSON.stringify(user));
}

export function clearAuth() {
  localStorage.removeItem(TOKEN_KEY);
  localStorage.removeItem(USER_KEY);
}

export async function login(
  email: string,
  password: string
): Promise<{ token: string; user: AuthUser }> {
  const res = await fetch("/api/v1/auth/login", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ email, password }),
  });
  if (!res.ok) {
    const body = await res.json().catch(() => ({}));
    throw new Error(body.error || `HTTP ${res.status}`);
  }
  const data = await res.json();
  setAuth(data.token, data.user);
  return data;
}
