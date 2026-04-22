import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { getUser, clearAuth, primaryRole, type AuthUser } from "./api/auth";
import { api } from "./api/client";
import { LoginPage } from "./components/LoginPage";
import { FacilitySelector } from "./components/FacilitySelector";
import { SampleResultsTable } from "./components/SampleResultsTable";
import { ComplianceView } from "./components/ComplianceView";
import { TrendingCharts } from "./components/TrendingCharts";
import { InstrumentsView } from "./components/InstrumentsView";
import { AlertsView } from "./components/AlertsView";

type Tab = "results" | "trending" | "compliance" | "instruments" | "alerts";

const TABS: { key: Tab; label: string }[] = [
  { key: "results", label: "Sample Results" },
  { key: "trending", label: "Trending" },
  { key: "compliance", label: "Compliance" },
  { key: "instruments", label: "Instruments" },
  { key: "alerts", label: "Alerts" },
];

function App() {
  const [user, setUser] = useState<AuthUser | null>(getUser);
  const [facilityId, setFacilityId] = useState<string | null>(null);
  const [tab, setTab] = useState<Tab>("results");

  const { data: activeAlerts } = useQuery({
    queryKey: ["alerts", { facility_id: facilityId ?? "", dismissed: false }],
    queryFn: () =>
      api.listAlerts({ facility_id: facilityId ?? undefined, dismissed: false }),
    enabled: !!user && !!facilityId,
    refetchInterval: 60_000,
  });
  const alertCount = activeAlerts?.length ?? 0;

  if (!user) {
    return <LoginPage onLogin={() => setUser(getUser())} />;
  }

  function handleLogout() {
    clearAuth();
    setUser(null);
  }

  return (
    <div className="min-h-screen bg-gray-100">
      <header className="border-b border-gray-200 bg-white">
        <div className="mx-auto flex max-w-6xl items-center justify-between px-4 py-4">
          <div>
            <h1 className="text-xl font-bold text-gray-900">
              Water Quality Data Management
            </h1>
            <p className="text-sm text-gray-500">
              Utility compliance and operations platform
            </p>
          </div>
          <div className="flex items-center gap-3">
            <div className="text-right">
              <div className="flex items-center gap-2 justify-end">
                <span className="text-sm font-medium text-gray-900">{user.name}</span>
                <span className="rounded-full bg-blue-100 px-2 py-0.5 text-xs font-medium text-blue-700 capitalize">
                  {primaryRole(user)}
                </span>
              </div>
              <div className="text-xs text-gray-500">{user.email}</div>
            </div>
            <button
              onClick={handleLogout}
              className="rounded border border-gray-300 px-3 py-1.5 text-sm text-gray-600 hover:bg-gray-50"
            >
              Sign out
            </button>
          </div>
        </div>
      </header>

      <main className="mx-auto max-w-6xl px-4 py-6">
        <section className="mb-6">
          <h2 className="mb-2 text-sm font-medium uppercase text-gray-500">
            Facility
          </h2>
          <FacilitySelector
            orgId={user.organization_id}
            selectedId={facilityId}
            onSelect={setFacilityId}
          />
        </section>

        {facilityId && (
          <>
            <div className="mb-4 flex gap-1 border-b border-gray-200">
              {TABS.map((t) => (
                <button
                  key={t.key}
                  onClick={() => setTab(t.key)}
                  className={`flex items-center gap-2 px-4 py-2 text-sm font-medium ${
                    tab === t.key
                      ? "border-b-2 border-blue-500 text-blue-600"
                      : "text-gray-500 hover:text-gray-700"
                  }`}
                >
                  {t.label}
                  {t.key === "alerts" && alertCount > 0 && (
                    <span className="rounded-full bg-red-100 px-2 py-0.5 text-xs font-medium text-red-700">
                      {alertCount}
                    </span>
                  )}
                </button>
              ))}
            </div>

            {tab === "results" && (
              <SampleResultsTable facilityId={facilityId} orgId={user.organization_id} user={user} />
            )}
            {tab === "trending" && (
              <TrendingCharts facilityId={facilityId} />
            )}
            {tab === "compliance" && (
              <ComplianceView facilityId={facilityId} />
            )}
            {tab === "instruments" && (
              <InstrumentsView facilityId={facilityId} />
            )}
            {tab === "alerts" && (
              <AlertsView facilityId={facilityId} user={user} />
            )}
          </>
        )}

        {!facilityId && (
          <div className="mt-12 text-center text-gray-400">
            Select a facility to get started
          </div>
        )}
      </main>
    </div>
  );
}

export default App;
