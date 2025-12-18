'use client';

import { useEffect, useState } from 'react';
import { useParams, useRouter } from 'next/navigation';
import Link from 'next/link';
import { 
  getMonitoringJob, 
  getMonitoringJobRuns,
  enableMonitoringJob,
  disableMonitoringJob,
  deleteMonitoringJob,
  MonitoringJob, 
  MonitoringJobRun 
} from '@/lib/api';
import { Card } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { toast } from 'sonner';

export default function JobDetailsPage() {
  const params = useParams();
  const router = useRouter();
  const jobId = params.id as string;

  const [job, setJob] = useState<MonitoringJob | null>(null);
  const [runs, setRuns] = useState<MonitoringJobRun[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    loadJobDetails();
  }, [jobId]);

  async function loadJobDetails() {
    try {
      setLoading(true);
      const [jobResponse, runsResponse] = await Promise.all([
        getMonitoringJob(jobId),
        getMonitoringJobRuns(jobId, 20),
      ]);
      setJob(jobResponse.job);
      setRuns(runsResponse.runs);
    } catch (error) {
      toast.error('Failed to load job details');
      console.error('Load job details error:', error);
    } finally {
      setLoading(false);
    }
  }

  async function handleToggleEnabled() {
    if (!job) return;

    try {
      if (job.is_enabled) {
        await disableMonitoringJob(job.id);
        toast.success('Job disabled');
      } else {
        await enableMonitoringJob(job.id);
        toast.success('Job enabled and scheduled');
      }
      loadJobDetails();
    } catch (error) {
      toast.error('Failed to toggle job status');
      console.error('Toggle error:', error);
    }
  }

  async function handleDelete() {
    if (!job) return;
    if (!confirm('Are you sure you want to delete this monitoring job?')) {
      return;
    }

    try {
      await deleteMonitoringJob(job.id);
      toast.success('Job deleted');
      router.push('/monitoring/jobs');
    } catch (error) {
      toast.error('Failed to delete job');
      console.error('Delete error:', error);
    }
  }

  function getRunStatusColor(status: string) {
    switch (status) {
      case 'success': return 'bg-green-600';
      case 'failed': return 'bg-red-600';
      case 'partial': return 'bg-yellow-600';
      case 'running': return 'bg-blue-600';
      default: return 'bg-gray-600';
    }
  }

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="text-gray-400">Loading job details...</div>
      </div>
    );
  }

  if (!job) {
    return (
      <div className="text-center py-12">
        <h2 className="text-2xl font-bold text-white mb-4">Job Not Found</h2>
        <Link href="/monitoring/jobs">
          <Button className="bg-orange hover:bg-orange/90">Back to Jobs</Button>
        </Link>
      </div>
    );
  }

  const metrics = job.metrics ? JSON.parse(job.metrics) : [];
  const rules = job.rules ? JSON.parse(job.rules) : [];

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex justify-between items-start">
        <div>
          <Link href="/monitoring/jobs" className="text-orange hover:underline text-sm mb-2 inline-block">
            ← Back to Jobs
          </Link>
          <h1 className="text-3xl font-bold text-white">{job.name}</h1>
          <p className="text-gray-400 mt-1">{job.description || 'No description'}</p>
        </div>
        <div className="flex gap-2">
          <Button
            variant="outline"
            onClick={handleToggleEnabled}
            className="border-blue hover:bg-blue"
          >
            {job.is_enabled ? 'Disable Job' : 'Enable Job'}
          </Button>
          <Button
            variant="outline"
            onClick={handleDelete}
            className="border-red-600 text-red-600 hover:bg-red-600 hover:text-white"
          >
            Delete Job
          </Button>
        </div>
      </div>

      {/* Job Info Cards */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        <Card className="bg-navy border-blue p-6">
          <div className="text-sm text-gray-400">Status</div>
          <div className="mt-2">
            <Badge className={job.is_enabled ? 'bg-green-600' : 'bg-gray-600'}>
              {job.is_enabled ? 'ENABLED' : 'DISABLED'}
            </Badge>
          </div>
        </Card>

        <Card className="bg-navy border-blue p-6">
          <div className="text-sm text-gray-400">Schedule</div>
          <div className="text-lg font-mono text-white mt-2">{job.cron_expr}</div>
        </Card>

        <Card className="bg-navy border-blue p-6">
          <div className="text-sm text-gray-400">Metrics Monitored</div>
          <div className="text-2xl font-bold text-white mt-2">{metrics.length}</div>
        </Card>

        <Card className="bg-navy border-blue p-6">
          <div className="text-sm text-gray-400">Last Run Alerts</div>
          <div className="text-2xl font-bold text-orange mt-2">
            {job.last_run_alerts !== undefined ? job.last_run_alerts : '-'}
          </div>
        </Card>
      </div>

      {/* Configuration Details */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        {/* Metrics */}
        <Card className="bg-navy border-blue">
          <div className="p-6 border-b border-blue">
            <h2 className="text-xl font-semibold text-white">Monitored Metrics</h2>
          </div>
          <div className="p-6">
            {metrics.length === 0 ? (
              <div className="text-gray-400">No metrics configured</div>
            ) : (
              <ul className="space-y-2">
                {metrics.map((metric: string, index: number) => (
                  <li key={index} className="flex items-center gap-2">
                    <div className="w-2 h-2 bg-orange rounded-full"></div>
                    <span className="text-white">{metric}</span>
                  </li>
                ))}
              </ul>
            )}
          </div>
        </Card>

        {/* Rules */}
        <Card className="bg-navy border-blue">
          <div className="p-6 border-b border-blue">
            <h2 className="text-xl font-semibold text-white">Monitoring Rules</h2>
          </div>
          <div className="p-6">
            {rules.length === 0 ? (
              <div className="text-gray-400">No rules configured</div>
            ) : (
              <ul className="space-y-2">
                {rules.map((ruleId: string, index: number) => (
                  <li key={index} className="flex items-center gap-2">
                    <div className="w-2 h-2 bg-blue rounded-full"></div>
                    <span className="text-white font-mono text-sm">{ruleId}</span>
                  </li>
                ))}
              </ul>
            )}
          </div>
        </Card>
      </div>

      {/* Last Run Info */}
      {job.last_run_at && (
        <Card className="bg-navy border-blue p-6">
          <h2 className="text-xl font-semibold text-white mb-4">Last Execution</h2>
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            <div>
              <div className="text-sm text-gray-400">Time</div>
              <div className="text-white mt-1">{new Date(job.last_run_at).toLocaleString()}</div>
            </div>
            <div>
              <div className="text-sm text-gray-400">Status</div>
              <div className="mt-1">
                <Badge className={getRunStatusColor(job.last_run_status || 'unknown')}>
                  {(job.last_run_status || 'unknown').toUpperCase()}
                </Badge>
              </div>
            </div>
            <div>
              <div className="text-sm text-gray-400">Alerts Created</div>
              <div className="text-white mt-1">{job.last_run_alerts || 0}</div>
            </div>
          </div>
        </Card>
      )}

      {/* Execution History */}
      <Card className="bg-navy border-blue">
        <div className="p-6 border-b border-blue">
          <h2 className="text-xl font-semibold text-white">Execution History</h2>
        </div>
        <div className="overflow-x-auto">
          {runs.length === 0 ? (
            <div className="p-8 text-center text-gray-400">No execution history available</div>
          ) : (
            <table className="w-full">
              <thead>
                <tr className="border-b border-blue">
                  <th className="text-left p-4 text-gray-400 font-medium">Started At</th>
                  <th className="text-left p-4 text-gray-400 font-medium">Completed At</th>
                  <th className="text-left p-4 text-gray-400 font-medium">Status</th>
                  <th className="text-left p-4 text-gray-400 font-medium">Metrics Checked</th>
                  <th className="text-left p-4 text-gray-400 font-medium">Alerts Created</th>
                  <th className="text-left p-4 text-gray-400 font-medium">Error</th>
                </tr>
              </thead>
              <tbody>
                {runs.map((run) => (
                  <tr key={run.id} className="border-b border-blue hover:bg-blue/20">
                    <td className="p-4 text-gray-300">
                      {new Date(run.started_at).toLocaleString()}
                    </td>
                    <td className="p-4 text-gray-300">
                      {run.completed_at ? new Date(run.completed_at).toLocaleString() : '-'}
                    </td>
                    <td className="p-4">
                      <Badge className={`${getRunStatusColor(run.status)} text-white`}>
                        {run.status.toUpperCase()}
                      </Badge>
                    </td>
                    <td className="p-4 text-white">{run.metrics_checked}</td>
                    <td className="p-4">
                      {run.alerts_created > 0 ? (
                        <Badge className="bg-red-600 text-white">{run.alerts_created}</Badge>
                      ) : (
                        <span className="text-gray-500">0</span>
                      )}
                    </td>
                    <td className="p-4 text-red-400 text-sm">
                      {run.error_message || '-'}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </div>
      </Card>
    </div>
  );
}
