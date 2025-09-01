"use client";
import Link from "next/link";
import { useEffect, useState } from "react";
import { Card } from "@/components/ui/card";
import { getPipelines } from "@/lib/api";

export default function PipelinesPage() {
  const [pipelines, setPipelines] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    async function fetchData() {
      try {
        setLoading(true);
        const data = await getPipelines();
        setPipelines(data);
      } catch (err: any) {
        setError(err.message);
      } finally {
        setLoading(false);
      }
    }
    fetchData();
  }, []);

  return (
    <div>
      <h1 className="text-2xl font-bold text-orange mb-6">Pipelines</h1>
      {loading && <p>Loading...</p>}
      {error && <p className="text-red-500">Error: {error}</p>}
      <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-6">
        {pipelines.map((pipeline) => (
          <Card key={pipeline.id} className="bg-navy text-white border-blue">
            <h2 className="text-xl font-bold text-orange mb-2">{pipeline.name}</h2>
            <p className="mb-2">ID: {pipeline.id}</p>
            <p className="mb-2">Status: {pipeline.status || "Unknown"}</p>
            <div className="flex space-x-2 mt-4">
              <Link href={`/pipelines/${pipeline.id}`} className="px-3 py-1 bg-blue text-white rounded hover:bg-orange">View</Link>
              {/* TODO: Add Run, Clone, Delete actions */}
            </div>
          </Card>
        ))}
      </div>
    </div>
  );
}
