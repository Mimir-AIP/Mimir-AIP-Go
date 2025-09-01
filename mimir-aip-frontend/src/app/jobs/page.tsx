"use client";
import { useEffect, useState } from "react";
import { Card } from "@/components/ui/card";
import { getJobs } from "@/lib/api";

export default function JobsPage() {
  const [jobs, setJobs] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    async function fetchData() {
      try {
        setLoading(true);
        const data = await getJobs();
        setJobs(data);
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
      <h1 className="text-2xl font-bold text-orange mb-6">Jobs</h1>
      {loading && <p>Loading...</p>}
      {error && <p className="text-red-500">Error: {error}</p>}
      <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-6">
        {jobs.map((job) => (
          <Card key={job.id} className="bg-navy text-white border-blue">
            <h2 className="text-xl font-bold text-orange mb-2">{job.name || job.id}</h2>
            <p className="mb-2">ID: {job.id}</p>
            <p className="mb-2">Status: {job.status || "Unknown"}</p>
            <p className="mb-2">Pipeline: {job.pipelineId || "N/A"}</p>
            <p className="mb-2">Created: {job.createdAt || "N/A"}</p>
            <div className="flex space-x-2 mt-4">
              {/* TODO: Add Enable/Disable, Delete, View actions */}
            </div>
          </Card>
        ))}
      </div>
    </div>
  );
}
