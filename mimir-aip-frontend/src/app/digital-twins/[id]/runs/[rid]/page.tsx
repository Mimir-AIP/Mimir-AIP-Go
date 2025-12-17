"use client";

import { useEffect, useState } from "react";
import { useParams } from "next/navigation";
import Link from "next/link";
import {
  getSimulationRun,
  getSimulationTimeline,
  analyzeSimulationImpact,
  type SimulationRun,
  type StateSnapshot,
  type ImpactAnalysis,
} from "@/lib/api";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { LoadingSkeleton } from "@/components/LoadingSkeleton";
import { toast } from "sonner";
import {
  ArrowLeft,
  CheckCircle2,
  XCircle,
  AlertTriangle,
  TrendingUp,
  TrendingDown,
  Activity,
  Clock,
  BarChart3,
  Loader2,
} from "lucide-react";

export default function SimulationRunPage() {
  const params = useParams();
  const twinId = params.id as string;
  const runId = params.rid as string;

  const [run, setRun] = useState<SimulationRun | null>(null);
  const [timeline, setTimeline] = useState<StateSnapshot[]>([]);
  const [analysis, setAnalysis] = useState<ImpactAnalysis | null>(null);
  const [loading, setLoading] = useState(true);
  const [analyzingImpact, setAnalyzingImpact] = useState(false);

  useEffect(() => {
    loadData();
  }, [twinId, runId]);

  async function loadData() {
    try {
      setLoading(true);
      const [runData, timelineData] = await Promise.all([
        getSimulationRun(twinId, runId),
        getSimulationTimeline(twinId, runId),
      ]);
      setRun(runData);
      setTimeline(timelineData.snapshots);
    } catch (err) {
      const message = err instanceof Error ? err.message : "Failed to load simulation run";
      toast.error(message);
    } finally {
      setLoading(false);
    }
  }

  async function handleAnalyzeImpact() {
    try {
      setAnalyzingImpact(true);
      const analysisData = await analyzeSimulationImpact(twinId, runId);
      setAnalysis(analysisData);
      toast.success("Impact analysis completed");
    } catch (err) {
      const message = err instanceof Error ? err.message : "Failed to analyze impact";
      toast.error(message);
    } finally {
      setAnalyzingImpact(false);
    }
  }

  function formatDate(dateString: string): string {
    return new Date(dateString).toLocaleString("en-US", {
      year: "numeric",
      month: "short",
      day: "numeric",
      hour: "2-digit",
      minute: "2-digit",
      second: "2-digit",
    });
  }

  function getStatusIcon(status: string) {
    switch (status) {
      case "completed":
        return <CheckCircle2 className="h-5 w-5 text-green-600" />;
      case "failed":
        return <XCircle className="h-5 w-5 text-red-600" />;
      case "running":
        return <Loader2 className="h-5 w-5 text-blue-600 animate-spin" />;
      default:
        return <Clock className="h-5 w-5 text-gray-600" />;
    }
  }

  function getStatusColor(status: string) {
    switch (status) {
      case "completed":
        return "bg-green-900/40 text-green-400 border-green-500";
      case "failed":
        return "bg-red-900/40 text-red-400 border-red-500";
      case "running":
        return "bg-blue-900/40 text-blue-400 border-blue-500";
      default:
        return "bg-gray-800 text-gray-400 border-gray-600";
    }
  }

  function getImpactIcon(impact: string) {
    switch (impact) {
      case "severe":
      case "critical":
        return <AlertTriangle className="h-5 w-5 text-red-600" />;
      case "moderate":
        return <TrendingDown className="h-5 w-5 text-yellow-600" />;
      case "minimal":
      case "low":
        return <TrendingUp className="h-5 w-5 text-green-600" />;
      default:
        return <Activity className="h-5 w-5 text-blue-600" />;
    }
  }

  if (loading) {
    return (
      <div className="container mx-auto py-8">
        <LoadingSkeleton />
      </div>
    );
  }

  if (!run) {
    return (
      <div className="container mx-auto py-8">
        <Card>
          <CardContent className="pt-6">
            <p className="text-red-600">Simulation run not found</p>
            <Link href={`/digital-twins/${twinId}`}>
              <Button className="mt-4">Back to Twin</Button>
            </Link>
          </CardContent>
        </Card>
      </div>
    );
  }

  return (
    <div className="container mx-auto py-8">
      {/* Header */}
      <Link href={`/digital-twins/${twinId}`}>
        <Button variant="ghost" className="mb-4">
          <ArrowLeft className="h-4 w-4 mr-2" />
          Back to Digital Twin
        </Button>
      </Link>

      <div className="flex items-start justify-between mb-6">
        <div>
          <h1 className="text-3xl font-bold flex items-center gap-3">
            <BarChart3 className="h-8 w-8" />
            Simulation Results
          </h1>
          <p className="text-muted-foreground mt-2">Run ID: {runId}</p>
        </div>
        <div className="flex items-center gap-3">
          <div className={`flex items-center gap-2 px-4 py-2 rounded-lg border ${getStatusColor(run.status)}`}>
            {getStatusIcon(run.status)}
            <span className="font-medium capitalize">{run.status}</span>
          </div>
        </div>
      </div>

      {/* Key Metrics */}
      <div className="grid grid-cols-4 gap-4 mb-6">
        <Card>
          <CardContent className="pt-6">
            <div className="text-center">
              <p className="text-3xl font-bold">{run.metrics.total_steps}</p>
              <p className="text-sm text-muted-foreground">Total Steps</p>
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="pt-6">
            <div className="text-center">
              <p className="text-3xl font-bold">{run.metrics.events_processed}</p>
              <p className="text-sm text-muted-foreground">Events Processed</p>
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="pt-6">
            <div className="text-center">
              <p className="text-3xl font-bold">{run.metrics.entities_affected}</p>
              <p className="text-sm text-muted-foreground">Entities Affected</p>
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="pt-6">
            <div className="text-center">
              <p className="text-3xl font-bold">{(run.metrics.system_stability * 100).toFixed(0)}%</p>
              <p className="text-sm text-muted-foreground">System Stability</p>
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Performance Metrics */}
      <div className="grid grid-cols-2 gap-6 mb-6">
        <Card>
          <CardHeader>
            <CardTitle>Utilization Metrics</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="space-y-4">
              <div>
                <div className="flex justify-between text-sm mb-1">
                  <span className="text-muted-foreground">Average Utilization</span>
                  <span className="font-medium">{(run.metrics.average_utilization * 100).toFixed(1)}%</span>
                </div>
                <div className="w-full bg-gray-200 rounded-full h-2">
                  <div
                    className="bg-blue-600 h-2 rounded-full"
                    style={{ width: `${run.metrics.average_utilization * 100}%` }}
                  />
                </div>
              </div>
              <div>
                <div className="flex justify-between text-sm mb-1">
                  <span className="text-muted-foreground">Peak Utilization</span>
                  <span className="font-medium">{(run.metrics.peak_utilization * 100).toFixed(1)}%</span>
                </div>
                <div className="w-full bg-gray-200 rounded-full h-2">
                  <div
                    className={`h-2 rounded-full ${
                      run.metrics.peak_utilization > 0.9 ? "bg-red-600" : run.metrics.peak_utilization > 0.7 ? "bg-yellow-600" : "bg-green-600"
                    }`}
                    style={{ width: `${run.metrics.peak_utilization * 100}%` }}
                  />
                </div>
              </div>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Critical Events</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="flex items-center justify-center py-6">
              <div className="text-center">
                <p className="text-5xl font-bold text-red-600">{run.metrics.critical_events}</p>
                <p className="text-sm text-muted-foreground mt-2">Critical Events Detected</p>
              </div>
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Bottlenecks */}
      {run.metrics.bottleneck_entities.length > 0 && (
        <Card className="mb-6">
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <AlertTriangle className="h-5 w-5 text-yellow-600" />
              Bottleneck Entities
            </CardTitle>
            <CardDescription>Entities that became bottlenecks during simulation</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="flex flex-wrap gap-2">
              {run.metrics.bottleneck_entities.map((entity) => (
                <Badge key={entity} variant="outline" className="border-yellow-600 text-yellow-700">
                  {entity}
                </Badge>
              ))}
            </div>
          </CardContent>
        </Card>
      )}

      {/* Recommendations */}
      {run.metrics.recommendations.length > 0 && (
        <Card className="mb-6">
          <CardHeader>
            <CardTitle>Recommendations</CardTitle>
            <CardDescription>Suggested actions based on simulation results</CardDescription>
          </CardHeader>
          <CardContent>
            <ul className="space-y-2">
              {run.metrics.recommendations.map((rec, index) => (
                <li key={index} className="flex items-start gap-2">
                  <CheckCircle2 className="h-5 w-5 text-green-600 mt-0.5 flex-shrink-0" />
                  <span className="text-sm">{rec}</span>
                </li>
              ))}
            </ul>
          </CardContent>
        </Card>
      )}

      {/* Impact Summary */}
      <Card className="mb-6">
        <CardHeader>
          <CardTitle>Impact Summary</CardTitle>
        </CardHeader>
        <CardContent>
          <p className="text-sm">{run.metrics.impact_summary}</p>
        </CardContent>
      </Card>

      {/* Impact Analysis */}
      <Card className="mb-6">
        <CardHeader>
          <div className="flex items-center justify-between">
            <div>
              <CardTitle>Detailed Impact Analysis</CardTitle>
              <CardDescription>Advanced analysis of scenario impact</CardDescription>
            </div>
            {!analysis && (
              <Button onClick={handleAnalyzeImpact} disabled={analyzingImpact}>
                {analyzingImpact ? (
                  <>
                    <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                    Analyzing...
                  </>
                ) : (
                  "Run Analysis"
                )}
              </Button>
            )}
          </div>
        </CardHeader>
        <CardContent>
          {analysis ? (
            <div className="space-y-4">
              <div className="flex items-center gap-3">
                {getImpactIcon(analysis.overall_impact)}
                <div>
                  <p className="font-medium capitalize">{analysis.overall_impact} Impact</p>
                  <p className="text-sm text-muted-foreground">
                    Risk Score: {(analysis.risk_score * 100).toFixed(0)}%
                  </p>
                </div>
              </div>

              {analysis.critical_path.length > 0 && (
                <div>
                  <p className="text-sm font-medium mb-2">Critical Path:</p>
                  <div className="flex flex-wrap gap-2">
                    {analysis.critical_path.map((entity, idx) => (
                      <Badge key={idx} variant="destructive">
                        {entity}
                      </Badge>
                    ))}
                  </div>
                </div>
              )}

              {analysis.mitigation_strategies.length > 0 && (
                <div>
                  <p className="text-sm font-medium mb-2">Mitigation Strategies:</p>
                  <ul className="space-y-1">
                    {analysis.mitigation_strategies.map((strategy, idx) => (
                      <li key={idx} className="text-sm">
                        • {strategy}
                      </li>
                    ))}
                  </ul>
                </div>
              )}
            </div>
          ) : (
            <p className="text-sm text-muted-foreground">Click "Run Analysis" to generate detailed impact report</p>
          )}
        </CardContent>
      </Card>

      {/* Timeline */}
      {timeline.length > 0 && (
        <Card>
          <CardHeader>
            <CardTitle>Simulation Timeline</CardTitle>
            <CardDescription>{timeline.length} snapshots captured</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="space-y-2 max-h-96 overflow-y-auto">
              {timeline.map((snapshot, index) => (
                <div key={index} className="flex items-center gap-3 py-2 border-b last:border-b-0">
                  <div className="flex-shrink-0 w-16 text-sm font-medium text-muted-foreground">
                    Step {snapshot.step_number}
                  </div>
                  <div className="flex-1">
                    <p className="text-sm">{snapshot.description || "State snapshot"}</p>
                  </div>
                  <div className="text-xs text-muted-foreground">
                    {new Date(snapshot.timestamp).toLocaleTimeString()}
                  </div>
                </div>
              ))}
            </div>
          </CardContent>
        </Card>
      )}
    </div>
  );
}
