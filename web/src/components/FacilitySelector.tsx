import { useQuery } from "@tanstack/react-query";
import { api } from "../api/client";

interface Props {
  orgId: string;
  selectedId: string | null;
  onSelect: (id: string) => void;
}

export function FacilitySelector({ orgId, selectedId, onSelect }: Props) {
  const { data: facilities, isLoading } = useQuery({
    queryKey: ["facilities", orgId],
    queryFn: () => api.listFacilities(orgId),
  });

  if (isLoading) return <div className="text-sm text-gray-500">Loading...</div>;

  return (
    <div className="flex gap-2">
      {facilities?.map((f) => (
        <button
          key={f.id}
          onClick={() => onSelect(f.id)}
          className={`rounded-lg border px-4 py-3 text-left transition ${
            selectedId === f.id
              ? "border-blue-500 bg-blue-50 shadow-sm"
              : "border-gray-200 bg-white hover:border-gray-300"
          }`}
        >
          <div className="text-sm font-medium text-gray-900">{f.name}</div>
          <div className="text-xs text-gray-500">
            {f.facility_type.replace("_", " ")}
          </div>
        </button>
      ))}
    </div>
  );
}
