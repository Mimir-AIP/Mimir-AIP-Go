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
  const [addStepDialogOpen, setAddStepDialogOpen] = useState(false);
  const [editorMode, setEditorMode] = useState<"visual" | "yaml">("visual");

  // Create pipeline form state
  const [createFormData, setCreateFormData] = useState({
    name: "",
    description: "",
    pipelineType: "ingestion" as "ingestion" | "processing" | "output",
    schedule: "",
    yamlConfig: "",
  });

  // Visual steps state (for visual editor mode)
  const [visualSteps, setVisualSteps] = useState<Array<{
    id: string;
    name: string;
    plugin: string;
    config: Record<string, unknown>;
  }>>([]);

  // New step form state
  const [newStep, setNewStep] = useState({
    name: "",
    plugin: "Input.csv",
    config: '{\n  "file_path": "/data/input.csv",\n  "has_headers": true\n}',
  });

  // Templates for common pipeline types
  const PIPELINE_TEMPLATES = [
    {
      id: "json-api",
      name: "JSON API Import",
      description: "Fetch JSON data from an API endpoint",
      steps: [
        { name: "fetch-api", plugin: "Input.api", config: { url: "https://api.example.com/data", format: "json" } },
      ],
    },
    {
      id: "excel-web",
      name: "Excel Web Import",
      description: "Download and parse Excel file from URL",
      steps: [
        { name: "download-excel", plugin: "Input.http", config: { url: "https://example.com/data.xlsx", method: "GET" } },
        { name: "parse-excel", plugin: "Input.excel", config: { sheet: 0, has_headers: true } },
      ],
    },
    {
      id: "csv-file",
      name: "CSV File Import",
      description: "Read CSV file from local path",
      steps: [
        { name: "read-csv", plugin: "Input.csv", config: { file_path: "/data/input.csv", has_headers: true } },
      ],
    },
    {
      id: "database",
      name: "Database Import",
      description: "Query data from database",
      steps: [
        { name: "query-db", plugin: "Input.postgres", config: { query: "SELECT * FROM data", connection_string: "postgres://..." } },
      ],
    },
    {
      id: "email-alert",
      name: "Email Alert Output",
      description: "Send anomaly alerts via email",
      steps: [
        { name: "format-alert", plugin: "Transform.py", config: { operation: "format_alert", template: "Alert: {severity} - {message}" } },
        { name: "send-email", plugin: "Output.email", config: { to: "admin@example.com", subject: "Mimir Alert" } },
      ],
    },
    {
      id: "api-output",
      name: "API Export",
      description: "Export data to external API",
      steps: [
        { name: "post-data", plugin: "Output.api", config: { url: "https://api.example.com/webhook", method: "POST" } },
      ],
    },
    {
      id: "custom",
      name: "Custom Pipeline",
      description: "Build your own pipeline from scratch",
      steps: [],
    },
  ];

  // Generate YAML from visual steps
  function generateYamlFromSteps(): string {
    const config: any = {
      version: "1.0",
      name: createFormData.name || "my-pipeline",
      description: createFormData.description || "A pipeline created with Mimir",
      steps: visualSteps.map((step) => ({
        name: step.name,
        plugin: step.plugin,
        config: step.config,
      })),
    };
    return yaml.dump(config, { indent: 2 });
  }

  // Parse YAML to visual steps
  function parseYamlToSteps(yamlStr: string): Array<{
    id: string;
    name: string;
    plugin: string;
    config: Record<string, unknown>;
  }> {
    try {
      const parsed = yaml.load(yamlStr) as any;
      if (parsed.steps && Array.isArray(parsed.steps)) {
        return parsed.steps.map((step: any, index: number) => ({
          id: `step-${index}-${Date.now()}`,
          name: step.name || `step-${index + 1}`,
          plugin: step.plugin || "Filter.py",
          config: step.config || {},
        }));
      }
    } catch (e) {
      // Invalid YAML, return empty
    }
    return [];
  }

  // Apply template
  function applyTemplate(templateId: string) {
    const template = PIPELINE_TEMPLATES.find((t) => t.id === templateId);
    if (!template) return;

    const steps = template.steps.map((step, index) => ({
      id: `step-${Date.now()}-${index}`,
      name: step.name,
      plugin: step.plugin,
      config: step.config,
    }));

    setVisualSteps(steps);
    setCreateFormData({
      ...createFormData,
      name: template.name,
      description: template.description,
    });
    setEditorMode("visual");
    toast.success(`Applied "${template.name}" template`);
  }

  // Switch to visual mode - parse existing YAML
  function switchToVisualMode() {
    if (!visualSteps.length && createFormData.yamlConfig) {
      const steps = parseYamlToSteps(createFormData.yamlConfig);
      setVisualSteps(steps);
    }
    setEditorMode("visual");
  }

  // Switch to YAML mode - generate from visual steps
  function switchToYamlMode() {
    const yaml = generateYamlFromSteps();
    setCreateFormData({ ...createFormData, yamlConfig: yaml });
    setEditorMode("yaml");
  }

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

    if (visualSteps.length === 0 && !createFormData.yamlConfig.trim()) {
      toast.error("Pipeline must have at least one step");
      return;
    }

    try {
      setIsProcessing(true);

      // Get steps from visual editor or YAML
      let steps: any[];
      if (editorMode === "visual" && visualSteps.length > 0) {
        steps = visualSteps.map(({ id, ...step }) => step);
      } else {
        // Parse YAML
        try {
          const parsedConfig = yaml.load(createFormData.yamlConfig) as any;
          if (!parsedConfig?.steps || !Array.isArray(parsedConfig.steps) || parsedConfig.steps.length === 0) {
            throw new Error("Pipeline must have at least one step");
          }
          steps = parsedConfig.steps;
        } catch (parseErr) {
          const message = parseErr instanceof Error ? parseErr.message : "Invalid YAML";
          toast.error(`YAML parsing error: ${message}`);
          return;
        }
      }

      const metadata = {
        name: createFormData.name,
        description: createFormData.description,
        enabled: true,
        tags: [createFormData.pipelineType],
        schedule: createFormData.pipelineType === "ingestion" ? (createFormData.schedule || "*/5 * * * *") : undefined,
      };

      const config = {
        name: createFormData.name,
        description: createFormData.description,
        version: "1.0",
        enabled: true,
        steps,
      };

      await createPipeline(metadata, config);
      toast.success(`Pipeline "${createFormData.name}" created successfully`);
      setCreateDialogOpen(false);
      setCreateFormData({
        name: "",
        description: "",
        pipelineType: "ingestion",
        schedule: "",
        yamlConfig: "",
      });
      setVisualSteps([]);
      await fetchPipelines();
    } catch (err) {
      const message = err instanceof Error ? err.message : "Unknown error";
      toast.error(`Failed to create pipeline: ${message}`);
    } finally {
      setIsProcessing(false);
    }
  }

  function handleAddStep() {
    if (!newStep.name.trim()) {
      toast.error("Please enter a step name");
      return;
    }

    try {
      const config = JSON.parse(newStep.config);
      const step = {
        id: `step-${Date.now()}`,
        name: newStep.name,
        plugin: newStep.plugin,
        config,
      };
      setVisualSteps([...visualSteps, step]);
      setAddStepDialogOpen(false);
      setNewStep({
        name: "",
        plugin: "Filter.py",
        config: '{\n  "condition": "value > threshold"\n}',
      });
      toast.success(`Step "${newStep.name}" added`);
    } catch (err) {
      toast.error("Invalid JSON configuration");
    }
  }

  function removeStep(stepId: string) {
    setVisualSteps(visualSteps.filter((s) => s.id !== stepId));
    toast.success("Step removed");
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

            {/* Templates Section */}
            {editorMode === "visual" && visualSteps.length === 0 && (
              <div className="grid gap-2">
                <Label>Quick Start Templates</Label>
                <div className="grid grid-cols-2 gap-3">
                  {PIPELINE_TEMPLATES.map((template) => (
                    <button
                      key={template.id}
                      onClick={() => applyTemplate(template.id)}
                      className="p-4 rounded-lg border border-blue/50 bg-blue/10 hover:bg-blue/20 hover:border-orange/50 transition-all text-left"
                    >
                      <div className="font-medium text-white mb-1">{template.name}</div>
                      <div className="text-xs text-gray-400">{template.description}</div>
                      <div className="text-xs text-orange mt-2">
                        {template.steps.length} step{template.steps.length !== 1 ? "s" : ""}
                      </div>
                    </button>
                  ))}
                </div>
              </div>
            )}

            {/* Editor Mode Toggle */}
            <div className="flex gap-2 border-t border-blue/30 pt-4">
              <Button
                type="button"
                variant={editorMode === "visual" ? "default" : "outline"}
                size="sm"
                onClick={switchToVisualMode}
                className={editorMode === "visual" ? "bg-orange text-navy" : "border-blue text-gray-400"}
              >
                <span className="mr-1">🎨</span> Visual Editor
              </Button>
              <Button
                type="button"
                variant={editorMode === "yaml" ? "default" : "outline"}
                size="sm"
                onClick={switchToYamlMode}
                className={editorMode === "yaml" ? "bg-orange text-navy" : "border-blue text-gray-400"}
              >
                <span className="mr-1">{`{ }`}</span> YAML
              </Button>
            </div>

            {/* Visual Editor - Steps */}
            {editorMode === "visual" && (
              <div className="grid gap-4">
                <div className="flex justify-between items-center">
                  <Label>Pipeline Steps ({visualSteps.length})</Label>
                  <Button
                    type="button"
                    size="sm"
                    variant="outline"
                    className="border-orange text-orange hover:bg-orange/10"
                    onClick={() => setAddStepDialogOpen(true)}
                  >
                    + Add Step
                  </Button>
                </div>

                {visualSteps.length === 0 ? (
                  <div className="p-8 text-center border border-dashed border-blue/50 rounded-lg bg-blue/5">
                    <p className="text-gray-400 mb-2">No steps added yet</p>
                    <p className="text-sm text-gray-500">Use a template above or add steps manually</p>
                  </div>
                ) : (
                  <div className="space-y-3">
                    {visualSteps.map((step, index) => (
                      <div
                        key={step.id}
                        className="flex items-center gap-3 p-3 rounded-lg bg-blue/10 border border-blue/30"
                      >
                        <div className="flex-shrink-0 w-8 h-8 rounded-full bg-orange flex items-center justify-center text-navy font-bold text-sm">
                          {index + 1}
                        </div>
                        <div className="flex-1 min-w-0">
                          <div className="font-medium text-white">{step.name}</div>
                          <div className="text-xs text-gray-400">{step.plugin}</div>
                        </div>
                        <Button
                          type="button"
                          variant="ghost"
                          size="sm"
                          onClick={() => removeStep(step.id)}
                          className="text-red-400 hover:text-red-300 hover:bg-red-500/10"
                        >
                          ✕
                        </Button>
                      </div>
                    ))}
                  </div>
                )}

                {/* Auto-generated YAML Preview */}
                <div className="mt-2 p-3 rounded bg-navy/50 border border-blue/30">
                  <div className="text-xs text-gray-400 mb-2">Generated YAML Preview:</div>
                  <pre className="text-xs text-orange font-mono overflow-x-auto max-h-32">
                    {generateYamlFromSteps() || "(no steps yet)"}
                  </pre>
                </div>
              </div>
            )}

            {/* YAML Editor */}
            {editorMode === "yaml" && (
              <div className="grid gap-2">
                <Label htmlFor="create-yaml">Pipeline Configuration (YAML)</Label>
                <Textarea
                  id="create-yaml"
                  name="yamlConfig"
                  value={createFormData.yamlConfig}
                  onChange={(e) => setCreateFormData({ ...createFormData, yamlConfig: e.target.value })}
                  placeholder="version: '1.0'
name: my-pipeline
steps:
  - name: fetch-data
    plugin: Input.csv
    config:
      file_path: /data/input.csv"
                  className="bg-blue/10 border-blue text-white font-mono text-sm min-h-[300px]"
                />
                <p className="text-xs text-white/60">
                  Define pipeline steps, plugins, and configuration in YAML format
                </p>
              </div>
            )}
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

      {/* Add Step Dialog */}
      <Dialog open={addStepDialogOpen} onOpenChange={setAddStepDialogOpen}>
        <DialogContent className="bg-navy text-white border-blue max-w-md">
          <DialogHeader>
            <DialogTitle className="text-orange">Add Pipeline Step</DialogTitle>
            <DialogDescription className="text-white/60">
              Configure a new step for your pipeline
            </DialogDescription>
          </DialogHeader>
          <div className="grid gap-4 py-4">
            <div className="grid gap-2">
              <Label htmlFor="step-name">Step Name *</Label>
              <Input
                id="step-name"
                value={newStep.name}
                onChange={(e) => setNewStep({ ...newStep, name: e.target.value })}
                placeholder="e.g., fetch-api, parse-excel, filter-data"
                className="bg-blue/10 border-blue text-white"
              />
            </div>
            <div className="grid gap-2">
              <Label htmlFor="step-plugin">Plugin Type</Label>
              <Select
                value={newStep.plugin}
                onValueChange={(value) => {
                  // Set default config based on plugin
                  const configs: Record<string, string> = {
                    "Input.api": '{\n  "url": "https://api.example.com/data",\n  "method": "GET",\n  "format": "json"\n}',
                    "Input.http": '{\n  "url": "https://example.com/file.xlsx",\n  "method": "GET"\n}',
                    "Input.excel": '{\n  "file_path": "/data/input.xlsx",\n  "sheet": 0,\n  "has_headers": true\n}',
                    "Input.csv": '{\n  "file_path": "/data/input.csv",\n  "has_headers": true\n}',
                    "Input.postgres": '{\n  "query": "SELECT * FROM data",\n  "connection_string": "postgres://user:pass@localhost/db"\n}',
                    "Filter.py": '{\n  "condition": "value > threshold"\n}',
                    "Transform.py": '{\n  "operation": "normalize",\n  "columns": ["col1", "col2"]\n}',
                    "Output.csv": '{\n  "file_path": "/data/output.csv"\n}',
                    "Output.email": '{\n  "to": "admin@example.com",\n  "subject": "Mimir Alert",\n  "body_template": "Alert: {severity} - {message}"\n}',
                    "Output.api": '{\n  "url": "https://api.example.com/webhook",\n  "method": "POST",\n  "headers": {"Content-Type": "application/json"}\n}',
                    "AnomalyDetector.py": '{\n  "algorithm": "isolation_forest",\n  "contamination": 0.1\n}',
                  };
                  setNewStep({
                    name: newStep.name,
                    plugin: value,
                    config: configs[value] || '{}',
                  });
                }}
              >
                <SelectTrigger className="bg-blue/10 border-blue text-white">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="Input.csv">📄 Input.csv - Read CSV file</SelectItem>
                  <SelectItem value="Input.excel">📊 Input.excel - Parse Excel file</SelectItem>
                  <SelectItem value="Input.api">🌐 Input.api - Fetch from REST API</SelectItem>
                  <SelectItem value="Input.http">🔗 Input.http - Download from URL</SelectItem>
                  <SelectItem value="Input.postgres">🐘 Input.postgres - Query PostgreSQL</SelectItem>
                  <SelectItem value="Filter.py">🔍 Filter.py - Filter rows</SelectItem>
                  <SelectItem value="Transform.py">⚙️ Transform.py - Transform data</SelectItem>
                  <SelectItem value="Output.csv">📤 Output.csv - Write to CSV</SelectItem>
                  <SelectItem value="Output.email">📧 Output.email - Send email alert</SelectItem>
                  <SelectItem value="Output.api">🔗 Output.api - POST to webhook</SelectItem>
                  <SelectItem value="AnomalyDetector.py">⚠️ AnomalyDetector.py - Detect anomalies</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div className="grid gap-2">
              <Label htmlFor="step-config">Configuration (JSON)</Label>
              <Textarea
                id="step-config"
                value={newStep.config}
                onChange={(e) => setNewStep({ ...newStep, config: e.target.value })}
                placeholder='{"key": "value"}'
                className="bg-blue/10 border-blue text-white font-mono text-sm min-h-[120px]"
              />
              <p className="text-xs text-gray-500">
                {newStep.plugin.startsWith("Input") && "Configure source, URL, file path, or query"}
                {newStep.plugin === "Filter.py" && "Define condition: value > threshold, column == 'value'"}
                {newStep.plugin === "Transform.py" && "Specify operation and columns to transform"}
                {newStep.plugin === "Output.csv" && "Configure output file path for CSV export"}
                {newStep.plugin === "Output.email" && "Configure email recipient, subject, and body template"}
                {newStep.plugin === "Output.api" && "Configure webhook URL, method, and headers"}
                {newStep.plugin === "AnomalyDetector.py" && "Configure algorithm and contamination threshold"}
              </p>
            </div>
          </div>
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => setAddStepDialogOpen(false)}
            >
              Cancel
            </Button>
            <Button onClick={handleAddStep} disabled={!newStep.name.trim()}>
              Add Step
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
