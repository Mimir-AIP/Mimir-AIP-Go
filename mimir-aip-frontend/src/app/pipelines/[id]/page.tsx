"use client";
import { useEffect, useState, useCallback } from "react";
import { useParams, useRouter } from "next/navigation";
import { Card } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  getPipeline,
  executePipeline,
  deletePipeline,
  clonePipeline,
  validatePipeline,
  getPipelineHistory,
  getPipelineLogs,
  updatePipeline,
  type Pipeline,
  type ExecutionLog,
} from "@/lib/api";
import { DetailsSkeleton } from "@/components/LoadingSkeleton";
import { ErrorDisplay } from "@/components/ErrorBoundary";
import { toast } from "sonner";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { getStatusColor } from "@/lib/utils";
import { LogViewer } from "@/components/LogViewer";

export default function PipelineDetailPage() {
  const { id } = useParams();
  const router = useRouter();
  const [pipeline, setPipeline] = useState<Pipeline | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [isExecuting, setIsExecuting] = useState(false);
  const [isValidating, setIsValidating] = useState(false);
  const [validationResult, setValidationResult] = useState<{ valid: boolean; errors: string[] } | null>(null);
  
  // Dialog states
  const [cloneDialogOpen, setCloneDialogOpen] = useState(false);
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const [editDialogOpen, setEditDialogOpen] = useState(false);
  const [historyDialogOpen, setHistoryDialogOpen] = useState(false);
  const [logsDialogOpen, setLogsDialogOpen] = useState(false);
  const [cloneName, setCloneName] = useState("");
  const [history, setHistory] = useState<ExecutionLog[]>([]);
  const [logs, setLogs] = useState<ExecutionLog[]>([]);
  const [isProcessing, setIsProcessing] = useState(false);
  const [logsLoading, setLogsLoading] = useState(false);

  // Edit form state
  const [editFormData, setEditFormData] = useState({
    name: "",
    description: "",
    yamlConfig: "",
  });

  const fetchPipeline = useCallback(async () => {
    try {
      setLoading(true);
      setError(null);
      const data = await getPipeline(id as string);
      setPipeline(data);
      setCloneName(`${data.name}-copy`);
    } catch (err) {
      const message = err instanceof Error ? err.message : "Unknown error";
      setError(message);
      toast.error("Failed to load pipeline");
    } finally {
      setLoading(false);
    }
  }, [id]);

  useEffect(() => {
    if (id) fetchPipeline();
  }, [id, fetchPipeline]);

  async function handleExecute() {
    if (!pipeline) return;
    
    try {
      setIsExecuting(true);
      await executePipeline(pipeline.id, {});
      toast.success("Pipeline execution started successfully");
    } catch (err) {
      const message = err instanceof Error ? err.message : "Unknown error";
      toast.error(`Failed to execute pipeline: ${message}`);
    } finally {
      setIsExecuting(false);
    }
  }

  async function handleValidate() {
    if (!pipeline) return;
    
    try {
      setIsValidating(true);
      const result = await validatePipeline(pipeline.id);
      setValidationResult(result);
      
      if (result.valid) {
        toast.success("Pipeline configuration is valid");
      } else {
        toast.error(`Pipeline validation failed: ${result.errors.length} error(s)`);
      }
    } catch (err) {
      const message = err instanceof Error ? err.message : "Unknown error";
      toast.error(`Failed to validate pipeline: ${message}`);
      setValidationResult({ valid: false, errors: [message] });
    } finally {
      setIsValidating(false);
    }
  }

  async function handleClone() {
    if (!pipeline || !cloneName.trim()) {
      toast.error("Please enter a name for the cloned pipeline");
      return;
    }

    try {
      setIsProcessing(true);
      await clonePipeline(pipeline.id, cloneName);
      toast.success(`Pipeline "${cloneName}" cloned successfully`);
      setCloneDialogOpen(false);
      router.push("/pipelines");
    } catch (err) {
      const message = err instanceof Error ? err.message : "Unknown error";
      toast.error(`Failed to clone pipeline: ${message}`);
    } finally {
      setIsProcessing(false);
    }
  }

  async function handleDelete() {
    if (!pipeline) return;

    try {
      setIsProcessing(true);
      await deletePipeline(pipeline.id);
      toast.success(`Pipeline "${pipeline.name}" deleted successfully`);
      setDeleteDialogOpen(false);
      router.push("/pipelines");
    } catch (err) {
      const message = err instanceof Error ? err.message : "Unknown error";
      toast.error(`Failed to delete pipeline: ${message}`);
    } finally {
      setIsProcessing(false);
    }
  }

  async function handleViewHistory() {
    if (!pipeline) return;

    try {
      setIsProcessing(true);
      const result = await getPipelineHistory(pipeline.id);
      setHistory(result.history);
      setHistoryDialogOpen(true);
    } catch (err) {
      const message = err instanceof Error ? err.message : "Unknown error";
      toast.error(`Failed to load history: ${message}`);
    } finally {
      setIsProcessing(false);
    }
  }

  async function handleViewLogs() {
    if (!pipeline) return;

    try {
      setLogsLoading(true);
      const result = await getPipelineLogs(pipeline.id);
      setLogs(result.logs);
      setLogsDialogOpen(true);
    } catch (err) {
      const message = err instanceof Error ? err.message : "Unknown error";
      toast.error(`Failed to load logs: ${message}`);
    } finally {
      setLogsLoading(false);
    }
  }

  async function handleRefreshLogs() {
    if (!pipeline) return;
    try {
      setLogsLoading(true);
      const result = await getPipelineLogs(pipeline.id);
      setLogs(result.logs);
      toast.success("Logs refreshed");
    } catch (err) {
      const message = err instanceof Error ? err.message : "Unknown error";
      toast.error(`Failed to refresh logs: ${message}`);
    } finally {
      setLogsLoading(false);
    }
  }

  function openEditDialog() {
    if (!pipeline) return;
    setEditFormData({
      name: pipeline.name || "",
      description: (pipeline as any).description || "",
      yamlConfig: JSON.stringify(pipeline, null, 2),
    });
    setEditDialogOpen(true);
  }

  async function handleEdit() {
    if (!pipeline) return;
    if (!editFormData.name.trim()) {
      toast.error("Please enter a pipeline name");
      return;
    }

    try {
      setIsProcessing(true);
      
      const metadata = {
        name: editFormData.name,
        description: editFormData.description,
      };

      const config = {
        name: editFormData.name,
        description: editFormData.description,
        enabled: true,
        steps: [], // Would be parsed from YAML
      };

      await updatePipeline(pipeline.id, metadata, config);
      toast.success(`Pipeline "${editFormData.name}" updated successfully`);
      setEditDialogOpen(false);
      await fetchPipeline();
    } catch (err) {
      const message = err instanceof Error ? err.message : "Unknown error";
      toast.error(`Failed to update pipeline: ${message}`);
    } finally {
      setIsProcessing(false);
    }
  }


  return (
    <div>
      <div className="flex justify-between items-center mb-6">
        <h1 className="text-2xl font-bold text-orange">Pipeline Details</h1>
        <Button variant="outline" onClick={() => router.push("/pipelines")}>
          Back to Pipelines
        </Button>
      </div>

      {loading && <DetailsSkeleton />}
      {error && !loading && <ErrorDisplay error={error} onRetry={fetchPipeline} />}

      {pipeline && !loading && (
        <div className="space-y-6">
          {/* Main Info Card */}
          <Card className="bg-navy text-white border-blue p-6">
            <div className="flex justify-between items-start mb-4">
              <div>
                <h2 className="text-2xl font-bold text-orange mb-2">{pipeline.name}</h2>
                <p className="text-sm text-white/60">ID: {pipeline.id}</p>
              </div>
              {pipeline.status && (
                <Badge className={`${getStatusColor(pipeline.status)} text-white`}>
                  {pipeline.status}
                </Badge>
              )}
            </div>

            {validationResult && (
              <div className={`p-4 rounded mb-4 ${validationResult.valid ? 'bg-green-500/10 border border-green-500' : 'bg-red-500/10 border border-red-500'}`}>
                <p className={`font-semibold ${validationResult.valid ? 'text-green-500' : 'text-red-500'}`}>
                  {validationResult.valid ? '✓ Valid Configuration' : '✗ Invalid Configuration'}
                </p>
                {!validationResult.valid && validationResult.errors.length > 0 && (
                  <ul className="mt-2 text-sm text-red-400 list-disc list-inside">
                    {validationResult.errors.map((err, idx) => (
                      <li key={idx}>{err}</li>
                    ))}
                  </ul>
                )}
              </div>
            )}

            {/* Action Buttons */}
            <div className="flex flex-wrap gap-2 mb-6">
              <Button onClick={handleExecute} disabled={isExecuting}>
                {isExecuting ? "Executing..." : "Run Pipeline"}
              </Button>
              <Button variant="outline" onClick={handleValidate} disabled={isValidating}>
                {isValidating ? "Validating..." : "Validate"}
              </Button>
              <Button variant="outline" onClick={handleViewLogs} disabled={logsLoading}>
                {logsLoading ? "Loading Logs..." : "View Logs"}
              </Button>
              <Button variant="outline" onClick={handleViewHistory} disabled={isProcessing}>
                View History
              </Button>
              <Button variant="outline" onClick={() => setCloneDialogOpen(true)}>
                Clone
              </Button>
              <Button variant="outline" onClick={openEditDialog}>
                Edit
              </Button>
              <Button variant="destructive" onClick={() => setDeleteDialogOpen(true)}>
                Delete
              </Button>
            </div>

            {/* Pipeline Configuration */}
            <div className="bg-blue/10 p-4 rounded">
              <h3 className="text-lg font-semibold text-orange mb-2">Configuration</h3>
              <pre className="text-sm text-white overflow-x-auto whitespace-pre-wrap">
                {JSON.stringify(pipeline, null, 2)}
              </pre>
            </div>
          </Card>
        </div>
      )}

      {/* Clone Dialog */}
      <Dialog open={cloneDialogOpen} onOpenChange={setCloneDialogOpen}>
        <DialogContent className="bg-navy text-white border-blue">
          <DialogHeader>
            <DialogTitle className="text-orange">Clone Pipeline</DialogTitle>
            <DialogDescription className="text-white/60">
              Create a copy of &quot;{pipeline?.name}&quot;
            </DialogDescription>
          </DialogHeader>
          <div className="grid gap-4 py-4">
            <div className="grid gap-2">
              <Label htmlFor="clone-name">New Pipeline Name</Label>
              <Input
                id="clone-name"
                value={cloneName}
                onChange={(e) => setCloneName(e.target.value)}
                placeholder="Enter pipeline name"
                className="bg-blue/10 border-blue text-white"
              />
            </div>
          </div>
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => setCloneDialogOpen(false)}
              disabled={isProcessing}
            >
              Cancel
            </Button>
            <Button onClick={handleClone} disabled={isProcessing || !cloneName.trim()}>
              {isProcessing ? "Cloning..." : "Clone Pipeline"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Delete Dialog */}
      <Dialog open={deleteDialogOpen} onOpenChange={setDeleteDialogOpen}>
        <DialogContent className="bg-navy text-white border-blue">
          <DialogHeader>
            <DialogTitle className="text-red-500">Delete Pipeline</DialogTitle>
            <DialogDescription className="text-white/60">
              Are you sure you want to delete &quot;{pipeline?.name}&quot;? This action cannot be undone.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => setDeleteDialogOpen(false)}
              disabled={isProcessing}
            >
              Cancel
            </Button>
            <Button
              variant="destructive"
              onClick={handleDelete}
              disabled={isProcessing}
            >
              {isProcessing ? "Deleting..." : "Delete Pipeline"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Edit Pipeline Dialog */}
      <Dialog open={editDialogOpen} onOpenChange={setEditDialogOpen}>
        <DialogContent className="bg-navy text-white border-blue max-w-4xl max-h-[90vh]">
          <DialogHeader>
            <DialogTitle className="text-orange">Edit Pipeline</DialogTitle>
            <DialogDescription className="text-white/60">
              Update pipeline &quot;{pipeline?.name}&quot; configuration
            </DialogDescription>
          </DialogHeader>
          <div className="grid gap-4 py-4 overflow-y-auto max-h-[60vh]">
            <div className="grid gap-2">
              <Label htmlFor="edit-name">Pipeline Name *</Label>
              <Input
                id="edit-name"
                value={editFormData.name}
                onChange={(e) => setEditFormData({ ...editFormData, name: e.target.value })}
                placeholder="my-pipeline"
                className="bg-blue/10 border-blue text-white"
              />
            </div>
            <div className="grid gap-2">
              <Label htmlFor="edit-description">Description</Label>
              <Input
                id="edit-description"
                value={editFormData.description}
                onChange={(e) => setEditFormData({ ...editFormData, description: e.target.value })}
                placeholder="A brief description of this pipeline"
                className="bg-blue/10 border-blue text-white"
              />
            </div>
            <div className="grid gap-2">
              <Label htmlFor="edit-yaml">Pipeline Configuration (YAML)</Label>
              <Textarea
                id="edit-yaml"
                value={editFormData.yamlConfig}
                onChange={(e) => setEditFormData({ ...editFormData, yamlConfig: e.target.value })}
                placeholder="Enter YAML configuration"
                className="bg-blue/10 border-blue text-white font-mono text-sm min-h-[300px]"
              />
              <p className="text-xs text-white/60">
                Define pipeline steps, plugins, and configuration in YAML format
              </p>
            </div>
          </div>
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => setEditDialogOpen(false)}
              disabled={isProcessing}
            >
              Cancel
            </Button>
            <Button onClick={handleEdit} disabled={isProcessing || !editFormData.name.trim()}>
              {isProcessing ? "Saving..." : "Save Changes"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* History Dialog */}
      <Dialog open={historyDialogOpen} onOpenChange={setHistoryDialogOpen}>
        <DialogContent className="bg-navy text-white border-blue max-w-3xl">
          <DialogHeader>
            <DialogTitle className="text-orange">Execution History</DialogTitle>
            <DialogDescription className="text-white/60">
              Past executions of &quot;{pipeline?.name}&quot;
            </DialogDescription>
          </DialogHeader>
          <div className="max-h-96 overflow-y-auto">
            {history.length === 0 ? (
              <p className="text-white/60 text-center py-8">No execution history found</p>
            ) : (
              <pre className="text-sm text-white whitespace-pre-wrap">
                {JSON.stringify(history, null, 2)}
              </pre>
            )}
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setHistoryDialogOpen(false)}>
              Close
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Logs Dialog */}
      <Dialog open={logsDialogOpen} onOpenChange={setLogsDialogOpen}>
        <DialogContent className="bg-navy text-white border-blue max-w-6xl max-h-[80vh]">
          <DialogHeader>
            <DialogTitle className="text-orange">Pipeline Execution Logs</DialogTitle>
            <DialogDescription className="text-white/60">
              Execution logs for &quot;{pipeline?.name}&quot;
            </DialogDescription>
          </DialogHeader>
          <div className="overflow-y-auto max-h-[60vh]">
            <LogViewer logs={logs} loading={logsLoading} onRefresh={handleRefreshLogs} />
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setLogsDialogOpen(false)}>
              Close
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
