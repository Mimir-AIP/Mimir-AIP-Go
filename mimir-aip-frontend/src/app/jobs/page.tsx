"use client";
import { useEffect, useState } from "react";
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
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import {
  getJobs,
  deleteJob,
  enableJob,
  disableJob,
  updateJob,
  getJobLogs,
  createJob,
  getPipelines,
  type Job,
  type ExecutionLog,
  type Pipeline,
} from "@/lib/api";
import { CardListSkeleton, TableSkeleton } from "@/components/LoadingSkeleton";
import { ErrorDisplay } from "@/components/ErrorBoundary";
import { toast } from "sonner";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { LogViewer } from "@/components/LogViewer";

export default function JobsPage() {
  const [jobs, setJobs] = useState<Job[]>([]);
  const [pipelines, setPipelines] = useState<Pipeline[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const [editDialogOpen, setEditDialogOpen] = useState(false);
  const [createDialogOpen, setCreateDialogOpen] = useState(false);
  const [logsDialogOpen, setLogsDialogOpen] = useState(false);
  const [selectedJob, setSelectedJob] = useState<Job | null>(null);
  const [isProcessing, setIsProcessing] = useState(false);
  const [logsLoading, setLogsLoading] = useState(false);
  const [viewMode, setViewMode] = useState<"grid" | "table">("grid");
  const [logs, setLogs] = useState<ExecutionLog[]>([]);
  
  // Edit form state
  const [editFormData, setEditFormData] = useState({
    name: "",
    pipeline: "",
    cron_expr: "",
  });

  // Create form state
  const [createFormData, setCreateFormData] = useState({
    id: "",
    name: "",
    pipeline: "",
    cron_expr: "*/5 * * * *",
  });

  useEffect(() => {
    fetchJobs();
    fetchPipelines();
  }, []);

  async function fetchJobs() {
    try {
      setLoading(true);
      setError(null);
      const data = await getJobs();
      setJobs(data);
    } catch (err) {
      const message = err instanceof Error ? err.message : "Unknown error";
      setError(message);
      toast.error("Failed to load jobs");
    } finally {
      setLoading(false);
    }
  }

  async function fetchPipelines() {
    try {
      const data = await getPipelines();
      setPipelines(data);
    } catch (err) {
      console.error("Failed to load pipelines:", err);
    }
  }

  function getStatusColor(status?: string) {
    switch (status) {
      case "enabled":
        return "bg-green-600";
      case "disabled":
        return "bg-gray-600";
      case "running":
        return "bg-blue-600";
      case "completed":
        return "bg-green-600";
      case "failed":
        return "bg-red-600";
      default:
        return "bg-gray-600";
    }
  }

  async function handleEnable(job: Job) {
    try {
      setIsProcessing(true);
      await enableJob(job.id);
      toast.success(`Job "${job.name || job.id}" enabled successfully`);
      await fetchJobs();
    } catch (err) {
      const message = err instanceof Error ? err.message : "Unknown error";
      toast.error(`Failed to enable job: ${message}`);
    } finally {
      setIsProcessing(false);
    }
  }

  async function handleDisable(job: Job) {
    try {
      setIsProcessing(true);
      await disableJob(job.id);
      toast.success(`Job "${job.name || job.id}" disabled successfully`);
      await fetchJobs();
    } catch (err) {
      const message = err instanceof Error ? err.message : "Unknown error";
      toast.error(`Failed to disable job: ${message}`);
    } finally {
      setIsProcessing(false);
    }
  }

  async function handleDelete() {
    if (!selectedJob) return;

    try {
      setIsProcessing(true);
      await deleteJob(selectedJob.id);
      toast.success(`Job "${selectedJob.name || selectedJob.id}" deleted successfully`);
      setDeleteDialogOpen(false);
      setSelectedJob(null);
      await fetchJobs();
    } catch (err) {
      const message = err instanceof Error ? err.message : "Unknown error";
      toast.error(`Failed to delete job: ${message}`);
    } finally {
      setIsProcessing(false);
    }
  }

  function openDeleteDialog(job: Job) {
    setSelectedJob(job);
    setDeleteDialogOpen(true);
  }

  function openEditDialog(job: Job) {
    setSelectedJob(job);
    setEditFormData({
      name: job.name || "",
      pipeline: job.pipeline || "",
      cron_expr: job.cron_expr || "",
    });
    setEditDialogOpen(true);
  }

  async function handleEdit() {
    if (!selectedJob) return;

    try {
      setIsProcessing(true);
      const updates: { name?: string; pipeline?: string; cron_expr?: string } = {};
      
      if (editFormData.name !== selectedJob.name) updates.name = editFormData.name;
      if (editFormData.pipeline !== selectedJob.pipeline) updates.pipeline = editFormData.pipeline;
      if (editFormData.cron_expr !== selectedJob.cron_expr) updates.cron_expr = editFormData.cron_expr;
      
      if (Object.keys(updates).length === 0) {
        toast.info("No changes to save");
        setEditDialogOpen(false);
        return;
      }

      await updateJob(selectedJob.id, updates);
      toast.success(`Job "${selectedJob.name || selectedJob.id}" updated successfully`);
      setEditDialogOpen(false);
      setSelectedJob(null);
      await fetchJobs();
    } catch (err) {
      const message = err instanceof Error ? err.message : "Unknown error";
      toast.error(`Failed to update job: ${message}`);
    } finally {
      setIsProcessing(false);
    }
  }

  async function handleViewLogs(job: Job) {
    try {
      setLogsLoading(true);
      setSelectedJob(job);
      const result = await getJobLogs(job.id);
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
    if (!selectedJob) return;
    try {
      setLogsLoading(true);
      const result = await getJobLogs(selectedJob.id);
      setLogs(result.logs);
      toast.success("Logs refreshed");
    } catch (err) {
      const message = err instanceof Error ? err.message : "Unknown error";
      toast.error(`Failed to refresh logs: ${message}`);
    } finally {
      setLogsLoading(false);
    }
  }

  async function handleCreateJob() {
    if (!createFormData.name.trim() || !createFormData.pipeline) {
      toast.error("Please fill in all required fields");
      return;
    }

    try {
      setIsProcessing(true);
      const jobId = createFormData.id || `job-${Date.now()}`;
      await createJob(jobId, createFormData.name, createFormData.pipeline, createFormData.cron_expr);
      toast.success(`Job "${createFormData.name}" created successfully`);
      setCreateDialogOpen(false);
      setCreateFormData({ id: "", name: "", pipeline: "", cron_expr: "*/5 * * * *" });
      await fetchJobs();
    } catch (err) {
      const message = err instanceof Error ? err.message : "Unknown error";
      toast.error(`Failed to create job: ${message}`);
    } finally {
      setIsProcessing(false);
    }
  }

  function isJobDisabled(job: Job): boolean {
    return job.enabled === false;
  }

  return (
    <div>
      <div className="flex justify-between items-center mb-6">
        <h1 className="text-2xl font-bold text-orange">Scheduled Jobs</h1>
        <div className="flex gap-2">
          <Button
            variant={viewMode === "grid" ? "default" : "outline"}
            size="sm"
            onClick={() => setViewMode("grid")}
          >
            Grid
          </Button>
          <Button
            variant={viewMode === "table" ? "default" : "outline"}
            size="sm"
            onClick={() => setViewMode("table")}
          >
            Table
          </Button>
          <Button onClick={() => setCreateDialogOpen(true)}>
            Create Job
          </Button>
        </div>
      </div>

      {loading && viewMode === "grid" && <CardListSkeleton count={6} />}
      {loading && viewMode === "table" && <TableSkeleton rows={10} />}
      {error && !loading && <ErrorDisplay error={error} onRetry={fetchJobs} />}

      {!loading && !error && jobs.length === 0 && (
        <Card className="bg-navy text-white border-blue p-8 text-center">
          <p className="text-white/60 mb-4">No scheduled jobs found</p>
          <Button onClick={() => setCreateDialogOpen(true)}>
            Create Your First Job
          </Button>
        </Card>
      )}

      {/* Grid View */}
      {!loading && !error && jobs.length > 0 && viewMode === "grid" && (
        <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-6">
          {jobs.map((job) => (
            <Card key={job.id} className="bg-navy text-white border-blue p-6">
              <div className="flex justify-between items-start mb-3">
                <h2 className="text-xl font-bold text-orange">{job.name || job.id}</h2>
                {job.status && (
                  <Badge className={`${getStatusColor(job.status)} text-white`}>
                    {job.status}
                  </Badge>
                )}
              </div>
              <div className="space-y-1 text-sm text-white/60 mb-4">
                <p>ID: {job.id}</p>
                {job.pipeline && <p>Pipeline: {job.pipeline}</p>}
                {job.cron_expr && <p>Schedule: {job.cron_expr}</p>}
                {job.created_at && <p>Created: {new Date(job.created_at).toLocaleString()}</p>}
              </div>

              <div className="flex flex-wrap gap-2">
                {isJobDisabled(job) ? (
                  <Button
                    size="sm"
                    variant="default"
                    onClick={() => handleEnable(job)}
                    disabled={isProcessing}
                  >
                    Enable
                  </Button>
                ) : (
                  <Button
                    size="sm"
                    variant="outline"
                    onClick={() => handleDisable(job)}
                    disabled={isProcessing}
                  >
                    Disable
                  </Button>
                )}
                <Button
                  size="sm"
                  variant="outline"
                  onClick={() => openEditDialog(job)}
                  disabled={isProcessing}
                >
                  Edit
                </Button>
                <Button
                  size="sm"
                  variant="outline"
                  onClick={() => handleViewLogs(job)}
                  disabled={logsLoading}
                >
                  Logs
                </Button>
                <Button
                  size="sm"
                  variant="destructive"
                  onClick={() => openDeleteDialog(job)}
                  disabled={isProcessing}
                >
                  Delete
                </Button>
              </div>
            </Card>
          ))}
        </div>
      )}

      {/* Table View */}
      {!loading && !error && jobs.length > 0 && viewMode === "table" && (
        <Card className="bg-navy text-white border-blue">
          <Table>
            <TableHeader>
              <TableRow className="border-blue hover:bg-blue/10">
                <TableHead className="text-orange">Name</TableHead>
                <TableHead className="text-orange">ID</TableHead>
                <TableHead className="text-orange">Status</TableHead>
                <TableHead className="text-orange">Pipeline</TableHead>
                <TableHead className="text-orange">Created</TableHead>
                <TableHead className="text-orange text-right">Actions</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {jobs.map((job) => (
                <TableRow key={job.id} className="border-blue hover:bg-blue/10">
                  <TableCell className="font-medium">{job.name || job.id}</TableCell>
                  <TableCell className="text-white/60 text-sm">{job.id}</TableCell>
                  <TableCell>
                    <Badge className={`${getStatusColor(job.status)} text-white`}>
                      {job.status || (job.enabled ? "enabled" : "disabled")}
                    </Badge>
                  </TableCell>
                  <TableCell className="text-white/60">{job.pipeline || "N/A"}</TableCell>
                  <TableCell className="text-white/60 text-sm">
                    {job.created_at ? new Date(job.created_at).toLocaleString() : "N/A"}
                  </TableCell>
                  <TableCell className="text-right">
                    <div className="flex justify-end gap-2">
                      {isJobDisabled(job) ? (
                        <Button
                          size="sm"
                          variant="default"
                          onClick={() => handleEnable(job)}
                          disabled={isProcessing}
                        >
                          Enable
                        </Button>
                      ) : (
                        <Button
                          size="sm"
                          variant="outline"
                          onClick={() => handleDisable(job)}
                          disabled={isProcessing}
                        >
                          Disable
                        </Button>
                      )}
                      <Button
                        size="sm"
                        variant="outline"
                        onClick={() => openEditDialog(job)}
                        disabled={isProcessing}
                      >
                        Edit
                      </Button>
                      <Button
                        size="sm"
                        variant="outline"
                        onClick={() => handleViewLogs(job)}
                        disabled={logsLoading}
                      >
                        Logs
                      </Button>
                      <Button
                        size="sm"
                        variant="destructive"
                        onClick={() => openDeleteDialog(job)}
                        disabled={isProcessing}
                      >
                        Delete
                      </Button>
                    </div>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </Card>
      )}

      {/* Delete Confirmation Dialog */}
      <Dialog open={deleteDialogOpen} onOpenChange={setDeleteDialogOpen}>
        <DialogContent className="bg-navy text-white border-blue">
          <DialogHeader>
            <DialogTitle className="text-red-500">Delete Job</DialogTitle>
            <DialogDescription className="text-white/60">
              Are you sure you want to delete &quot;{selectedJob?.name || selectedJob?.id}&quot;? This action cannot be undone.
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
              {isProcessing ? "Deleting..." : "Delete Job"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Create Job Dialog */}
      <Dialog open={createDialogOpen} onOpenChange={setCreateDialogOpen}>
        <DialogContent className="bg-navy text-white border-blue">
          <DialogHeader>
            <DialogTitle className="text-orange">Create New Job</DialogTitle>
            <DialogDescription className="text-white/60">
              Schedule a pipeline to run automatically
            </DialogDescription>
          </DialogHeader>
          <div className="grid gap-4 py-4">
            <div className="grid gap-2">
              <Label htmlFor="create-name">Job Name *</Label>
              <Input
                id="create-name"
                value={createFormData.name}
                onChange={(e) => setCreateFormData({ ...createFormData, name: e.target.value })}
                placeholder="my-scheduled-job"
                className="bg-blue/10 border-blue text-white"
              />
            </div>
            <div className="grid gap-2">
              <Label htmlFor="create-pipeline">Pipeline *</Label>
              <select
                id="create-pipeline"
                value={createFormData.pipeline}
                onChange={(e) => setCreateFormData({ ...createFormData, pipeline: e.target.value })}
                className="bg-blue/10 border-blue text-white rounded-md px-3 py-2 border"
              >
                <option value="">Select a pipeline</option>
                {pipelines.map((p) => (
                  <option key={p.id} value={p.id}>{p.name}</option>
                ))}
              </select>
            </div>
            <div className="grid gap-2">
              <Label htmlFor="create-cron">Cron Expression *</Label>
              <Input
                id="create-cron"
                value={createFormData.cron_expr}
                onChange={(e) => setCreateFormData({ ...createFormData, cron_expr: e.target.value })}
                placeholder="*/5 * * * *"
                className="bg-blue/10 border-blue text-white"
              />
              <p className="text-xs text-white/60">
                Format: minute hour day month weekday (e.g., &quot;*/5 * * * *&quot; runs every 5 minutes)
              </p>
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setCreateDialogOpen(false)} disabled={isProcessing}>
              Cancel
            </Button>
            <Button onClick={handleCreateJob} disabled={isProcessing || !createFormData.name.trim() || !createFormData.pipeline}>
              {isProcessing ? "Creating..." : "Create Job"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Edit Job Dialog */}
      <Dialog open={editDialogOpen} onOpenChange={setEditDialogOpen}>
        <DialogContent className="bg-navy text-white border-blue">
          <DialogHeader>
            <DialogTitle className="text-orange">Edit Job</DialogTitle>
            <DialogDescription className="text-white/60">
              Update properties for &quot;{selectedJob?.name || selectedJob?.id}&quot;
            </DialogDescription>
          </DialogHeader>
          <div className="grid gap-4 py-4">
            <div className="grid gap-2">
              <Label htmlFor="edit-name">Job Name</Label>
              <Input
                id="edit-name"
                value={editFormData.name}
                onChange={(e) => setEditFormData({ ...editFormData, name: e.target.value })}
                placeholder="Enter job name"
                className="bg-blue/10 border-blue text-white"
              />
            </div>
            <div className="grid gap-2">
              <Label htmlFor="edit-pipeline">Pipeline ID</Label>
              <Input
                id="edit-pipeline"
                value={editFormData.pipeline}
                onChange={(e) => setEditFormData({ ...editFormData, pipeline: e.target.value })}
                placeholder="Enter pipeline ID"
                className="bg-blue/10 border-blue text-white"
              />
            </div>
            <div className="grid gap-2">
              <Label htmlFor="edit-cron">Cron Expression</Label>
              <Input
                id="edit-cron"
                value={editFormData.cron_expr}
                onChange={(e) => setEditFormData({ ...editFormData, cron_expr: e.target.value })}
                placeholder="e.g., */5 * * * *"
                className="bg-blue/10 border-blue text-white"
              />
              <p className="text-xs text-white/60">
                Format: minute hour day month weekday (e.g., &quot;*/5 * * * *&quot; runs every 5 minutes)
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
            <Button onClick={handleEdit} disabled={isProcessing}>
              {isProcessing ? "Saving..." : "Save Changes"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Logs Dialog */}
      <Dialog open={logsDialogOpen} onOpenChange={setLogsDialogOpen}>
        <DialogContent className="bg-navy text-white border-blue max-w-6xl max-h-[80vh]">
          <DialogHeader>
            <DialogTitle className="text-orange">Job Execution Logs</DialogTitle>
            <DialogDescription className="text-white/60">
              Execution logs for &quot;{selectedJob?.name || selectedJob?.id}&quot;
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
