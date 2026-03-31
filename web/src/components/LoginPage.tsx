import { useState } from "react";
import { login } from "../api/auth";

interface Props {
  onLogin: () => void;
}

export function LoginPage({ onLogin }: Props) {
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    setLoading(true);
    try {
      await login(email, password);
      onLogin();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Login failed");
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="flex min-h-screen items-center justify-center bg-gray-100">
      <div className="w-full max-w-sm">
        <div className="rounded-lg border border-gray-200 bg-white p-8 shadow-sm">
          <h1 className="mb-1 text-xl font-bold text-gray-900">
            Water Quality Data Management
          </h1>
          <p className="mb-6 text-sm text-gray-500">
            Sign in to continue
          </p>

          <form onSubmit={handleSubmit} className="space-y-4">
            <div>
              <label className="mb-1 block text-sm font-medium text-gray-700">
                Email
              </label>
              <input
                type="email"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                className="w-full rounded border border-gray-300 px-3 py-2 text-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
                placeholder="admin@clearwater.gov"
                required
              />
            </div>
            <div>
              <label className="mb-1 block text-sm font-medium text-gray-700">
                Password
              </label>
              <input
                type="password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                className="w-full rounded border border-gray-300 px-3 py-2 text-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
                placeholder="demo1234"
                required
              />
            </div>

            {error && (
              <div className="rounded bg-red-50 px-3 py-2 text-sm text-red-700">
                {error}
              </div>
            )}

            <button
              type="submit"
              disabled={loading}
              className="w-full rounded bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700 disabled:opacity-50"
            >
              {loading ? "Signing in..." : "Sign in"}
            </button>
          </form>

          <div className="mt-6 rounded bg-gray-50 p-3">
            <p className="mb-1 text-xs font-medium text-gray-500">Demo accounts</p>
            <div className="space-y-0.5 text-xs text-gray-500">
              <div><span className="font-mono">admin@clearwater.gov</span> — Admin</div>
              <div><span className="font-mono">jmartinez@clearwater.gov</span> — Operator (WTP)</div>
              <div><span className="font-mono">akim@clearwater.gov</span> — Reviewer</div>
              <div><span className="font-mono">rjohnson@clearwater.gov</span> — Operator (WWTP)</div>
              <div className="mt-1 text-gray-400">Password for all: <span className="font-mono">demo1234</span></div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
