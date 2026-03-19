const styles: Record<string, string> = {
  draft: "bg-yellow-100 text-yellow-800",
  reviewed: "bg-blue-100 text-blue-800",
  approved: "bg-green-100 text-green-800",
  EXCEEDANCE: "bg-red-100 text-red-800",
  OK: "bg-green-100 text-green-800",
  "N/A": "bg-gray-100 text-gray-600",
};

export function StatusBadge({ value }: { value: string }) {
  const cls = styles[value] ?? "bg-gray-100 text-gray-800";
  return (
    <span
      className={`inline-block rounded-full px-2.5 py-0.5 text-xs font-medium ${cls}`}
    >
      {value}
    </span>
  );
}
