import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { AlertTriangle, Info as InfoIcon, ShieldAlert, X } from "lucide-react";
import { api } from "../api/client";
import { canDismissAlert, type AuthUser } from "../api/auth";
import type { Alert, AlertSeverity, AlertType } from "../api/types";

const SEVERITY_STYLES: Record<
  AlertSeverity,
  { ring: string; text: string; bg: string; label: string; icon: typeof AlertTriangle }
> = {
  critical: {
    ring: "ring-red-600/20",
    text: "text-red-700",
    bg: "bg-red-50",
    label: "Critical",
    icon: ShieldAlert,
  },
  warning: {
    ring: "ring-amber-600/20",
    text: "text-amber-700",
    bg: "bg-amber-50",
    label: "Warning",
    icon: AlertTriangle,
  },
  info: {
    ring: "ring-blue-600/20",
    text: "text-blue-700",
    bg: "bg-blue-50",
    label: "Info",
    icon: InfoIcon,
  },
};

const TYPE_LABELS: Record<AlertType, string> = {
  exceedance: "Exceedance",
  overdue_calibration: "Overdue calibration",
};

function SeverityBadge({ severity }: { severity: AlertSeverity }) {
  const s = SEVERITY_STYLES[severity];
  const Icon = s.icon;
  return (
    <span
      className={`inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-xs font-medium ring-1 ring-inset ${s.bg} ${s.text} ${s.ring}`}
    >
      <Icon className="h-3 w-3" />
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

  const criticalCount = alerts?.filter((a) => a.severity === "critical" && !a.dismissed_at).length ?? 0;
  const warningCount = alerts?.filter((a) => a.severity === "warning" && !a.dismissed_at).length ?? 0;
  const infoCount = alerts?.filter((a) => a.severity === "info" && !a.dismissed_at).length ?? 0;

  return (
    <div className="space-y-4">
      <div className="grid grid-cols-1 gap-3 sm:grid-cols-3">
        <SeverityCard severity="critical" count={criticalCount} />
        <SeverityCard severity="warning" count={warningCount} />
        <SeverityCard severity="info" count={infoCount} />
      </div>

      <div className="overflow-hidden rounded-xl border border-slate-200 bg-white shadow-sm">
        <div className="flex flex-wrap items-center gap-3 border-b border-slate-200 px-4 py-3">
          <label className="flex items-center gap-2 text-sm">
            <span className="text-slate-600">Type</span>
            <select
              value={typeFilter}
              onChange={(e) => setTypeFilter(e.target.value as AlertType | "")}
              className="rounded-lg border border-slate-200 bg-white px-2.5 py-1 text-sm text-slate-700 focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
            >
              <option value="">All</option>
              <option value="exceedance">Exceedance</option>
              <option value="overdue_calibration">Overdue calibration</option>
            </select>
          </label>
          <label className="flex items-center gap-2 text-sm text-slate-600">
            <input
              type="checkbox"
              checked={showDismissed}
              onChange={(e) => setShowDismissed(e.target.checked)}
              className="rounded border-slate-300 text-blue-600 focus:ring-blue-500"
            />
            Include dismissed
          </label>
        </div>

        {isLoading ? (
          <div className="py-12 text-center text-sm text-slate-500">Loading...</div>
        ) : (
          <div className="overflow-x-auto">
            <table className="min-w-full divide-y divide-slate-200">
              <thead className="bg-slate-50">
                <tr>
                  <th className="px-4 py-2.5 text-left text-xs font-semibold uppercase tracking-wide text-slate-500">
                    Severity
                  </th>
                  <th className="px-4 py-2.5 text-left text-xs font-semibold uppercase tracking-wide text-slate-500">
                    Type
                  </th>
                  <th className="px-4 py-2.5 text-left text-xs font-semibold uppercase tracking-wide text-slate-500">
                    Message
                  </th>
                  <th className="px-4 py-2.5 text-left text-xs font-semibold uppercase tracking-wide text-slate-500">
                    Created
                  </th>
                  <th className="px-4 py-2.5 text-left text-xs font-semibold uppercase tracking-wide text-slate-500">
                    Status
                  </th>
                  <th className="px-4 py-2.5 text-right text-xs font-semibold uppercase tracking-wide text-slate-500">
                    Action
                  </th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-100 bg-white">
                {alerts?.map((alert) => (
                  <AlertRow
                    key={alert.id}
                    alert={alert}
                    canDismiss={canDismiss}
                    onDismiss={() => dismissMutation.mutate(alert.id)}
                    pending={
                      dismissMutation.isPending && dismissMutation.variables === alert.id
                    }
                  />
                ))}
                {alerts?.length === 0 && (
                  <tr>
                    <td colSpan={6} className="py-12 text-center text-sm text-slate-400">
                      No alerts
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
    ? "bg-slate-50 opacity-60"
    : alert.severity === "critical"
    ? "bg-red-50/40"
    : "";

  return (
    <tr className={`hover:bg-slate-50 ${rowBg}`}>
      <td className="px-4 py-2.5">
        <SeverityBadge severity={alert.severity} />
      </td>
      <td className="px-4 py-2.5 text-sm text-slate-600">
        {TYPE_LABELS[alert.type]}
      </td>
      <td className="px-4 py-2.5 text-sm text-slate-900">{alert.message}</td>
      <td className="px-4 py-2.5 text-sm text-slate-500">
        {new Date(alert.created_at).toLocaleString()}
      </td>
      <td className="px-4 py-2.5 text-sm">
        {isDismissed ? (
          <span className="text-slate-500">
            Dismissed {new Date(alert.dismissed_at!).toLocaleDateString()}
          </span>
        ) : (
          <span className="font-medium text-slate-700">Active</span>
        )}
      </td>
      <td className="px-4 py-2.5 text-right">
        {!isDismissed && canDismiss && (
          <button
            onClick={onDismiss}
            disabled={pending}
            className="inline-flex items-center gap-1 rounded-md border border-slate-200 px-2 py-1 text-xs text-slate-600 hover:bg-slate-50 disabled:opacity-50"
          >
            <X className="h-3 w-3" />
            {pending ? "Dismissing..." : "Dismiss"}
          </button>
        )}
      </td>
    </tr>
  );
}

function SeverityCard({ severity, count }: { severity: AlertSeverity; count: number }) {
  const s = SEVERITY_STYLES[severity];
  const Icon = s.icon;
  return (
    <div className="flex items-center gap-3 rounded-xl border border-slate-200 bg-white p-4 shadow-sm">
      <div className={`flex h-10 w-10 items-center justify-center rounded-lg ${s.bg} ${s.text}`}>
        <Icon className="h-5 w-5" />
      </div>
      <div>
        <div className="text-xs font-medium uppercase tracking-wide text-slate-500">
          {s.label}
        </div>
        <div className={`text-2xl font-semibold ${s.text}`}>{count}</div>
      </div>
    </div>
  );
}
