"use client";
import { useEffect, useState } from "react";
import { useParams } from "next/navigation";
import { Card } from "@/components/ui/card";
import { getPipeline, executePipeline } from "@/lib/api";

export default function PipelineDetailPage() {
  const { id } = useParams();
  const [pipeline, setPipeline] = useState<any>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [runStatus, setRunStatus] = useState<string | null>(null);

  useEffect(() => {
    async function fetchData() {
      try {
        setLoading(true);
        const data = await getPipeline(id as string);
        setPipeline(data);
      } catch (err: any) {
        setError(err.message);
      } finally {
        setLoading(false);
      }
    }
    if (id) fetchData();
  }, [id]);

  async function handleRun() {
    try {
      setRunStatus("Running...");
      await executePipeline(id as string, {}); // Pass any required body
      setRunStatus("Pipeline executed successfully.");
    } catch (err: any) {
      setRunStatus("Error: " + err.message);
    }
  }

  return (
    <div>
      <h1 className="text-2xl font-bold text-orange mb-6">Pipeline Details</h1>
      {loading && <p>Loading...</p>}
      {error && <p className="text-red-500">Error: {error}</p>}
      {pipeline && (
        <Card className="bg-navy text-white border-blue">
          <h2 className="text-xl font-bold text-orange mb-2">{pipeline.name}</h2>
          <p className="mb-2">ID: {pipeline.id}</p>
          <p className="mb-2">Status: {pipeline.status || "Unknown"}</p>
          <pre className="bg-blue/10 p-4 rounded text-white overflow-x-auto mb-4">
            {JSON.stringify(pipeline, null, 2)}
          </pre>
          <button onClick={handleRun} className="px-4 py-2 bg-orange text-navy rounded hover:bg-blue hover:text-white font-bold">Run Pipeline</button>
          {runStatus && <p className="mt-2">{runStatus}</p>}
        </Card>
      )}
    </div>
  );
}
