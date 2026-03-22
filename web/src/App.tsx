import { useState } from "react";
import { FacilitySelector } from "./components/FacilitySelector";
import { SampleResultsTable } from "./components/SampleResultsTable";
import { ComplianceView } from "./components/ComplianceView";
import { TrendingCharts } from "./components/TrendingCharts";
import { InstrumentsView } from "./components/InstrumentsView";

// Seed data org ID. In a real app this comes from auth.
const ORG_ID = "019558a0-0000-7000-a000-000000000001";

type Tab = "results" | "trending" | "compliance" | "instruments";

const TABS: { key: Tab; label: string }[] = [
  { key: "results", label: "Sample Results" },
  { key: "trending", label: "Trending" },
  { key: "compliance", label: "Compliance" },
  { key: "instruments", label: "Instruments" },
];

function App() {
  const [facilityId, setFacilityId] = useState<string | null>(null);
  const [tab, setTab] = useState<Tab>("results");

  return (
    <div className="min-h-screen bg-gray-100">
      <header className="border-b border-gray-200 bg-white">
        <div className="mx-auto max-w-6xl px-4 py-4">
          <h1 className="text-xl font-bold text-gray-900">
            Water Quality Data Management
          </h1>
          <p className="text-sm text-gray-500">
            Utility compliance and operations platform
          </p>
        </div>
      </header>

      <main className="mx-auto max-w-6xl px-4 py-6">
        <section className="mb-6">
          <h2 className="mb-2 text-sm font-medium uppercase text-gray-500">
            Facility
          </h2>
          <FacilitySelector
            orgId={ORG_ID}
            selectedId={facilityId}
            onSelect={setFacilityId}
          />
        </section>

        {facilityId && (
          <>
            <div className="mb-4 flex gap-1 border-b border-gray-200">
              {TABS.map((t) => (
                <button
                  key={t.key}
                  onClick={() => setTab(t.key)}
                  className={`px-4 py-2 text-sm font-medium ${
                    tab === t.key
                      ? "border-b-2 border-blue-500 text-blue-600"
                      : "text-gray-500 hover:text-gray-700"
                  }`}
                >
                  {t.label}
                </button>
              ))}
            </div>

            {tab === "results" && (
              <SampleResultsTable facilityId={facilityId} orgId={ORG_ID} />
            )}
            {tab === "trending" && (
              <TrendingCharts facilityId={facilityId} />
            )}
            {tab === "compliance" && (
              <ComplianceView facilityId={facilityId} />
            )}
            {tab === "instruments" && (
              <InstrumentsView facilityId={facilityId} />
            )}
          </>
        )}

        {!facilityId && (
          <div className="mt-12 text-center text-gray-400">
            Select a facility to get started
          </div>
        )}
      </main>
    </div>
  );
}

export default App;
