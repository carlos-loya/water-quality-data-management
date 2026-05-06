import { useQuery } from "@tanstack/react-query";
import {
  Bar,
  BarChart,
  CartesianGrid,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from "recharts";
import {
  AlertTriangle,
  ArrowRight,
  ClipboardCheck,
  Droplets,
  FlaskConical,
  ShieldCheck,
} from "lucide-react";
import { api } from "../api/client";
import type { RecentSampleResult } from "../api/types";
import { StatusBadge } from "./StatusBadge";

interface Props {
  facilityId: string;
  onNavigate: (
    tab: "results" | "trending" | "compliance" | "instruments" | "alerts"
  ) => void;
}

export function OverviewPage({ facilityId, onNavigate }: Props) {
  const overviewQuery = useQuery({
    queryKey: ["overview", facilityId],
    queryFn: () => api.getFacilityOverview(facilityId),
  });

  const alertsQuery = useQuery({
    queryKey: ["alerts", { facility_id: facilityId, dismissed: false }],
    queryFn: () => api.listAlerts({ facility_id: facilityId, dismissed: false }),
  });

  const instrumentsQuery = useQuery({
    queryKey: ["instruments", facilityId],
    queryFn: () => api.listInstrumentStatuses(facilityId),
  });

  const overview = overviewQuery.data;
  const alerts = alertsQuery.data ?? [];
  const instruments = instrumentsQuery.data ?? [];

  const overdueCalibrations = instruments.filter(
    (i) => i.calibration_status === "overdue"
  ).length;
  const criticalAlerts = alerts.filter((a) => a.severity === "critical").length;

  const isLoading =
    overviewQuery.isLoading ||
    alertsQuery.isLoading ||
    instrumentsQuery.isLoading;

  if (isLoading) {
    return (
      <div className="rounded-xl border border-slate-200 bg-white py-16 text-center text-sm text-slate-500 shadow-sm">
        Loading overview...
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-4">
        <KpiCard
          icon={Droplets}
          label="Samples (7 days)"
          value={overview?.samples_last_7d ?? 0}
          subtext={`${overview?.samples_last_30d ?? 0} in last 30 days`}
          tone="blue"
        />
        <KpiCard
          icon={ClipboardCheck}
          label="Pending review"
          value={overview?.pending_review ?? 0}
          subtext={`${overview?.pending_approval ?? 0} awaiting approval`}
          tone="amber"
          onClick={() => onNavigate("results")}
        />
        <KpiCard
          icon={AlertTriangle}
          label="Open alerts"
          value={alerts.length}
          subtext={
            criticalAlerts > 0
              ? `${criticalAlerts} critical`
              : "No critical alerts"
          }
          tone={criticalAlerts > 0 ? "red" : "slate"}
          onClick={() => onNavigate("alerts")}
        />
        <KpiCard
          icon={FlaskConical}
          label="Overdue calibrations"
          value={overdueCalibrations}
          subtext={`${instruments.length} instruments tracked`}
          tone={overdueCalibrations > 0 ? "red" : "emerald"}
          onClick={() => onNavigate("instruments")}
        />
      </div>

      <div className="grid grid-cols-1 gap-4 lg:grid-cols-3">
        <div className="lg:col-span-2">
          <SamplesByDayCard buckets={overview?.samples_by_day ?? []} />
        </div>
        <div>
          <CompliancePulseCard
            critical={criticalAlerts}
            warning={alerts.filter((a) => a.severity === "warning").length}
            overdueCalibrations={overdueCalibrations}
          />
        </div>
      </div>

      <RecentResultsCard
        recent={overview?.recent_results ?? []}
        onSeeAll={() => onNavigate("results")}
      />
    </div>
  );
}

const KPI_TONES: Record<
  string,
  { iconBg: string; iconText: string; valueText: string }
> = {
  blue: {
    iconBg: "bg-blue-50",
    iconText: "text-blue-600",
    valueText: "text-slate-900",
  },
  amber: {
    iconBg: "bg-amber-50",
    iconText: "text-amber-600",
    valueText: "text-slate-900",
  },
  red: {
    iconBg: "bg-red-50",
    iconText: "text-red-600",
    valueText: "text-red-700",
  },
  emerald: {
    iconBg: "bg-emerald-50",
    iconText: "text-emerald-600",
    valueText: "text-slate-900",
  },
  slate: {
    iconBg: "bg-slate-100",
    iconText: "text-slate-500",
    valueText: "text-slate-900",
  },
};

function KpiCard({
  icon: Icon,
  label,
  value,
  subtext,
  tone,
  onClick,
}: {
  icon: typeof Droplets;
  label: string;
  value: number;
  subtext: string;
  tone: keyof typeof KPI_TONES;
  onClick?: () => void;
}) {
  const t = KPI_TONES[tone];
  const interactive = onClick !== undefined;
  return (
    <button
      type="button"
      onClick={onClick}
      disabled={!interactive}
      className={`group flex flex-col rounded-xl border border-slate-200 bg-white p-4 text-left shadow-sm transition ${
        interactive
          ? "hover:border-slate-300 hover:shadow-md"
          : "cursor-default"
      }`}
    >
      <div className="flex items-center justify-between">
        <div
          className={`flex h-9 w-9 items-center justify-center rounded-lg ${t.iconBg} ${t.iconText}`}
        >
          <Icon className="h-5 w-5" />
        </div>
        {interactive && (
          <ArrowRight className="h-4 w-4 text-slate-300 transition group-hover:translate-x-0.5 group-hover:text-slate-500" />
        )}
      </div>
      <div className="mt-3 text-xs font-medium uppercase tracking-wide text-slate-500">
        {label}
      </div>
      <div className={`mt-0.5 text-3xl font-semibold tracking-tight ${t.valueText}`}>
        {value}
      </div>
      <div className="mt-1 text-xs text-slate-500">{subtext}</div>
    </button>
  );
}

function SamplesByDayCard({
  buckets,
}: {
  buckets: { day: string; count: number }[];
}) {
  // Fill in missing days so the bar chart has continuous spacing.
  const data = fillDays(buckets, 30);

  const total = data.reduce((sum, d) => sum + d.count, 0);

  return (
    <div className="rounded-xl border border-slate-200 bg-white p-5 shadow-sm">
      <div className="mb-4 flex items-baseline justify-between">
        <div>
          <h3 className="text-sm font-semibold text-slate-900">
            Sampling activity
          </h3>
          <p className="text-xs text-slate-500">Last 30 days</p>
        </div>
        <div className="text-right">
          <div className="text-2xl font-semibold text-slate-900">{total}</div>
          <div className="text-xs text-slate-500">total samples</div>
        </div>
      </div>
      {total === 0 ? (
        <div className="flex h-48 items-center justify-center text-sm text-slate-400">
          No samples in the last 30 days
        </div>
      ) : (
        <ResponsiveContainer width="100%" height={200}>
          <BarChart data={data} margin={{ top: 5, right: 4, left: 0, bottom: 0 }}>
            <CartesianGrid strokeDasharray="3 3" stroke="#e2e8f0" vertical={false} />
            <XAxis
              dataKey="label"
              tick={{ fontSize: 10, fill: "#64748b" }}
              stroke="#cbd5e1"
              interval="preserveStartEnd"
            />
            <YAxis
              tick={{ fontSize: 10, fill: "#64748b" }}
              stroke="#cbd5e1"
              width={28}
              allowDecimals={false}
            />
            <Tooltip
              cursor={{ fill: "#f1f5f9" }}
              contentStyle={{
                fontSize: 12,
                borderRadius: 8,
                border: "1px solid #e2e8f0",
              }}
              formatter={(value) => [`${value} samples`, ""]}
              labelFormatter={(l) => l}
            />
            <Bar dataKey="count" fill="#2563eb" radius={[3, 3, 0, 0]} />
          </BarChart>
        </ResponsiveContainer>
      )}
    </div>
  );
}

function CompliancePulseCard({
  critical,
  warning,
  overdueCalibrations,
}: {
  critical: number;
  warning: number;
  overdueCalibrations: number;
}) {
  const allClear = critical === 0 && warning === 0 && overdueCalibrations === 0;

  return (
    <div className="flex h-full flex-col rounded-xl border border-slate-200 bg-white p-5 shadow-sm">
      <div className="mb-4 flex items-center gap-2">
        <ShieldCheck className="h-4 w-4 text-slate-400" />
        <h3 className="text-sm font-semibold text-slate-900">
          Compliance pulse
        </h3>
      </div>
      {allClear ? (
        <div className="flex flex-1 flex-col items-center justify-center text-center">
          <div className="flex h-12 w-12 items-center justify-center rounded-full bg-emerald-50 text-emerald-600">
            <ShieldCheck className="h-6 w-6" />
          </div>
          <p className="mt-3 text-sm font-medium text-slate-800">All clear</p>
          <p className="mt-0.5 text-xs text-slate-500">
            No active alerts or overdue calibrations
          </p>
        </div>
      ) : (
        <ul className="space-y-2.5 text-sm">
          <PulseRow label="Critical alerts" value={critical} tone="red" />
          <PulseRow label="Warning alerts" value={warning} tone="amber" />
          <PulseRow
            label="Overdue calibrations"
            value={overdueCalibrations}
            tone="red"
          />
        </ul>
      )}
    </div>
  );
}

function PulseRow({
  label,
  value,
  tone,
}: {
  label: string;
  value: number;
  tone: "red" | "amber" | "emerald";
}) {
  const dot =
    tone === "red"
      ? "bg-red-500"
      : tone === "amber"
      ? "bg-amber-500"
      : "bg-emerald-500";
  return (
    <li className="flex items-center justify-between">
      <span className="flex items-center gap-2 text-slate-600">
        <span className={`inline-block h-2 w-2 rounded-full ${dot}`} />
        {label}
      </span>
      <span
        className={`font-semibold ${
          value > 0 ? "text-slate-900" : "text-slate-400"
        }`}
      >
        {value}
      </span>
    </li>
  );
}

function RecentResultsCard({
  recent,
  onSeeAll,
}: {
  recent: RecentSampleResult[];
  onSeeAll: () => void;
}) {
  return (
    <div className="overflow-hidden rounded-xl border border-slate-200 bg-white shadow-sm">
      <div className="flex items-center justify-between border-b border-slate-200 px-4 py-3">
        <h3 className="text-sm font-semibold text-slate-900">Recent results</h3>
        <button
          onClick={onSeeAll}
          className="inline-flex items-center gap-1 text-xs font-medium text-blue-600 hover:text-blue-700"
        >
          View all
          <ArrowRight className="h-3 w-3" />
        </button>
      </div>
      {recent.length === 0 ? (
        <div className="py-12 text-center text-sm text-slate-400">
          No sample results yet
        </div>
      ) : (
        <ul className="divide-y divide-slate-100">
          {recent.map((r) => (
            <li
              key={r.id}
              className="flex items-center gap-4 px-4 py-2.5 hover:bg-slate-50"
            >
              <div className="min-w-0 flex-1">
                <div className="truncate text-sm font-medium text-slate-800">
                  {r.parameter_name}
                </div>
                <div className="truncate text-xs text-slate-500">
                  {r.location_name} · {new Date(r.collected_at).toLocaleString()}
                </div>
              </div>
              <div className="text-right font-mono text-sm text-slate-900">
                {formatRecentValue(r)}{" "}
                <span className="text-xs text-slate-500">{r.unit_code}</span>
              </div>
              <StatusBadge value={r.status} />
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}

function formatRecentValue(r: RecentSampleResult): string {
  if (r.result_qualifier) return `${r.result_qualifier}${r.result_value ?? ""}`;
  return r.result_value !== null ? String(r.result_value) : "—";
}

function fillDays(
  buckets: { day: string; count: number }[],
  days: number
): { label: string; count: number; key: string }[] {
  const map = new Map<string, number>();
  for (const b of buckets) {
    map.set(b.day.slice(0, 10), b.count);
  }
  const out: { label: string; count: number; key: string }[] = [];
  const today = new Date();
  today.setHours(0, 0, 0, 0);
  for (let i = days - 1; i >= 0; i--) {
    const d = new Date(today);
    d.setDate(today.getDate() - i);
    const key = d.toISOString().slice(0, 10);
    out.push({
      key,
      label: d.toLocaleDateString(undefined, { month: "short", day: "numeric" }),
      count: map.get(key) ?? 0,
    });
  }
  return out;
}
