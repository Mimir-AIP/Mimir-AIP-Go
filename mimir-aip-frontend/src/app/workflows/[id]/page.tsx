"use client";

import { useState, useEffect } from "react";
import { useRouter, useParams } from "next/navigation";
import Link from "next/link";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Progress } from "@/components/ui/progress";
import { toast } from "sonner";
import {
  Loader2,
  CheckCircle,
  XCircle,
  Clock,
  PlayCircle,
  AlertCircle,
  ArrowRight,
  Database,
  FileText,
  Brain,
  Boxes,
  Activity,
  ChevronRight,
} from "lucide-react";

interface WorkflowStep {
  id: number;
  workflow_id: number;
  step_name: string;
  step_order: number;
  status: string;
  started_at?: string;
  completed_at?: string;
  error_message?: string;
}

interface WorkflowArtifact {
  id: number;
  workflow_id: number;
  step_name: string;
  artifact_type: string;
  artifact_id: number;
  artifact_name: string;
  created_at: string;
}

interface WorkflowData {
  id: number;
  name: string;
  import_id: number;
  status: string;
  current_step: string;
  total_steps: number;
  completed_steps: number;
  error_message?: string;
  created_at: string;
  updated_at?: string;
  completed_at?: string;
  created_by: string;
  metadata?: Record<string, any>;
  steps?: WorkflowStep[];
  artifacts?: WorkflowArtifact[];
}

export default function WorkflowDetailPage() {
  const router = useRouter();
  const params = useParams();
  const workflowId = params.id as string;
  
  const [workflow, setWorkflow] = useState<WorkflowData | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    loadWorkflow();
    
    // Poll for updates every 3 seconds if workflow is running
    const interval = setInterval(() => {
      if (workflow?.status === "running") {
        loadWorkflow();
      }
    }, 3000);

    return () => clearInterval(interval);
  }, [workflowId, workflow?.status]);

  const loadWorkflow = async () => {
    try {
      const response = await fetch(`/api/v1/workflows/${workflowId}`);
      if (!response.ok) {
        throw new Error("Failed to load workflow");
      }
      const data = await response.json();
      setWorkflow(data);
    } catch (error) {
      console.error("Failed to load workflow:", error);
      toast.error("Failed to load workflow details");
    } finally {
      setLoading(false);
    }
  };

  const getStatusBadge = (status: string) => {
    switch (status) {
      case "completed":
        return <Badge className="bg-green-500"><CheckCircle className="h-3 w-3 mr-1" /> Completed</Badge>;
      case "running":
        return <Badge className="bg-blue-500"><Loader2 className="h-3 w-3 mr-1 animate-spin" /> Running</Badge>;
      case "failed":
        return <Badge className="bg-red-500"><XCircle className="h-3 w-3 mr-1" /> Failed</Badge>;
      case "pending":
        return <Badge variant="outline"><Clock className="h-3 w-3 mr-1" /> Pending</Badge>;
      default:
        return <Badge variant="outline">{status}</Badge>;
    }
  };

  const getStepIcon = (stepName: string) => {
    switch (stepName) {
      case "schema_inference":
        return <Database className="h-5 w-5" />;
      case "ontology_creation":
        return <FileText className="h-5 w-5" />;
      case "entity_extraction":
        return <Boxes className="h-5 w-5" />;
      case "ml_training":
        return <Brain className="h-5 w-5" />;
      case "twin_creation":
        return <Activity className="h-5 w-5" />;
      case "monitoring_setup":
        return <Activity className="h-5 w-5" />;
      default:
        return <CheckCircle className="h-5 w-5" />;
    }
  };

  const formatStepName = (stepName: string) => {
    return stepName.split("_").map(word => 
      word.charAt(0).toUpperCase() + word.slice(1)
    ).join(" ");
  };

  const formatDate = (dateString?: string) => {
    if (!dateString) return "N/A";
    return new Date(dateString).toLocaleString();
  };

  const getProgressPercentage = () => {
    if (!workflow) return 0;
    return (workflow.completed_steps / workflow.total_steps) * 100;
  };

  if (loading) {
    return (
      <div className="p-6 max-w-7xl mx-auto">
        <div className="flex items-center justify-center h-64">
          <Loader2 className="h-8 w-8 animate-spin text-blue-500" />
          <span className="ml-2 text-lg">Loading workflow...</span>
        </div>
      </div>
    );
  }

  if (!workflow) {
    return (
      <div className="p-6 max-w-7xl mx-auto">
        <Card className="p-12 text-center">
          <AlertCircle className="h-16 w-16 mx-auto text-gray-400 mb-4" />
          <h3 className="text-xl font-semibold mb-2">Workflow Not Found</h3>
          <p className="text-gray-500 mb-4">
            The requested workflow could not be found.
          </p>
          <Link href="/workflows">
            <Button>Back to Workflows</Button>
          </Link>
        </Card>
      </div>
    );
  }

  return (
    <div className="p-6 max-w-7xl mx-auto">
      <div className="mb-6">
        <Link href="/workflows" className="text-orange hover:underline mb-4 inline-block">
          ← Back to Workflows
        </Link>
        
        {/* Workflow Header */}
        <Card>
          <CardHeader>
            <div className="flex items-start justify-between">
              <div className="space-y-2">
                <div className="flex items-center space-x-3">
                  <CardTitle className="text-2xl">{workflow.name}</CardTitle>
                  {getStatusBadge(workflow.status)}
                </div>
                <CardDescription>
                  Created {formatDate(workflow.created_at)}
                  {workflow.completed_at && ` • Completed ${formatDate(workflow.completed_at)}`}
                </CardDescription>
                <div className="flex items-center space-x-4 text-sm text-gray-500">
                  <span>Import ID: {workflow.import_id}</span>
                  <span>Workflow ID: {workflow.id}</span>
                </div>
              </div>
              {workflow.status === "running" && (
                <Loader2 className="h-8 w-8 animate-spin text-blue-500" />
              )}
            </div>
          </CardHeader>
          <CardContent>
            <div className="space-y-2">
              <div className="flex items-center justify-between text-sm">
                <span className="font-medium">Overall Progress</span>
                <span className="text-gray-500">
                  {workflow.completed_steps} / {workflow.total_steps} steps completed
                </span>
              </div>
              <Progress value={getProgressPercentage()} className="w-full h-3" />
            </div>
            
            {workflow.error_message && (
              <div className="mt-4 flex items-start space-x-2 p-3 bg-red-50 border border-red-200 rounded-lg">
                <AlertCircle className="h-5 w-5 text-red-500 mt-0.5 flex-shrink-0" />
                <div>
                  <p className="text-sm font-medium text-red-800">Workflow Failed</p>
                  <p className="text-sm text-red-600">{workflow.error_message}</p>
                </div>
              </div>
            )}
          </CardContent>
        </Card>
      </div>

      {/* Workflow Steps */}
      <div className="space-y-4">
        <h2 className="text-2xl font-bold">Pipeline Steps</h2>
        
        {workflow.steps && workflow.steps.length > 0 ? (
          <div className="space-y-3">
            {workflow.steps.map((step, index) => {
              const artifacts = workflow.artifacts?.filter(a => a.step_name === step.step_name) || [];
              const isActive = workflow.current_step === step.step_name;
              
              return (
                <Card 
                  key={step.id} 
                  className={isActive ? "border-2 border-blue-500" : ""}
                >
                  <CardContent className="pt-6">
                    <div className="flex items-start justify-between">
                      <div className="flex items-start space-x-4 flex-1">
                        {/* Step Number & Icon */}
                        <div className="flex flex-col items-center">
                          <div className={`
                            w-10 h-10 rounded-full flex items-center justify-center
                            ${step.status === "completed" ? "bg-green-100 text-green-600" :
                              step.status === "running" ? "bg-blue-100 text-blue-600" :
                              step.status === "failed" ? "bg-red-100 text-red-600" :
                              "bg-gray-100 text-gray-400"}
                          `}>
                            {getStepIcon(step.step_name)}
                          </div>
                          {index < (workflow.steps?.length || 0) - 1 && (
                            <div className="w-0.5 h-12 bg-gray-200 mt-2" />
                          )}
                        </div>

                        {/* Step Details */}
                        <div className="flex-1 space-y-2">
                          <div className="flex items-center space-x-3">
                            <h3 className="text-lg font-semibold">
                              {step.step_order}. {formatStepName(step.step_name)}
                            </h3>
                            {getStatusBadge(step.status)}
                            {isActive && <Badge variant="outline">Current</Badge>}
                          </div>

                          {/* Timestamps */}
                          <div className="flex items-center space-x-4 text-sm text-gray-500">
                            {step.started_at && (
                              <span>Started: {formatDate(step.started_at)}</span>
                            )}
                            {step.completed_at && (
                              <span>Completed: {formatDate(step.completed_at)}</span>
                            )}
                          </div>

                          {/* Error Message */}
                          {step.error_message && (
                            <div className="flex items-start space-x-2 p-3 bg-red-50 border border-red-200 rounded-lg">
                              <AlertCircle className="h-4 w-4 text-red-500 mt-0.5 flex-shrink-0" />
                              <div>
                                <p className="text-sm font-medium text-red-800">Step Failed</p>
                                <p className="text-sm text-red-600">{step.error_message}</p>
                              </div>
                            </div>
                          )}

                          {/* Artifacts */}
                          {artifacts.length > 0 && (
                            <div className="mt-3 space-y-2">
                              <p className="text-sm font-medium text-gray-700">Generated Artifacts:</p>
                              <div className="space-y-1">
                                {artifacts.map((artifact) => (
                                  <div 
                                    key={artifact.id}
                                    className="flex items-center justify-between p-2 bg-blue-50 border border-blue-200 rounded"
                                  >
                                    <div className="flex items-center space-x-2">
                                      <Badge variant="outline" className="text-xs">
                                        {artifact.artifact_type}
                                      </Badge>
                                      <span className="text-sm font-medium">{artifact.artifact_name}</span>
                                    </div>
                                    <Button
                                      variant="ghost"
                                      size="sm"
                                      onClick={() => {
                                        // Navigate to artifact based on type
                                        if (artifact.artifact_type === "ontology") {
                                          router.push(`/ontology/${artifact.artifact_id}`);
                                        } else if (artifact.artifact_type === "model") {
                                          router.push(`/ml/models/${artifact.artifact_id}`);
                                        } else if (artifact.artifact_type === "twin") {
                                          router.push(`/twin/${artifact.artifact_id}`);
                                        }
                                      }}
                                    >
                                      View <ChevronRight className="h-4 w-4 ml-1" />
                                    </Button>
                                  </div>
                                ))}
                              </div>
                            </div>
                          )}
                        </div>
                      </div>
                    </div>
                  </CardContent>
                </Card>
              );
            })}
          </div>
        ) : (
          <Card className="p-8 text-center">
            <AlertCircle className="h-12 w-12 mx-auto text-gray-400 mb-3" />
            <p className="text-gray-500">No steps found for this workflow</p>
          </Card>
        )}
      </div>

      {/* Actions */}
      <div className="mt-6 flex justify-between">
        <Link href={`/data/preview/${workflow.import_id}`}>
          <Button variant="outline">
            View Source Data
          </Button>
        </Link>
        
        {workflow.status === "failed" && (
          <Button
            onClick={async () => {
              try {
                const response = await fetch(`/api/v1/workflows/${workflow.id}/execute`, {
                  method: "POST",
                });
                if (response.ok) {
                  toast.success("Workflow restarted");
                  loadWorkflow();
                } else {
                  throw new Error("Failed to restart workflow");
                }
              } catch (error) {
                toast.error("Failed to restart workflow");
              }
            }}
          >
            <PlayCircle className="h-4 w-4 mr-2" />
            Retry Workflow
          </Button>
        )}
      </div>
    </div>
  );
}
