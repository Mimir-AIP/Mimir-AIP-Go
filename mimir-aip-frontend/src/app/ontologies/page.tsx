"use client";

import { useState, useEffect } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import {
  listOntologies,
  deleteOntology,
  getPipelines,
  type Ontology,
  type Pipeline,
} from "@/lib/api";
import { Button } from "@/components/ui/button";
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
import { Checkbox } from "@/components/ui/checkbox";
import { toast } from "sonner";

export default function OntologiesPage() {
  const router = useRouter();
  const [ontologies, setOntologies] = useState<Ontology[]>([]);
  const [pipelines, setPipelines] = useState<Pipeline[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [statusFilter, setStatusFilter] = useState<string>("");
  
  // Create from pipeline dialog
  const [createDialogOpen, setCreateDialogOpen] = useState(false);
  const [selectedPipelines, setSelectedPipelines] = useState<string[]>([]);
  const [ontologyName, setOntologyName] = useState("");
  const [isCreating, setIsCreating] = useState(false);

  useEffect(() => {
    loadOntologies();
    loadPipelines();
  }, [statusFilter]);

  const loadOntologies = async () => {
    try {
      setLoading(true);
      setError(null);
      const data = await listOntologies(statusFilter);
      setOntologies(data || []); // Handle null/undefined responses
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load ontologies");
      setOntologies([]); // Set empty array on error
    } finally {
      setLoading(false);
    }
  };

  const loadPipelines = async () => {
    try {
      const data = await getPipelines();
      // Filter to only show ingestion pipelines
      const ingestionPipelines = data.filter(p => p.tags?.includes("ingestion"));
      setPipelines(ingestionPipelines.length > 0 ? ingestionPipelines : data);
    } catch (err) {
      console.error("Failed to load pipelines:", err);
    }
  };

  const handleCreateFromPipeline = async () => {
    if (selectedPipelines.length === 0) {
      toast.error("Please select at least one pipeline");
      return;
    }
    if (!ontologyName.trim()) {
      toast.error("Please enter an ontology name");
      return;
    }

    setIsCreating(true);
    try {
      const apiBase = process.env.NEXT_PUBLIC_API_URL || "";
      
      // Create workflow that will execute the full autonomous flow
      const workflowResponse = await fetch(`${apiBase}/api/v1/workflows`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          name: `Ontology: ${ontologyName}`,
          pipeline_ids: selectedPipelines,
          ontology_name: ontologyName,
        }),
      });

      if (!workflowResponse.ok) {
        throw new Error("Failed to create workflow");
      }

      const workflowData = await workflowResponse.json();
      
      // Execute workflow
      const executeResponse = await fetch(`${apiBase}/api/v1/workflows/${workflowData.workflow_id}/execute`, {
        method: "POST",
      });

      if (!executeResponse.ok) {
        throw new Error("Failed to start workflow");
      }

      toast.success("Ontology creation started! Mimir will automatically extract entities, train ML models, and create a digital twin.");
      setCreateDialogOpen(false);
      setSelectedPipelines([]);
      setOntologyName("");
      
      // Navigate to workflows to track progress
      router.push(`/workflows/${workflowData.workflow_id}`);
    } catch (err) {
      toast.error(`Failed to create ontology: ${err instanceof Error ? err.message : "Unknown error"}`);
    } finally {
      setIsCreating(false);
    }
  };

  const togglePipelineSelection = (pipelineId: string) => {
    setSelectedPipelines(prev => 
      prev.includes(pipelineId) 
        ? prev.filter(id => id !== pipelineId)
        : [...prev, pipelineId]
    );
  };

  const handleDelete = async (id: string, name: string) => {
    if (!confirm(`Are you sure you want to delete ontology "${name}"?`)) {
      return;
    }

    try {
      await deleteOntology(id);
      await loadOntologies();
    } catch (err) {
      alert(`Failed to delete ontology: ${err instanceof Error ? err.message : "Unknown error"}`);
    }
  };

  const handleExport = (id: string, name: string, format: string) => {
    window.open(`/api/v1/ontology/${id}/export?format=${format}`, "_blank");
  };

  return (
    <div className="space-y-6">
      {/* Page Header */}
      <div className="flex flex-col md:flex-row justify-between items-start md:items-center gap-4">
        <div>
          <h1 className="text-3xl font-bold text-orange">Ontologies</h1>
          <p className="text-white/60 mt-1">
            Define knowledge schemas and let Mimir auto-generate from your data
          </p>
        </div>
        <div className="flex flex-wrap gap-3">
          <Button 
            onClick={() => setCreateDialogOpen(true)}
            className="bg-gradient-to-r from-blue-500 to-purple-500 hover:from-blue-600 hover:to-purple-600 text-white font-semibold"
          >
            ✨ Create from Pipeline
          </Button>
          <Link
            href="/ontologies/upload"
            className="inline-flex items-center gap-2 bg-orange hover:bg-orange/90 text-navy px-4 py-2 rounded-lg font-semibold transition-colors"
          >
            📤 Upload Ontology
          </Link>
        </div>
      </div>

      <div className="mb-4 flex gap-4 items-center">
        <label className="flex items-center gap-2">
          <span className="text-sm font-medium text-white">Status:</span>
          <select
            value={statusFilter}
            onChange={(e) => setStatusFilter(e.target.value)}
            className="bg-navy border border-blue rounded px-3 py-1 text-white focus:border-orange focus:ring-1 focus:ring-orange"
          >
            <option value="">All</option>
            <option value="active">Active</option>
            <option value="deprecated">Deprecated</option>
            <option value="draft">Draft</option>
            <option value="archived">Archived</option>
          </select>
        </label>
        <button
          onClick={loadOntologies}
          className="bg-blue hover:bg-orange text-white px-4 py-1 rounded border border-blue"
        >
          Refresh
        </button>
      </div>

      {loading && (
        <div className="text-center py-12">
          <p className="text-gray-400">Loading ontologies...</p>
        </div>
      )}

      {error && (
        <div className="bg-red-900/20 border border-red-500 text-red-400 px-4 py-3 rounded mb-4">
          {error}
        </div>
      )}

      {!loading && !error && ontologies.length === 0 && (
        <div className="text-center py-12">
          <p className="text-gray-400 mb-4">No ontologies found</p>
          <Link
            href="/ontologies/upload"
            className="text-orange hover:underline"
          >
            Upload your first ontology
          </Link>
        </div>
      )}

      {!loading && !error && ontologies.length > 0 && (
        <div className="bg-blue border border-blue rounded-lg shadow overflow-hidden">
          <table className="min-w-full divide-y divide-gray-700">
            <thead className="bg-navy">
              <tr>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-400 uppercase tracking-wider">
                  Name
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-400 uppercase tracking-wider">
                  Version
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-400 uppercase tracking-wider">
                  Format
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-400 uppercase tracking-wider">
                  Status
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-400 uppercase tracking-wider">
                  Created
                </th>
                <th className="px-6 py-3 text-right text-xs font-medium text-gray-400 uppercase tracking-wider">
                  Actions
                </th>
              </tr>
            </thead>
            <tbody className="bg-blue divide-y divide-gray-700">
              {ontologies.map((ontology) => (
                <tr key={ontology.id} className="hover:bg-navy transition-colors">
                  <td className="px-6 py-4 whitespace-nowrap">
                    <div className="text-sm font-medium text-white">
                      {ontology.name}
                    </div>
                    {ontology.description && (
                      <div className="text-sm text-gray-400">
                        {ontology.description}
                      </div>
                    )}
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-white">
                    {ontology.version}
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-white">
                    {ontology.format}
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap">
                    <span
                      className={`px-2 inline-flex text-xs leading-5 font-semibold rounded-full ${
                        ontology.status === "active"
                          ? "bg-green-900/40 text-green-400 border border-green-500"
                          : ontology.status === "deprecated"
                          ? "bg-yellow-900/40 text-yellow-400 border border-yellow-500"
                          : ontology.status === "draft"
                          ? "bg-blue-900/40 text-blue-400 border border-blue-500"
                          : "bg-gray-800 text-gray-400 border border-gray-600"
                      }`}
                    >
                      {ontology.status}
                    </span>
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-400">
                    {new Date(ontology.created_at).toLocaleDateString()}
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-right text-sm font-medium">
                    <div className="flex justify-end gap-2">
                      <Link
                        href={`/ontologies/${ontology.id}`}
                        className="text-orange hover:text-orange/80"
                      >
                        View
                      </Link>
                      <button
                        onClick={() =>
                          handleExport(ontology.id, ontology.name, ontology.format)
                        }
                        className="text-green-400 hover:text-green-300"
                      >
                        Export
                      </button>
                      <button
                        onClick={() => handleDelete(ontology.id, ontology.name)}
                        className="text-red-400 hover:text-red-300"
                      >
                        Delete
                      </button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      <div className="mt-6">
        <p className="text-sm text-gray-400">
          Total: {ontologies.length} ontolog{ontologies.length === 1 ? "y" : "ies"}
        </p>
      </div>

      {/* Create from Pipeline Dialog */}
      <Dialog open={createDialogOpen} onOpenChange={setCreateDialogOpen}>
        <DialogContent className="bg-navy text-white border-blue max-w-2xl">
          <DialogHeader>
            <DialogTitle className="text-orange">✨ Create Ontology from Pipeline</DialogTitle>
            <DialogDescription className="text-white/60">
              Select ingestion pipeline(s) to generate an ontology. Mimir will automatically:
              <ul className="list-disc ml-6 mt-2 space-y-1">
                <li>Run the pipeline(s) to fetch data</li>
                <li>Extract entities and relationships</li>
                <li>Generate the ontology</li>
                <li>Train ML models automatically</li>
                <li>Create a digital twin</li>
              </ul>
            </DialogDescription>
          </DialogHeader>
          
          <div className="grid gap-4 py-4">
            <div className="grid gap-2">
              <Label htmlFor="ontology-name">Ontology Name *</Label>
              <Input
                id="ontology-name"
                value={ontologyName}
                onChange={(e) => setOntologyName(e.target.value)}
                placeholder="e.g., Customer Data Ontology"
                className="bg-blue/10 border-blue text-white"
              />
            </div>
            
            <div className="grid gap-2">
              <Label>Select Ingestion Pipeline(s) *</Label>
              <div className="border border-blue rounded-lg p-3 max-h-60 overflow-y-auto">
                {pipelines.length === 0 ? (
                  <p className="text-white/60 text-center py-4">
                    No pipelines found. <Link href="/pipelines" className="text-orange underline">Create a pipeline first</Link>
                  </p>
                ) : (
                  <div className="space-y-2">
                    {pipelines.map(pipeline => (
                      <div 
                        key={pipeline.id}
                        className={`flex items-center gap-3 p-2 rounded cursor-pointer hover:bg-blue/20 ${
                          selectedPipelines.includes(pipeline.id) ? 'bg-blue/30 border border-orange' : ''
                        }`}
                        onClick={() => togglePipelineSelection(pipeline.id)}
                      >
                        <Checkbox 
                          checked={selectedPipelines.includes(pipeline.id)}
                          onCheckedChange={() => togglePipelineSelection(pipeline.id)}
                        />
                        <div>
                          <p className="font-medium text-white">{pipeline.name}</p>
                          {pipeline.tags?.includes("ingestion") && (
                            <span className="text-xs bg-blue-500 text-white px-2 py-0.5 rounded">📥 Ingestion</span>
                          )}
                        </div>
                      </div>
                    ))}
                  </div>
                )}
              </div>
              <p className="text-xs text-white/60">
                Selected: {selectedPipelines.length} pipeline(s)
              </p>
            </div>
          </div>
          
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => setCreateDialogOpen(false)}
              disabled={isCreating}
            >
              Cancel
            </Button>
            <Button 
              onClick={handleCreateFromPipeline} 
              disabled={isCreating || selectedPipelines.length === 0 || !ontologyName.trim()}
              className="bg-orange hover:bg-orange/80 text-navy"
            >
              {isCreating ? "Creating..." : "🚀 Start Autonomous Creation"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
