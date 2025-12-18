'use client';

import Link from 'next/link';
import { Card } from '@/components/ui/card';
import { Button } from '@/components/ui/button';

export default function CreateRulePage() {
  return (
    <div className="space-y-6">
      <div>
        <Link href="/monitoring/rules" className="text-orange hover:underline text-sm mb-2 inline-block">
          ← Back to Rules
        </Link>
        <h1 className="text-3xl font-bold text-orange">Create Monitoring Rule</h1>
        <p className="text-gray-400 mt-1">Create a new monitoring rule</p>
      </div>

      <Card className="bg-navy border-blue p-8">
        <div className="text-center py-12">
          <h2 className="text-2xl font-bold text-white mb-4">Rule Creation Form</h2>
          <p className="text-gray-400 mb-6">
            This form will allow you to create monitoring rules manually.
          </p>
          <p className="text-gray-400 mb-6">
            For now, monitoring rules are created automatically when you use the Auto-Train feature.
          </p>
          <div className="flex gap-4 justify-center">
            <Link href="/ontologies">
              <Button className="bg-orange hover:bg-orange/90">
                Go to Ontologies
              </Button>
            </Link>
            <Link href="/monitoring/rules">
              <Button variant="outline" className="border-blue hover:bg-blue">
                Back to Rules
              </Button>
            </Link>
          </div>
        </div>
      </Card>

      <Card className="bg-navy border-blue p-6">
        <h3 className="text-lg font-semibold text-white mb-4">Rule Types</h3>
        <div className="space-y-4">
          <div>
            <h4 className="text-white font-medium">Threshold Rules</h4>
            <p className="text-gray-400 text-sm mt-1">
              Alert when a metric crosses a specific threshold (e.g., value &gt; 100)
            </p>
          </div>
          <div>
            <h4 className="text-white font-medium">Trend Rules</h4>
            <p className="text-gray-400 text-sm mt-1">
              Alert when a metric shows increasing/decreasing trends beyond a percentage
            </p>
          </div>
          <div>
            <h4 className="text-white font-medium">Anomaly Rules</h4>
            <p className="text-gray-400 text-sm mt-1">
              Alert when a metric value is anomalous based on Z-score analysis
            </p>
          </div>
        </div>
      </Card>
    </div>
  );
}
