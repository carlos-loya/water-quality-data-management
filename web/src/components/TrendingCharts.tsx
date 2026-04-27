import { useQuery } from "@tanstack/react-query";
import { useState } from "react";
import {
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ReferenceLine,
  ResponsiveContainer,
} from "recharts";
import { TrendingUp } from "lucide-react";
import { api } from "../api/client";
import type { TrendingSeries } from "../api/types";

interface Props {
  facilityId: string;
}

const LIMIT_COLORS: Record<string, string> = {
  daily_max: "#ef4444",
  daily_min: "#3b82f6",
  monthly_avg: "#f97316",
  weekly_avg: "#a855f7",
  instantaneous_max: "#dc2626",
};

function formatLimitLabel(type: string): string {
  return type
    .split("_")
    .map((w) => w.charAt(0).toUpperCase() + w.slice(1))
    .join(" ");
}

function SeriesChart({ series }: { series: TrendingSeries }) {
  const data = series.points
    .filter((p) => p.result_value !== null)
    .map((p) => ({
      date: new Date(p.collected_at).toLocaleDateString(),
      timestamp: new Date(p.collected_at).getTime(),
      value: p.result_value,
    }));

  if (data.length === 0) {
    return (
      <div className="rounded-xl border border-slate-200 bg-white p-6 text-center text-sm text-slate-400 shadow-sm">
        No numeric data points for {series.parameter_name} at {series.location_name}
      </div>
    );
  }

  return (
    <div className="rounded-xl border border-slate-200 bg-white p-5 shadow-sm">
      <div className="mb-4 flex items-start justify-between gap-4">
        <div>
          <h3 className="text-sm font-semibold text-slate-900">
            {series.parameter_name}
          </h3>
          <p className="text-xs text-slate-500">
            {series.location_name} · {series.unit_code}
          </p>
        </div>
        {series.limits?.length > 0 && (
          <div className="flex flex-wrap justify-end gap-x-3 gap-y-1">
            {series.limits.map((l) => (
              <span
                key={l.limit_type}
                className="flex items-center gap-1.5 text-xs text-slate-600"
              >
                <span
                  className="inline-block h-2 w-2 rounded-full"
                  style={{
                    backgroundColor: LIMIT_COLORS[l.limit_type] ?? "#9ca3af",
                  }}
                />
                {formatLimitLabel(l.limit_type)}: {l.limit_value}
              </span>
            ))}
          </div>
        )}
      </div>
      <ResponsiveContainer width="100%" height={240}>
        <LineChart data={data} margin={{ top: 5, right: 8, left: 0, bottom: 0 }}>
          <CartesianGrid strokeDasharray="3 3" stroke="#e2e8f0" />
          <XAxis
            dataKey="date"
            tick={{ fontSize: 11, fill: "#64748b" }}
            stroke="#cbd5e1"
          />
          <YAxis
            tick={{ fontSize: 11, fill: "#64748b" }}
            stroke="#cbd5e1"
            width={50}
          />
          <Tooltip
            contentStyle={{
              fontSize: 12,
              borderRadius: 8,
              border: "1px solid #e2e8f0",
              boxShadow: "0 4px 6px -1px rgba(0,0,0,0.05)",
            }}
            formatter={(value) => [
              `${value} ${series.unit_code}`,
              series.parameter_name,
            ]}
          />
          <Line
            type="monotone"
            dataKey="value"
            stroke="#2563eb"
            strokeWidth={2}
            dot={{ r: 3, fill: "#2563eb" }}
            activeDot={{ r: 5 }}
          />
          {(series.limits ?? []).map((l) => (
            <ReferenceLine
              key={l.limit_type}
              y={l.limit_value}
              stroke={LIMIT_COLORS[l.limit_type] ?? "#9ca3af"}
              strokeDasharray="6 3"
              strokeWidth={1.5}
            />
          ))}
        </LineChart>
      </ResponsiveContainer>
    </div>
  );
}

export function TrendingCharts({ facilityId }: Props) {
  const [days, setDays] = useState(30);

  const { data: seriesList, isLoading } = useQuery({
    queryKey: ["trending", facilityId, days],
    queryFn: () => api.getTrending(facilityId, days),
  });

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between rounded-xl border border-slate-200 bg-white px-4 py-3 shadow-sm">
        <div className="flex items-center gap-2 text-sm text-slate-600">
          <TrendingUp className="h-4 w-4 text-slate-400" />
          {seriesList?.length ?? 0} parameter series
        </div>
        <select
          value={days}
          onChange={(e) => setDays(Number(e.target.value))}
          className="rounded-lg border border-slate-200 bg-white px-3 py-1.5 text-sm text-slate-700 focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
        >
          <option value={7}>Last 7 days</option>
          <option value={30}>Last 30 days</option>
          <option value={90}>Last 90 days</option>
          <option value={365}>Last year</option>
        </select>
      </div>

      {isLoading ? (
        <div className="rounded-xl border border-slate-200 bg-white py-12 text-center text-sm text-slate-500 shadow-sm">
          Loading...
        </div>
      ) : seriesList?.length === 0 ? (
        <div className="rounded-xl border border-dashed border-slate-300 bg-white py-12 text-center text-sm text-slate-400">
          No trending data for this period
        </div>
      ) : (
        <div className="grid gap-4">
          {seriesList?.map((s) => (
            <SeriesChart
              key={`${s.parameter_code}-${s.location_name}`}
              series={s}
            />
          ))}
        </div>
      )}
    </div>
  );
}
