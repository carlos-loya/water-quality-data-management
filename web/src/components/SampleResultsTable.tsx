import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { api } from "../api/client";
import type { MonitoringLocation, Parameter, SampleResult } from "../api/types";
import { StatusBadge } from "./StatusBadge";
import { AuditPanel } from "./AuditPanel";

// Hardcoded reviewer for demo. In a real app this comes from auth context.
const DEMO_REVIEWER = "019558a0-0001-7000-a000-000000000003"; // Amy Kim

interface Props {
  facilityId: string;
  orgId: string;
}

export function SampleResultsTable({ facilityId, orgId }: Props) {
  const queryClient = useQueryClient();
  const [statusFilter, setStatusFilter] = useState<string>("");
  const [auditResultId, setAuditResultId] = useState<string | null>(null);

  const { data: locations } = useQuery({
    queryKey: ["locations", facilityId],
    queryFn: () => api.listMonitoringLocations(facilityId),
  });

  const { data: parameters } = useQuery({
    queryKey: ["parameters", orgId],
    queryFn: () => api.listParameters(orgId),
  });

  const locationIds = locations?.map((l) => l.id) ?? [];

  const { data: results, isLoading } = useQuery({
    queryKey: ["sample-results", facilityId, statusFilter],
    queryFn: async () => {
      if (locationIds.length === 0) return [];
      // Fetch results for all locations in this facility
      const all = await Promise.all(
        locationIds.map((locId) => {
          const params: Record<string, string> = {
            monitoring_location_id: locId,
            limit: "100",
          };
          if (statusFilter) params.status = statusFilter;
          return api.listSampleResults(params);
        })
      );
      return all
        .flat()
        .sort(
          (a, b) =>
            new Date(b.collected_at).getTime() -
            new Date(a.collected_at).getTime()
        );
    },
    enabled: locationIds.length > 0,
  });

  const reviewMutation = useMutation({
    mutationFn: (id: string) => api.reviewSampleResult(id, DEMO_REVIEWER),
    onSuccess: () =>
      queryClient.invalidateQueries({ queryKey: ["sample-results"] }),
  });

  const approveMutation = useMutation({
    mutationFn: (id: string) => api.approveSampleResult(id, DEMO_REVIEWER),
    onSuccess: () =>
      queryClient.invalidateQueries({ queryKey: ["sample-results"] }),
  });

  const locMap = new Map<string, MonitoringLocation>(
    locations?.map((l) => [l.id, l]) ?? []
  );
  const paramMap = new Map<string, Parameter>(
    parameters?.map((p) => [p.id, p]) ?? []
  );

  function formatValue(r: SampleResult): string {
    if (r.result_qualifier) {
      return `${r.result_qualifier}${r.detection_limit ?? ""}`;
    }
    return r.result_value?.toString() ?? "-";
  }

  return (
    <div>
      <div className="mb-4 flex items-center justify-between">
        <h2 className="text-lg font-semibold text-gray-900">Sample Results</h2>
        <select
          value={statusFilter}
          onChange={(e) => setStatusFilter(e.target.value)}
          className="rounded border border-gray-300 px-3 py-1.5 text-sm"
        >
          <option value="">All statuses</option>
          <option value="draft">Draft</option>
          <option value="reviewed">Reviewed</option>
          <option value="approved">Approved</option>
        </select>
      </div>

      {isLoading ? (
        <div className="py-8 text-center text-sm text-gray-500">Loading...</div>
      ) : (
        <div className="overflow-x-auto rounded-lg border border-gray-200">
          <table className="min-w-full divide-y divide-gray-200">
            <thead className="bg-gray-50">
              <tr>
                <th className="px-4 py-2 text-left text-xs font-medium uppercase text-gray-500">
                  Date
                </th>
                <th className="px-4 py-2 text-left text-xs font-medium uppercase text-gray-500">
                  Location
                </th>
                <th className="px-4 py-2 text-left text-xs font-medium uppercase text-gray-500">
                  Parameter
                </th>
                <th className="px-4 py-2 text-right text-xs font-medium uppercase text-gray-500">
                  Value
                </th>
                <th className="px-4 py-2 text-center text-xs font-medium uppercase text-gray-500">
                  Status
                </th>
                <th className="px-4 py-2 text-center text-xs font-medium uppercase text-gray-500">
                  Actions
                </th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-200 bg-white">
              {results?.map((r) => (
                <tr key={r.id} className="hover:bg-gray-50">
                  <td className="whitespace-nowrap px-4 py-2 text-sm text-gray-700">
                    {new Date(r.collected_at).toLocaleDateString()}
                  </td>
                  <td className="px-4 py-2 text-sm text-gray-700">
                    {locMap.get(r.monitoring_location_id)?.name ?? "—"}
                  </td>
                  <td className="px-4 py-2 text-sm text-gray-700">
                    {paramMap.get(r.parameter_id)?.name ?? "—"}
                  </td>
                  <td className="whitespace-nowrap px-4 py-2 text-right text-sm font-mono text-gray-900">
                    {formatValue(r)}
                  </td>
                  <td className="px-4 py-2 text-center">
                    <StatusBadge value={r.status} />
                  </td>
                  <td className="whitespace-nowrap px-4 py-2 text-center">
                    <div className="flex items-center justify-center gap-1">
                      {r.status === "draft" && (
                        <button
                          onClick={() => reviewMutation.mutate(r.id)}
                          disabled={reviewMutation.isPending}
                          className="rounded bg-blue-500 px-2 py-1 text-xs text-white hover:bg-blue-600 disabled:opacity-50"
                        >
                          Review
                        </button>
                      )}
                      {r.status === "reviewed" && (
                        <button
                          onClick={() => approveMutation.mutate(r.id)}
                          disabled={approveMutation.isPending}
                          className="rounded bg-green-500 px-2 py-1 text-xs text-white hover:bg-green-600 disabled:opacity-50"
                        >
                          Approve
                        </button>
                      )}
                      <button
                        onClick={() => setAuditResultId(r.id)}
                        className="rounded border border-gray-300 px-2 py-1 text-xs text-gray-600 hover:bg-gray-100"
                      >
                        History
                      </button>
                    </div>
                  </td>
                </tr>
              ))}
              {results?.length === 0 && (
                <tr>
                  <td
                    colSpan={6}
                    className="py-8 text-center text-sm text-gray-400"
                  >
                    No results found
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
      )}

      {auditResultId && (
        <AuditPanel
          recordId={auditResultId}
          onClose={() => setAuditResultId(null)}
        />
      )}
    </div>
  );
}
