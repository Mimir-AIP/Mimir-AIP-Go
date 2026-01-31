"use client";

import { useState, useEffect } from "react";
import { useParams } from "next/navigation";
import Link from "next/link";
import { getOntology, getOntologyStats, type Ontology, executeSPARQLQuery, updateOntology, uploadOntology } from "@/lib/api";
import { Card } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogTrigger } from "@/components/ui/dialog";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Textarea } from "@/components/ui/textarea";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { ArrowLeft, Database, FileText, Calendar, Info, ChevronDown, ChevronRight, Edit, Upload, RefreshCw, Code, Layers, Box } from "lucide-react";
import { toast } from "sonner";

// Entity, Class, and Property types
interface Entity {
  entity: { value: string; type: string };
  type: { value: string; type: string };
  label?: { value: string; type: string };
}

interface ClassInfo {
  class: { value: string; type: string };
  label?: { value: string; type: string };
  count?: number;
}

interface PropertyInfo {
  property: { value: string; type: string };
  label?: { value: string; type: string };
  count?: number;
}

export default function OntologyDetailPage() {
  const params = useParams();
  const id = params.id as string;

  const [ontology, setOntology] = useState<Ontology | null>(null);
  const [stats, setStats] = useState<any>(null);
  const [entities, setEntities] = useState<Entity[]>([]);
  const [classes, setClasses] = useState<ClassInfo[]>([]);
  const [properties, setProperties] = useState<PropertyInfo[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [expandedEntities, setExpandedEntities] = useState<Set<string>>(new Set());
  const [activeTab, setActiveTab] = useState("entities");
  
  // Edit mode states
  const [isEditing, setIsEditing] = useState(false);
  const [extractionMethod, setExtractionMethod] = useState("hybrid");
  const [editDescription, setEditDescription] = useState("");
  
  // Import states
  const [importFormat, setImportFormat] = useState("turtle");
  const [importData, setImportData] = useState("");
  const [isImporting, setIsImporting] = useState(false);

  useEffect(() => {
    loadOntologyData();
  }, [id]);

  async function loadOntologyData() {
    try {
      setLoading(true);
      setError(null);

      // Load ontology details
      const [ontologyRes, statsRes] = await Promise.all([
        getOntology(id),
        getOntologyStats(id).catch(() => null),
      ]);

      const ontData = ontologyRes.data?.ontology;
      setOntology(ontData);
      setStats(statsRes?.data?.stats);
      setEditDescription(ontData?.description || "");

      // Get graph URI for queries
      const graphUri = ontData?.tdb2_graph || `http://mimir-aip.io/ontology/${id}`;

      // Load all data in parallel
      await Promise.all([
        loadEntities(graphUri),
        loadClasses(graphUri),
        loadProperties(graphUri),
      ]);

    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load ontology");
      toast.error("Failed to load ontology data");
    } finally {
      setLoading(false);
    }
  }

  async function loadEntities(graphUri: string) {
    try {
      const query = `
        PREFIX rdfs: <http://www.w3.org/2000/01/rdf-schema#>
        SELECT DISTINCT ?entity ?type ?label
        WHERE {
          GRAPH <${graphUri}> {
            ?entity a ?type .
            OPTIONAL { ?entity rdfs:label ?label }
            FILTER(?type != <http://www.w3.org/2002/07/owl#Class>)
            FILTER(?type != <http://www.w3.org/2002/07/owl#ObjectProperty>)
            FILTER(?type != <http://www.w3.org/2002/07/owl#DatatypeProperty>)
            FILTER(?type != <http://www.w3.org/2002/07/owl#Ontology>)
          }
        }
        ORDER BY ?label
      `;
      const result = await executeSPARQLQuery(query);
      const entityData: Entity[] = (result.data?.bindings || []).map((binding: Record<string, unknown>) => ({
        entity: binding.entity as { value: string; type: string },
        type: binding.type as { value: string; type: string },
        label: binding.label as { value: string; type: string } | undefined,
      }));
      setEntities(entityData);
    } catch (err) {
      console.error("Failed to load entities:", err);
    }
  }

  async function loadClasses(graphUri: string) {
    try {
      const query = `
        PREFIX rdfs: <http://www.w3.org/2000/01/rdf-schema#>
        SELECT DISTINCT ?class ?label (COUNT(?instance) as ?instanceCount)
        WHERE {
          GRAPH <${graphUri}> {
            ?class a <http://www.w3.org/2002/07/owl#Class> .
            OPTIONAL { ?class rdfs:label ?label }
            OPTIONAL { ?instance a ?class }
          }
        }
        GROUP BY ?class ?label
        ORDER BY DESC(?instanceCount)
      `;
      const result = await executeSPARQLQuery(query);
      const classData: ClassInfo[] = (result.data?.bindings || []).map((binding: Record<string, unknown>) => ({
        class: binding.class as { value: string; type: string },
        label: binding.label as { value: string; type: string } | undefined,
        count: parseInt((binding.instanceCount as { value: string } | undefined)?.value || "0"),
      }));
      setClasses(classData);
    } catch (err) {
      console.error("Failed to load classes:", err);
    }
  }

  async function loadProperties(graphUri: string) {
    try {
      const query = `
        PREFIX rdfs: <http://www.w3.org/2000/01/rdf-schema#>
        SELECT DISTINCT ?property ?label (COUNT(?subject) as ?usageCount)
        WHERE {
          GRAPH <${graphUri}> {
            ?subject ?property ?object .
            OPTIONAL { ?property rdfs:label ?label }
            FILTER(?property != <http://www.w3.org/1999/02/22-rdf-syntax-ns#type>)
            FILTER(isIRI(?object))
          }
        }
        GROUP BY ?property ?label
        ORDER BY DESC(?usageCount)
        LIMIT 100
      `;
      const result = await executeSPARQLQuery(query);
      const propData: PropertyInfo[] = (result.data?.bindings || []).map((binding: Record<string, unknown>) => ({
        property: binding.property as { value: string; type: string },
        label: binding.label as { value: string; type: string } | undefined,
        count: parseInt((binding.usageCount as { value: string } | undefined)?.value || "0"),
      }));
      setProperties(propData);
    } catch (err) {
      console.error("Failed to load properties:", err);
    }
  }

  async function handleSaveEdit() {
    try {
      await updateOntology(id, {
        description: editDescription,
        metadata: JSON.stringify({
          extraction_method: extractionMethod,
        }),
      });
      toast.success("Ontology updated successfully");
      setIsEditing(false);
      loadOntologyData();
    } catch (err) {
      toast.error("Failed to update ontology");
    }
  }

  async function handleImport() {
    if (!importData.trim()) {
      toast.error("Please enter ontology data");
      return;
    }

    setIsImporting(true);
    try {
      await uploadOntology({
        name: ontology?.name || "Imported Ontology",
        description: ontology?.description || "",
        format: importFormat,
        version: ontology?.version || "1.0.0",
        ontology_data: importData,
      });
      toast.success("Ontology imported successfully");
      setImportData("");
      loadOntologyData();
    } catch (err) {
      toast.error("Failed to import ontology");
    } finally {
      setIsImporting(false);
    }
  }

  function toggleEntity(entityUri: string) {
    const newExpanded = new Set(expandedEntities);
    if (newExpanded.has(entityUri)) {
      newExpanded.delete(entityUri);
    } else {
      newExpanded.add(entityUri);
    }
    setExpandedEntities(newExpanded);
  }

  if (loading) {
    return (
      <div className="space-y-6">
        <div className="flex items-center gap-4">
          <div className="h-8 w-32 bg-blue/20 animate-pulse rounded" />
        </div>
        <Card className="bg-navy border-blue p-6">
          <div className="h-6 w-48 bg-blue/20 animate-pulse rounded mb-4" />
          <div className="h-4 w-full bg-blue/20 animate-pulse rounded" />
        </Card>
      </div>
    );
  }

  if (error) {
    return (
      <div className="space-y-6">
        <Link href="/ontologies" className="text-white/60 hover:text-orange flex items-center gap-2">
          <ArrowLeft className="h-4 w-4" />
          Back to Ontologies
        </Link>
        <Card className="bg-navy border-blue p-6">
          <h1 className="text-xl font-bold text-red-400 mb-2">Error</h1>
          <p className="text-white/60">{error}</p>
        </Card>
      </div>
    );
  }

  if (!ontology) {
    return (
      <div className="space-y-6">
        <Link href="/ontologies" className="text-white/60 hover:text-orange flex items-center gap-2">
          <ArrowLeft className="h-4 w-4" />
          Back to Ontologies
        </Link>
        <Card className="bg-navy border-blue p-6">
          <p className="text-white/60">Ontology not found</p>
        </Card>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <Link href="/ontologies" className="text-white/60 hover:text-orange flex items-center gap-2">
          <ArrowLeft className="h-4 w-4" />
          Back to Ontologies
        </Link>
        <div className="flex gap-2">
          <Dialog>
            <DialogTrigger asChild>
              <Button variant="outline" className="border-blue hover:border-orange">
                <Upload className="h-4 w-4 mr-2" />
                Import/Edit
              </Button>
            </DialogTrigger>
            <DialogContent className="bg-navy border-blue text-white max-w-2xl">
              <DialogHeader>
                <DialogTitle className="text-orange">Ontology Management</DialogTitle>
              </DialogHeader>
              
              <Tabs defaultValue="edit" className="mt-4">
                <TabsList className="bg-blue/20">
                  <TabsTrigger value="edit" className="data-[state=active]:bg-orange">Edit Settings</TabsTrigger>
                  <TabsTrigger value="import" className="data-[state=active]:bg-orange">Import Data</TabsTrigger>
                </TabsList>
                
                <TabsContent value="edit" className="space-y-4 mt-4">
                  <div>
                    <Label className="text-white/60">Description</Label>
                    <Textarea
                      value={editDescription}
                      onChange={(e) => setEditDescription(e.target.value)}
                      className="bg-blue/20 border-blue text-white mt-2"
                      rows={3}
                    />
                  </div>
                  
                  <div>
                    <Label className="text-white/60">Extraction Method</Label>
                    <Select value={extractionMethod} onValueChange={setExtractionMethod}>
                      <SelectTrigger className="bg-blue/20 border-blue text-white mt-2">
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="deterministic">Deterministic (Rule-based)</SelectItem>
                        <SelectItem value="llm">LLM-Driven (AI-powered)</SelectItem>
                        <SelectItem value="hybrid">Hybrid (Combined)</SelectItem>
                      </SelectContent>
                    </Select>
                  </div>
                  
                  <Button 
                    onClick={handleSaveEdit}
                    className="bg-orange hover:bg-orange/90 text-navy w-full"
                  >
                    <RefreshCw className="h-4 w-4 mr-2" />
                    Save Changes
                  </Button>
                </TabsContent>
                
                <TabsContent value="import" className="space-y-4 mt-4">
                  <div>
                    <Label className="text-white/60">Format</Label>
                    <Select value={importFormat} onValueChange={setImportFormat}>
                      <SelectTrigger className="bg-blue/20 border-blue text-white mt-2">
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="turtle">Turtle (.ttl)</SelectItem>
                        <SelectItem value="owl">OWL/XML (.owl)</SelectItem>
                        <SelectItem value="rdfxml">RDF/XML (.rdf)</SelectItem>
                        <SelectItem value="ntriples">N-Triples (.nt)</SelectItem>
                        <SelectItem value="jsonld">JSON-LD (.jsonld)</SelectItem>
                      </SelectContent>
                    </Select>
                  </div>
                  
                  <div>
                    <Label className="text-white/60">Ontology Data</Label>
                    <Textarea
                      value={importData}
                      onChange={(e) => setImportData(e.target.value)}
                      placeholder={`@prefix ex: <http://example.org/> .\nex:Entity1 a ex:Class ;\n  ex:property "value" .`}
                      className="bg-blue/20 border-blue text-white mt-2 font-mono text-sm"
                      rows={10}
                    />
                  </div>
                  
                  <Button 
                    onClick={handleImport}
                    disabled={isImporting}
                    className="bg-orange hover:bg-orange/90 text-navy w-full"
                  >
                    {isImporting ? (
                      <RefreshCw className="h-4 w-4 mr-2 animate-spin" />
                    ) : (
                      <Upload className="h-4 w-4 mr-2" />
                    )}
                    Import Ontology
                  </Button>
                </TabsContent>
              </Tabs>
            </DialogContent>
          </Dialog>
        </div>
      </div>

      {/* Basic Info */}
      <Card className="bg-navy border-blue p-6">
        <div className="flex items-start justify-between mb-4">
          <div>
            <h1 className="text-2xl font-bold text-orange">{ontology.name}</h1>
            <p className="text-white/60 mt-2">{ontology.description}</p>
          </div>
          <Badge variant={ontology.status === "active" ? "default" : "secondary"} className={ontology.status === "active" ? "bg-green-600" : ""}>
            {ontology.status}
          </Badge>
        </div>
        
        <div className="grid grid-cols-4 gap-4 pt-4 border-t border-blue/30">
          <div>
            <span className="text-white/40 text-sm">Version</span>
            <p className="text-white font-medium">{ontology.version}</p>
          </div>
          <div>
            <span className="text-white/40 text-sm">Format</span>
            <p className="text-white font-medium">{ontology.format}</p>
          </div>
          <div>
            <span className="text-white/40 text-sm">Created</span>
            <p className="text-white font-medium">{new Date(ontology.created_at).toLocaleDateString()}</p>
          </div>
          <div>
            <span className="text-white/40 text-sm">Entities</span>
            <p className="text-white font-medium">{stats?.total_entities ?? entities.length}</p>
          </div>
        </div>
      </Card>

      {/* Tabs for Entities, Classes, Properties */}
      <Tabs value={activeTab} onValueChange={setActiveTab} className="w-full">
        <TabsList className="bg-blue/20 w-full justify-start">
          <TabsTrigger value="entities" className="data-[state=active]:bg-orange flex items-center gap-2">
            <Box className="h-4 w-4" />
            Entities ({entities.length})
          </TabsTrigger>
          <TabsTrigger value="classes" className="data-[state=active]:bg-orange flex items-center gap-2">
            <Layers className="h-4 w-4" />
            Classes ({classes.length})
          </TabsTrigger>
          <TabsTrigger value="properties" className="data-[state=active]:bg-orange flex items-center gap-2">
            <Code className="h-4 w-4" />
            Properties ({properties.length})
          </TabsTrigger>
        </TabsList>

        <TabsContent value="entities" className="mt-4">
          <Card className="bg-navy border-blue">
            <div className="p-4 border-b border-blue/30 flex justify-between items-center">
              <div>
                <h2 className="text-lg font-semibold text-white">All Entities</h2>
                <p className="text-white/60 text-sm">{entities.length} entities in this ontology</p>
              </div>
              <Button variant="outline" size="sm" className="border-blue" onClick={() => loadOntologyData()}>
                <RefreshCw className="h-4 w-4 mr-2" />
                Refresh
              </Button>
            </div>
            <div className="divide-y divide-blue/30 max-h-[600px] overflow-y-auto">
              {entities.length === 0 ? (
                <p className="p-4 text-white/60">No entities found</p>
              ) : (
                entities.map((entity, index) => {
                  const entityUri = entity.entity?.value || `entity-${index}`;
                  const entityType = entity.type?.value || "Unknown";
                  const entityLabel = entity.label?.value || entityUri.split("/").pop() || entityUri;
                  const isExpanded = expandedEntities.has(entityUri);

                  return (
                    <div key={entityUri} className="p-4 hover:bg-blue/5">
                      <button
                        onClick={() => toggleEntity(entityUri)}
                        className="flex items-center justify-between w-full text-left"
                      >
                        <div className="flex items-center gap-3">
                          {isExpanded ? (
                            <ChevronDown className="h-4 w-4 text-orange" />
                          ) : (
                            <ChevronRight className="h-4 w-4 text-white/60" />
                          )}
                          <div>
                            <span className="text-white font-medium">{entityLabel}</span>
                            <p className="text-white/40 text-xs truncate max-w-md">{entityUri}</p>
                          </div>
                        </div>
                        <Badge variant="outline" className="text-xs">
                          {entityType.split("/").pop() || entityType}
                        </Badge>
                      </button>
                      {isExpanded && (
                        <div className="mt-3 ml-8 p-3 bg-blue/10 rounded text-sm space-y-2">
                          <p className="text-white/60">URI: <span className="text-blue font-mono text-xs">{entityUri}</span></p>
                          <p className="text-white/60">Type: <span className="text-blue">{entityType}</span></p>
                          {entity.label && (
                            <p className="text-white/60">Label: <span className="text-white">{entity.label.value}</span></p>
                          )}
                        </div>
                      )}
                    </div>
                  );
                })
              )}
            </div>
          </Card>
        </TabsContent>

        <TabsContent value="classes" className="mt-4">
          <Card className="bg-navy border-blue">
            <div className="p-4 border-b border-blue/30">
              <h2 className="text-lg font-semibold text-white">All Classes</h2>
              <p className="text-white/60 text-sm">{classes.length} classes defined in this ontology</p>
            </div>
            <div className="divide-y divide-blue/30">
              {classes.length === 0 ? (
                <p className="p-4 text-white/60">No classes found</p>
              ) : (
                classes.map((cls, index) => (
                  <div key={index} className="p-4 flex justify-between items-center hover:bg-blue/5">
                    <div>
                      <p className="text-white font-medium">
                        {cls.label?.value || cls.class?.value?.split("/").pop() || "Unknown Class"}
                      </p>
                      <p className="text-white/40 text-xs font-mono">{cls.class?.value}</p>
                    </div>
                    <Badge variant="secondary" className="bg-blue/30">
                      {cls.count || 0} instances
                    </Badge>
                  </div>
                ))
              )}
            </div>
          </Card>
        </TabsContent>

        <TabsContent value="properties" className="mt-4">
          <Card className="bg-navy border-blue">
            <div className="p-4 border-b border-blue/30">
              <h2 className="text-lg font-semibold text-white">All Properties</h2>
              <p className="text-white/60 text-sm">{properties.length} properties in this ontology</p>
            </div>
            <div className="divide-y divide-blue/30">
              {properties.length === 0 ? (
                <p className="p-4 text-white/60">No properties found</p>
              ) : (
                properties.map((prop, index) => (
                  <div key={index} className="p-4 flex justify-between items-center hover:bg-blue/5">
                    <div>
                      <p className="text-white font-medium">
                        {prop.label?.value || prop.property?.value?.split("/").pop()?.split("#").pop() || "Unknown Property"}
                      </p>
                      <p className="text-white/40 text-xs font-mono">{prop.property?.value}</p>
                    </div>
                    <Badge variant="secondary" className="bg-blue/30">
                      {prop.count || 0} uses
                    </Badge>
                  </div>
                ))
              )}
            </div>
          </Card>
        </TabsContent>
      </Tabs>

      {/* Advanced Details */}
      <Card className="bg-navy border-blue p-6">
        <h3 className="text-lg font-semibold text-white mb-4 flex items-center gap-2">
          <Info className="h-5 w-5 text-orange" />
          Technical Details
        </h3>
        <div className="space-y-3 text-sm">
          <div className="grid grid-cols-3 gap-4">
            <span className="text-white/60">Ontology ID:</span>
            <span className="text-white col-span-2 font-mono">{ontology.id}</span>
          </div>
          <div className="grid grid-cols-3 gap-4">
            <span className="text-white/60">TDB2 Graph:</span>
            <span className="text-white col-span-2 font-mono break-all">{ontology.tdb2_graph || "N/A"}</span>
          </div>
          <div className="grid grid-cols-3 gap-4">
            <span className="text-white/60">File Path:</span>
            <span className="text-white col-span-2 font-mono">{ontology.file_path || "N/A"}</span>
          </div>
          <div className="grid grid-cols-3 gap-4">
            <span className="text-white/60">Created By:</span>
            <span className="text-white col-span-2">{ontology.created_by || "Unknown"}</span>
          </div>
          {stats && (
            <>
              <div className="grid grid-cols-3 gap-4 pt-2 border-t border-blue/30">
                <span className="text-white/60">Entity Count:</span>
                <span className="text-white col-span-2">{stats.total_entities ?? stats.entity_count ?? "N/A"}</span>
              </div>
              <div className="grid grid-cols-3 gap-4">
                <span className="text-white/60">Class Count:</span>
                <span className="text-white col-span-2">{stats.total_classes ?? stats.class_count ?? "N/A"}</span>
              </div>
              <div className="grid grid-cols-3 gap-4">
                <span className="text-white/60">Property Count:</span>
                <span className="text-white col-span-2">{stats.total_properties ?? stats.property_count ?? "N/A"}</span>
              </div>
            </>
          )}
        </div>
      </Card>

      {/* Actions */}
      <div className="flex justify-end gap-4">
        <Link href={`/chat?ontology=${id}`}>
          <Button className="bg-orange hover:bg-orange/90 text-navy">
            <Database className="h-4 w-4 mr-2" />
            Chat About This Ontology
          </Button>
        </Link>
      </div>
    </div>
  );
}
