'use client';

import { useEffect, useState } from 'react';
import { listAlerts, updateAlertStatus, Alert } from '@/lib/api';
import { Card } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { toast } from 'sonner';

export default function AlertsPage() {
  const [alerts, setAlerts] = useState<Alert[]>([]);
  const [loading, setLoading] = useState(true);
  const [filter, setFilter] = useState<{ status?: string; severity?: string }>({});

  useEffect(() => {
    loadAlerts();
  }, [filter]);

  async function loadAlerts() {
    try {
      setLoading(true);
      const response = await listAlerts(filter);
      setAlerts(response.alerts || []);
    } catch (error) {
      toast.error('Failed to load alerts');
      console.error('Load alerts error:', error);
      setAlerts([]);
    } finally {
      setLoading(false);
    }
  }

  async function handleAcknowledge(id: string) {
    try {
      await updateAlertStatus(id, 'acknowledged');
      toast.success('Alert acknowledged');
      loadAlerts();
    } catch (error) {
      toast.error('Failed to acknowledge alert');
      console.error('Acknowledge error:', error);
    }
  }

  async function handleResolve(id: string) {
    try {
      await updateAlertStatus(id, 'resolved');
      toast.success('Alert resolved');
      loadAlerts();
    } catch (error) {
      toast.error('Failed to resolve alert');
      console.error('Resolve error:', error);
    }
  }

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
      case 'active': return 'bg-red-600';
      case 'acknowledged': return 'bg-yellow-600';
      case 'resolved': return 'bg-green-600';
      default: return 'bg-gray-600';
    }
  }

  const activeAlerts = alerts.filter(a => a.status === 'active');
  const acknowledgedAlerts = alerts.filter(a => a.status === 'acknowledged');
  const resolvedAlerts = alerts.filter(a => a.status === 'resolved');

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-3xl font-bold text-orange">Alerts</h1>
        <p className="text-gray-400 mt-1">Monitor and manage system alerts</p>
      </div>

      {/* Summary Cards */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        <Card className="bg-navy border-blue p-6">
          <div className="text-sm text-gray-400">Active Alerts</div>
          <div className="text-3xl font-bold text-red-500 mt-2">{activeAlerts.length}</div>
        </Card>
        <Card className="bg-navy border-blue p-6">
          <div className="text-sm text-gray-400">Acknowledged</div>
          <div className="text-3xl font-bold text-yellow-500 mt-2">{acknowledgedAlerts.length}</div>
        </Card>
        <Card className="bg-navy border-blue p-6">
          <div className="text-sm text-gray-400">Resolved</div>
          <div className="text-3xl font-bold text-green-500 mt-2">{resolvedAlerts.length}</div>
        </Card>
      </div>

      {/* Filters */}
      <Card className="bg-navy border-blue p-4">
        <div className="flex gap-4">
          <div>
            <label className="text-sm text-gray-400">Status:</label>
            <select
              className="ml-2 bg-navy border border-blue rounded px-3 py-1 text-white"
              value={filter.status || ''}
              onChange={(e) => setFilter({ ...filter, status: e.target.value || undefined })}
            >
              <option value="">All</option>
              <option value="active">Active</option>
              <option value="acknowledged">Acknowledged</option>
              <option value="resolved">Resolved</option>
            </select>
          </div>
          <div>
            <label className="text-sm text-gray-400">Severity:</label>
            <select
              className="ml-2 bg-navy border border-blue rounded px-3 py-1 text-white"
              value={filter.severity || ''}
              onChange={(e) => setFilter({ ...filter, severity: e.target.value || undefined })}
            >
              <option value="">All</option>
              <option value="critical">Critical</option>
              <option value="high">High</option>
              <option value="medium">Medium</option>
              <option value="low">Low</option>
            </select>
          </div>
        </div>
      </Card>

      {/* Alerts List */}
      <Card className="bg-navy border-blue">
        <div className="overflow-x-auto">
          {loading ? (
            <div className="p-8 text-center text-gray-400">Loading alerts...</div>
          ) : alerts.length === 0 ? (
            <div className="p-8 text-center text-gray-400">No alerts found</div>
          ) : (
            <table className="w-full">
              <thead>
                <tr className="border-b border-blue">
                  <th className="text-left p-4 text-gray-400 font-medium">Metric</th>
                  <th className="text-left p-4 text-gray-400 font-medium">Severity</th>
                  <th className="text-left p-4 text-gray-400 font-medium">Status</th>
                  <th className="text-left p-4 text-gray-400 font-medium">Message</th>
                  <th className="text-left p-4 text-gray-400 font-medium">Value</th>
                  <th className="text-left p-4 text-gray-400 font-medium">Created</th>
                  <th className="text-left p-4 text-gray-400 font-medium">Actions</th>
                </tr>
              </thead>
              <tbody>
                {alerts.map((alert) => (
                  <tr key={alert.id} className="border-b border-blue hover:bg-blue/20">
                    <td className="p-4 text-white font-medium">{alert.metric_name}</td>
                    <td className="p-4">
                      <Badge className={`${getSeverityColor(alert.severity)} text-white`}>
                        {alert.severity.toUpperCase()}
                      </Badge>
                    </td>
                    <td className="p-4">
                      <Badge className={`${getStatusColor(alert.status)} text-white`}>
                        {alert.status.toUpperCase()}
                      </Badge>
                    </td>
                    <td className="p-4 text-gray-300">{alert.message}</td>
                    <td className="p-4 text-white">{alert.value.toFixed(2)}</td>
                    <td className="p-4 text-gray-400">
                      {new Date(alert.created_at).toLocaleString()}
                    </td>
                    <td className="p-4">
                      <div className="flex gap-2">
                        {alert.status === 'active' && (
                          <Button
                            size="sm"
                            variant="outline"
                            onClick={() => handleAcknowledge(alert.id)}
                            className="border-blue hover:bg-blue"
                          >
                            Acknowledge
                          </Button>
                        )}
                        {alert.status !== 'resolved' && (
                          <Button
                            size="sm"
                            onClick={() => handleResolve(alert.id)}
                            className="bg-green-600 hover:bg-green-700"
                          >
                            Resolve
                          </Button>
                        )}
                      </div>
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
