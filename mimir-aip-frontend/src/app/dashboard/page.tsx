"use client";
import { useEffect, useState } from "react";
import { Card } from "@/components/ui/card";
import { getJobs, getRunningJobs, getRecentJobs, getPerformanceMetrics } from "@/lib/api";
// If shadcn/ui chart components are not installed, you can use recharts or chart.js as a fallback

export default function DashboardPage() {
  const [jobs, setJobs] = useState([]);
  const [runningJobs, setRunningJobs] = useState([]);
  const [recentJobs, setRecentJobs] = useState([]);
  const [metrics, setMetrics] = useState<any>({});
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    async function fetchData() {
      try {
        setLoading(true);
        const [jobsData, runningData, recentData, metricsData] = await Promise.all([
          getJobs(),
          getRunningJobs(),
          getRecentJobs(),
          getPerformanceMetrics(),
        ]);
        setJobs(jobsData);
        setRunningJobs(runningData);
        setRecentJobs(recentData);
        setMetrics(metricsData);
      } catch (err: any) {
        setError(err.message);
      } finally {
        setLoading(false);
      }
    }
    fetchData();
  }, []);

  return (
    <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-6">
      <Card className="bg-navy text-white border-blue">
        <h2 className="text-xl font-bold text-orange mb-2">Total Jobs</h2>
        <p className="text-3xl font-semibold">{jobs.length}</p>
      </Card>
      <Card className="bg-navy text-white border-blue">
        <h2 className="text-xl font-bold text-orange mb-2">Running Jobs</h2>
        <p className="text-3xl font-semibold">{runningJobs.length}</p>
      </Card>
      <Card className="bg-navy text-white border-blue">
        <h2 className="text-xl font-bold text-orange mb-2">Recent Jobs</h2>
        <p className="text-3xl font-semibold">{recentJobs.length}</p>
      </Card>
      <Card className="col-span-1 md:col-span-2 xl:col-span-3 bg-navy text-white border-blue">
        <h2 className="text-xl font-bold text-orange mb-4">Performance Metrics</h2>
        <pre className="bg-blue/10 p-4 rounded text-white overflow-x-auto">
          {JSON.stringify(metrics, null, 2)}
        </pre>
      </Card>
      {/* TODO: Add shadcn/ui chart components for job status and trends */}
      {error && (
        <Card className="col-span-full bg-red-900 text-white border-orange">
          <p>Error: {error}</p>
        </Card>
      )}
      {loading && (
        <Card className="col-span-full bg-blue text-white border-orange">
          <p>Loading...</p>
        </Card>
      )}
    </div>
  );
}
