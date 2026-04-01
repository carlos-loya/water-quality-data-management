import { useState, useEffect } from "react";
import { useQuery } from "@tanstack/react-query";
import { api } from "../api/client";
import type { CreateSampleResultInput, MonitoringLocation, Parameter, UnitOfMeasure } from "../api/types";

interface Props {
  facilityId: string;
  orgId: string;
  userId: string;
  locations: MonitoringLocation[];
  parameters: Parameter[];
  onSubmit: (input: CreateSampleResultInput) => void;
  onCancel: () => void;
  isPending: boolean;
  error?: string | null;
}

export function SampleResultForm({
  facilityId,
  orgId,
  userId,
  locations,
  parameters,
  onSubmit,
  onCancel,
  isPending,
  error,
}: Props) {
  const [locationId, setLocationId] = useState("");
  const [parameterId, setParameterId] = useState("");
  const [unitId, setUnitId] = useState("");
  const [collectedAt, setCollectedAt] = useState(
    new Date().toISOString().slice(0, 16)
  );
  const [resultValue, setResultValue] = useState("");
  const [resultQualifier, setResultQualifier] = useState("");
  const [notes, setNotes] = useState("");

  const { data: units } = useQuery({
    queryKey: ["units", orgId],
    queryFn: () => api.listUnits(orgId),
  });

  const unitMap = new Map<string, UnitOfMeasure>(
    units?.map((u) => [u.id, u]) ?? []
  );

  // Auto-select unit when parameter changes
  useEffect(() => {
    if (!parameterId) return;
    const param = parameters.find((p) => p.id === parameterId);
    if (param?.default_unit_id && unitMap.has(param.default_unit_id)) {
      setUnitId(param.default_unit_id);
    }
  }, [parameterId, parameters, units]);

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault();

    const input: CreateSampleResultInput = {
      monitoring_location_id: locationId,
      parameter_id: parameterId,
      unit_id: unitId,
      collected_at: new Date(collectedAt).toISOString(),
      entered_by: userId,
      source: "manual",
    };

    if (resultQualifier) {
      input.result_qualifier = resultQualifier;
      if (resultValue) {
        input.detection_limit = parseFloat(resultValue);
      }
    } else {
      input.result_value = parseFloat(resultValue);
    }

    if (notes.trim()) {
      input.notes = notes.trim();
    }

    onSubmit(input);
  }

  const isValid =
    locationId &&
    parameterId &&
    unitId &&
    collectedAt &&
    (resultValue || resultQualifier);

  return (
    <form
      onSubmit={handleSubmit}
      className="mb-6 rounded-lg border border-blue-200 bg-blue-50 p-4"
    >
      <h3 className="mb-4 text-sm font-semibold text-gray-900">
        New Sample Result
      </h3>

      {error && (
        <div className="mb-3 rounded border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700">
          {error}
        </div>
      )}

      <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3">
        <label className="block">
          <span className="text-xs font-medium text-gray-600">Location *</span>
          <select
            required
            value={locationId}
            onChange={(e) => setLocationId(e.target.value)}
            className="mt-1 block w-full rounded border border-gray-300 px-3 py-1.5 text-sm"
          >
            <option value="">Select location...</option>
            {locations.map((l) => (
              <option key={l.id} value={l.id}>
                {l.name}
              </option>
            ))}
          </select>
        </label>

        <label className="block">
          <span className="text-xs font-medium text-gray-600">
            Parameter *
          </span>
          <select
            required
            value={parameterId}
            onChange={(e) => setParameterId(e.target.value)}
            className="mt-1 block w-full rounded border border-gray-300 px-3 py-1.5 text-sm"
          >
            <option value="">Select parameter...</option>
            {parameters.map((p) => (
              <option key={p.id} value={p.id}>
                {p.name} ({p.code})
              </option>
            ))}
          </select>
        </label>

        <label className="block">
          <span className="text-xs font-medium text-gray-600">Unit *</span>
          <select
            required
            value={unitId}
            onChange={(e) => setUnitId(e.target.value)}
            className="mt-1 block w-full rounded border border-gray-300 px-3 py-1.5 text-sm"
          >
            <option value="">Select unit...</option>
            {units?.map((u) => (
              <option key={u.id} value={u.id}>
                {u.code} — {u.name}
              </option>
            ))}
          </select>
        </label>

        <label className="block">
          <span className="text-xs font-medium text-gray-600">
            Collected At *
          </span>
          <input
            type="datetime-local"
            required
            value={collectedAt}
            onChange={(e) => setCollectedAt(e.target.value)}
            className="mt-1 block w-full rounded border border-gray-300 px-3 py-1.5 text-sm"
          />
        </label>

        <label className="block">
          <span className="text-xs font-medium text-gray-600">
            Result Value {resultQualifier ? "" : "*"}
          </span>
          <input
            type="number"
            step="any"
            required={!resultQualifier}
            value={resultValue}
            onChange={(e) => setResultValue(e.target.value)}
            placeholder="e.g. 7.2"
            className="mt-1 block w-full rounded border border-gray-300 px-3 py-1.5 text-sm font-mono"
          />
        </label>

        <label className="block">
          <span className="text-xs font-medium text-gray-600">
            Qualifier
          </span>
          <select
            value={resultQualifier}
            onChange={(e) => setResultQualifier(e.target.value)}
            className="mt-1 block w-full rounded border border-gray-300 px-3 py-1.5 text-sm"
          >
            <option value="">None</option>
            <option value="<">{"<"} (Below detection)</option>
            <option value=">">{">"} (Above range)</option>
            <option value="ND">ND (Not detected)</option>
          </select>
        </label>

        <label className="block sm:col-span-2 lg:col-span-3">
          <span className="text-xs font-medium text-gray-600">Notes</span>
          <input
            type="text"
            value={notes}
            onChange={(e) => setNotes(e.target.value)}
            placeholder="Optional notes..."
            className="mt-1 block w-full rounded border border-gray-300 px-3 py-1.5 text-sm"
          />
        </label>
      </div>

      <div className="mt-4 flex items-center gap-2">
        <button
          type="submit"
          disabled={!isValid || isPending}
          className="rounded bg-blue-600 px-4 py-1.5 text-sm font-medium text-white hover:bg-blue-700 disabled:opacity-50"
        >
          {isPending ? "Saving..." : "Save Result"}
        </button>
        <button
          type="button"
          onClick={onCancel}
          className="rounded border border-gray-300 px-4 py-1.5 text-sm text-gray-600 hover:bg-gray-50"
        >
          Cancel
        </button>
      </div>
    </form>
  );
}
