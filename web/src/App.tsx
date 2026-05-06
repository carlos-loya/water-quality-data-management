import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import {
  Activity,
  Bell,
  Building2,
  ChevronDown,
  ClipboardCheck,
  Droplets,
  FileBarChart,
  FlaskConical,
  LayoutDashboard,
  LineChart,
  LogOut,
  Settings,
} from "lucide-react";
import { getUser, clearAuth, primaryRole, type AuthUser } from "./api/auth";
import { api } from "./api/client";
import type { Facility } from "./api/types";
import { LoginPage } from "./components/LoginPage";
import { OverviewPage } from "./components/OverviewPage";
import { SampleResultsTable } from "./components/SampleResultsTable";
import { ComplianceView } from "./components/ComplianceView";
import { TrendingCharts } from "./components/TrendingCharts";
import { InstrumentsView } from "./components/InstrumentsView";
import { AlertsView } from "./components/AlertsView";

type Tab =
  | "overview"
  | "results"
  | "trending"
  | "compliance"
  | "instruments"
  | "alerts";

const NAV: { key: Tab; label: string; icon: typeof Activity }[] = [
  { key: "overview", label: "Overview", icon: LayoutDashboard },
  { key: "results", label: "Sample Results", icon: ClipboardCheck },
  { key: "trending", label: "Trending", icon: LineChart },
  { key: "compliance", label: "Compliance", icon: FileBarChart },
  { key: "instruments", label: "Instruments", icon: FlaskConical },
  { key: "alerts", label: "Alerts", icon: Bell },
];

function App() {
  const [user, setUser] = useState<AuthUser | null>(getUser);
  const [selectedFacilityId, setSelectedFacilityId] = useState<string | null>(null);
  const [tab, setTab] = useState<Tab>("overview");

  const { data: facilities } = useQuery({
    queryKey: ["facilities", user?.organization_id ?? ""],
    queryFn: () => api.listFacilities(user!.organization_id),
    enabled: !!user,
  });

  // Default to the first facility until the user picks one explicitly. Derived
  // (not stored) so we don't need an effect to keep state in sync with the query.
  const facilityId = selectedFacilityId ?? facilities?.[0]?.id ?? null;

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

  const currentFacility = facilities?.find((f) => f.id === facilityId) ?? null;
  const currentNav = NAV.find((n) => n.key === tab);

  return (
    <div className="flex min-h-screen bg-slate-50 text-slate-900">
      <Sidebar tab={tab} onTab={setTab} alertCount={alertCount} />

      <div className="flex min-w-0 flex-1 flex-col">
        <TopBar
          user={user}
          facilities={facilities ?? []}
          currentFacility={currentFacility}
          onSelectFacility={setSelectedFacilityId}
          alertCount={alertCount}
          onLogout={handleLogout}
          onOpenAlerts={() => setTab("alerts")}
        />

        <main className="flex-1 px-6 py-6 lg:px-8">
          <div className="mx-auto max-w-7xl">
            <PageHeading
              title={currentNav?.label ?? ""}
              icon={currentNav?.icon ?? Activity}
              facilityName={currentFacility?.name}
            />

            {!facilityId ? (
              <EmptyState />
            ) : (
              <>
                {tab === "overview" && (
                  <OverviewPage
                    facilityId={facilityId}
                    onNavigate={(t) => setTab(t)}
                  />
                )}
                {tab === "results" && (
                  <SampleResultsTable
                    facilityId={facilityId}
                    orgId={user.organization_id}
                    user={user}
                  />
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
          </div>
        </main>
      </div>
    </div>
  );
}

function Sidebar({
  tab,
  onTab,
  alertCount,
}: {
  tab: Tab;
  onTab: (t: Tab) => void;
  alertCount: number;
}) {
  return (
    <aside className="hidden w-60 shrink-0 flex-col border-r border-slate-200 bg-white lg:flex">
      <div className="flex h-16 items-center gap-2 border-b border-slate-200 px-5">
        <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-blue-600 text-white">
          <Droplets className="h-5 w-5" />
        </div>
        <div className="leading-tight">
          <div className="text-sm font-semibold text-slate-900">Aquaflow</div>
          <div className="text-xs text-slate-500">Water Quality</div>
        </div>
      </div>

      <nav className="flex-1 space-y-0.5 p-3">
        {NAV.map((item) => {
          const Icon = item.icon;
          const active = item.key === tab;
          const showAlertCount = item.key === "alerts" && alertCount > 0;
          return (
            <button
              key={item.key}
              onClick={() => onTab(item.key)}
              className={`flex w-full items-center gap-3 rounded-lg px-3 py-2 text-sm font-medium transition ${
                active
                  ? "bg-blue-50 text-blue-700"
                  : "text-slate-600 hover:bg-slate-50 hover:text-slate-900"
              }`}
            >
              <Icon
                className={`h-4 w-4 ${active ? "text-blue-600" : "text-slate-400"}`}
              />
              <span className="flex-1 text-left">{item.label}</span>
              {showAlertCount && (
                <span className="rounded-full bg-red-100 px-2 py-0.5 text-xs font-semibold text-red-700">
                  {alertCount}
                </span>
              )}
            </button>
          );
        })}
      </nav>

      <div className="border-t border-slate-200 p-3">
        <button
          disabled
          className="flex w-full items-center gap-3 rounded-lg px-3 py-2 text-sm font-medium text-slate-400"
          title="Settings (coming soon)"
        >
          <Settings className="h-4 w-4" />
          Settings
        </button>
      </div>
    </aside>
  );
}

function TopBar({
  user,
  facilities,
  currentFacility,
  onSelectFacility,
  alertCount,
  onLogout,
  onOpenAlerts,
}: {
  user: AuthUser;
  facilities: Facility[];
  currentFacility: Facility | null;
  onSelectFacility: (id: string) => void;
  alertCount: number;
  onLogout: () => void;
  onOpenAlerts: () => void;
}) {
  const [facilityOpen, setFacilityOpen] = useState(false);
  const [userOpen, setUserOpen] = useState(false);

  return (
    <header className="sticky top-0 z-30 flex h-16 items-center justify-between border-b border-slate-200 bg-white/80 px-6 backdrop-blur lg:px-8">
      <div className="flex items-center gap-3">
        <div className="relative">
          <button
            onClick={() => {
              setFacilityOpen((v) => !v);
              setUserOpen(false);
            }}
            className="flex items-center gap-2 rounded-lg border border-slate-200 bg-white px-3 py-1.5 text-sm font-medium text-slate-700 hover:border-slate-300 hover:bg-slate-50"
          >
            <Building2 className="h-4 w-4 text-slate-400" />
            <span>{currentFacility?.name ?? "Select facility"}</span>
            <ChevronDown className="h-3.5 w-3.5 text-slate-400" />
          </button>
          {facilityOpen && (
            <div className="absolute left-0 top-full z-40 mt-1 w-72 overflow-hidden rounded-lg border border-slate-200 bg-white shadow-lg">
              <div className="border-b border-slate-100 px-3 py-2 text-xs font-semibold uppercase tracking-wide text-slate-400">
                Facilities
              </div>
              <ul className="max-h-72 overflow-y-auto py-1">
                {facilities.map((f) => (
                  <li key={f.id}>
                    <button
                      onClick={() => {
                        onSelectFacility(f.id);
                        setFacilityOpen(false);
                      }}
                      className={`flex w-full items-start gap-2 px-3 py-2 text-left text-sm hover:bg-slate-50 ${
                        f.id === currentFacility?.id ? "bg-blue-50/50" : ""
                      }`}
                    >
                      <Building2
                        className={`mt-0.5 h-4 w-4 shrink-0 ${
                          f.id === currentFacility?.id
                            ? "text-blue-600"
                            : "text-slate-400"
                        }`}
                      />
                      <div className="min-w-0">
                        <div className="truncate font-medium text-slate-800">
                          {f.name}
                        </div>
                        <div className="text-xs capitalize text-slate-500">
                          {f.facility_type.replace("_", " ")}
                        </div>
                      </div>
                    </button>
                  </li>
                ))}
                {facilities.length === 0 && (
                  <li className="px-3 py-3 text-sm text-slate-400">
                    No facilities
                  </li>
                )}
              </ul>
            </div>
          )}
        </div>
      </div>

      <div className="flex items-center gap-2">
        <button
          onClick={onOpenAlerts}
          className="relative flex h-9 w-9 items-center justify-center rounded-lg text-slate-500 hover:bg-slate-100 hover:text-slate-700"
          aria-label="Alerts"
        >
          <Bell className="h-4.5 w-4.5" />
          {alertCount > 0 && (
            <span className="absolute -right-0.5 -top-0.5 flex h-4 min-w-4 items-center justify-center rounded-full bg-red-500 px-1 text-[10px] font-semibold text-white">
              {alertCount > 9 ? "9+" : alertCount}
            </span>
          )}
        </button>

        <div className="relative">
          <button
            onClick={() => {
              setUserOpen((v) => !v);
              setFacilityOpen(false);
            }}
            className="flex items-center gap-2 rounded-lg p-1 pr-2 hover:bg-slate-100"
          >
            <div className="flex h-8 w-8 items-center justify-center rounded-full bg-gradient-to-br from-blue-500 to-indigo-600 text-xs font-semibold text-white">
              {initials(user.name)}
            </div>
            <div className="hidden text-left leading-tight md:block">
              <div className="text-sm font-medium text-slate-800">
                {user.name}
              </div>
              <div className="text-xs capitalize text-slate-500">
                {primaryRole(user)}
              </div>
            </div>
            <ChevronDown className="h-3.5 w-3.5 text-slate-400" />
          </button>
          {userOpen && (
            <div className="absolute right-0 top-full z-40 mt-1 w-56 overflow-hidden rounded-lg border border-slate-200 bg-white shadow-lg">
              <div className="border-b border-slate-100 px-3 py-3">
                <div className="text-sm font-medium text-slate-800">
                  {user.name}
                </div>
                <div className="truncate text-xs text-slate-500">
                  {user.email}
                </div>
              </div>
              <button
                onClick={onLogout}
                className="flex w-full items-center gap-2 px-3 py-2 text-sm text-slate-700 hover:bg-slate-50"
              >
                <LogOut className="h-4 w-4 text-slate-400" />
                Sign out
              </button>
            </div>
          )}
        </div>
      </div>
    </header>
  );
}

function PageHeading({
  title,
  icon: Icon,
  facilityName,
}: {
  title: string;
  icon: typeof Activity;
  facilityName?: string;
}) {
  return (
    <div className="mb-6 flex items-center justify-between">
      <div className="flex items-center gap-3">
        <div className="flex h-9 w-9 items-center justify-center rounded-lg bg-white text-blue-600 ring-1 ring-slate-200">
          <Icon className="h-5 w-5" />
        </div>
        <div>
          <h1 className="text-xl font-semibold tracking-tight text-slate-900">
            {title}
          </h1>
          {facilityName && (
            <p className="text-sm text-slate-500">{facilityName}</p>
          )}
        </div>
      </div>
    </div>
  );
}

function EmptyState() {
  return (
    <div className="flex flex-col items-center justify-center rounded-xl border border-dashed border-slate-300 bg-white py-20 text-center">
      <div className="mb-3 flex h-12 w-12 items-center justify-center rounded-full bg-slate-100 text-slate-400">
        <Building2 className="h-6 w-6" />
      </div>
      <p className="text-sm font-medium text-slate-700">No facility selected</p>
      <p className="mt-1 text-sm text-slate-500">
        Pick a facility from the top bar to get started.
      </p>
    </div>
  );
}

function initials(name: string): string {
  return name
    .split(/\s+/)
    .filter(Boolean)
    .slice(0, 2)
    .map((p) => p[0]!.toUpperCase())
    .join("");
}

export default App;
