import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { api } from "../api/client";
import { canDismissAlert, type AuthUser } from "../api/auth";
import type { Alert, AlertSeverity, AlertType } from "../api/types";

const SEVERITY_STYLES: Record<AlertSeverity, { bg: string; text: string; label: string }> = {
  critical: { bg: "bg-red-100", text: "text-red-800", label: "Critical" },
  warning: { bg: "bg-yellow-100", text: "text-yellow-800", label: "Warning" },
  info: { bg: "bg-blue-100", text: "text-blue-800", label: "Info" },
};

const TYPE_LABELS: Record<AlertType, string> = {
  exceedance: "Exceedance",
  overdue_calibration: "Overdue Calibration",
};

function SeverityBadge({ severity }: { severity: AlertSeverity }) {
  const s = SEVERITY_STYLES[severity];
  return (
    <span className={`inline-block rounded-full px-2.5 py-0.5 text-xs font-medium ${s.bg} ${s.text}`}>
      {s.label}
    </span>
  );
}

interface Props {
  facilityId: string;
  user: AuthUser;
}

export function AlertsView({ facilityId, user }: Props) {
  const queryClient = useQueryClient();
  const [typeFilter, setTypeFilter] = useState<AlertType | "">("");
  const [showDismissed, setShowDismissed] = useState(false);

  const filter = {
    facility_id: facilityId,
    ...(typeFilter ? { type: typeFilter } : {}),
    dismissed: showDismissed,
  };

  const { data: alerts, isLoading } = useQuery({
    queryKey: ["alerts", filter],
    queryFn: () => api.listAlerts(filter),
  });

  const dismissMutation = useMutation({
    mutationFn: (id: string) => api.dismissAlert(id),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["alerts"] }),
  });

  const canDismiss = canDismissAlert(user);

  if (isLoading) {
    return <div className="py-8 text-center text-sm text-gray-500">Loading...</div>;
  }

  const criticalCount = alerts?.filter((a) => a.severity === "critical" && !a.dismissed_at).length ?? 0;
  const warningCount = alerts?.filter((a) => a.severity === "warning" && !a.dismissed_at).length ?? 0;

  return (
    <div>
      <div className="mb-4 flex items-center justify-between">
        <h2 className="text-lg font-semibold text-gray-900">Alerts</h2>
        <div className="flex gap-3 text-sm">
          {criticalCount > 0 && (
            <span className="font-medium text-red-600">{criticalCount} critical</span>
          )}
          {warningCount > 0 && (
            <span className="font-medium text-yellow-600">{warningCount} warning</span>
          )}
        </div>
      </div>

      <div className="mb-4 flex flex-wrap gap-3 text-sm">
        <label className="flex items-center gap-2">
          <span className="text-gray-600">Type:</span>
          <select
            value={typeFilter}
            onChange={(e) => setTypeFilter(e.target.value as AlertType | "")}
            className="rounded border border-gray-300 px-2 py-1 text-sm"
          >
            <option value="">All</option>
            <option value="exceedance">Exceedance</option>
            <option value="overdue_calibration">Overdue Calibration</option>
          </select>
        </label>
        <label className="flex items-center gap-2">
          <input
            type="checkbox"
            checked={showDismissed}
            onChange={(e) => setShowDismissed(e.target.checked)}
          />
          <span className="text-gray-600">Include dismissed</span>
        </label>
      </div>

      <div className="overflow-x-auto rounded-lg border border-gray-200">
        <table className="min-w-full divide-y divide-gray-200">
          <thead className="bg-gray-50">
            <tr>
              <th className="px-4 py-2 text-left text-xs font-medium uppercase text-gray-500">
                Severity
              </th>
              <th className="px-4 py-2 text-left text-xs font-medium uppercase text-gray-500">
                Type
              </th>
              <th className="px-4 py-2 text-left text-xs font-medium uppercase text-gray-500">
                Message
              </th>
              <th className="px-4 py-2 text-left text-xs font-medium uppercase text-gray-500">
                Created
              </th>
              <th className="px-4 py-2 text-left text-xs font-medium uppercase text-gray-500">
                Status
              </th>
              <th className="px-4 py-2 text-center text-xs font-medium uppercase text-gray-500">
                Action
              </th>
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-200 bg-white">
            {alerts?.map((alert) => (
              <AlertRow
                key={alert.id}
                alert={alert}
                canDismiss={canDismiss}
                onDismiss={() => dismissMutation.mutate(alert.id)}
                pending={dismissMutation.isPending && dismissMutation.variables === alert.id}
              />
            ))}
            {alerts?.length === 0 && (
              <tr>
                <td colSpan={6} className="py-8 text-center text-sm text-gray-400">
                  No alerts
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>
    </div>
  );
}

function AlertRow({
  alert,
  canDismiss,
  onDismiss,
  pending,
}: {
  alert: Alert;
  canDismiss: boolean;
  onDismiss: () => void;
  pending: boolean;
}) {
  const isDismissed = !!alert.dismissed_at;
  const rowBg = isDismissed
    ? "bg-gray-50 opacity-60"
    : alert.severity === "critical"
    ? "bg-red-50"
    : "";

  return (
    <tr className={`hover:bg-gray-100 ${rowBg}`}>
      <td className="px-4 py-2">
        <SeverityBadge severity={alert.severity} />
      </td>
      <td className="px-4 py-2 text-sm text-gray-600">{TYPE_LABELS[alert.type]}</td>
      <td className="px-4 py-2 text-sm text-gray-900">{alert.message}</td>
      <td className="px-4 py-2 text-sm text-gray-500">
        {new Date(alert.created_at).toLocaleString()}
      </td>
      <td className="px-4 py-2 text-sm">
        {isDismissed ? (
          <span className="text-gray-500">
            Dismissed {new Date(alert.dismissed_at!).toLocaleDateString()}
          </span>
        ) : (
          <span className="font-medium text-gray-700">Active</span>
        )}
      </td>
      <td className="px-4 py-2 text-center">
        {!isDismissed && canDismiss && (
          <button
            onClick={onDismiss}
            disabled={pending}
            className="rounded border border-gray-300 px-2 py-1 text-xs text-gray-600 hover:bg-gray-100 disabled:opacity-50"
          >
            {pending ? "Dismissing..." : "Dismiss"}
          </button>
        )}
      </td>
    </tr>
  );
}
