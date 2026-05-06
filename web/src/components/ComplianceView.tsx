import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { FileSpreadsheet, FileText } from "lucide-react";
import { api } from "../api/client";
import { StatusBadge } from "./StatusBadge";

interface Props {
  facilityId: string;
}

export function ComplianceView({ facilityId }: Props) {
  const [downloading, setDownloading] = useState<"xlsx" | "pdf" | null>(null);
  const [downloadError, setDownloadError] = useState<string | null>(null);

  const { data: results, isLoading } = useQuery({
    queryKey: ["compliance", facilityId],
    queryFn: () => api.evaluateCompliance(facilityId),
  });

  async function handleDownload(ext: "xlsx" | "pdf") {
    setDownloadError(null);
    setDownloading(ext);
    try {
      const blob = await api.downloadComplianceReport(facilityId, ext);
      const url = URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = `compliance-${facilityId}.${ext}`;
      document.body.appendChild(a);
      a.click();
      document.body.removeChild(a);
      URL.revokeObjectURL(url);
    } catch (err) {
      setDownloadError(err instanceof Error ? err.message : "Download failed");
    } finally {
      setDownloading(null);
    }
  }

  const exceedances = results?.filter((r) => r.compliance === "EXCEEDANCE").length ?? 0;
  const ok = results?.filter((r) => r.compliance === "OK").length ?? 0;

  return (
    <div className="space-y-4">
      <div className="grid grid-cols-1 gap-3 sm:grid-cols-3">
        <SummaryCard label="Total evaluated" value={results?.length ?? 0} tone="slate" />
        <SummaryCard label="In compliance" value={ok} tone="emerald" />
        <SummaryCard label="Exceedances" value={exceedances} tone="red" />
      </div>

      <div className="overflow-hidden rounded-xl border border-slate-200 bg-white shadow-sm">
        <div className="flex flex-wrap items-center justify-between gap-3 border-b border-slate-200 px-4 py-3">
          <h2 className="text-sm font-semibold text-slate-700">
            Permit limit evaluation
          </h2>
          <div className="flex items-center gap-2">
            {downloadError && (
              <span className="text-xs text-red-600">{downloadError}</span>
            )}
            <button
              type="button"
              onClick={() => handleDownload("xlsx")}
              disabled={downloading !== null}
              className="inline-flex items-center gap-1.5 rounded-lg border border-slate-200 bg-white px-3 py-1.5 text-sm font-medium text-slate-700 shadow-sm hover:bg-slate-50 disabled:opacity-50"
            >
              <FileSpreadsheet className="h-4 w-4 text-emerald-600" />
              {downloading === "xlsx" ? "Exporting..." : "Excel"}
            </button>
            <button
              type="button"
              onClick={() => handleDownload("pdf")}
              disabled={downloading !== null}
              className="inline-flex items-center gap-1.5 rounded-lg border border-slate-200 bg-white px-3 py-1.5 text-sm font-medium text-slate-700 shadow-sm hover:bg-slate-50 disabled:opacity-50"
            >
              <FileText className="h-4 w-4 text-red-600" />
              {downloading === "pdf" ? "Exporting..." : "PDF"}
            </button>
          </div>
        </div>
        {isLoading ? (
          <div className="py-12 text-center text-sm text-slate-500">Loading...</div>
        ) : (
          <div className="overflow-x-auto">
            <table className="min-w-full divide-y divide-slate-200">
              <thead className="bg-slate-50">
                <tr>
                  <th className="px-4 py-2.5 text-left text-xs font-semibold uppercase tracking-wide text-slate-500">
                    Date
                  </th>
                  <th className="px-4 py-2.5 text-left text-xs font-semibold uppercase tracking-wide text-slate-500">
                    Location
                  </th>
                  <th className="px-4 py-2.5 text-left text-xs font-semibold uppercase tracking-wide text-slate-500">
                    Parameter
                  </th>
                  <th className="px-4 py-2.5 text-right text-xs font-semibold uppercase tracking-wide text-slate-500">
                    Result
                  </th>
                  <th className="px-4 py-2.5 text-left text-xs font-semibold uppercase tracking-wide text-slate-500">
                    Limit type
                  </th>
                  <th className="px-4 py-2.5 text-right text-xs font-semibold uppercase tracking-wide text-slate-500">
                    Limit
                  </th>
                  <th className="px-4 py-2.5 text-center text-xs font-semibold uppercase tracking-wide text-slate-500">
                    Status
                  </th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-100 bg-white">
                {results?.map((r, i) => (
                  <tr
                    key={i}
                    className={
                      r.compliance === "EXCEEDANCE"
                        ? "bg-red-50/50 hover:bg-red-50"
                        : "hover:bg-slate-50"
                    }
                  >
                    <td className="whitespace-nowrap px-4 py-2.5 text-sm text-slate-700">
                      {new Date(r.collected_at).toLocaleDateString()}
                    </td>
                    <td className="px-4 py-2.5 text-sm text-slate-700">
                      {r.location_name}
                    </td>
                    <td className="px-4 py-2.5 text-sm text-slate-700">
                      {r.parameter_name}
                    </td>
                    <td className="whitespace-nowrap px-4 py-2.5 text-right font-mono text-sm text-slate-900">
                      {r.result_value ?? "ND"} {r.unit_code}
                    </td>
                    <td className="px-4 py-2.5 text-sm capitalize text-slate-500">
                      {r.limit_type.replace("_", " ")}
                    </td>
                    <td className="whitespace-nowrap px-4 py-2.5 text-right font-mono text-sm text-slate-700">
                      {r.limit_value} {r.unit_code}
                    </td>
                    <td className="px-4 py-2.5 text-center">
                      <StatusBadge value={r.compliance} />
                    </td>
                  </tr>
                ))}
                {results?.length === 0 && (
                  <tr>
                    <td colSpan={7} className="py-12 text-center text-sm text-slate-400">
                      No compliance evaluations available
                    </td>
                  </tr>
                )}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </div>
  );
}

const TONE_STYLES: Record<string, { bg: string; text: string; ring: string }> = {
  slate: { bg: "bg-slate-50", text: "text-slate-900", ring: "ring-slate-200" },
  emerald: { bg: "bg-emerald-50", text: "text-emerald-700", ring: "ring-emerald-200" },
  red: { bg: "bg-red-50", text: "text-red-700", ring: "ring-red-200" },
};

function SummaryCard({
  label,
  value,
  tone,
}: {
  label: string;
  value: number;
  tone: keyof typeof TONE_STYLES;
}) {
  const t = TONE_STYLES[tone];
  return (
    <div className="rounded-xl border border-slate-200 bg-white p-4 shadow-sm">
      <div className="text-xs font-medium uppercase tracking-wide text-slate-500">
        {label}
      </div>
      <div className="mt-1 flex items-baseline gap-2">
        <span className={`text-2xl font-semibold ${t.text}`}>{value}</span>
        <span
          className={`inline-flex h-1.5 w-1.5 rounded-full ${t.bg} ring-2 ${t.ring}`}
        />
      </div>
    </div>
  );
}
