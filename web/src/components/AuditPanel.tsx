import { useQuery } from "@tanstack/react-query";
import { X } from "lucide-react";
import { api } from "../api/client";

interface Props {
  recordId: string;
  onClose: () => void;
}

export function AuditPanel({ recordId, onClose }: Props) {
  const { data: entries, isLoading } = useQuery({
    queryKey: ["audit-log", recordId],
    queryFn: () => api.getAuditLog(recordId),
  });

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-slate-900/50 p-4 backdrop-blur-sm"
      onClick={onClose}
    >
      <div
        className="max-h-[80vh] w-full max-w-lg overflow-y-auto rounded-xl bg-white p-6 shadow-xl"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="mb-4 flex items-start justify-between">
          <div>
            <h3 className="text-lg font-semibold text-slate-900">Audit history</h3>
            <p className="mt-0.5 font-mono text-xs text-slate-400">{recordId}</p>
          </div>
          <button
            onClick={onClose}
            className="rounded-lg p-1 text-slate-400 hover:bg-slate-100 hover:text-slate-600"
          >
            <X className="h-4 w-4" />
          </button>
        </div>

        {isLoading ? (
          <div className="py-8 text-center text-sm text-slate-500">Loading...</div>
        ) : entries?.length === 0 ? (
          <div className="py-8 text-center text-sm text-slate-400">
            No audit entries yet
          </div>
        ) : (
          <ol className="relative space-y-3 border-l border-slate-200 pl-4">
            {entries?.map((e) => {
              const oldStatus =
                e.old_values && typeof e.old_values === "object"
                  ? (e.old_values as Record<string, unknown>).status
                  : null;
              const newStatus =
                e.new_values && typeof e.new_values === "object"
                  ? (e.new_values as Record<string, unknown>).status
                  : null;

              return (
                <li key={e.id} className="relative">
                  <span className="absolute -left-[21px] top-1.5 h-2.5 w-2.5 rounded-full border-2 border-white bg-blue-500 ring-1 ring-slate-200" />
                  <div className="rounded-lg border border-slate-200 bg-slate-50 p-3">
                    <div className="flex items-center justify-between">
                      <span className="text-sm font-medium capitalize text-slate-800">
                        {e.action}
                      </span>
                      <span className="text-xs text-slate-500">
                        {new Date(e.changed_at).toLocaleString()}
                      </span>
                    </div>
                    <div className="mt-1 text-xs text-slate-500">
                      by{" "}
                      <span className="font-mono">
                        {e.changed_by.slice(0, 12)}...
                      </span>
                    </div>
                    {(oldStatus != null || newStatus != null) && (
                      <div className="mt-1 text-xs text-slate-700">
                        {oldStatus ? `${oldStatus}` : "(new)"} →{" "}
                        <span className="font-medium">{String(newStatus)}</span>
                      </div>
                    )}
                  </div>
                </li>
              );
            })}
          </ol>
        )}
      </div>
    </div>
  );
}
