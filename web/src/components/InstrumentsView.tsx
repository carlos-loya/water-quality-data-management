import { useQuery } from "@tanstack/react-query";
import { useState } from "react";
import { api } from "../api/client";
import type { InstrumentStatus } from "../api/types";

const STATUS_STYLES: Record<string, { bg: string; text: string; label: string }> = {
  overdue: { bg: "bg-red-100", text: "text-red-800", label: "Overdue" },
  due_soon: { bg: "bg-yellow-100", text: "text-yellow-800", label: "Due Soon" },
  current: { bg: "bg-green-100", text: "text-green-800", label: "Current" },
  no_schedule: { bg: "bg-gray-100", text: "text-gray-600", label: "No Schedule" },
};

function CalStatusBadge({ status }: { status: string }) {
  const s = STATUS_STYLES[status] ?? STATUS_STYLES.no_schedule;
  return (
    <span className={`inline-block rounded-full px-2.5 py-0.5 text-xs font-medium ${s.bg} ${s.text}`}>
      {s.label}
    </span>
  );
}

function CalibrationHistory({ instrumentId, instrumentName }: { instrumentId: string; instrumentName: string }) {
  const { data: records, isLoading } = useQuery({
    queryKey: ["calibrations", instrumentId],
    queryFn: () => api.listCalibrationRecords(instrumentId),
  });

  return (
    <div className="mt-3 rounded border border-gray-200 bg-gray-50 p-4">
      <h4 className="mb-3 text-sm font-medium text-gray-700">
        Calibration History — {instrumentName}
      </h4>
      {isLoading ? (
        <div className="text-sm text-gray-500">Loading...</div>
      ) : records?.length === 0 ? (
        <div className="text-sm text-gray-400">No calibration records</div>
      ) : (
        <div className="space-y-2">
          {records?.map((r) => (
            <div key={r.id} className="rounded border border-gray-200 bg-white p-3">
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-2">
                  <span className="text-sm font-medium text-gray-800">
                    {r.calibration_type.charAt(0).toUpperCase() + r.calibration_type.slice(1)}
                  </span>
                  <span
                    className={`inline-block rounded-full px-2 py-0.5 text-xs font-medium ${
                      r.status === "pass"
                        ? "bg-green-100 text-green-800"
                        : "bg-red-100 text-red-800"
                    }`}
                  >
                    {r.status.toUpperCase()}
                  </span>
                </div>
                <span className="text-xs text-gray-500">
                  {new Date(r.performed_at).toLocaleString()}
                </span>
              </div>
              <div className="mt-1 flex gap-4 text-xs text-gray-500">
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
                <div className="mt-1 text-xs text-red-600">
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
    return <div className="py-8 text-center text-sm text-gray-500">Loading...</div>;
  }

  const overdueCount = instruments?.filter((i) => i.calibration_status === "overdue").length ?? 0;
  const dueSoonCount = instruments?.filter((i) => i.calibration_status === "due_soon").length ?? 0;

  return (
    <div>
      <div className="mb-4 flex items-center justify-between">
        <h2 className="text-lg font-semibold text-gray-900">
          Instruments & Calibration
        </h2>
        <div className="flex gap-3 text-sm">
          {overdueCount > 0 && (
            <span className="font-medium text-red-600">
              {overdueCount} overdue
            </span>
          )}
          {dueSoonCount > 0 && (
            <span className="font-medium text-yellow-600">
              {dueSoonCount} due soon
            </span>
          )}
        </div>
      </div>

      <div className="overflow-x-auto rounded-lg border border-gray-200">
        <table className="min-w-full divide-y divide-gray-200">
          <thead className="bg-gray-50">
            <tr>
              <th className="px-4 py-2 text-left text-xs font-medium uppercase text-gray-500">
                Instrument
              </th>
              <th className="px-4 py-2 text-left text-xs font-medium uppercase text-gray-500">
                Type
              </th>
              <th className="px-4 py-2 text-left text-xs font-medium uppercase text-gray-500">
                Serial / Make
              </th>
              <th className="px-4 py-2 text-left text-xs font-medium uppercase text-gray-500">
                Last Calibration
              </th>
              <th className="px-4 py-2 text-left text-xs font-medium uppercase text-gray-500">
                Next Due
              </th>
              <th className="px-4 py-2 text-center text-xs font-medium uppercase text-gray-500">
                Status
              </th>
              <th className="px-4 py-2 text-center text-xs font-medium uppercase text-gray-500">
                History
              </th>
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-200 bg-white">
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
                <td colSpan={7} className="py-8 text-center text-sm text-gray-400">
                  No instruments configured
                </td>
              </tr>
            )}
          </tbody>
        </table>
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
    <tr className={`hover:bg-gray-50 ${inst.calibration_status === "overdue" ? "bg-red-50" : ""}`}>
      <td className="px-4 py-2 text-sm font-medium text-gray-900">{inst.name}</td>
      <td className="px-4 py-2 text-sm text-gray-600">{inst.instrument_type}</td>
      <td className="px-4 py-2 text-sm text-gray-600">
        {inst.serial_number && <div className="font-mono text-xs">{inst.serial_number}</div>}
        {inst.manufacturer && inst.model && (
          <div className="text-xs text-gray-400">
            {inst.manufacturer} {inst.model}
          </div>
        )}
      </td>
      <td className="px-4 py-2 text-sm text-gray-600">
        {inst.last_performed_at ? (
          <div>
            <div>{new Date(inst.last_performed_at).toLocaleDateString()}</div>
            <div className="text-xs text-gray-400">
              {inst.last_calibration_type} — {inst.last_status}
            </div>
          </div>
        ) : (
          <span className="text-gray-400">—</span>
        )}
      </td>
      <td className="px-4 py-2 text-sm text-gray-600">
        {inst.due_at ? new Date(inst.due_at).toLocaleDateString() : "—"}
      </td>
      <td className="px-4 py-2 text-center">
        <CalStatusBadge status={inst.calibration_status} />
      </td>
      <td className="px-4 py-2 text-center">
        <button
          onClick={onToggle}
          className={`rounded border px-2 py-1 text-xs ${
            expanded
              ? "border-blue-500 bg-blue-50 text-blue-600"
              : "border-gray-300 text-gray-600 hover:bg-gray-100"
          }`}
        >
          {expanded ? "Hide" : "View"}
        </button>
      </td>
    </tr>
  );
}
