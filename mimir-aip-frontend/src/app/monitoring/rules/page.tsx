'use client';

import { useEffect, useState } from 'react';
import { listMonitoringRules, deleteMonitoringRule, MonitoringRule } from '@/lib/api';
import { Card } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { toast } from 'sonner';
import Link from 'next/link';

export default function RulesPage() {
  const [rules, setRules] = useState<MonitoringRule[]>([]);
  const [loading, setLoading] = useState(true);
  const [filter, setFilter] = useState<{ metric_name?: string }>({});

  useEffect(() => {
    loadRules();
  }, [filter]);

  async function loadRules() {
    try {
      setLoading(true);
      const response = await listMonitoringRules(filter);
      setRules(response.rules || []);
    } catch (error) {
      toast.error('Failed to load monitoring rules');
      console.error('Load rules error:', error);
      setRules([]);
    } finally {
      setLoading(false);
    }
  }

  async function handleDelete(id: string) {
    if (!confirm('Are you sure you want to delete this rule?')) {
      return;
    }

    try {
      await deleteMonitoringRule(id);
      toast.success('Rule deleted');
      loadRules();
    } catch (error) {
      toast.error('Failed to delete rule');
      console.error('Delete error:', error);
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

  function getRuleTypeColor(type: string) {
    switch (type) {
      case 'threshold': return 'bg-purple-600';
      case 'trend': return 'bg-cyan-600';
      case 'anomaly': return 'bg-pink-600';
      default: return 'bg-gray-600';
    }
  }

  function parseCondition(conditionStr: string): string {
    try {
      const cond = JSON.parse(conditionStr);
      return JSON.stringify(cond, null, 2);
    } catch {
      return conditionStr;
    }
  }

  const enabledRules = rules.filter(r => r.is_enabled);
  const disabledRules = rules.filter(r => !r.is_enabled);

  return (
    <div className="space-y-6">
      <div className="flex justify-between items-center">
        <div>
          <h1 className="text-3xl font-bold text-orange">Monitoring Rules</h1>
          <p className="text-gray-400 mt-1">Configure threshold, trend, and anomaly detection rules</p>
        </div>
        <Link href="/monitoring/rules/create">
          <Button className="bg-orange hover:bg-orange/90">Create Rule</Button>
        </Link>
      </div>

      {/* Summary Cards */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        <Card className="bg-navy border-blue p-6">
          <div className="text-sm text-gray-400">Total Rules</div>
          <div className="text-3xl font-bold text-white mt-2">{rules.length}</div>
        </Card>
        <Card className="bg-navy border-blue p-6">
          <div className="text-sm text-gray-400">Enabled</div>
          <div className="text-3xl font-bold text-green-500 mt-2">{enabledRules.length}</div>
        </Card>
        <Card className="bg-navy border-blue p-6">
          <div className="text-sm text-gray-400">Disabled</div>
          <div className="text-3xl font-bold text-gray-500 mt-2">{disabledRules.length}</div>
        </Card>
      </div>

      {/* Filters */}
      <Card className="bg-navy border-blue p-4">
        <div className="flex gap-4">
          <div>
            <label className="text-sm text-gray-400">Filter by metric:</label>
            <input
              type="text"
              className="ml-2 bg-navy border border-blue rounded px-3 py-1 text-white"
              placeholder="Enter metric name..."
              value={filter.metric_name || ''}
              onChange={(e) => setFilter({ metric_name: e.target.value || undefined })}
            />
          </div>
        </div>
      </Card>

      {/* Rules List */}
      <Card className="bg-navy border-blue">
        <div className="overflow-x-auto">
          {loading ? (
            <div className="p-8 text-center text-gray-400">Loading rules...</div>
          ) : rules.length === 0 ? (
            <div className="p-8 text-center text-gray-400">
              No monitoring rules found.{' '}
              <Link href="/monitoring/rules/create" className="text-orange hover:underline">
                Create one
              </Link>
            </div>
          ) : (
            <table className="w-full">
              <thead>
                <tr className="border-b border-blue">
                  <th className="text-left p-4 text-gray-400 font-medium">Metric</th>
                  <th className="text-left p-4 text-gray-400 font-medium">Rule Type</th>
                  <th className="text-left p-4 text-gray-400 font-medium">Severity</th>
                  <th className="text-left p-4 text-gray-400 font-medium">Status</th>
                  <th className="text-left p-4 text-gray-400 font-medium">Condition</th>
                  <th className="text-left p-4 text-gray-400 font-medium">Actions</th>
                </tr>
              </thead>
              <tbody>
                {rules.map((rule) => (
                  <tr key={rule.id} className="border-b border-blue hover:bg-blue/20">
                    <td className="p-4 text-white font-medium">{rule.metric_name}</td>
                    <td className="p-4">
                      <Badge className={`${getRuleTypeColor(rule.rule_type)} text-white`}>
                        {rule.rule_type.toUpperCase()}
                      </Badge>
                    </td>
                    <td className="p-4">
                      <Badge className={`${getSeverityColor(rule.severity)} text-white`}>
                        {rule.severity.toUpperCase()}
                      </Badge>
                    </td>
                    <td className="p-4">
                      <Badge className={rule.is_enabled ? 'bg-green-600' : 'bg-gray-600'}>
                        {rule.is_enabled ? 'ENABLED' : 'DISABLED'}
                      </Badge>
                    </td>
                    <td className="p-4">
                      <pre className="text-xs text-gray-300 font-mono max-w-xs overflow-x-auto">
                        {parseCondition(rule.condition)}
                      </pre>
                    </td>
                    <td className="p-4">
                      <div className="flex gap-2">
                        <Button
                          size="sm"
                          variant="outline"
                          onClick={() => handleDelete(rule.id)}
                          className="border-red-600 text-red-600 hover:bg-red-600 hover:text-white"
                        >
                          Delete
                        </Button>
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
