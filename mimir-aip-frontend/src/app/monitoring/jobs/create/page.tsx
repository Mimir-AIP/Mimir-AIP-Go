'use client';

import { useState, useEffect } from 'react';
import { useRouter } from 'next/navigation';
import Link from 'next/link';
import { Card } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { toast } from 'sonner';
import { createMonitoringJob, listOntologies, type CreateMonitoringJobRequest } from '@/lib/api';

export default function CreateMonitoringJobPage() {
  const router = useRouter();
  const [loading, setLoading] = useState(false);
  const [ontologies, setOntologies] = useState<any[]>([]);
  const [formData, setFormData] = useState<CreateMonitoringJobRequest>({
    name: '',
    ontology_id: '',
    description: '',
    cron_expr: '*/15 * * * *', // Every 15 minutes
    metrics: [],
    rules: [],
    is_enabled: true,
  });
  const [metricsInput, setMetricsInput] = useState('');

  useEffect(() => {
    loadOntologies();
  }, []);

  async function loadOntologies() {
    try {
      const data = await listOntologies();
      setOntologies(Array.isArray(data) ? data : []);
    } catch (error) {
      console.error('Failed to load ontologies:', error);
      toast.error('Failed to load ontologies');
    }
  }

  function handleMetricsChange(value: string) {
    setMetricsInput(value);
    // Parse comma-separated metrics
    const metrics = value.split(',').map(m => m.trim()).filter(m => m.length > 0);
    setFormData({ ...formData, metrics });
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    
    if (!formData.name || !formData.ontology_id || !formData.cron_expr) {
      toast.error('Please fill in all required fields');
      return;
    }

    if (formData.metrics.length === 0) {
      toast.error('Please specify at least one metric to monitor');
      return;
    }

    try {
      setLoading(true);
      await createMonitoringJob(formData);
      toast.success('Monitoring job created successfully');
      router.push('/monitoring/jobs');
    } catch (error) {
      console.error('Failed to create job:', error);
      toast.error(error instanceof Error ? error.message : 'Failed to create monitoring job');
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="space-y-6">
      <div>
        <Link href="/monitoring/jobs" className="text-orange hover:underline text-sm mb-2 inline-block">
          ← Back to Jobs
        </Link>
        <h1 className="text-3xl font-bold text-orange">Create Monitoring Job</h1>
        <p className="text-gray-400 mt-1">Create a new automated monitoring job</p>
      </div>

      <form onSubmit={handleSubmit}>
        <Card className="bg-navy border-blue p-6 space-y-6">
          {/* Job Name */}
          <div>
            <label className="block text-sm font-medium text-white mb-2">
              Job Name <span className="text-red-500">*</span>
            </label>
            <input
              type="text"
              name="name"
              value={formData.name}
              onChange={(e) => setFormData({ ...formData, name: e.target.value })}
              className="w-full bg-navy border border-blue rounded px-3 py-2 text-white focus:border-orange focus:ring-1 focus:ring-orange"
              placeholder="My Monitoring Job"
              required
            />
          </div>

          {/* Ontology Selection */}
          <div>
            <label className="block text-sm font-medium text-white mb-2">
              Ontology <span className="text-red-500">*</span>
            </label>
            <select
              name="pipeline"
              value={formData.ontology_id}
              onChange={(e) => setFormData({ ...formData, ontology_id: e.target.value })}
              className="w-full bg-navy border border-blue rounded px-3 py-2 text-white focus:border-orange focus:ring-1 focus:ring-orange"
              required
            >
              <option value="">Select Pipeline</option>
              {ontologies.map((ont) => (
                <option key={ont.id} value={ont.id}>
                  {ont.name} (v{ont.version})
                </option>
              ))}
            </select>
            {ontologies.length === 0 && (
              <p className="text-sm text-gray-400 mt-1">
                No ontologies found.{' '}
                <Link href="/ontologies/upload" className="text-orange hover:underline">
                  Upload one first
                </Link>
              </p>
            )}
          </div>

          {/* Description */}
          <div>
            <label className="block text-sm font-medium text-white mb-2">
              Description
            </label>
            <textarea
              value={formData.description}
              onChange={(e) => setFormData({ ...formData, description: e.target.value })}
              className="w-full bg-navy border border-blue rounded px-3 py-2 text-white focus:border-orange focus:ring-1 focus:ring-orange"
              rows={3}
              placeholder="Monitor supply chain metrics for anomalies"
            />
          </div>

          {/* Metrics */}
          <div>
            <label className="block text-sm font-medium text-white mb-2">
              Metrics to Monitor <span className="text-red-500">*</span>
            </label>
            <input
              type="text"
              value={metricsInput}
              onChange={(e) => handleMetricsChange(e.target.value)}
              className="w-full bg-navy border border-blue rounded px-3 py-2 text-white focus:border-orange focus:ring-1 focus:ring-orange"
              placeholder="delivery_time, inventory_level, order_volume"
              required
            />
            <p className="text-sm text-gray-400 mt-1">
              Enter comma-separated metric names from your ontology data
            </p>
            {formData.metrics.length > 0 && (
              <div className="mt-2 flex flex-wrap gap-2">
                {formData.metrics.map((metric, idx) => (
                  <span
                    key={idx}
                    className="inline-flex items-center px-2 py-1 rounded bg-blue text-white text-sm"
                  >
                    {metric}
                  </span>
                ))}
              </div>
            )}
          </div>

          {/* Cron Schedule */}
          <div>
            <label className="block text-sm font-medium text-white mb-2">
              Schedule (Cron Expression) <span className="text-red-500">*</span>
            </label>
            <input
              type="text"
              name="schedule"
              value={formData.cron_expr}
              onChange={(e) => setFormData({ ...formData, cron_expr: e.target.value })}
              className="w-full bg-navy border border-blue rounded px-3 py-2 text-white focus:border-orange focus:ring-1 focus:ring-orange font-mono"
              placeholder="*/5 * * * *"
              required
            />
            <div className="mt-2 space-y-1 text-sm text-gray-400">
              <p>Examples:</p>
              <ul className="list-disc list-inside ml-2">
                <li><code className="text-orange">*/15 * * * *</code> - Every 15 minutes</li>
                <li><code className="text-orange">0 * * * *</code> - Every hour</li>
                <li><code className="text-orange">0 0 * * *</code> - Daily at midnight</li>
                <li><code className="text-orange">0 9 * * 1</code> - Every Monday at 9 AM</li>
              </ul>
            </div>
          </div>

          {/* Enabled Toggle */}
          <div className="flex items-center">
            <input
              type="checkbox"
              id="is_enabled"
              checked={formData.is_enabled}
              onChange={(e) => setFormData({ ...formData, is_enabled: e.target.checked })}
              className="w-4 h-4 text-orange bg-navy border-blue rounded focus:ring-orange focus:ring-2"
            />
            <label htmlFor="is_enabled" className="ml-2 text-sm text-white">
              Enable job immediately after creation
            </label>
          </div>

          {/* Submit Buttons */}
          <div className="flex gap-3 pt-4">
            <Button
              type="submit"
              disabled={loading}
              className="bg-orange hover:bg-orange/90"
            >
              {loading ? 'Creating...' : 'Create Monitoring Job'}
            </Button>
            <Link href="/monitoring/jobs">
              <Button type="button" variant="outline" className="border-blue hover:bg-blue">
                Cancel
              </Button>
            </Link>
          </div>
        </Card>
      </form>

      {/* Help Section */}
      <Card className="bg-navy border-blue p-6">
        <h3 className="text-lg font-semibold text-white mb-4">How Monitoring Jobs Work</h3>
        <ol className="list-decimal list-inside space-y-2 text-gray-300">
          <li>Select an ontology that contains time-series or metric data</li>
          <li>Specify which metrics you want to monitor (column names from your data)</li>
          <li>Set a cron schedule for how often to check the metrics</li>
          <li>The job will run automatically and check for anomalies, thresholds, and trends</li>
          <li>Alerts will be created when issues are detected</li>
          <li>View job runs and alerts on the monitoring dashboard</li>
        </ol>
      </Card>
    </div>
  );
}
