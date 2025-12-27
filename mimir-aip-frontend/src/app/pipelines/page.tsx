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
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
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
    pipelineType: "ingestion" as "ingestion" | "processing" | "output",
    schedule: "",
    yamlConfig: `version: "1.0"
name: my-pipeline
description: A sample pipeline
steps:
  - name: fetch-data
    plugin: Input.csv
    config:
      file_path: /data/input.csv
      has_headers: true`,
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
        tags: [createFormData.pipelineType],  // Add pipeline type as tag
        schedule: createFormData.pipelineType === "ingestion" ? (createFormData.schedule || "*/5 * * * *") : undefined, // Default 5 min schedule for ingestion
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
        pipelineType: "ingestion",
        schedule: "",
        yamlConfig: `version: "1.0"
name: my-pipeline
description: A sample pipeline
steps:
  - name: fetch-data
    plugin: Input.csv
    config:
      file_path: /data/input.csv
      has_headers: true`,
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
    <div className="space-y-6">
      {/* Page Header */}
      <div className="flex flex-col sm:flex-row justify-between items-start sm:items-center gap-4">
        <div>
          <h1 className="text-3xl font-bold text-orange">Pipelines</h1>
          <p className="text-white/60 mt-1">Create and manage data ingestion pipelines</p>
        </div>
        <Button 
          onClick={() => setCreateDialogOpen(true)}
          className="bg-orange hover:bg-orange/90 text-navy font-semibold"
        >
          + Create Pipeline
        </Button>
      </div>

      {loading && <CardListSkeleton count={6} />}
      {error && !loading && <ErrorDisplay error={error} onRetry={fetchPipelines} />}

      {!loading && !error && pipelines.length === 0 && (
        <Card className="bg-gradient-to-br from-navy to-blue/20 text-white border-blue border-dashed p-12 text-center">
          <div className="max-w-md mx-auto">
            <div className="w-16 h-16 mx-auto mb-4 rounded-full bg-blue/30 flex items-center justify-center">
              <span className="text-3xl">🔄</span>
            </div>
            <h3 className="text-xl font-semibold text-white mb-2">No Pipelines Yet</h3>
            <p className="text-white/60 mb-6">
              Pipelines define how data flows into Mimir. Create an ingestion pipeline to start pulling data from databases, APIs, or files.
            </p>
            <Button 
              onClick={() => setCreateDialogOpen(true)}
              className="bg-orange hover:bg-orange/90 text-navy"
            >
              Create Your First Pipeline
            </Button>
          </div>
        </Card>
      )}

      {!loading && !error && pipelines.length > 0 && (
        <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-6">
          {pipelines.map((pipeline) => (
            <Card key={pipeline.id} className="bg-gradient-to-br from-navy to-blue/30 text-white border-blue hover:border-orange/50 transition-all duration-200 p-6 group">
              <div className="flex justify-between items-start mb-3">
                <h2 className="text-xl font-bold text-orange group-hover:text-orange/90">{pipeline.name || 'Unnamed Pipeline'}</h2>
                <div className="flex flex-wrap gap-1.5 justify-end">
                  {pipeline.tags?.includes("ingestion") && (
                    <Badge className="bg-blue-500/80 text-white text-xs">📥 Ingestion</Badge>
                  )}
                  {pipeline.tags?.includes("processing") && (
                    <Badge className="bg-purple-500/80 text-white text-xs">⚙️ Processing</Badge>
                  )}
                  {pipeline.tags?.includes("output") && (
                    <Badge className="bg-green-500/80 text-white text-xs">📤 Output</Badge>
                  )}
                  {pipeline.status && (
                    <Badge className={`${getStatusColor(pipeline.status)} text-white text-xs`}>
                      {pipeline.status}
                    </Badge>
                  )}
                </div>
              </div>
              
              {pipeline.description && (
                <p className="text-sm text-white/70 mb-3 line-clamp-2">{pipeline.description}</p>
              )}
              
              <div className="flex items-center gap-4 text-xs text-white/50 mb-4">
                <span className="font-mono bg-blue/30 px-2 py-1 rounded">ID: {pipeline.id?.slice(0, 8) || 'N/A'}...</span>
                {pipeline.steps && Array.isArray(pipeline.steps) && (
                  <span>{pipeline.steps.length} step{pipeline.steps.length !== 1 ? "s" : ""}</span>
                )}
              </div>
              
              <div className="flex flex-wrap gap-2 pt-3 border-t border-blue/30">
                <Button asChild size="sm" className="bg-orange hover:bg-orange/90 text-navy">
                  <Link href={`/pipelines/${pipeline.id}`}>View Details</Link>
                </Button>
                <Button
                  size="sm"
                  variant="outline"
                  className="border-blue hover:border-orange hover:text-orange"
                  onClick={() => openCloneDialog(pipeline)}
                >
                  Clone
                </Button>
                <Button
                  size="sm"
                  variant="ghost"
                  className="text-red-400 hover:text-red-300 hover:bg-red-500/10"
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
            <div className="grid grid-cols-2 gap-4">
              <div className="grid gap-2">
                <Label htmlFor="create-type">Pipeline Type *</Label>
                <Select
                  value={createFormData.pipelineType}
                  onValueChange={(value) => 
                    setCreateFormData({ ...createFormData, pipelineType: value as "ingestion" | "processing" | "output" })
                  }
                >
                  <SelectTrigger className="bg-blue/10 border-blue text-white">
                    <SelectValue placeholder="Select type" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="ingestion">📥 Ingestion (Data Import)</SelectItem>
                    <SelectItem value="processing">⚙️ Processing (Transform)</SelectItem>
                    <SelectItem value="output">📤 Output (Export)</SelectItem>
                  </SelectContent>
                </Select>
                <p className="text-xs text-white/60">
                  Ingestion pipelines pull data from sources (DB, API, files)
                </p>
              </div>
              {createFormData.pipelineType === "ingestion" && (
                <div className="grid gap-2">
                  <Label htmlFor="create-schedule">Schedule (Cron)</Label>
                  <Input
                    id="create-schedule"
                    value={createFormData.schedule}
                    onChange={(e) => setCreateFormData({ ...createFormData, schedule: e.target.value })}
                    placeholder="*/5 * * * * (every 5 mins)"
                    className="bg-blue/10 border-blue text-white"
                  />
                  <p className="text-xs text-white/60">
                    Leave empty for default: every 5 minutes
                  </p>
                </div>
              )}
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
