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
      <div className="py-6 text-center text-sm text-gray-400">
        No numeric data points
      </div>
    );
  }

  return (
    <div className="rounded-lg border border-gray-200 bg-white p-4">
      <div className="mb-3 flex items-baseline justify-between">
        <div>
          <h3 className="text-sm font-semibold text-gray-900">
            {series.parameter_name}
          </h3>
          <p className="text-xs text-gray-500">
            {series.location_name} &middot; {series.unit_code}
          </p>
        </div>
        {series.limits.length > 0 && (
          <div className="flex gap-3">
            {series.limits.map((l) => (
              <span
                key={l.limit_type}
                className="flex items-center gap-1 text-xs text-gray-500"
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
      <ResponsiveContainer width="100%" height={220}>
        <LineChart data={data}>
          <CartesianGrid strokeDasharray="3 3" stroke="#f0f0f0" />
          <XAxis
            dataKey="date"
            tick={{ fontSize: 11 }}
            stroke="#9ca3af"
          />
          <YAxis
            tick={{ fontSize: 11 }}
            stroke="#9ca3af"
            width={50}
          />
          <Tooltip
            contentStyle={{
              fontSize: 12,
              borderRadius: 8,
              border: "1px solid #e5e7eb",
            }}
            formatter={(value: number) => [
              `${value} ${series.unit_code}`,
              series.parameter_name,
            ]}
          />
          <Line
            type="monotone"
            dataKey="value"
            stroke="#3b82f6"
            strokeWidth={2}
            dot={{ r: 3, fill: "#3b82f6" }}
            activeDot={{ r: 5 }}
          />
          {series.limits.map((l) => (
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
    <div>
      <div className="mb-4 flex items-center justify-between">
        <h2 className="text-lg font-semibold text-gray-900">
          Parameter Trends
        </h2>
        <select
          value={days}
          onChange={(e) => setDays(Number(e.target.value))}
          className="rounded border border-gray-300 px-3 py-1.5 text-sm"
        >
          <option value={7}>Last 7 days</option>
          <option value={30}>Last 30 days</option>
          <option value={90}>Last 90 days</option>
          <option value={365}>Last year</option>
        </select>
      </div>

      {isLoading ? (
        <div className="py-8 text-center text-sm text-gray-500">Loading...</div>
      ) : seriesList?.length === 0 ? (
        <div className="py-8 text-center text-sm text-gray-400">
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
