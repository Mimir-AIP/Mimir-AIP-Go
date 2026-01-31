"use client";

import { useState, useEffect } from "react";
import { useParams } from "next/navigation";
import Link from "next/link";
import { getOntology, getOntologyStats, type Ontology, executeSPARQLQuery } from "@/lib/api";
import { Card } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { ArrowLeft, Database, FileText, Calendar, Info, ChevronDown, ChevronRight } from "lucide-react";

export default function OntologyDetailPage() {
  const params = useParams();
  const id = params.id as string;

  const [ontology, setOntology] = useState<Ontology | null>(null);
  const [stats, setStats] = useState<any>(null);
  const [entities, setEntities] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [showDetails, setShowDetails] = useState(false);
  const [expandedEntities, setExpandedEntities] = useState<Set<string>>(new Set());

  useEffect(() => {
    loadOntology();
  }, [id]);

  async function loadOntology() {
    try {
      setLoading(true);
      setError(null);

      const [ontologyRes, statsRes] = await Promise.all([
        getOntology(id),
        getOntologyStats(id).catch(() => null),
      ]);

      setOntology(ontologyRes.data?.ontology);
      setStats(statsRes?.data?.stats);

      // Query entities
      const entityQuery = `
        PREFIX rdfs: <http://www.w3.org/2000/01/rdf-schema#>
        SELECT ?entity ?type ?label
        WHERE {
          ?entity a ?type .
          OPTIONAL { ?entity rdfs:label ?label }
        }
        LIMIT 50
      `;
      const entityRes = await executeSPARQLQuery(entityQuery);
      setEntities(entityRes.data?.bindings || []);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load ontology");
    } finally {
      setLoading(false);
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

  function formatDate(dateString: string): string {
    return new Date(dateString).toLocaleDateString("en-US", {
      year: "numeric",
      month: "short",
      day: "numeric",
    });
  }

  if (loading) {
    return (
      <div className="space-y-6">
        <div className="flex items-center gap-4">
          <Link href="/ontologies" className="text-white/60 hover:text-orange">
            <ArrowLeft className="h-5 w-5" />
          </Link>
          <h1 className="text-2xl font-bold text-orange">Loading...</h1>
        </div>
        <Card className="bg-navy border-blue p-6 animate-pulse h-96"></Card>
      </div>
    );
  }

  if (error || !ontology) {
    return (
      <div className="space-y-6">
        <div className="flex items-center gap-4">
          <Link href="/ontologies" className="text-white/60 hover:text-orange">
            <ArrowLeft className="h-5 w-5" />
          </Link>
          <h1 className="text-2xl font-bold text-orange">Error</h1>
        </div>
        <Card className="bg-red-900/20 border-red-500 text-red-400 p-6">
          <p>{error || "Ontology not found"}</p>
        </Card>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center gap-4">
        <Link href="/ontologies" className="text-white/60 hover:text-orange transition-colors">
          <ArrowLeft className="h-5 w-5" />
        </Link>
        <div className="flex-1">
          <h1 className="text-2xl font-bold text-orange">{ontology.name}</h1>
          <p className="text-white/60 text-sm">{ontology.description || "No description"}</p>
        </div>
        <Badge className={
          ontology.status === "active" ? "bg-green-500/20 text-green-400 border-green-500" :
          ontology.status === "deprecated" ? "bg-red-500/20 text-red-400 border-red-500" :
          "bg-yellow-500/20 text-yellow-400 border-yellow-500"
        }>
          {ontology.status}
        </Badge>
      </div>

      {/* Basic Info Cards */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        <Card className="bg-navy border-blue p-4">
          <div className="flex items-center gap-3">
            <Database className="h-5 w-5 text-blue" />
            <div>
              <p className="text-white/60 text-xs">Version</p>
              <p className="text-white font-semibold">{ontology.version}</p>
            </div>
          </div>
        </Card>
        <Card className="bg-navy border-blue p-4">
          <div className="flex items-center gap-3">
            <FileText className="h-5 w-5 text-blue" />
            <div>
              <p className="text-white/60 text-xs">Format</p>
              <p className="text-white font-semibold capitalize">{ontology.format}</p>
            </div>
          </div>
        </Card>
        <Card className="bg-navy border-blue p-4">
          <div className="flex items-center gap-3">
            <Calendar className="h-5 w-5 text-blue" />
            <div>
              <p className="text-white/60 text-xs">Created</p>
              <p className="text-white font-semibold">{formatDate(ontology.created_at)}</p>
            </div>
          </div>
        </Card>
      </div>

      {/* Entities List */}
      <Card className="bg-navy border-blue">
        <div className="p-4 border-b border-blue/30">
          <h2 className="text-lg font-semibold text-white">Entities ({entities.length})</h2>
          <p className="text-white/60 text-sm">Click to view entity details</p>
        </div>
        <div className="divide-y divide-blue/30">
          {entities.length === 0 ? (
            <p className="p-4 text-white/60">No entities found in this ontology</p>
          ) : (
            entities.map((entity, index) => {
              const entityUri = entity.entity?.value || `entity-${index}`;
              const entityType = entity.type?.value || "Unknown";
              const entityLabel = entity.label?.value || entityUri.split("/").pop() || entityUri;
              const isExpanded = expandedEntities.has(entityUri);

              return (
                <div key={entityUri} className="p-4">
                  <button
                    onClick={() => toggleEntity(entityUri)}
                    className="flex items-center justify-between w-full text-left hover:bg-blue/10 p-2 rounded transition-colors"
                  >
                    <div className="flex items-center gap-3">
                      {isExpanded ? (
                        <ChevronDown className="h-4 w-4 text-orange" />
                      ) : (
                        <ChevronRight className="h-4 w-4 text-white/60" />
                      )}
                      <span className="text-white font-medium">{entityLabel}</span>
                    </div>
                    <Badge variant="outline" className="text-xs">
                      {entityType.split("/").pop() || entityType}
                    </Badge>
                  </button>
                  {isExpanded && (
                    <div className="mt-2 ml-8 p-3 bg-blue/10 rounded text-sm">
                      <p className="text-white/60">URI: <span className="text-blue">{entityUri}</span></p>
                      <p className="text-white/60 mt-1">Type: <span className="text-blue">{entityType}</span></p>
                    </div>
                  )}
                </div>
              );
            })
          )}
        </div>
      </Card>

      {/* Read More Button */}
      <button
        onClick={() => setShowDetails(!showDetails)}
        className="w-full py-3 bg-blue/20 hover:bg-blue/30 border border-blue text-blue hover:text-white rounded transition-colors flex items-center justify-center gap-2"
      >
        <Info className="h-4 w-4" />
        {showDetails ? "Hide Advanced Details" : "Read More - Advanced Details"}
      </button>

      {/* Advanced Details */}
      {showDetails && (
        <Card className="bg-navy border-blue p-6">
          <h3 className="text-lg font-semibold text-white mb-4">Advanced Details</h3>
          <div className="space-y-3 text-sm">
            <div className="grid grid-cols-3 gap-4">
              <span className="text-white/60">Ontology ID:</span>
              <span className="text-white col-span-2 font-mono">{ontology.id}</span>
            </div>
            <div className="grid grid-cols-3 gap-4">
              <span className="text-white/60">TDB2 Graph:</span>
              <span className="text-white col-span-2 font-mono">{ontology.tdb2_graph}</span>
            </div>
            <div className="grid grid-cols-3 gap-4">
              <span className="text-white/60">File Path:</span>
              <span className="text-white col-span-2 font-mono">{ontology.file_path}</span>
            </div>
            <div className="grid grid-cols-3 gap-4">
              <span className="text-white/60">Created By:</span>
              <span className="text-white col-span-2">{ontology.created_by || "Unknown"}</span>
            </div>
            {stats && (
              <>
                <div className="grid grid-cols-3 gap-4 pt-2 border-t border-blue/30">
                  <span className="text-white/60">Entity Count:</span>
                  <span className="text-white col-span-2">{stats.entity_count || "N/A"}</span>
                </div>
                <div className="grid grid-cols-3 gap-4">
                  <span className="text-white/60">Class Count:</span>
                  <span className="text-white col-span-2">{stats.class_count || "N/A"}</span>
                </div>
                <div className="grid grid-cols-3 gap-4">
                  <span className="text-white/60">Property Count:</span>
                  <span className="text-white col-span-2">{stats.property_count || "N/A"}</span>
                </div>
              </>
            )}
          </div>
        </Card>
      )}

      {/* Actions */}
      <div className="flex gap-4">
        <Link
          href={`/chat?twin_id=${id}`}
          className="flex-1 py-3 bg-orange hover:bg-orange/80 text-white rounded text-center transition-colors"
        >
          Chat About This Ontology
        </Link>
      </div>
    </div>
  );
}
