"use client";

import { useEffect, useState } from "react";
import { useParams, useRouter } from "next/navigation";
import Link from "next/link";
import { getPipeline, executePipeline, getPipelineLogs, type Pipeline, type ExecutionLog } from "@/lib/api";
import { Card } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { ArrowLeft, GitBranch, Play, Clock, CheckCircle, XCircle, AlertCircle, ExternalLink } from "lucide-react";
import { toast } from "sonner";

export default function PipelineDetailPage() {
  const params = useParams();
  const router = useRouter();
  const id = params.id as string;

  const [pipeline, setPipeline] = useState<Pipeline | null>(null);
  const [executionLogs, setExecutionLogs] = useState<ExecutionLog[]>([]);
  const [loading, setLoading] = useState(true);
  const [executing, setExecuting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    loadPipelineData();
  }, [id]);

  async function loadPipelineData() {
    try {
      setLoading(true);
      setError(null);
      
      const [pipelineData, logsData] = await Promise.all([
        getPipeline(id),
        getPipelineLogs(id).catch(() => ({ pipeline_id: id, logs: [] })),
      ]);
      
      setPipeline(pipelineData);
      setExecutionLogs(logsData.logs || []);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load pipeline");
    } finally {
      setLoading(false);
    }
  }

  async function handleExecute() {
    if (!pipeline) return;
    
    try {
      setExecuting(true);
      await executePipeline(pipeline.id, {});
      toast.success("Pipeline execution started");
      setTimeout(loadPipelineData, 3000);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Execution failed");
    } finally {
      setExecuting(false);
    }
  }

  if (loading) {
    return (
      <div className="space-y-6">
        <div className="flex items-center gap-4">
          <Link href="/pipelines" className="text-white/60 hover:text-orange">
            <ArrowLeft className="h-5 w-5" />
          </Link>
          <h1 className="text-2xl font-bold text-orange">Loading...</h1>
        </div>
        <Card className="bg-navy border-blue p-6 animate-pulse h-96"></Card>
      </div>
    );
  }

  if (error || !pipeline) {
    return (
      <div className="space-y-6">
        <div className="flex items-center gap-4">
          <Link href="/pipelines" className="text-white/60 hover:text-orange">
            <ArrowLeft className="h-5 w-5" />
          </Link>
          <h1 className="text-2xl font-bold text-orange">Error</h1>
        </div>
        <Card className="bg-red-900/20 border-red-500 text-red-400 p-6">
          <p>{error || "Pipeline not found"}</p>
        </Card>
      </div>
    );
  }

  const steps = pipeline.steps || [];
  const metadata = pipeline.metadata || {};

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-4">
          <Link href="/pipelines" className="text-white/60 hover:text-orange transition-colors">
            <ArrowLeft className="h-5 w-5" />
          </Link>
          <div>
            <h1 className="text-2xl font-bold text-orange">{pipeline.name}</h1>
            <p className="text-white/60 text-sm">ID: {pipeline.id}</p>
          </div>
        </div>
        <div className="flex gap-3">
          <Button
            onClick={handleExecute}
            disabled={executing}
            className="bg-orange hover:bg-orange/80 text-white"
          >
            <Play className="h-4 w-4 mr-2" />
            {executing ? "Running..." : "Execute Now"}
          </Button>
        </div>
      </div>

      {/* Status Card */}
      <Card className="bg-navy border-blue p-6">
        <div className="grid grid-cols-1 md:grid-cols-4 gap-6">
          <div>
            <p className="text-white/60 text-sm mb-1">Status</p>
            <Badge className={pipeline.enabled ? "bg-green-500/20 text-green-400" : "bg-gray-500/20 text-gray-400"}>
              {pipeline.enabled ? "Enabled" : "Disabled"}
            </Badge>
          </div>
          <div>
            <p className="text-white/60 text-sm mb-1">Steps</p>
            <p className="text-white text-lg font-semibold">{steps.length}</p>
          </div>
          <div>
            <p className="text-white/60 text-sm mb-1">Executions</p>
            <p className="text-white text-lg font-semibold">{executionLogs.length}</p>
          </div>
          <div>
            <p className="text-white/60 text-sm mb-1">Created</p>
            <p className="text-white text-sm">
              {pipeline.created_at ? new Date(pipeline.created_at).toLocaleDateString() : "Unknown"}
            </p>
          </div>
        </div>
      </Card>

      {/* Linked Ontology */}
      {(metadata as any).target_ontology_id && (
        <Card className="bg-navy border-blue p-6">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-3">
              <GitBranch className="h-5 w-5 text-blue" />
              <div>
                <h3 className="text-white font-semibold">Linked Ontology</h3>
                <p className="text-white/60 text-sm">{(metadata as any).target_ontology_id}</p>
              </div>
            </div>
            <Link
              href={`/ontologies/${(metadata as any).target_ontology_id}`}
              className="flex items-center gap-2 px-4 py-2 bg-blue/20 hover:bg-blue/30 text-blue rounded transition-colors"
            >
              View Ontology
              <ExternalLink className="h-4 w-4" />
            </Link>
          </div>
        </Card>
      )}

      {/* Pipeline Steps */}
      <Card className="bg-navy border-blue">
        <div className="p-4 border-b border-blue/30">
          <h2 className="text-lg font-semibold text-white">Pipeline Steps ({steps.length})</h2>
        </div>
        <div className="divide-y divide-blue/30">
          {steps.length === 0 ? (
            <div className="p-8 text-center text-white/40">
              No steps configured
            </div>
          ) : (
            steps.map((step: any, index: number) => (
              <div key={index} className="p-4">
                <div className="flex items-start gap-4">
                  <div className="flex-shrink-0 w-8 h-8 bg-blue/20 rounded-full flex items-center justify-center text-blue font-semibold">
                    {index + 1}
                  </div>
                  <div className="flex-1">
                    <div className="flex items-center justify-between mb-2">
                      <h3 className="text-white font-medium">
                        {step.name || step.Name || `Step ${index + 1}`}
                      </h3>
                      <Badge variant="outline" className="text-xs">
                        {step.plugin || step.Plugin || "Unknown"}
                      </Badge>
                    </div>
                    {step.config && Object.keys(step.config).length > 0 && (
                      <div className="bg-blue/10 rounded p-3 mt-2">
                        <p className="text-white/40 text-xs mb-1">Configuration:</p>
                        <pre className="text-xs text-white/60 overflow-x-auto">
                          {JSON.stringify(step.config, null, 2)}
                        </pre>
                      </div>
                    )}
                  </div>
                </div>
              </div>
            ))
          )}
        </div>
      </Card>

      {/* Execution History */}
      <Card className="bg-navy border-blue">
        <div className="p-4 border-b border-blue/30">
          <h2 className="text-lg font-semibold text-white">Execution History ({executionLogs.length})</h2>
        </div>
        <div className="divide-y divide-blue/30 max-h-96 overflow-y-auto">
          {executionLogs.length === 0 ? (
            <div className="p-8 text-center text-white/40">
              <Clock className="h-12 w-12 mx-auto mb-3 text-white/20" />
              <p>No executions yet</p>
              <p className="text-sm mt-1">Click "Execute Now" to run this pipeline</p>
            </div>
          ) : (
            executionLogs.map((log: ExecutionLog, index: number) => (
              <div key={log.id || index} className="p-4">
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-3">
                    {log.status === 'completed' ? (
                      <CheckCircle className="h-5 w-5 text-green-400" />
                    ) : log.status === 'failed' ? (
                      <XCircle className="h-5 w-5 text-red-400" />
                    ) : (
                      <AlertCircle className="h-5 w-5 text-orange" />
                    )}
                    <div>
                      <p className="text-white font-medium">
                        {log.status === 'completed' ? 'Success' : log.status === 'failed' ? 'Failed' : 'Running'}
                      </p>
                      <p className="text-white/40 text-xs">
                        {log.started_at ? new Date(log.started_at).toLocaleString() : 'Unknown time'}
                      </p>
                    </div>
                  </div>
                  <Badge className={
                    log.status === 'completed' ? 'bg-green-500/20 text-green-400' :
                    log.status === 'failed' ? 'bg-red-500/20 text-red-400' :
                    'bg-orange/20 text-orange'
                  }>
                    {log.status}
                  </Badge>
                </div>
              </div>
            ))
          )}
        </div>
      </Card>

      {/* Description */}
      {pipeline.description && (
        <Card className="bg-navy border-blue p-6">
          <h3 className="text-white font-semibold mb-2">Description</h3>
          <p className="text-white/60">{pipeline.description}</p>
        </Card>
      )}
    </div>
  );
}
