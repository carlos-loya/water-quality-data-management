const styles: Record<string, string> = {
  draft: "bg-amber-50 text-amber-700 ring-amber-600/20",
  reviewed: "bg-blue-50 text-blue-700 ring-blue-600/20",
  approved: "bg-emerald-50 text-emerald-700 ring-emerald-600/20",
  EXCEEDANCE: "bg-red-50 text-red-700 ring-red-600/20",
  OK: "bg-emerald-50 text-emerald-700 ring-emerald-600/20",
  "N/A": "bg-slate-50 text-slate-600 ring-slate-500/20",
};

export function StatusBadge({ value }: { value: string }) {
  const cls = styles[value] ?? "bg-slate-50 text-slate-700 ring-slate-500/20";
  return (
    <span
      className={`inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium capitalize ring-1 ring-inset ${cls}`}
    >
      {value}
    </span>
  );
}
