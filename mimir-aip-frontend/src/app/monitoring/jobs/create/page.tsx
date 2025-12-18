'use client';

import Link from 'next/link';
import { Card } from '@/components/ui/card';
import { Button } from '@/components/ui/button';

export default function CreateJobPage() {
  return (
    <div className="space-y-6">
      <div>
        <Link href="/monitoring/jobs" className="text-orange hover:underline text-sm mb-2 inline-block">
          ← Back to Jobs
        </Link>
        <h1 className="text-3xl font-bold text-orange">Create Monitoring Job</h1>
        <p className="text-gray-400 mt-1">Create a new automated monitoring job</p>
      </div>

      <Card className="bg-navy border-blue p-8">
        <div className="text-center py-12">
          <h2 className="text-2xl font-bold text-white mb-4">Job Creation Form</h2>
          <p className="text-gray-400 mb-6">
            This form will allow you to create monitoring jobs manually.
          </p>
          <p className="text-gray-400 mb-6">
            For now, monitoring jobs are created automatically when you use the Auto-Train feature
            on an ontology with time-series data.
          </p>
          <div className="flex gap-4 justify-center">
            <Link href="/ontologies">
              <Button className="bg-orange hover:bg-orange/90">
                Go to Ontologies
              </Button>
            </Link>
            <Link href="/monitoring/jobs">
              <Button variant="outline" className="border-blue hover:bg-blue">
                Back to Jobs
              </Button>
            </Link>
          </div>
        </div>
      </Card>

      <Card className="bg-navy border-blue p-6">
        <h3 className="text-lg font-semibold text-white mb-4">How to Set Up Monitoring</h3>
        <ol className="list-decimal list-inside space-y-2 text-gray-300">
          <li>Upload a CSV file with time-series data (e.g., supply chain data with dates and metrics)</li>
          <li>Navigate to the Ontologies page</li>
          <li>Select your ontology and click "Auto-Train"</li>
          <li>The system will automatically detect time-series metrics and create monitoring jobs</li>
          <li>Monitoring will run every 15 minutes by default</li>
          <li>View alerts on the Alerts page when thresholds are exceeded</li>
        </ol>
      </Card>
    </div>
  );
}
