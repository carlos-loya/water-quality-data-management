import { useState, useMemo } from "react";
import { useQuery } from "@tanstack/react-query";
import { api } from "../api/client";
import type { CreateSampleResultInput, MonitoringLocation, Parameter, UnitOfMeasure, ValidationRule } from "../api/types";

interface Props {
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
  const [overrideReason, setOverrideReason] = useState("");

  const { data: units } = useQuery({
    queryKey: ["units", orgId],
    queryFn: () => api.listUnits(orgId),
  });

  const { data: validationRules } = useQuery({
    queryKey: ["validation-rules", orgId],
    queryFn: () => api.listValidationRules(orgId),
  });

  const unitMap = new Map<string, UnitOfMeasure>(
    units?.map((u) => [u.id, u]) ?? []
  );

  // Index rules by parameter_id for fast lookup
  const ruleMap = useMemo(
    () => new Map<string, ValidationRule>(
      validationRules?.map((r) => [r.parameter_id, r]) ?? []
    ),
    [validationRules]
  );

  const activeRule = parameterId ? ruleMap.get(parameterId) : undefined;

  // Auto-select unit when parameter changes
  function handleParameterChange(newParameterId: string) {
    setParameterId(newParameterId);
    if (!newParameterId) return;
    const param = parameters.find((p) => p.id === newParameterId);
    if (param?.default_unit_id && unitMap.has(param.default_unit_id)) {
      setUnitId(param.default_unit_id);
    }
  }

  // Range check: produces a soft warning (overridable) for out-of-range values
  // and a hard error for precision/format problems.
  const rangeWarning = useMemo(() => {
    if (!resultValue || !activeRule) return null;
    const v = parseFloat(resultValue);
    if (isNaN(v)) return null;
    if (activeRule.min_value !== null && v < activeRule.min_value) {
      return `Below minimum (${activeRule.min_value})`;
    }
    if (activeRule.max_value !== null && v > activeRule.max_value) {
      return `Exceeds maximum (${activeRule.max_value})`;
    }
    return null;
  }, [resultValue, activeRule]);

  const valueError = useMemo(() => {
    if (!resultValue) return null;
    const v = parseFloat(resultValue);
    if (isNaN(v)) return "Must be a number";
    if (activeRule?.precision_digits !== null && activeRule?.precision_digits !== undefined) {
      const parts = resultValue.split(".");
      if (parts[1] && parts[1].length > activeRule.precision_digits) {
        return `Max ${activeRule.precision_digits} decimal place(s)`;
      }
    }
    return null;
  }, [resultValue, activeRule]);

  const overrideRequired = !!rangeWarning && !resultQualifier;

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (valueError) return;
    if (overrideRequired && !overrideReason.trim()) return;

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

    if (overrideRequired && overrideReason.trim()) {
      input.override_reason = overrideReason.trim();
    }

    onSubmit(input);
  }

  const isValueRequired = activeRule?.is_required && !resultQualifier;

  const isValid =
    locationId &&
    parameterId &&
    unitId &&
    collectedAt &&
    (resultValue || resultQualifier) &&
    !valueError &&
    (!overrideRequired || overrideReason.trim().length > 0);

  // Build hint text for value field
  function rangeHint(): string | null {
    if (!activeRule) return null;
    const parts: string[] = [];
    if (activeRule.min_value !== null && activeRule.max_value !== null) {
      parts.push(`Range: ${activeRule.min_value} – ${activeRule.max_value}`);
    } else if (activeRule.min_value !== null) {
      parts.push(`Min: ${activeRule.min_value}`);
    } else if (activeRule.max_value !== null) {
      parts.push(`Max: ${activeRule.max_value}`);
    }
    if (activeRule.precision_digits !== null) {
      parts.push(`${activeRule.precision_digits} decimal(s)`);
    }
    return parts.length > 0 ? parts.join(" · ") : null;
  }

  const inputCls =
    "mt-1 block w-full rounded-lg border border-slate-300 bg-white px-3 py-1.5 text-sm text-slate-900 focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500";
  const labelCls = "text-xs font-medium text-slate-600";

  return (
    <form
      onSubmit={handleSubmit}
      className="overflow-hidden rounded-xl border border-slate-200 bg-white shadow-sm"
    >
      <div className="border-b border-slate-200 bg-slate-50 px-5 py-3">
        <h3 className="text-sm font-semibold text-slate-900">New sample result</h3>
        <p className="text-xs text-slate-500">
          Enter a new measurement. Defaults follow the parameter's configured unit and validation range.
        </p>
      </div>

      <div className="p-5">
        {error && (
          <div className="mb-3 rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700">
            {error}
          </div>
        )}

        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
          <label className="block">
            <span className={labelCls}>Location *</span>
            <select
              required
              value={locationId}
              onChange={(e) => setLocationId(e.target.value)}
              className={inputCls}
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
            <span className={labelCls}>Parameter *</span>
            <select
              required
              value={parameterId}
              onChange={(e) => handleParameterChange(e.target.value)}
              className={inputCls}
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
            <span className={labelCls}>Unit *</span>
            <select
              required
              value={unitId}
              onChange={(e) => setUnitId(e.target.value)}
              className={inputCls}
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
            <span className={labelCls}>Collected at *</span>
            <input
              type="datetime-local"
              required
              value={collectedAt}
              onChange={(e) => setCollectedAt(e.target.value)}
              className={inputCls}
            />
          </label>

          <div className="block">
            <label>
              <span className={labelCls}>
                Result value {isValueRequired ? "*" : resultQualifier ? "" : "*"}
              </span>
              <input
                type="number"
                step="any"
                required={!resultQualifier}
                value={resultValue}
                onChange={(e) => setResultValue(e.target.value)}
                placeholder="e.g. 7.2"
                className={`mt-1 block w-full rounded-lg border px-3 py-1.5 font-mono text-sm focus:outline-none focus:ring-1 ${
                  valueError
                    ? "border-red-400 bg-red-50 focus:border-red-500 focus:ring-red-500"
                    : overrideRequired
                    ? "border-amber-400 bg-amber-50 focus:border-amber-500 focus:ring-amber-500"
                    : "border-slate-300 bg-white focus:border-blue-500 focus:ring-blue-500"
                }`}
              />
            </label>
            {valueError && (
              <p className="mt-1 text-xs text-red-600">{valueError}</p>
            )}
            {!valueError && overrideRequired && (
              <p className="mt-1 text-xs text-amber-700">
                {rangeWarning} — override reason required below
              </p>
            )}
            {!valueError && !overrideRequired && rangeHint() && (
              <p className="mt-1 text-xs text-slate-400">{rangeHint()}</p>
            )}
          </div>

          <label className="block">
            <span className={labelCls}>Qualifier</span>
            <select
              value={resultQualifier}
              onChange={(e) => setResultQualifier(e.target.value)}
              className={inputCls}
            >
              <option value="">None</option>
              <option value="<">{"<"} (Below detection)</option>
              <option value=">">{">"} (Above range)</option>
              <option value="ND">ND (Not detected)</option>
            </select>
          </label>

          <label className="block sm:col-span-2 lg:col-span-3">
            <span className={labelCls}>Notes</span>
            <input
              type="text"
              value={notes}
              onChange={(e) => setNotes(e.target.value)}
              placeholder="Optional notes..."
              className={inputCls}
            />
          </label>

          {overrideRequired && (
            <label className="block sm:col-span-2 lg:col-span-3">
              <span className="text-xs font-medium text-amber-800">
                Override reason *
              </span>
              <textarea
                required
                rows={2}
                value={overrideReason}
                onChange={(e) => setOverrideReason(e.target.value)}
                placeholder="Explain why this out-of-range value is defensible (e.g., verified with duplicate sample, matrix interference, etc.)"
                className="mt-1 block w-full rounded-lg border border-amber-300 bg-amber-50 px-3 py-1.5 text-sm focus:border-amber-500 focus:outline-none focus:ring-1 focus:ring-amber-500"
              />
              <p className="mt-1 text-xs text-amber-700">
                Required because the value is outside the configured validation range.
              </p>
            </label>
          )}
        </div>
      </div>

      <div className="flex items-center justify-end gap-2 border-t border-slate-200 bg-slate-50 px-5 py-3">
        <button
          type="button"
          onClick={onCancel}
          className="rounded-lg border border-slate-200 bg-white px-3 py-1.5 text-sm font-medium text-slate-700 hover:bg-slate-50"
        >
          Cancel
        </button>
        <button
          type="submit"
          disabled={!isValid || isPending}
          className="rounded-lg bg-blue-600 px-4 py-1.5 text-sm font-medium text-white shadow-sm hover:bg-blue-700 disabled:opacity-50"
        >
          {isPending ? "Saving..." : "Save result"}
        </button>
      </div>
    </form>
  );
}
