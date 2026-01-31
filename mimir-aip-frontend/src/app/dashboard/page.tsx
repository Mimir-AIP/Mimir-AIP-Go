"use client";
import { useEffect, useState } from "react";
import Link from "next/link";
import { Card } from "@/components/ui/card";
import { 
  GitBranch, 
  Network, 
  Copy, 
  CheckCircle, 
  Clock,
  AlertCircle,
  Activity
} from "lucide-react";
import { getPipelines, listOntologies, listDigitalTwins, getRecentJobs, type Job, type Pipeline, type Ontology, type DigitalTwin } from "@/lib/api";

export default function DashboardPage() {
  const [pipelines, setPipelines] = useState<Pipeline[]>([]);
  const [ontologies, setOntologies] = useState<Ontology[]>([]);
  const [twins, setTwins] = useState<DigitalTwin[]>([]);
  const [recentJobs, setRecentJobs] = useState<Job[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let mounted = true;
    
    async function fetchData() {
      try {
        setLoading(true);
        setError(null);
        
        const [pipelinesData, ontologiesData, twinsData, jobsData] = await Promise.all([
          getPipelines().catch(() => []),
          listOntologies().catch(() => []),
          listDigitalTwins().catch(() => []),
          getRecentJobs().catch(() => []),
        ]);
        
        if (!mounted) return;
        
        setPipelines(pipelinesData);
        setOntologies(ontologiesData);
        setTwins(twinsData);
        setRecentJobs(jobsData);
      } catch (err) {
        if (mounted) {
          setError(err instanceof Error ? err.message : "Unknown error");
        }
      } finally {
        if (mounted) {
          setLoading(false);
        }
      }
    }
    
    fetchData();
    
    return () => {
      mounted = false;
    };
  }, []);

  // Determine system health based on recent job status
  const failedJobs = recentJobs.filter(j => j.status === 'failed');
  const hasJobs = recentJobs.length > 0;
  const systemHealthy = failedJobs.length === 0;

  if (loading) {
    return (
      <div className="space-y-6">
        <div className="mb-8">
          <h1 className="text-3xl font-bold text-orange mb-2">Dashboard</h1>
          <p className="text-white/60">Loading system status...</p>
        </div>
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
          {[1, 2, 3, 4].map(i => (
            <Card key={i} className="bg-navy border-blue p-6 animate-pulse">
              <div className="h-8 bg-blue/30 rounded mb-2"></div>
              <div className="h-4 bg-blue/20 rounded"></div>
            </Card>
          ))}
        </div>
      </div>
    );
  }
  
  if (error) {
    return (
      <div className="space-y-6">
        <div className="mb-8">
          <h1 className="text-3xl font-bold text-orange mb-2">Dashboard</h1>
        </div>
        <Card className="col-span-full bg-red-900/20 text-white border-red-500 p-8">
          <h2 className="text-2xl font-bold mb-2">Error Loading Dashboard</h2>
          <p className="font-mono text-sm text-red-400">{error}</p>
        </Card>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Page Header */}
      <div className="mb-8">
        <h1 className="text-3xl font-bold text-orange mb-2">Dashboard</h1>
        <p className="text-white/60">System monitoring and overview</p>
      </div>

      {/* System Status */}
      <Card className={`p-6 border-l-4 ${!hasJobs ? 'border-l-blue-500 bg-navy' : systemHealthy ? 'border-l-green-500 bg-navy' : 'border-l-red-500 bg-red-900/10'}`}>
        <div className="flex items-center gap-4">
          <div className={`w-12 h-12 rounded-full flex items-center justify-center ${!hasJobs ? 'bg-blue-500/20' : systemHealthy ? 'bg-green-500/20' : 'bg-red-500/20'}`}>
            <Activity className={`w-6 h-6 ${!hasJobs ? 'text-blue-400' : systemHealthy ? 'text-green-500' : 'text-red-500'}`} />
          </div>
          <div>
            <h2 className="text-xl font-semibold text-white">System Status</h2>
            <p className={!hasJobs ? 'text-blue-400' : systemHealthy ? 'text-green-400' : 'text-red-400'}>
              {!hasJobs ? 'No recent executions - ready to process' : systemHealthy ? 'All systems operational' : `${failedJobs.length} recent job(s) failed`}
            </p>
          </div>
        </div>
      </Card>

      {/* Stats Overview */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
        <Link href="/pipelines">
          <Card className="bg-navy border-blue p-6 hover:border-orange/50 transition-colors cursor-pointer">
            <div className="flex items-center justify-between mb-4">
              <GitBranch className="w-10 h-10 text-orange p-2 bg-orange/10 rounded-lg" />
              <span className="text-xs text-white/40 uppercase tracking-wide">Active</span>
            </div>
            <p className="text-4xl font-bold text-white mb-1">{pipelines.length}</p>
            <p className="text-sm text-white/60">Data Pipelines</p>
          </Card>
        </Link>
        
        <Link href="/ontologies">
          <Card className="bg-navy border-blue p-6 hover:border-orange/50 transition-colors cursor-pointer">
            <div className="flex items-center justify-between mb-4">
              <Network className="w-10 h-10 text-blue-400 p-2 bg-blue-400/10 rounded-lg" />
              <span className="text-xs text-white/40 uppercase tracking-wide">Active</span>
            </div>
            <p className="text-4xl font-bold text-white mb-1">{ontologies.filter(o => o.status === 'active').length}</p>
            <p className="text-sm text-white/60">Ontologies</p>
          </Card>
        </Link>
        
        <Link href="/digital-twins">
          <Card className="bg-navy border-blue p-6 hover:border-orange/50 transition-colors cursor-pointer">
            <div className="flex items-center justify-between mb-4">
              <Copy className="w-10 h-10 text-purple-400 p-2 bg-purple-400/10 rounded-lg" />
              <span className="text-xs text-white/40 uppercase tracking-wide">Total</span>
            </div>
            <p className="text-4xl font-bold text-white mb-1">{twins.length}</p>
            <p className="text-sm text-white/60">Digital Twins</p>
          </Card>
        </Link>
        
        <Card className="bg-navy border-blue p-6">
          <div className="flex items-center justify-between mb-4">
            <Clock className="w-10 h-10 text-yellow-400 p-2 bg-yellow-400/10 rounded-lg" />
            <span className="text-xs text-white/40 uppercase tracking-wide">24h</span>
          </div>
          <p className="text-4xl font-bold text-white mb-1">{recentJobs.length}</p>
          <p className="text-sm text-white/60">Recent Executions</p>
        </Card>
      </div>

      {/* Recent Activity */}
      <Card className="bg-navy border-blue p-6">
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-xl font-bold text-orange">Recent Pipeline Executions</h2>
          <span className="text-xs text-white/40">Last 24 hours</span>
        </div>
        
        {recentJobs.length > 0 ? (
          <div className="space-y-3">
            {recentJobs.slice(0, 5).map((job, i) => (
              <div key={job.id || i} className="flex items-center justify-between p-3 bg-blue/20 rounded-lg">
                <div className="flex items-center gap-3">
                  {job.status === 'completed' ? (
                    <CheckCircle className="w-5 h-5 text-green-400" />
                  ) : job.status === 'failed' ? (
                    <AlertCircle className="w-5 h-5 text-red-400" />
                  ) : (
                    <Clock className="w-5 h-5 text-orange" />
                  )}
                  <div>
                    <span className="text-white font-medium">{job.name || job.id}</span>
                    <span className="text-xs text-white/40 ml-2">{job.pipeline}</span>
                  </div>
                </div>
                <span className={`text-xs px-2 py-1 rounded-full ${
                  job.status === 'completed' ? 'bg-green-500/20 text-green-400' :
                  job.status === 'failed' ? 'bg-red-500/20 text-red-400' :
                  'bg-orange/20 text-orange'
                }`}>
                  {job.status}
                </span>
              </div>
            ))}
          </div>
        ) : (
          <p className="text-white/40 text-center py-8">No recent pipeline executions</p>
        )}
      </Card>

      {/* Quick Links */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
        <Link href="/pipelines">
          <Card className="bg-gradient-to-br from-blue to-blue/80 border-blue hover:border-orange transition-all p-4 cursor-pointer group">
            <GitBranch className="w-8 h-8 text-orange mb-2 group-hover:scale-110 transition-transform" />
            <h3 className="font-semibold text-white">View Pipelines</h3>
            <p className="text-xs text-white/60">Monitor data ingestion</p>
          </Card>
        </Link>
        <Link href="/ontologies">
          <Card className="bg-gradient-to-br from-blue to-blue/80 border-blue hover:border-orange transition-all p-4 cursor-pointer group">
            <Network className="w-8 h-8 text-orange mb-2 group-hover:scale-110 transition-transform" />
            <h3 className="font-semibold text-white">View Ontologies</h3>
            <p className="text-xs text-white/60">Knowledge schemas</p>
          </Card>
        </Link>
        <Link href="/chat">
          <Card className="bg-gradient-to-br from-blue to-blue/80 border-blue hover:border-orange transition-all p-4 cursor-pointer group">
            <Activity className="w-8 h-8 text-orange mb-2 group-hover:scale-110 transition-transform" />
            <h3 className="font-semibold text-white">Agent Chat</h3>
            <p className="text-xs text-white/60">Interact with AI agent</p>
          </Card>
        </Link>
      </div>
    </div>
  );
}
