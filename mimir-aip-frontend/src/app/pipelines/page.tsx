"use client";
import Link from "next/link";
import { useEffect, useState } from "react";
import { Card } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import yaml from "js-yaml";
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
import { Textarea } from "@/components/ui/textarea";
import { getPipelines, deletePipeline, clonePipeline, createPipeline, type Pipeline } from "@/lib/api";
import { CardListSkeleton } from "@/components/LoadingSkeleton";
import { ErrorDisplay } from "@/components/ErrorBoundary";
import { toast } from "sonner";

export default function PipelinesPage() {
  const [pipelines, setPipelines] = useState<Pipeline[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [cloneDialogOpen, setCloneDialogOpen] = useState(false);
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const [createDialogOpen, setCreateDialogOpen] = useState(false);
  const [selectedPipeline, setSelectedPipeline] = useState<Pipeline | null>(null);
  const [cloneName, setCloneName] = useState("");
  const [isProcessing, setIsProcessing] = useState(false);

  // Create pipeline form state
  const [createFormData, setCreateFormData] = useState({
    name: "",
    description: "",
    yamlConfig: `version: "1.0"
name: my-pipeline
description: A sample pipeline
steps:
  - name: step1
    plugin: input/http
    config:
      url: https://example.com/api
  - name: step2
    plugin: output/json
    config:
      file: output.json`,
  });

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

  async function handleCreate() {
    if (!createFormData.name.trim()) {
      toast.error("Please enter a pipeline name");
      return;
    }

    try {
      setIsProcessing(true);
      
      // Parse YAML config
      let parsedConfig: any;
      try {
        parsedConfig = yaml.load(createFormData.yamlConfig) as any;
        
        if (!parsedConfig || typeof parsedConfig !== 'object') {
          throw new Error("Invalid YAML structure");
        }
        
        if (!parsedConfig.steps || !Array.isArray(parsedConfig.steps) || parsedConfig.steps.length === 0) {
          throw new Error("Pipeline must have at least one step");
        }
      } catch (parseErr) {
        const message = parseErr instanceof Error ? parseErr.message : "Invalid YAML";
        toast.error(`YAML parsing error: ${message}`);
        return;
      }

      const metadata = {
        name: createFormData.name,
        description: createFormData.description,
        enabled: true,
        tags: [],
      };

      const config = {
        name: parsedConfig.name || createFormData.name,
        description: parsedConfig.description || createFormData.description,
        version: parsedConfig.version || "1.0",
        enabled: true,
        steps: parsedConfig.steps,
      };

      await createPipeline(metadata, config);
      toast.success(`Pipeline "${createFormData.name}" created successfully`);
      setCreateDialogOpen(false);
      setCreateFormData({
        name: "",
        description: "",
        yamlConfig: `version: "1.0"
name: my-pipeline
description: A sample pipeline
steps:
  - name: step1
    plugin: input/http
    config:
      url: https://example.com/api
  - name: step2
    plugin: output/json
    config:
      file: output.json`,
      });
      await fetchPipelines();
    } catch (err) {
      const message = err instanceof Error ? err.message : "Unknown error";
      toast.error(`Failed to create pipeline: ${message}`);
    } finally {
      setIsProcessing(false);
    }
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
        <Button onClick={() => setCreateDialogOpen(true)}>
          Create Pipeline
        </Button>
      </div>

      {loading && <CardListSkeleton count={6} />}
      {error && !loading && <ErrorDisplay error={error} onRetry={fetchPipelines} />}

      {!loading && !error && pipelines.length === 0 && (
        <Card className="bg-navy text-white border-blue p-8 text-center">
          <p className="text-white/60 mb-4">No pipelines found</p>
          <Button onClick={() => setCreateDialogOpen(true)}>
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

      {/* Create Pipeline Dialog */}
      <Dialog open={createDialogOpen} onOpenChange={setCreateDialogOpen}>
        <DialogContent className="bg-navy text-white border-blue max-w-4xl max-h-[90vh]">
          <DialogHeader>
            <DialogTitle className="text-orange">Create New Pipeline</DialogTitle>
            <DialogDescription className="text-white/60">
              Define a new pipeline with steps and configuration
            </DialogDescription>
          </DialogHeader>
          <div className="grid gap-4 py-4 overflow-y-auto max-h-[60vh]">
            <div className="grid gap-2">
              <Label htmlFor="create-name">Pipeline Name *</Label>
              <Input
                id="create-name"
                name="name"
                value={createFormData.name}
                onChange={(e) => setCreateFormData({ ...createFormData, name: e.target.value })}
                placeholder="my-pipeline"
                className="bg-blue/10 border-blue text-white"
              />
            </div>
            <div className="grid gap-2">
              <Label htmlFor="create-description">Description</Label>
              <Input
                id="create-description"
                value={createFormData.description}
                onChange={(e) => setCreateFormData({ ...createFormData, description: e.target.value })}
                placeholder="A brief description of this pipeline"
                className="bg-blue/10 border-blue text-white"
              />
            </div>
            <div className="grid gap-2">
              <Label htmlFor="create-yaml">Pipeline Configuration (YAML)</Label>
              <Textarea
                id="create-yaml"
                name="yamlConfig"
                value={createFormData.yamlConfig}
                onChange={(e) => setCreateFormData({ ...createFormData, yamlConfig: e.target.value })}
                placeholder="Enter YAML config"
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
              onClick={() => setCreateDialogOpen(false)}
              disabled={isProcessing}
            >
              Cancel
            </Button>
            <Button onClick={handleCreate} disabled={isProcessing || !createFormData.name.trim()}>
              {isProcessing ? "Creating..." : "Create Pipeline"}
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
