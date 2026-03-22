import { useQuery } from "@tanstack/react-query";
import { api } from "../api/client";
import { StatusBadge } from "./StatusBadge";

interface Props {
  facilityId: string;
}

export function ComplianceView({ facilityId }: Props) {
  const { data: results, isLoading } = useQuery({
    queryKey: ["compliance", facilityId],
    queryFn: () => api.evaluateCompliance(facilityId),
  });

  if (isLoading) {
    return <div className="py-8 text-center text-sm text-gray-500">Loading...</div>;
  }

  // Group by parameter for a cleaner view
  const grouped = new Map<string, typeof results>();
  results?.forEach((r) => {
    const key = r.parameter_code;
    if (!grouped.has(key)) grouped.set(key, []);
    grouped.get(key)!.push(r);
  });

  const downloadUrl = (ext: string) =>
    `/api/v1/facilities/${facilityId}/reports/compliance.${ext}`;

  return (
    <div>
      <div className="mb-4 flex items-center justify-between">
        <h2 className="text-lg font-semibold text-gray-900">
          Compliance Evaluation
        </h2>
        <div className="flex gap-2">
          <a
            href={downloadUrl("xlsx")}
            className="rounded border border-gray-300 bg-white px-3 py-1.5 text-sm text-gray-700 hover:bg-gray-50"
          >
            Export Excel
          </a>
          <a
            href={downloadUrl("pdf")}
            className="rounded border border-gray-300 bg-white px-3 py-1.5 text-sm text-gray-700 hover:bg-gray-50"
          >
            Export PDF
          </a>
        </div>
      </div>
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
                Result
              </th>
              <th className="px-4 py-2 text-left text-xs font-medium uppercase text-gray-500">
                Limit Type
              </th>
              <th className="px-4 py-2 text-right text-xs font-medium uppercase text-gray-500">
                Limit
              </th>
              <th className="px-4 py-2 text-center text-xs font-medium uppercase text-gray-500">
                Status
              </th>
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-200 bg-white">
            {results?.map((r, i) => (
              <tr
                key={i}
                className={
                  r.compliance === "EXCEEDANCE"
                    ? "bg-red-50 hover:bg-red-100"
                    : "hover:bg-gray-50"
                }
              >
                <td className="whitespace-nowrap px-4 py-2 text-sm text-gray-700">
                  {new Date(r.collected_at).toLocaleDateString()}
                </td>
                <td className="px-4 py-2 text-sm text-gray-700">
                  {r.location_name}
                </td>
                <td className="px-4 py-2 text-sm text-gray-700">
                  {r.parameter_name}
                </td>
                <td className="whitespace-nowrap px-4 py-2 text-right text-sm font-mono text-gray-900">
                  {r.result_value ?? "ND"} {r.unit_code}
                </td>
                <td className="px-4 py-2 text-sm text-gray-500">
                  {r.limit_type.replace("_", " ")}
                </td>
                <td className="whitespace-nowrap px-4 py-2 text-right text-sm font-mono text-gray-700">
                  {r.limit_value} {r.unit_code}
                </td>
                <td className="px-4 py-2 text-center">
                  <StatusBadge value={r.compliance} />
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}
