'use client';

import { useEffect, useState } from 'react';
import Link from 'next/link';
import { 
  listMonitoringJobs, 
  enableMonitoringJob, 
  disableMonitoringJob,
  deleteMonitoringJob,
  MonitoringJob 
} from '@/lib/api';
import { Card } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { toast } from 'sonner';

export default function MonitoringJobsPage() {
  const [jobs, setJobs] = useState<MonitoringJob[]>([]);
  const [loading, setLoading] = useState(true);
  const [filter, setFilter] = useState<{ enabled_only?: boolean }>({});

  useEffect(() => {
    loadJobs();
  }, [filter]);

  async function loadJobs() {
    try {
      setLoading(true);
      const response = await listMonitoringJobs(filter);
      setJobs(response?.jobs || []);
    } catch (error) {
      toast.error('Failed to load monitoring jobs');
      console.error('Load jobs error:', error);
      setJobs([]);
    } finally {
      setLoading(false);
    }
  }

  async function handleToggleEnabled(job: MonitoringJob) {
    try {
      if (job.is_enabled) {
        await disableMonitoringJob(job.id);
        toast.success('Job disabled');
      } else {
        await enableMonitoringJob(job.id);
        toast.success('Job enabled');
      }
      loadJobs();
    } catch (error) {
      toast.error('Failed to toggle job status');
      console.error('Toggle error:', error);
    }
  }

  async function handleDelete(id: string) {
    if (!confirm('Are you sure you want to delete this monitoring job?')) {
      return;
    }

    try {
      await deleteMonitoringJob(id);
      toast.success('Job deleted');
      loadJobs();
    } catch (error) {
      toast.error('Failed to delete job');
      console.error('Delete error:', error);
    }
  }

  function parseMetrics(metricsStr: string): string[] {
    try {
      return JSON.parse(metricsStr);
    } catch {
      return [];
    }
  }

  function getStatusBadge(job: MonitoringJob) {
    if (!job.is_enabled) {
      return <Badge className="bg-gray-600 text-white">DISABLED</Badge>;
    }
    
    if (job.last_run_status === 'success') {
      return <Badge className="bg-green-600 text-white">ACTIVE</Badge>;
    }
    
    if (job.last_run_status === 'failed') {
      return <Badge className="bg-red-600 text-white">FAILED</Badge>;
    }
    
    if (job.last_run_status === 'partial') {
      return <Badge className="bg-yellow-600 text-white">PARTIAL</Badge>;
    }
    
    return <Badge className="bg-blue-600 text-white">READY</Badge>;
  }

  const enabledJobs = jobs.filter(j => j.is_enabled);
  const disabledJobs = jobs.filter(j => !j.is_enabled);

  return (
    <div className="space-y-6">
      <div className="flex justify-between items-center">
        <div>
          <h1 className="text-3xl font-bold text-orange">Monitoring Jobs</h1>
          <p className="text-gray-400 mt-1">Manage automated monitoring tasks</p>
        </div>
        <Link href="/monitoring/jobs/create">
          <Button className="bg-orange hover:bg-orange/90">Create Job</Button>
        </Link>
      </div>

      {/* Summary Cards */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        <Card className="bg-navy border-blue p-6">
          <div className="text-sm text-gray-400">Total Jobs</div>
          <div className="text-3xl font-bold text-white mt-2">{jobs.length}</div>
        </Card>
        <Card className="bg-navy border-blue p-6">
          <div className="text-sm text-gray-400">Enabled</div>
          <div className="text-3xl font-bold text-green-500 mt-2">{enabledJobs.length}</div>
        </Card>
        <Card className="bg-navy border-blue p-6">
          <div className="text-sm text-gray-400">Disabled</div>
          <div className="text-3xl font-bold text-gray-500 mt-2">{disabledJobs.length}</div>
        </Card>
      </div>

      {/* Filters */}
      <Card className="bg-navy border-blue p-4">
        <div className="flex gap-4">
          <div>
            <label className="text-sm text-gray-400">Show:</label>
            <select
              className="ml-2 bg-navy border border-blue rounded px-3 py-1 text-white"
              value={filter.enabled_only ? 'enabled' : 'all'}
              onChange={(e) => setFilter({ enabled_only: e.target.value === 'enabled' ? true : undefined })}
            >
              <option value="all">All Jobs</option>
              <option value="enabled">Enabled Only</option>
            </select>
          </div>
        </div>
      </Card>

      {/* Jobs List */}
      <Card className="bg-navy border-blue">
        <div className="overflow-x-auto">
          {loading ? (
            <div className="p-8 text-center text-gray-400">Loading jobs...</div>
          ) : jobs.length === 0 ? (
            <div className="p-8 text-center text-gray-400">
              No monitoring jobs found.{' '}
              <Link href="/monitoring/jobs/create" className="text-orange hover:underline">
                Create one
              </Link>
            </div>
          ) : (
            <table className="w-full">
              <thead>
                <tr className="border-b border-blue">
                  <th className="text-left p-4 text-gray-400 font-medium">Name</th>
                  <th className="text-left p-4 text-gray-400 font-medium">Status</th>
                  <th className="text-left p-4 text-gray-400 font-medium">Schedule</th>
                  <th className="text-left p-4 text-gray-400 font-medium">Metrics</th>
                  <th className="text-left p-4 text-gray-400 font-medium">Last Run</th>
                  <th className="text-left p-4 text-gray-400 font-medium">Alerts</th>
                  <th className="text-left p-4 text-gray-400 font-medium">Actions</th>
                </tr>
              </thead>
              <tbody>
                {jobs.map((job) => {
                  const metrics = parseMetrics(job.metrics);
                  return (
                    <tr key={job.id} className="border-b border-blue hover:bg-blue/20">
                      <td className="p-4">
                        <Link 
                          href={`/monitoring/jobs/${job.id}`}
                          className="text-orange hover:underline font-medium"
                        >
                          {job.name}
                        </Link>
                        {job.description && (
                          <div className="text-sm text-gray-400 mt-1">{job.description}</div>
                        )}
                      </td>
                      <td className="p-4">{getStatusBadge(job)}</td>
                      <td className="p-4 text-gray-300 font-mono text-sm">{job.cron_expr}</td>
                      <td className="p-4 text-gray-300">{metrics.length} metrics</td>
                      <td className="p-4 text-gray-400">
                        {job.last_run_at ? new Date(job.last_run_at).toLocaleString() : 'Never'}
                      </td>
                      <td className="p-4">
                        {job.last_run_alerts !== undefined && job.last_run_alerts > 0 ? (
                          <Badge className="bg-red-600 text-white">{job.last_run_alerts}</Badge>
                        ) : (
                          <span className="text-gray-500">0</span>
                        )}
                      </td>
                      <td className="p-4">
                        <div className="flex gap-2">
                          <Button
                            size="sm"
                            variant="outline"
                            onClick={() => handleToggleEnabled(job)}
                            className="border-blue hover:bg-blue"
                          >
                            {job.is_enabled ? 'Disable' : 'Enable'}
                          </Button>
                          <Link href={`/monitoring/jobs/${job.id}`}>
                            <Button
                              size="sm"
                              variant="outline"
                              className="border-blue hover:bg-blue"
                            >
                              Details
                            </Button>
                          </Link>
                          <Button
                            size="sm"
                            variant="outline"
                            onClick={() => handleDelete(job.id)}
                            className="border-red-600 text-red-600 hover:bg-red-600 hover:text-white"
                          >
                            Delete
                          </Button>
                        </div>
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          )}
        </div>
      </Card>
    </div>
  );
}
