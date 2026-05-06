import { useState } from "react";
import { Droplets } from "lucide-react";
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
    <div className="grid min-h-screen lg:grid-cols-2">
      <div className="flex items-center justify-center bg-slate-50 px-4 py-10">
        <div className="w-full max-w-sm">
          <div className="mb-6 flex items-center gap-2">
            <div className="flex h-9 w-9 items-center justify-center rounded-lg bg-blue-600 text-white">
              <Droplets className="h-5 w-5" />
            </div>
            <div>
              <div className="text-base font-semibold text-slate-900">Aquaflow</div>
              <div className="text-xs text-slate-500">Water Quality Data Management</div>
            </div>
          </div>

          <div className="rounded-xl border border-slate-200 bg-white p-6 shadow-sm">
            <h1 className="text-lg font-semibold text-slate-900">
              Sign in to your account
            </h1>
            <p className="mb-5 mt-1 text-sm text-slate-500">
              Use your utility credentials to continue.
            </p>

            <form onSubmit={handleSubmit} className="space-y-4">
              <div>
                <label className="mb-1 block text-xs font-medium text-slate-600">
                  Email
                </label>
                <input
                  type="email"
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  className="w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
                  placeholder="admin@clearwater.gov"
                  required
                  autoFocus
                />
              </div>
              <div>
                <label className="mb-1 block text-xs font-medium text-slate-600">
                  Password
                </label>
                <input
                  type="password"
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  className="w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
                  placeholder="••••••••"
                  required
                />
              </div>

              {error && (
                <div className="rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700">
                  {error}
                </div>
              )}

              <button
                type="submit"
                disabled={loading}
                className="w-full rounded-lg bg-blue-600 px-4 py-2 text-sm font-medium text-white shadow-sm hover:bg-blue-700 disabled:opacity-50"
              >
                {loading ? "Signing in..." : "Sign in"}
              </button>
            </form>

            <div className="mt-6 rounded-lg border border-slate-200 bg-slate-50 p-3">
              <p className="mb-1.5 text-xs font-semibold uppercase tracking-wide text-slate-500">
                Demo accounts
              </p>
              <ul className="space-y-0.5 text-xs text-slate-600">
                <li><span className="font-mono">admin@clearwater.gov</span> — Admin</li>
                <li><span className="font-mono">jmartinez@clearwater.gov</span> — Operator (WTP)</li>
                <li><span className="font-mono">akim@clearwater.gov</span> — Reviewer</li>
                <li><span className="font-mono">rjohnson@clearwater.gov</span> — Operator (WWTP)</li>
                <li><span className="font-mono">tlee@clearwater.gov</span> — Viewer</li>
              </ul>
              <p className="mt-1.5 text-xs text-slate-500">
                Password for all: <span className="font-mono">demo1234</span>
              </p>
            </div>
          </div>
        </div>
      </div>

      <div className="relative hidden overflow-hidden bg-gradient-to-br from-blue-600 via-blue-700 to-indigo-800 lg:block">
        <div
          className="absolute inset-0 opacity-20"
          style={{
            backgroundImage:
              "radial-gradient(circle at 20% 20%, rgba(255,255,255,0.4) 0, transparent 40%), radial-gradient(circle at 80% 60%, rgba(255,255,255,0.3) 0, transparent 40%)",
          }}
        />
        <div className="relative flex h-full flex-col justify-between p-12 text-white">
          <div className="flex items-center gap-2">
            <Droplets className="h-6 w-6" />
            <span className="text-sm font-semibold tracking-wide">AQUAFLOW</span>
          </div>
          <div>
            <h2 className="text-3xl font-semibold leading-tight">
              Compliance, calibration, and quality data.
              <br />
              All in one place.
            </h2>
            <p className="mt-4 max-w-md text-sm text-blue-100">
              Sample results, permit limit evaluation, instrument calibration,
              and audit-ready reports for municipal water and wastewater operations.
            </p>
          </div>
          <div className="text-xs text-blue-200">
            Built for municipal utilities · TimescaleDB · Event-driven audit trail
          </div>
        </div>
      </div>
    </div>
  );
}
