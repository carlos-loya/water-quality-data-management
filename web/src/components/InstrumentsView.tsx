import { useQuery } from "@tanstack/react-query";
import { useState } from "react";
import { ChevronDown, ChevronRight } from "lucide-react";
import { api } from "../api/client";
import type { InstrumentStatus } from "../api/types";

const STATUS_STYLES: Record<string, { cls: string; label: string }> = {
  overdue: { cls: "bg-red-50 text-red-700 ring-red-600/20", label: "Overdue" },
  due_soon: { cls: "bg-amber-50 text-amber-700 ring-amber-600/20", label: "Due soon" },
  current: { cls: "bg-emerald-50 text-emerald-700 ring-emerald-600/20", label: "Current" },
  no_schedule: { cls: "bg-slate-50 text-slate-600 ring-slate-500/20", label: "No schedule" },
};

function CalStatusBadge({ status }: { status: string }) {
  const s = STATUS_STYLES[status] ?? STATUS_STYLES.no_schedule;
  return (
    <span
      className={`inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium ring-1 ring-inset ${s.cls}`}
    >
      {s.label}
    </span>
  );
}

function CalibrationHistory({
  instrumentId,
  instrumentName,
}: {
  instrumentId: string;
  instrumentName: string;
}) {
  const { data: records, isLoading } = useQuery({
    queryKey: ["calibrations", instrumentId],
    queryFn: () => api.listCalibrationRecords(instrumentId),
  });

  return (
    <div className="rounded-xl border border-slate-200 bg-white p-5 shadow-sm">
      <h4 className="mb-3 text-sm font-semibold text-slate-700">
        Calibration history — {instrumentName}
      </h4>
      {isLoading ? (
        <div className="text-sm text-slate-500">Loading...</div>
      ) : records?.length === 0 ? (
        <div className="text-sm text-slate-400">No calibration records</div>
      ) : (
        <div className="space-y-2">
          {records?.map((r) => (
            <div
              key={r.id}
              className="rounded-lg border border-slate-200 bg-slate-50 p-3"
            >
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-2">
                  <span className="text-sm font-medium capitalize text-slate-800">
                    {r.calibration_type}
                  </span>
                  <span
                    className={`inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium ring-1 ring-inset ${
                      r.status === "pass"
                        ? "bg-emerald-50 text-emerald-700 ring-emerald-600/20"
                        : "bg-red-50 text-red-700 ring-red-600/20"
                    }`}
                  >
                    {r.status.toUpperCase()}
                  </span>
                </div>
                <span className="text-xs text-slate-500">
                  {new Date(r.performed_at).toLocaleString()}
                </span>
              </div>
              <div className="mt-1.5 flex flex-wrap gap-x-4 gap-y-1 text-xs text-slate-500">
                {r.pre_value != null && r.post_value != null && (
                  <span>
                    Pre: {r.pre_value} → Post: {r.post_value}
                  </span>
                )}
                {r.method_reference && <span>Method: {r.method_reference}</span>}
                {r.due_at && (
                  <span>Next due: {new Date(r.due_at).toLocaleDateString()}</span>
                )}
              </div>
              {r.corrective_action && (
                <div className="mt-1.5 text-xs text-red-600">
                  Corrective action: {r.corrective_action}
                </div>
              )}
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

interface Props {
  facilityId: string;
}

export function InstrumentsView({ facilityId }: Props) {
  const [expandedId, setExpandedId] = useState<string | null>(null);

  const { data: instruments, isLoading } = useQuery({
    queryKey: ["instruments", facilityId],
    queryFn: () => api.listInstrumentStatuses(facilityId),
  });

  if (isLoading) {
    return (
      <div className="rounded-xl border border-slate-200 bg-white py-12 text-center text-sm text-slate-500 shadow-sm">
        Loading...
      </div>
    );
  }

  const overdueCount = instruments?.filter((i) => i.calibration_status === "overdue").length ?? 0;
  const dueSoonCount = instruments?.filter((i) => i.calibration_status === "due_soon").length ?? 0;
  const currentCount = instruments?.filter((i) => i.calibration_status === "current").length ?? 0;

  return (
    <div className="space-y-4">
      <div className="grid grid-cols-1 gap-3 sm:grid-cols-3">
        <SummaryStat label="Overdue" value={overdueCount} tone="red" />
        <SummaryStat label="Due soon" value={dueSoonCount} tone="amber" />
        <SummaryStat label="Current" value={currentCount} tone="emerald" />
      </div>

      <div className="overflow-hidden rounded-xl border border-slate-200 bg-white shadow-sm">
        <div className="overflow-x-auto">
          <table className="min-w-full divide-y divide-slate-200">
            <thead className="bg-slate-50">
              <tr>
                <th className="w-8 px-2 py-2.5"></th>
                <th className="px-4 py-2.5 text-left text-xs font-semibold uppercase tracking-wide text-slate-500">
                  Instrument
                </th>
                <th className="px-4 py-2.5 text-left text-xs font-semibold uppercase tracking-wide text-slate-500">
                  Type
                </th>
                <th className="px-4 py-2.5 text-left text-xs font-semibold uppercase tracking-wide text-slate-500">
                  Serial / Make
                </th>
                <th className="px-4 py-2.5 text-left text-xs font-semibold uppercase tracking-wide text-slate-500">
                  Last calibration
                </th>
                <th className="px-4 py-2.5 text-left text-xs font-semibold uppercase tracking-wide text-slate-500">
                  Next due
                </th>
                <th className="px-4 py-2.5 text-center text-xs font-semibold uppercase tracking-wide text-slate-500">
                  Status
                </th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-100 bg-white">
              {instruments?.map((inst) => (
                <TableRow
                  key={inst.id}
                  instrument={inst}
                  expanded={expandedId === inst.id}
                  onToggle={() =>
                    setExpandedId(expandedId === inst.id ? null : inst.id)
                  }
                />
              ))}
              {instruments?.length === 0 && (
                <tr>
                  <td colSpan={7} className="py-12 text-center text-sm text-slate-400">
                    No instruments configured
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
      </div>

      {expandedId && (
        <CalibrationHistory
          instrumentId={expandedId}
          instrumentName={instruments?.find((i) => i.id === expandedId)?.name ?? ""}
        />
      )}
    </div>
  );
}

function TableRow({
  instrument: inst,
  expanded,
  onToggle,
}: {
  instrument: InstrumentStatus;
  expanded: boolean;
  onToggle: () => void;
}) {
  return (
    <tr
      className={`cursor-pointer transition hover:bg-slate-50 ${
        inst.calibration_status === "overdue" ? "bg-red-50/40" : ""
      } ${expanded ? "bg-blue-50/30" : ""}`}
      onClick={onToggle}
    >
      <td className="px-2 py-2.5 text-center text-slate-400">
        {expanded ? (
          <ChevronDown className="mx-auto h-4 w-4" />
        ) : (
          <ChevronRight className="mx-auto h-4 w-4" />
        )}
      </td>
      <td className="px-4 py-2.5 text-sm font-medium text-slate-900">
        {inst.name}
      </td>
      <td className="px-4 py-2.5 text-sm text-slate-600">{inst.instrument_type}</td>
      <td className="px-4 py-2.5 text-sm text-slate-600">
        {inst.serial_number && (
          <div className="font-mono text-xs">{inst.serial_number}</div>
        )}
        {inst.manufacturer && inst.model && (
          <div className="text-xs text-slate-400">
            {inst.manufacturer} {inst.model}
          </div>
        )}
      </td>
      <td className="px-4 py-2.5 text-sm text-slate-600">
        {inst.last_performed_at ? (
          <div>
            <div>{new Date(inst.last_performed_at).toLocaleDateString()}</div>
            <div className="text-xs text-slate-400">
              {inst.last_calibration_type} — {inst.last_status}
            </div>
          </div>
        ) : (
          <span className="text-slate-400">—</span>
        )}
      </td>
      <td className="px-4 py-2.5 text-sm text-slate-600">
        {inst.due_at ? new Date(inst.due_at).toLocaleDateString() : "—"}
      </td>
      <td className="px-4 py-2.5 text-center">
        <CalStatusBadge status={inst.calibration_status} />
      </td>
    </tr>
  );
}

const STAT_TONES: Record<string, { text: string; bar: string }> = {
  red: { text: "text-red-700", bar: "bg-red-500" },
  amber: { text: "text-amber-700", bar: "bg-amber-500" },
  emerald: { text: "text-emerald-700", bar: "bg-emerald-500" },
};

function SummaryStat({
  label,
  value,
  tone,
}: {
  label: string;
  value: number;
  tone: keyof typeof STAT_TONES;
}) {
  const t = STAT_TONES[tone];
  return (
    <div className="overflow-hidden rounded-xl border border-slate-200 bg-white shadow-sm">
      <div className={`h-1 ${t.bar}`} />
      <div className="p-4">
        <div className="text-xs font-medium uppercase tracking-wide text-slate-500">
          {label}
        </div>
        <div className={`mt-1 text-2xl font-semibold ${t.text}`}>{value}</div>
      </div>
    </div>
  );
}
