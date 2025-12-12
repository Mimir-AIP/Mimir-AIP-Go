"use client";
import { Card } from "@/components/ui/card";

export default function LoginPage() {
  return (
    <div className="flex items-center justify-center min-h-[60vh]">
      <Card className="bg-navy text-white border-blue p-8 max-w-md w-full">
        <h1 className="text-3xl font-bold text-orange mb-4">Authentication</h1>
        <p className="text-white/80 mb-6">
          Authentication is disabled for local deployments. 
          You have full access to all Mimir AIP features.
        </p>
        <div className="bg-blue/20 border border-blue/50 rounded p-4">
          <p className="text-sm text-white/70">
            <strong className="text-orange">Note:</strong> For production deployments with authentication requirements, 
            configure the auth settings in <code className="bg-black/30 px-1 rounded">config.yaml</code>
          </p>
        </div>
      </Card>
    </div>
  );
}
