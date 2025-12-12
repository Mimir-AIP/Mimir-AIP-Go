"use client";
import { useEffect, useState } from "react";
import { Card } from "@/components/ui/card";
import { getJobs, getRunningJobs, getRecentJobs, getPerformanceMetrics, type Job } from "@/lib/api";
// If shadcn/ui chart components are not installed, you can use recharts or chart.js as a fallback

export default function DashboardPage() {
  const [jobs, setJobs] = useState<Job[]>([]);
  const [runningJobs, setRunningJobs] = useState<Job[]>([]);
  const [recentJobs, setRecentJobs] = useState<Job[]>([]);
  const [metrics, setMetrics] = useState<Record<string, unknown>>({});
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let mounted = true;
    
    async function fetchData() {
      try {
        console.log("[Dashboard] Fetching data...");
        setLoading(true);
        setError(null);
        
        const [jobsData, runningData, recentData, metricsData] = await Promise.all([
          getJobs().catch(err => { console.error("[Dashboard] getJobs failed:", err); return []; }),
          getRunningJobs().catch(err => { console.error("[Dashboard] getRunningJobs failed:", err); return []; }),
          getRecentJobs().catch(err => { console.error("[Dashboard] getRecentJobs failed:", err); return []; }),
          getPerformanceMetrics().catch(err => { console.error("[Dashboard] getPerformanceMetrics failed:", err); return {}; }),
        ]);
        
        if (!mounted) return;
        
        console.log("[Dashboard] Data fetched successfully", { 
          jobs: jobsData.length, 
          running: runningData.length, 
          recent: recentData.length 
        });
        
        setJobs(jobsData);
        setRunningJobs(runningData);
        setRecentJobs(recentData);
        setMetrics(metricsData);
      } catch (err) {
        console.error("[Dashboard] Fetch error:", err);
        if (mounted) {
          setError(err instanceof Error ? err.message : "Unknown error");
        }
      } finally {
        if (mounted) {
          console.log("[Dashboard] Loading complete");
          setLoading(false);
        }
      }
    }
    
    fetchData();
    
    return () => {
      mounted = false;
    };
  }, []);

  if (loading) {
    return (
      <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-6">
        <Card className="col-span-full bg-blue text-white border-orange p-8 text-center">
          <h2 className="text-2xl font-bold mb-2">Loading Dashboard...</h2>
          <p className="text-sm opacity-75">Fetching data from API</p>
        </Card>
      </div>
    );
  }
  
  if (error) {
    return (
      <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-6">
        <Card className="col-span-full bg-red-900 text-white border-orange p-8">
          <h2 className="text-2xl font-bold mb-2">Error Loading Dashboard</h2>
          <p className="font-mono text-sm">{error}</p>
          <button 
            onClick={() => window.location.reload()} 
            className="mt-4 px-4 py-2 bg-orange text-navy rounded hover:bg-orange/90"
          >
            Retry
          </button>
        </Card>
      </div>
    );
  }

  return (
    <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-6">
      <Card className="bg-navy text-white border-blue p-6">
        <h2 className="text-xl font-bold text-orange mb-2">Total Jobs</h2>
        <p className="text-3xl font-semibold">{jobs.length}</p>
      </Card>
      <Card className="bg-navy text-white border-blue p-6">
        <h2 className="text-xl font-bold text-orange mb-2">Running Jobs</h2>
        <p className="text-3xl font-semibold">{runningJobs.length}</p>
      </Card>
      <Card className="bg-navy text-white border-blue p-6">
        <h2 className="text-xl font-bold text-orange mb-2">Recent Jobs</h2>
        <p className="text-3xl font-semibold">{recentJobs.length}</p>
      </Card>
      <Card className="col-span-1 md:col-span-2 xl:col-span-3 bg-navy text-white border-blue p-6">
        <h2 className="text-xl font-bold text-orange mb-4">Performance Metrics</h2>
        <pre className="bg-blue/10 p-4 rounded text-white overflow-x-auto text-sm">
          {JSON.stringify(metrics, null, 2)}
        </pre>
      </Card>
    </div>
  );
}
