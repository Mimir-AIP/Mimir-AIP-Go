"use client";

import { useState, useEffect } from "react";
import { useRouter } from "next/navigation";
import { createDigitalTwin, listOntologies, type Ontology } from "@/lib/api";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { toast } from "sonner";
import { ArrowLeft, Loader2, Network } from "lucide-react";
import Link from "next/link";

const MODEL_TYPES = [
  { value: "organization", label: "Organization", description: "Complete organizational structure" },
  { value: "department", label: "Department", description: "Single department or team" },
  { value: "process", label: "Process", description: "Business process or workflow" },
  { value: "supply_chain", label: "Supply Chain", description: "Supply chain network" },
  { value: "system", label: "System", description: "Technical system or infrastructure" },
  { value: "custom", label: "Custom", description: "Custom model type" },
];

export default function CreateTwinPage() {
  const router = useRouter();
  const [loading, setLoading] = useState(false);
  const [ontologies, setOntologies] = useState<Ontology[]>([]);
  const [loadingOntologies, setLoadingOntologies] = useState(true);

  const [formData, setFormData] = useState({
    name: "",
    description: "",
    ontology_id: "",
    model_type: "organization",
    query: "",
  });

  useEffect(() => {
    loadOntologies();
  }, []);

  async function loadOntologies() {
    try {
      setLoadingOntologies(true);
      const data = await listOntologies();
      setOntologies(data.filter((o) => o.status === "active"));
    } catch (err) {
      toast.error("Failed to load ontologies");
    } finally {
      setLoadingOntologies(false);
    }
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();

    if (!formData.name || !formData.ontology_id || !formData.model_type) {
      toast.error("Please fill in all required fields");
      return;
    }

    setLoading(true);
    try {
      const response = await createDigitalTwin({
        name: formData.name,
        description: formData.description || undefined,
        ontology_id: formData.ontology_id,
        model_type: formData.model_type,
        query: formData.query || undefined,
      });

      toast.success(
        `Digital twin created successfully! ${response.entity_count} entities, ${response.relationship_count} relationships`
      );
      router.push(`/digital-twins/${response.twin_id}`);
    } catch (err) {
      const message = err instanceof Error ? err.message : "Failed to create digital twin";
      toast.error(message);
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="container mx-auto py-8 max-w-3xl">
      <Link href="/digital-twins">
        <Button variant="ghost" className="mb-4">
          <ArrowLeft className="h-4 w-4 mr-2" />
          Back to Digital Twins
        </Button>
      </Link>

      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Network className="h-6 w-6" />
            Create Digital Twin
          </CardTitle>
          <CardDescription>
            Create a digital twin from your knowledge graph to simulate business scenarios
          </CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleSubmit} className="space-y-6">
            {/* Twin Name */}
            <div className="space-y-2">
              <Label htmlFor="name">
                Twin Name <span className="text-red-500">*</span>
              </Label>
              <Input
                id="name"
                name="name"
                placeholder="e.g., Q4 2024 Organization Model"
                value={formData.name}
                onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                required
              />
            </div>

            {/* Description */}
            <div className="space-y-2">
              <Label htmlFor="description">Description</Label>
              <Textarea
                id="description"
                name="description"
                placeholder="Enter description for this digital twin"
                value={formData.description}
                onChange={(e) => setFormData({ ...formData, description: e.target.value })}
                rows={3}
              />
            </div>

            {/* Ontology Selection */}
            <div className="space-y-2">
              <Label htmlFor="ontology">
                Source Ontology <span className="text-red-500">*</span>
              </Label>
              {loadingOntologies ? (
                <div className="flex items-center gap-2 text-sm text-muted-foreground">
                  <Loader2 className="h-4 w-4 animate-spin" />
                  Loading ontologies...
                </div>
              ) : ontologies.length === 0 ? (
                <div className="text-sm text-muted-foreground">
                  No active ontologies found.{" "}
                  <Link href="/ontologies/upload" className="underline">
                    Upload an ontology first
                  </Link>
                </div>
              ) : (
                <select
                  id="ontology"
                  name="ontology"
                  className="w-full border rounded-md p-2"
                  value={formData.ontology_id}
                  onChange={(e) => setFormData({ ...formData, ontology_id: e.target.value })}
                  required
                >
                  <option value="">Select an ontology...</option>
                  {ontologies.map((ontology) => (
                    <option key={ontology.id} value={ontology.id}>
                      {ontology.name} (v{ontology.version})
                    </option>
                  ))}
                </select>
              )}
            </div>

            {/* Model Type */}
            <div className="space-y-2">
              <Label htmlFor="model_type">
                Model Type <span className="text-red-500">*</span>
              </Label>
              <select
                id="model_type"
                name="type"
                className="w-full border rounded-md p-2"
                value={formData.model_type}
                onChange={(e) => setFormData({ ...formData, model_type: e.target.value })}
                required
              >
                {MODEL_TYPES.map((type) => (
                  <option key={type.value} value={type.value}>
                    {type.label} - {type.description}
                  </option>
                ))}
              </select>
            </div>

            {/* Optional SPARQL Query */}
            <div className="space-y-2">
              <Label htmlFor="query">Custom SPARQL Query (Optional)</Label>
              <Textarea
                id="query"
                placeholder="Leave empty to include all entities, or provide a custom SPARQL query..."
                value={formData.query}
                onChange={(e) => setFormData({ ...formData, query: e.target.value })}
                rows={6}
                className="font-mono text-sm"
              />
              <p className="text-xs text-muted-foreground">
                Advanced: Provide a SPARQL SELECT query to filter which entities to include in the twin.
                Query must return ?entity, ?type, and optionally ?label.
              </p>
            </div>

            {/* Submit Button */}
            <div className="flex justify-end gap-3 pt-4 border-t">
              <Link href="/digital-twins">
                <Button type="button" variant="outline" disabled={loading}>
                  Cancel
                </Button>
              </Link>
              <Button type="submit" disabled={loading || loadingOntologies || ontologies.length === 0}>
                {loading ? (
                  <>
                    <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                    Creating Twin...
                  </>
                ) : (
                  "Create Digital Twin"
                )}
              </Button>
            </div>
          </form>
        </CardContent>
      </Card>
    </div>
  );
}
