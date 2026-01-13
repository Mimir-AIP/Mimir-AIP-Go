"use client";

import { useState, useEffect } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Progress } from "@/components/ui/progress";
import { toast } from "sonner";
import {
  listWorkflows,
  createWorkflow,
  getWorkflow,
  executeWorkflow,
  type Workflow as WorkflowType,
} from "@/lib/api";
import {
  Workflow,
  Loader2,
  CheckCircle,
  XCircle,
  Clock,
  PlayCircle,
  Eye,
  AlertCircle,
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
  steps?: WorkflowStep[];
}

export default function WorkflowsPage() {
  const router = useRouter();
  const [workflows, setWorkflows] = useState<WorkflowData[]>([]);
  const [loading, setLoading] = useState(true);
  const [statusFilter, setStatusFilter] = useState<string>("all");

  useEffect(() => {
    loadWorkflows();
    
    // Poll for updates every 5 seconds if there are running workflows
    const interval = setInterval(() => {
      if (workflows.some(w => w.status === "running")) {
        loadWorkflows();
      }
    }, 5000);

    return () => clearInterval(interval);
  }, []);

  const loadWorkflows = async () => {
    try {
      const url = statusFilter === "all" 
        ? "/api/v1/workflows"
        : `/api/v1/workflows?status=${statusFilter}`;
      
      const response = await fetch(url);
      if (!response.ok) {
        throw new Error("Failed to load workflows");
      }
      const data = await response.json();
      setWorkflows(Array.isArray(data.workflows) ? data.workflows : []);
    } catch (error) {
      console.error("Failed to load workflows:", error);
      toast.error("Failed to load workflows");
      setWorkflows([]);
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

  const getProgressPercentage = (workflow: WorkflowData) => {
    return (workflow.completed_steps / workflow.total_steps) * 100;
  };

  const formatDate = (dateString?: string) => {
    if (!dateString) return "N/A";
    return new Date(dateString).toLocaleString();
  };

  if (loading) {
    return (
      <div className="p-6 max-w-7xl mx-auto">
        <div className="flex items-center justify-center h-64">
          <Loader2 className="h-8 w-8 animate-spin text-blue-500" />
          <span className="ml-2 text-lg">Loading workflows...</span>
        </div>
      </div>
    );
  }

  return (
    <div className="p-6 max-w-7xl mx-auto">
      <div className="mb-6">
        <Link href="/" className="text-orange hover:underline mb-4 inline-block">
          ← Back to Dashboard
        </Link>
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-3xl font-bold text-orange">Autonomous Workflows</h1>
            <p className="text-gray-400 mt-1">
              Track automated data processing pipelines
            </p>
          </div>
          <Link href="/ontologies">
            <Button>
              <PlayCircle className="h-4 w-4 mr-2" />
              Start New Workflow
            </Button>
          </Link>
        </div>
      </div>

      {/* Status Filter */}
      <div className="mb-6 flex space-x-2">
        {["all", "pending", "running", "completed", "failed"].map((status) => (
          <Button
            key={status}
            variant={statusFilter === status ? "default" : "outline"}
            size="sm"
            onClick={() => setStatusFilter(status)}
          >
            {status.charAt(0).toUpperCase() + status.slice(1)}
          </Button>
        ))}
      </div>

      {/* Workflows List */}
      {workflows.length === 0 ? (
        <Card className="p-12 text-center bg-gradient-to-br from-navy to-blue/20 border-blue">
          <Workflow className="h-16 w-16 mx-auto text-orange mb-4" />
          <h3 className="text-xl font-semibold mb-2 text-white">No Workflows Found</h3>
          <p className="text-gray-400 mb-6">
            Start an autonomous workflow by creating an ontology from your ingestion pipelines.
            Mimir will automatically extract entities, train ML models, and create digital twins.
          </p>
          <div className="flex justify-center gap-3">
            <Link href="/pipelines">
              <Button variant="outline" className="border-blue hover:border-orange">
                <PlayCircle className="h-4 w-4 mr-2" />
                Create Pipeline
              </Button>
            </Link>
            <Link href="/ontologies">
              <Button className="bg-orange hover:bg-orange/90 text-navy">
                ✨ Create Ontology
              </Button>
            </Link>
          </div>
        </Card>
      ) : (
        <div className="space-y-4">
          {workflows.map((workflow) => (
            <Card key={workflow.id} className="hover:shadow-lg transition-shadow">
              <CardHeader>
                <div className="flex items-start justify-between">
                  <div className="space-y-1">
                    <div className="flex items-center space-x-3">
                      <CardTitle>{workflow.name}</CardTitle>
                      {getStatusBadge(workflow.status)}
                    </div>
                    <CardDescription>
                      Created {formatDate(workflow.created_at)}
                      {workflow.completed_at && ` • Completed ${formatDate(workflow.completed_at)}`}
                    </CardDescription>
                  </div>
                  <Link href={`/workflows/${workflow.id}`}>
                    <Button variant="outline" size="sm">
                      <Eye className="h-4 w-4 mr-2" />
                      View Details
                    </Button>
                  </Link>
                </div>
              </CardHeader>
              <CardContent>
                <div className="space-y-4">
                  {/* Progress Bar */}
                  <div className="space-y-2">
                    <div className="flex items-center justify-between text-sm">
                      <span className="font-medium">Progress</span>
                      <span className="text-gray-500">
                        {workflow.completed_steps} / {workflow.total_steps} steps
                      </span>
                    </div>
                    <Progress value={getProgressPercentage(workflow)} className="w-full" />
                  </div>

                  {/* Current Step */}
                  <div className="flex items-center justify-between">
                    <div className="flex items-center space-x-2">
                      <span className="text-sm font-medium">Current Step:</span>
                      <Badge variant="outline">{workflow.current_step.replace(/_/g, " ")}</Badge>
                    </div>
                    <span className="text-xs font-mono text-gray-500 bg-blue/10 px-2 py-1 rounded">
                      {String(workflow.id).slice(0, 8)}...
                    </span>
                  </div>

                  {/* Error Message */}
                  {workflow.error_message && (
                    <div className="flex items-start space-x-2 p-3 bg-red-50 border border-red-200 rounded-lg">
                      <AlertCircle className="h-5 w-5 text-red-500 mt-0.5 flex-shrink-0" />
                      <div>
                        <p className="text-sm font-medium text-red-800">Error</p>
                        <p className="text-sm text-red-600">{workflow.error_message}</p>
                      </div>
                    </div>
                  )}
                </div>
              </CardContent>
            </Card>
          ))}
        </div>
      )}
    </div>
  );
}
