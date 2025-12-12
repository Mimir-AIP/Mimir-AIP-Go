"use client";
import Link from "next/link";
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
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { getPipelines, deletePipeline, clonePipeline, type Pipeline } from "@/lib/api";
import { CardListSkeleton } from "@/components/LoadingSkeleton";
import { ErrorDisplay } from "@/components/ErrorBoundary";
import { toast } from "sonner";

export default function PipelinesPage() {
  const [pipelines, setPipelines] = useState<Pipeline[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [cloneDialogOpen, setCloneDialogOpen] = useState(false);
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const [selectedPipeline, setSelectedPipeline] = useState<Pipeline | null>(null);
  const [cloneName, setCloneName] = useState("");
  const [isProcessing, setIsProcessing] = useState(false);

  useEffect(() => {
    fetchPipelines();
  }, []);

  async function fetchPipelines() {
    try {
      setLoading(true);
      setError(null);
      const data = await getPipelines();
      setPipelines(data);
    } catch (err) {
      const message = err instanceof Error ? err.message : "Unknown error";
      setError(message);
      toast.error("Failed to load pipelines");
    } finally {
      setLoading(false);
    }
  }

  async function handleClone() {
    if (!selectedPipeline || !cloneName.trim()) {
      toast.error("Please enter a name for the cloned pipeline");
      return;
    }

    try {
      setIsProcessing(true);
      await clonePipeline(selectedPipeline.id, cloneName);
      toast.success(`Pipeline "${cloneName}" cloned successfully`);
      setCloneDialogOpen(false);
      setCloneName("");
      setSelectedPipeline(null);
      await fetchPipelines();
    } catch (err) {
      const message = err instanceof Error ? err.message : "Unknown error";
      toast.error(`Failed to clone pipeline: ${message}`);
    } finally {
      setIsProcessing(false);
    }
  }

  async function handleDelete() {
    if (!selectedPipeline) return;

    try {
      setIsProcessing(true);
      await deletePipeline(selectedPipeline.id);
      toast.success(`Pipeline "${selectedPipeline.name}" deleted successfully`);
      setDeleteDialogOpen(false);
      setSelectedPipeline(null);
      await fetchPipelines();
    } catch (err) {
      const message = err instanceof Error ? err.message : "Unknown error";
      toast.error(`Failed to delete pipeline: ${message}`);
    } finally {
      setIsProcessing(false);
    }
  }

  function openCloneDialog(pipeline: Pipeline) {
    setSelectedPipeline(pipeline);
    setCloneName(`${pipeline.name}-copy`);
    setCloneDialogOpen(true);
  }

  function openDeleteDialog(pipeline: Pipeline) {
    setSelectedPipeline(pipeline);
    setDeleteDialogOpen(true);
  }

  function getStatusColor(status?: string) {
    switch (status?.toLowerCase()) {
      case "active":
      case "running":
        return "bg-green-500";
      case "failed":
      case "error":
        return "bg-red-500";
      case "pending":
        return "bg-yellow-500";
      default:
        return "bg-gray-500";
    }
  }

  return (
    <div>
      <div className="flex justify-between items-center mb-6">
        <h1 className="text-2xl font-bold text-orange">Pipelines</h1>
        <Button onClick={() => toast.info("Create pipeline feature coming soon")}>
          Create Pipeline
        </Button>
      </div>

      {loading && <CardListSkeleton count={6} />}
      {error && !loading && <ErrorDisplay error={error} onRetry={fetchPipelines} />}

      {!loading && !error && pipelines.length === 0 && (
        <Card className="bg-navy text-white border-blue p-8 text-center">
          <p className="text-white/60 mb-4">No pipelines found</p>
          <Button onClick={() => toast.info("Create pipeline feature coming soon")}>
            Create Your First Pipeline
          </Button>
        </Card>
      )}

      {!loading && !error && pipelines.length > 0 && (
        <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-6">
          {pipelines.map((pipeline) => (
            <Card key={pipeline.id} className="bg-navy text-white border-blue p-6">
              <div className="flex justify-between items-start mb-3">
                <h2 className="text-xl font-bold text-orange">{pipeline.name}</h2>
                {pipeline.status && (
                  <Badge className={`${getStatusColor(pipeline.status)} text-white`}>
                    {pipeline.status}
                  </Badge>
                )}
              </div>
              <p className="text-sm text-white/60 mb-2">ID: {pipeline.id}</p>
              {pipeline.steps && Array.isArray(pipeline.steps) && (
                <p className="text-sm text-white/60 mb-4">
                  {pipeline.steps.length} step{pipeline.steps.length !== 1 ? "s" : ""}
                </p>
              )}
              
              <div className="flex flex-wrap gap-2 mt-4">
                <Button asChild size="sm" variant="default">
                  <Link href={`/pipelines/${pipeline.id}`}>View</Link>
                </Button>
                <Button
                  size="sm"
                  variant="outline"
                  onClick={() => openCloneDialog(pipeline)}
                >
                  Clone
                </Button>
                <Button
                  size="sm"
                  variant="destructive"
                  onClick={() => openDeleteDialog(pipeline)}
                >
                  Delete
                </Button>
              </div>
            </Card>
          ))}
        </div>
      )}

      {/* Clone Dialog */}
      <Dialog open={cloneDialogOpen} onOpenChange={setCloneDialogOpen}>
        <DialogContent className="bg-navy text-white border-blue">
          <DialogHeader>
            <DialogTitle className="text-orange">Clone Pipeline</DialogTitle>
            <DialogDescription className="text-white/60">
              Create a copy of &quot;{selectedPipeline?.name}&quot;
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

      {/* Delete Confirmation Dialog */}
      <Dialog open={deleteDialogOpen} onOpenChange={setDeleteDialogOpen}>
        <DialogContent className="bg-navy text-white border-blue">
          <DialogHeader>
            <DialogTitle className="text-red-500">Delete Pipeline</DialogTitle>
            <DialogDescription className="text-white/60">
              Are you sure you want to delete &quot;{selectedPipeline?.name}&quot;? This action cannot be undone.
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
    </div>
  );
}
