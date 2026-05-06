import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { Check, History, Info, Plus, Search } from "lucide-react";
import { api } from "../api/client";
import type { MonitoringLocation, Parameter, SampleResult, CreateSampleResultInput } from "../api/types";
import { type AuthUser, canReview, canApprove, hasAnyRole } from "../api/auth";
import { StatusBadge } from "./StatusBadge";
import { AuditPanel } from "./AuditPanel";
import { SampleResultForm } from "./SampleResultForm";
import { SampleResultDetails } from "./SampleResultDetails";

interface Props {
  facilityId: string;
  orgId: string;
  user: AuthUser;
}

export function SampleResultsTable({ facilityId, orgId, user }: Props) {
  const queryClient = useQueryClient();
  const [statusFilter, setStatusFilter] = useState<string>("");
  const [auditResultId, setAuditResultId] = useState<string | null>(null);
  const [detailsResultId, setDetailsResultId] = useState<string | null>(null);
  const [showForm, setShowForm] = useState(false);
  const [createError, setCreateError] = useState<string | null>(null);

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
    mutationFn: (id: string) => api.reviewSampleResult(id),
    onSuccess: () =>
      queryClient.invalidateQueries({ queryKey: ["sample-results"] }),
  });

  const approveMutation = useMutation({
    mutationFn: (id: string) => api.approveSampleResult(id),
    onSuccess: () =>
      queryClient.invalidateQueries({ queryKey: ["sample-results"] }),
  });

  const createMutation = useMutation({
    mutationFn: (input: CreateSampleResultInput) =>
      api.createSampleResult(input),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["sample-results"] });
      setShowForm(false);
      setCreateError(null);
    },
    onError: (err: Error) => setCreateError(err.message),
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
    <div className="space-y-4">
      {showForm && locations && parameters && (
        <SampleResultForm
          orgId={orgId}
          userId={user.id}
          locations={locations}
          parameters={parameters}
          onSubmit={(input) => createMutation.mutate(input)}
          onCancel={() => { setShowForm(false); setCreateError(null); }}
          isPending={createMutation.isPending}
          error={createError}
        />
      )}

      <div className="overflow-hidden rounded-xl border border-slate-200 bg-white shadow-sm">
        <div className="flex flex-wrap items-center justify-between gap-3 border-b border-slate-200 px-4 py-3">
          <div className="flex items-center gap-2">
            <div className="relative">
              <Search className="pointer-events-none absolute left-2.5 top-1/2 h-4 w-4 -translate-y-1/2 text-slate-400" />
              <select
                value={statusFilter}
                onChange={(e) => setStatusFilter(e.target.value)}
                className="appearance-none rounded-lg border border-slate-200 bg-white py-1.5 pl-8 pr-8 text-sm text-slate-700 focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
              >
                <option value="">All statuses</option>
                <option value="draft">Draft</option>
                <option value="reviewed">Reviewed</option>
                <option value="approved">Approved</option>
              </select>
            </div>
            <span className="text-sm text-slate-500">
              {results?.length ?? 0} results
            </span>
          </div>
          {hasAnyRole(user, ["admin", "operator"]) && !showForm && (
            <button
              onClick={() => { setShowForm(true); setCreateError(null); }}
              className="inline-flex items-center gap-1.5 rounded-lg bg-blue-600 px-3 py-1.5 text-sm font-medium text-white shadow-sm hover:bg-blue-700"
            >
              <Plus className="h-4 w-4" />
              New result
            </button>
          )}
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
                    Value
                  </th>
                  <th className="px-4 py-2.5 text-center text-xs font-semibold uppercase tracking-wide text-slate-500">
                    Status
                  </th>
                  <th className="px-4 py-2.5 text-right text-xs font-semibold uppercase tracking-wide text-slate-500">
                    Actions
                  </th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-100 bg-white">
                {results?.map((r) => (
                  <tr key={r.id} className="hover:bg-slate-50">
                    <td className="whitespace-nowrap px-4 py-2.5 text-sm text-slate-700">
                      {new Date(r.collected_at).toLocaleDateString()}
                    </td>
                    <td className="px-4 py-2.5 text-sm text-slate-700">
                      {locMap.get(r.monitoring_location_id)?.name ?? "—"}
                    </td>
                    <td className="px-4 py-2.5 text-sm text-slate-700">
                      {paramMap.get(r.parameter_id)?.name ?? "—"}
                    </td>
                    <td className="whitespace-nowrap px-4 py-2.5 text-right font-mono text-sm text-slate-900">
                      {formatValue(r)}
                    </td>
                    <td className="px-4 py-2.5 text-center">
                      <StatusBadge value={r.status} />
                    </td>
                    <td className="whitespace-nowrap px-4 py-2.5 text-right">
                      <div className="flex items-center justify-end gap-1">
                        {r.status === "draft" && canReview(user) && (
                          <button
                            onClick={() => reviewMutation.mutate(r.id)}
                            disabled={reviewMutation.isPending}
                            className="inline-flex items-center gap-1 rounded-md bg-blue-600 px-2 py-1 text-xs font-medium text-white hover:bg-blue-700 disabled:opacity-50"
                          >
                            <Check className="h-3 w-3" />
                            Review
                          </button>
                        )}
                        {r.status === "reviewed" && canApprove(user) && (
                          <button
                            onClick={() => approveMutation.mutate(r.id)}
                            disabled={approveMutation.isPending}
                            className="inline-flex items-center gap-1 rounded-md bg-emerald-600 px-2 py-1 text-xs font-medium text-white hover:bg-emerald-700 disabled:opacity-50"
                          >
                            <Check className="h-3 w-3" />
                            Approve
                          </button>
                        )}
                        <button
                          onClick={() => setDetailsResultId(r.id)}
                          className="inline-flex items-center gap-1 rounded-md border border-slate-200 px-2 py-1 text-xs text-slate-600 hover:bg-slate-50"
                        >
                          <Info className="h-3 w-3" />
                          Details
                        </button>
                        <button
                          onClick={() => setAuditResultId(r.id)}
                          className="inline-flex items-center gap-1 rounded-md border border-slate-200 px-2 py-1 text-xs text-slate-600 hover:bg-slate-50"
                        >
                          <History className="h-3 w-3" />
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
                      className="py-12 text-center text-sm text-slate-400"
                    >
                      No results found
                    </td>
                  </tr>
                )}
              </tbody>
            </table>
          </div>
        )}
      </div>

      {auditResultId && (
        <AuditPanel
          recordId={auditResultId}
          onClose={() => setAuditResultId(null)}
        />
      )}

      {detailsResultId && (
        <SampleResultDetails
          sampleResultId={detailsResultId}
          user={user}
          onClose={() => setDetailsResultId(null)}
        />
      )}
    </div>
  );
}
