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
  type Job,
} from "@/lib/api";
import { CardListSkeleton, TableSkeleton } from "@/components/LoadingSkeleton";
import { ErrorDisplay } from "@/components/ErrorBoundary";
import { toast } from "sonner";

export default function JobsPage() {
  const [jobs, setJobs] = useState<Job[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const [selectedJob, setSelectedJob] = useState<Job | null>(null);
  const [isProcessing, setIsProcessing] = useState(false);
  const [viewMode, setViewMode] = useState<"grid" | "table">("grid");

  useEffect(() => {
    fetchJobs();
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

  function getStatusColor(status?: string) {
    switch (status?.toLowerCase()) {
      case "active":
      case "running":
      case "enabled":
        return "bg-green-500";
      case "failed":
      case "error":
        return "bg-red-500";
      case "disabled":
        return "bg-gray-500";
      case "pending":
        return "bg-yellow-500";
      default:
        return "bg-blue-500";
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
          <Button onClick={() => toast.info("Create job feature coming soon")}>
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
          <Button onClick={() => toast.info("Create job feature coming soon")}>
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
    </div>
  );
}
