import { useQuery } from "@tanstack/react-query";
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
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40">
      <div className="max-h-[80vh] w-full max-w-lg overflow-y-auto rounded-lg bg-white p-6 shadow-xl">
        <div className="mb-4 flex items-center justify-between">
          <h3 className="text-lg font-semibold text-gray-900">Audit History</h3>
          <button
            onClick={onClose}
            className="text-gray-400 hover:text-gray-600"
          >
            &times;
          </button>
        </div>

        <p className="mb-4 font-mono text-xs text-gray-400">
          {recordId}
        </p>

        {isLoading ? (
          <div className="py-4 text-center text-sm text-gray-500">Loading...</div>
        ) : entries?.length === 0 ? (
          <div className="py-4 text-center text-sm text-gray-400">
            No audit entries yet
          </div>
        ) : (
          <div className="space-y-3">
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
                <div
                  key={e.id}
                  className="rounded border border-gray-200 bg-gray-50 p-3"
                >
                  <div className="flex items-center justify-between">
                    <span className="text-sm font-medium text-gray-800">
                      {e.action}
                    </span>
                    <span className="text-xs text-gray-500">
                      {new Date(e.changed_at).toLocaleString()}
                    </span>
                  </div>
                  <div className="mt-1 text-xs text-gray-500">
                    by{" "}
                    <span className="font-mono">
                      {e.changed_by.slice(0, 12)}...
                    </span>
                  </div>
                  {(oldStatus || newStatus) && (
                    <div className="mt-1 text-xs text-gray-600">
                      {oldStatus ? `${oldStatus}` : "(new)"} &rarr;{" "}
                      <span className="font-medium">{String(newStatus)}</span>
                    </div>
                  )}
                </div>
              );
            })}
          </div>
        )}
      </div>
    </div>
  );
}
