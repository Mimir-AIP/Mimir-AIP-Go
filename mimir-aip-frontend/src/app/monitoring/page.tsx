'use client';

import { useEffect, useState } from 'react';
import Link from 'next/link';
import { listMonitoringJobs, listAlerts, MonitoringJob, Alert } from '@/lib/api';
import { Card } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';

export default function MonitoringDashboard() {
  const [jobs, setJobs] = useState<MonitoringJob[]>([]);
  const [alerts, setAlerts] = useState<Alert[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    loadData();
  }, []);

  async function loadData() {
    try {
      setLoading(true);
      const [jobsResponse, alertsResponse] = await Promise.all([
        listMonitoringJobs(),
        listAlerts(),
      ]);
      setJobs(Array.isArray(jobsResponse.jobs) ? jobsResponse.jobs : []);
      setAlerts(Array.isArray(alertsResponse.alerts) ? alertsResponse.alerts : []);
    } catch (error) {
      console.error('Load data error:', error);
      setJobs([]);
      setAlerts([]);
    } finally {
      setLoading(false);
    }
  }

  const enabledJobs = jobs.filter(j => j.is_enabled);
  const activeAlerts = alerts.filter(a => a.status === 'active');
  const criticalAlerts = alerts.filter(a => a.severity === 'critical' && a.status === 'active');
  const highAlerts = alerts.filter(a => a.severity === 'high' && a.status === 'active');

  const recentAlerts = alerts
    .sort((a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime())
    .slice(0, 10);

  function getSeverityColor(severity: string) {
    switch (severity) {
      case 'critical': return 'bg-red-600';
      case 'high': return 'bg-orange-600';
      case 'medium': return 'bg-yellow-600';
      case 'low': return 'bg-blue-600';
      default: return 'bg-gray-600';
    }
  }

  function getStatusColor(status: string) {
    switch (status) {
      case 'active': return 'text-red-500';
      case 'acknowledged': return 'text-yellow-500';
      case 'resolved': return 'text-green-500';
      default: return 'text-gray-500';
    }
  }

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="text-gray-400">Loading monitoring dashboard...</div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-3xl font-bold text-orange">Monitoring Dashboard</h1>
        <p className="text-gray-400 mt-1">Overview of system monitoring and alerts</p>
      </div>

      {/* Summary Cards */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        <Card className="bg-navy border-blue p-6">
          <div className="text-sm text-gray-400">Active Jobs</div>
          <div className="text-3xl font-bold text-green-500 mt-2">{enabledJobs.length}</div>
          <div className="text-xs text-gray-500 mt-1">of {jobs.length} total</div>
        </Card>
        
        <Card className="bg-navy border-blue p-6">
          <div className="text-sm text-gray-400">Active Alerts</div>
          <div className="text-3xl font-bold text-orange mt-2">{activeAlerts.length}</div>
          <Link href="/monitoring/alerts?status=active" className="text-xs text-orange hover:underline mt-1 inline-block">
            View all
          </Link>
        </Card>
        
        <Card className="bg-navy border-blue p-6">
          <div className="text-sm text-gray-400">Critical Alerts</div>
          <div className="text-3xl font-bold text-red-500 mt-2">{criticalAlerts.length}</div>
          {criticalAlerts.length > 0 && (
            <Link href="/monitoring/alerts?severity=critical&status=active" className="text-xs text-red-400 hover:underline mt-1 inline-block">
              Needs attention
            </Link>
          )}
        </Card>
        
        <Card className="bg-navy border-blue p-6">
          <div className="text-sm text-gray-400">High Priority</div>
          <div className="text-3xl font-bold text-orange mt-2">{highAlerts.length}</div>
          {highAlerts.length > 0 && (
            <Link href="/monitoring/alerts?severity=high&status=active" className="text-xs text-orange hover:underline mt-1 inline-block">
              Review now
            </Link>
          )}
        </Card>
      </div>

      {/* Quick Actions */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        <Link href="/monitoring/jobs">
          <Card className="bg-navy border-blue p-6 hover:border-orange transition-colors cursor-pointer">
            <h3 className="text-lg font-semibold text-white">Monitoring Jobs</h3>
            <p className="text-gray-400 mt-2">Manage automated monitoring tasks</p>
            <Button className="mt-4 bg-orange hover:bg-orange/90 w-full">
              View Jobs
            </Button>
          </Card>
        </Link>
        
        <Link href="/monitoring/rules">
          <Card className="bg-navy border-blue p-6 hover:border-orange transition-colors cursor-pointer">
            <h3 className="text-lg font-semibold text-white">Monitoring Rules</h3>
            <p className="text-gray-400 mt-2">Configure threshold and anomaly detection</p>
            <Button className="mt-4 bg-orange hover:bg-orange/90 w-full">
              View Rules
            </Button>
          </Card>
        </Link>
        
        <Link href="/monitoring/alerts">
          <Card className="bg-navy border-blue p-6 hover:border-orange transition-colors cursor-pointer">
            <h3 className="text-lg font-semibold text-white">Alerts</h3>
            <p className="text-gray-400 mt-2">View and manage system alerts</p>
            <Button className="mt-4 bg-orange hover:bg-orange/90 w-full">
              View Alerts
            </Button>
          </Card>
        </Link>
      </div>

      {/* Recent Alerts */}
      <Card className="bg-navy border-blue">
        <div className="p-6 border-b border-blue">
          <div className="flex justify-between items-center">
            <h2 className="text-xl font-semibold text-white">Recent Alerts</h2>
            <Link href="/monitoring/alerts">
              <Button variant="outline" size="sm" className="border-blue hover:bg-blue">
                View All
              </Button>
            </Link>
          </div>
        </div>
        
        <div className="overflow-x-auto">
          {recentAlerts.length === 0 ? (
            <div className="p-8 text-center text-gray-400">No alerts to display</div>
          ) : (
            <table className="w-full">
              <thead>
                <tr className="border-b border-blue">
                  <th className="text-left p-4 text-gray-400 font-medium">Time</th>
                  <th className="text-left p-4 text-gray-400 font-medium">Metric</th>
                  <th className="text-left p-4 text-gray-400 font-medium">Severity</th>
                  <th className="text-left p-4 text-gray-400 font-medium">Status</th>
                  <th className="text-left p-4 text-gray-400 font-medium">Message</th>
                  <th className="text-left p-4 text-gray-400 font-medium">Value</th>
                </tr>
              </thead>
              <tbody>
                {recentAlerts.map((alert) => (
                  <tr key={alert.id} className="border-b border-blue hover:bg-blue/20">
                    <td className="p-4 text-gray-400">
                      {new Date(alert.created_at).toLocaleString()}
                    </td>
                    <td className="p-4 text-white font-medium">{alert.metric_name}</td>
                    <td className="p-4">
                      <Badge className={`${getSeverityColor(alert.severity)} text-white`}>
                        {alert.severity.toUpperCase()}
                      </Badge>
                    </td>
                    <td className="p-4">
                      <span className={`font-medium ${getStatusColor(alert.status)}`}>
                        {alert.status.toUpperCase()}
                      </span>
                    </td>
                    <td className="p-4 text-gray-300">{alert.message}</td>
                    <td className="p-4 text-white">{alert.value.toFixed(2)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </div>
      </Card>

      {/* Active Jobs Status */}
      <Card className="bg-navy border-blue">
        <div className="p-6 border-b border-blue">
          <div className="flex justify-between items-center">
            <h2 className="text-xl font-semibold text-white">Active Monitoring Jobs</h2>
            <Link href="/monitoring/jobs">
              <Button variant="outline" size="sm" className="border-blue hover:bg-blue">
                Manage Jobs
              </Button>
            </Link>
          </div>
        </div>
        
        <div className="p-6">
          {enabledJobs.length === 0 ? (
            <div className="text-center text-gray-400 py-4">
              No active monitoring jobs.{' '}
              <Link href="/monitoring/jobs/create" className="text-orange hover:underline">
                Create one
              </Link>
            </div>
          ) : (
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              {enabledJobs.map((job) => {
                const metrics = job.metrics ? JSON.parse(job.metrics).length : 0;
                return (
                  <Card key={job.id} className="bg-navy/50 border-blue p-4">
                    <div className="flex justify-between items-start">
                      <div>
                        <Link 
                          href={`/monitoring/jobs/${job.id}`}
                          className="text-orange hover:underline font-medium"
                        >
                          {job.name}
                        </Link>
                        <div className="text-sm text-gray-400 mt-1">
                          {metrics} metrics | {job.cron_expr}
                        </div>
                      </div>
                      {job.last_run_alerts !== undefined && job.last_run_alerts > 0 && (
                        <Badge className="bg-red-600 text-white">{job.last_run_alerts}</Badge>
                      )}
                    </div>
                    <div className="text-xs text-gray-500 mt-2">
                      Last run: {job.last_run_at ? new Date(job.last_run_at).toLocaleString() : 'Never'}
                    </div>
                  </Card>
                );
              })}
            </div>
          )}
        </div>
      </Card>
    </div>
  );
}
