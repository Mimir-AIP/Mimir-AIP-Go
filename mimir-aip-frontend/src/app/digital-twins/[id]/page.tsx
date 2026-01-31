"use client";

import { useState, useEffect } from "react";
import { useParams } from "next/navigation";
import Link from "next/link";
import { getDigitalTwin, getTwinState, listScenarios, type DigitalTwin, type SimulationScenario } from "@/lib/api";
import { Card } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { ArrowLeft, Network, Database, Calendar, Activity, ChevronDown, ChevronRight, Zap } from "lucide-react";

export default function DigitalTwinDetailPage() {
  const params = useParams();
  const id = params.id as string;

  const [twin, setTwin] = useState<DigitalTwin | null>(null);
  const [twinState, setTwinState] = useState<any>(null);
  const [scenarios, setScenarios] = useState<SimulationScenario[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [expandedEntities, setExpandedEntities] = useState<Set<string>>(new Set());

  useEffect(() => {
    loadTwin();
  }, [id]);

  async function loadTwin() {
    try {
      setLoading(true);
      setError(null);

      const [twinData, stateData, scenariosData] = await Promise.all([
        getDigitalTwin(id),
        getTwinState(id).catch(() => null),
        listScenarios(id).catch(() => []),
      ]);

      setTwin(twinData);
      setTwinState(stateData);
      setScenarios(scenariosData);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load digital twin");
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

  function getStatusColor(status: string) {
    switch (status?.toLowerCase()) {
      case "active":
      case "running":
        return "bg-green-500/20 text-green-400 border-green-500";
      case "inactive":
      case "stopped":
        return "bg-red-500/20 text-red-400 border-red-500";
      case "warning":
        return "bg-yellow-500/20 text-yellow-400 border-yellow-500";
      default:
        return "bg-blue/20 text-blue border-blue";
    }
  }

  if (loading) {
    return (
      <div className="space-y-6">
        <div className="flex items-center gap-4">
          <Link href="/digital-twins" className="text-white/60 hover:text-orange">
            <ArrowLeft className="h-5 w-5" />
          </Link>
          <h1 className="text-2xl font-bold text-orange">Loading...</h1>
        </div>
        <Card className="bg-navy border-blue p-6 animate-pulse h-96"></Card>
      </div>
    );
  }

  if (error || !twin) {
    return (
      <div className="space-y-6">
        <div className="flex items-center gap-4">
          <Link href="/digital-twins" className="text-white/60 hover:text-orange">
            <ArrowLeft className="h-5 w-5" />
          </Link>
          <h1 className="text-2xl font-bold text-orange">Error</h1>
        </div>
        <Card className="bg-red-900/20 border-red-500 text-red-400 p-6">
          <p>{error || "Digital twin not found"}</p>
        </Card>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center gap-4">
        <Link href="/digital-twins" className="text-white/60 hover:text-orange transition-colors">
          <ArrowLeft className="h-5 w-5" />
        </Link>
        <div className="flex-1">
          <h1 className="text-2xl font-bold text-orange">{twin.name}</h1>
          <p className="text-white/60 text-sm">{twin.description || "No description"}</p>
        </div>
        <Badge className={getStatusColor(twinState?.state?.status || "unknown")}>
          {twinState?.state?.status || "Unknown"}
        </Badge>
      </div>

      {/* Basic Info Cards */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        <Card className="bg-navy border-blue p-4">
          <div className="flex items-center gap-3">
            <Network className="h-5 w-5 text-blue" />
            <div>
              <p className="text-white/60 text-xs">Model Type</p>
              <p className="text-white font-semibold capitalize">{twin.model_type}</p>
            </div>
          </div>
        </Card>
        <Card className="bg-navy border-blue p-4">
          <div className="flex items-center gap-3">
            <Database className="h-5 w-5 text-blue" />
            <div>
              <p className="text-white/60 text-xs">Entities</p>
              <p className="text-white font-semibold">{twin.entity_count || twin.entities?.length || 0}</p>
            </div>
          </div>
        </Card>
        <Card className="bg-navy border-blue p-4">
          <div className="flex items-center gap-3">
            <Activity className="h-5 w-5 text-blue" />
            <div>
              <p className="text-white/60 text-xs">Relationships</p>
              <p className="text-white font-semibold">{twin.relationship_count || twin.relationships?.length || 0}</p>
            </div>
          </div>
        </Card>
        <Card className="bg-navy border-blue p-4">
          <div className="flex items-center gap-3">
            <Calendar className="h-5 w-5 text-blue" />
            <div>
              <p className="text-white/60 text-xs">Created</p>
              <p className="text-white font-semibold">{formatDate(twin.created_at)}</p>
            </div>
          </div>
        </Card>
      </div>

      {/* Entities List */}
      {twin.entities && twin.entities.length > 0 && (
        <Card className="bg-navy border-blue">
          <div className="p-4 border-b border-blue/30">
            <h2 className="text-lg font-semibold text-white">Entities ({twin.entities.length})</h2>
            <p className="text-white/60 text-sm">Click to view entity state and details</p>
          </div>
          <div className="divide-y divide-blue/30 max-h-96 overflow-y-auto">
            {twin.entities.map((entity) => {
              const isExpanded = expandedEntities.has(entity.uri);
              const entityState = twinState?.entity_states?.[entity.uri];

              return (
                <div key={entity.uri} className="p-4">
                  <button
                    onClick={() => toggleEntity(entity.uri)}
                    className="flex items-center justify-between w-full text-left hover:bg-blue/10 p-2 rounded transition-colors"
                  >
                    <div className="flex items-center gap-3">
                      {isExpanded ? (
                        <ChevronDown className="h-4 w-4 text-orange" />
                      ) : (
                        <ChevronRight className="h-4 w-4 text-white/60" />
                      )}
                      <span className="text-white font-medium">{entity.label}</span>
                    </div>
                    <div className="flex items-center gap-2">
                      {entityState && (
                        <Badge className={getStatusColor(entityState.status)}>
                          {entityState.status}
                        </Badge>
                      )}
                      <Badge variant="outline" className="text-xs">
                        {entity.type.split("/").pop() || entity.type}
                      </Badge>
                    </div>
                  </button>
                  {isExpanded && (
                    <div className="mt-2 ml-8 p-3 bg-blue/10 rounded text-sm space-y-2">
                      <p className="text-white/60">URI: <span className="text-blue">{entity.uri}</span></p>
                      <p className="text-white/60">Type: <span className="text-blue">{entity.type}</span></p>
                      {entityState && (
                        <>
                          <div className="grid grid-cols-3 gap-2 mt-2 pt-2 border-t border-blue/30">
                            <div>
                              <p className="text-white/40 text-xs">Utilization</p>
                              <p className="text-white">{(entityState.utilization * 100).toFixed(1)}%</p>
                            </div>
                            <div>
                              <p className="text-white/40 text-xs">Capacity</p>
                              <p className="text-white">{entityState.capacity}</p>
                            </div>
                            <div>
                              <p className="text-white/40 text-xs">Available</p>
                              <p className="text-white">{entityState.available ? "Yes" : "No"}</p>
                            </div>
                          </div>
                        </>
                      )}
                    </div>
                  )}
                </div>
              );
            })}
          </div>
        </Card>
      )}

      {/* Scenarios */}
      {scenarios.length > 0 && (
        <Card className="bg-navy border-blue">
          <div className="p-4 border-b border-blue/30">
            <h2 className="text-lg font-semibold text-white">Scenarios ({scenarios.length})</h2>
          </div>
          <div className="divide-y divide-blue/30">
            {scenarios.map((scenario) => (
              <div key={scenario.id} className="p-4 flex items-center justify-between">
                <div>
                  <p className="text-white font-medium">{scenario.name}</p>
                  <p className="text-white/60 text-sm">{scenario.description || "No description"}</p>
                  <p className="text-white/40 text-xs mt-1">{scenario.events?.length || 0} events • {scenario.duration} steps</p>
                </div>
                <Badge variant="outline">{scenario.scenario_type || "Standard"}</Badge>
              </div>
            ))}
          </div>
        </Card>
      )}

      {/* Twin ID Info */}
      <Card className="bg-navy border-blue p-4">
        <h3 className="text-sm font-semibold text-white/60 mb-2">Twin ID</h3>
        <p className="text-white font-mono text-sm">{twin.id}</p>
        <p className="text-white/40 text-xs mt-2">Based on ontology: {twin.ontology_id}</p>
      </Card>

      {/* Actions */}
      <div className="grid grid-cols-2 gap-4">
        <Link
          href={`/chat?twin_id=${id}`}
          className="py-3 bg-orange hover:bg-orange/80 text-white rounded text-center transition-colors flex items-center justify-center gap-2"
        >
          <Zap className="h-4 w-4" />
          Chat with Twin
        </Link>
        <Link
          href="/pipelines"
          className="py-3 bg-blue hover:bg-blue/80 text-white rounded text-center transition-colors"
        >
          Run Pipeline
        </Link>
      </div>
    </div>
  );
}
