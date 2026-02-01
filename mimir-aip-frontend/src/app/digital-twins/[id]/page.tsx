"use client";

import { useState, useEffect } from "react";
import { useParams } from "next/navigation";
import Link from "next/link";
import { getDigitalTwin, getTwinState, listScenarios, runSimulation, createScenario, type DigitalTwin, type SimulationScenario } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from "@/components/ui/dialog";
import { Plus, Wand2 } from "lucide-react";
import { Card } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { ArrowLeft, Network, Database, Calendar, Activity, ChevronDown, ChevronRight, Zap, Play } from "lucide-react";

export default function DigitalTwinDetailPage() {
  const params = useParams();
  const id = params.id as string;

  const [twin, setTwin] = useState<DigitalTwin | null>(null);
  const [twinState, setTwinState] = useState<any>(null);
  const [scenarios, setScenarios] = useState<SimulationScenario[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [expandedEntities, setExpandedEntities] = useState<Set<string>>(new Set());
  
  // Scenario execution state
  const [runningScenario, setRunningScenario] = useState<string | null>(null);
  const [simulationResults, setSimulationResults] = useState<Record<string, any>>({});
  const [simulationError, setSimulationError] = useState<string | null>(null);
  
  // Scenario creation state
  const [createDialogOpen, setCreateDialogOpen] = useState(false);
  const [creatingScenario, setCreatingScenario] = useState(false);
  const [newScenario, setNewScenario] = useState({
    name: "",
    description: "",
    scenario_type: "custom",
    duration: 10,
  });

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

  async function handleRunScenario(scenarioId: string) {
    try {
      setRunningScenario(scenarioId);
      setSimulationError(null);
      const result = await runSimulation(id, scenarioId);
      setSimulationResults(prev => ({ ...prev, [scenarioId]: result }));
    } catch (err) {
      setSimulationError(err instanceof Error ? err.message : "Simulation failed");
    } finally {
      setRunningScenario(null);
    }
  }

  async function handleCreateScenario() {
    try {
      setCreatingScenario(true);
      await createScenario(id, {
        name: newScenario.name,
        description: newScenario.description,
        scenario_type: newScenario.scenario_type,
        duration: newScenario.duration,
        events: []
      });
      // Reload scenarios
      const scenariosData = await listScenarios(id);
      setScenarios(scenariosData);
      setCreateDialogOpen(false);
      // Reset form
      setNewScenario({
        name: "",
        description: "",
        scenario_type: "custom",
        duration: 10,
      });
    } catch (err) {
      setSimulationError(err instanceof Error ? err.message : "Failed to create scenario");
    } finally {
      setCreatingScenario(false);
    }
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

      {/* Scenarios with Run Controls */}
      <Card className="bg-navy border-blue">
        <div className="p-4 border-b border-blue/30 flex items-center justify-between">
          <h2 className="text-lg font-semibold text-white">Scenarios ({scenarios.length})</h2>
          <Button
            onClick={() => setCreateDialogOpen(true)}
            variant="outline"
            className="text-xs border-orange text-orange hover:bg-orange/20"
          >
            <Plus className="h-3 w-3 mr-1" />
            Create Custom Scenario
          </Button>
        </div>
          
          {simulationError && (
            <div className="p-3 bg-red-500/20 border-b border-red-500 text-red-400 text-sm">
              {simulationError}
            </div>
          )}
          
          <div className="divide-y divide-blue/30">
            {scenarios.map((scenario) => (
              <div key={scenario.id} className="p-4">
                <div className="flex items-center justify-between mb-3">
                  <div className="flex-1">
                    <p className="text-white font-medium">{scenario.name}</p>
                    <p className="text-white/60 text-sm">{scenario.description || "No description"}</p>
                    <p className="text-white/40 text-xs mt-1">{scenario.events?.length || 0} events • {scenario.duration} steps</p>
                  </div>
                  <div className="flex items-center gap-2">
                    <Badge variant="outline">{scenario.scenario_type || "Standard"}</Badge>
                    <button
                      onClick={() => handleRunScenario(scenario.id)}
                      disabled={runningScenario === scenario.id}
                      className="px-3 py-1 bg-orange hover:bg-orange/80 disabled:bg-orange/50 text-white text-sm rounded transition-colors flex items-center gap-1"
                    >
                      {runningScenario === scenario.id ? (
                        <>
                          <Activity className="h-3 w-3 animate-spin" />
                          Running...
                        </>
                      ) : (
                        <>
                          <Play className="h-3 w-3" />
                          Run
                        </>
                      )}
                    </button>
                  </div>
                </div>
                
                {/* Simulation Results */}
                {simulationResults[scenario.id] && (
                  <div className="mt-3 p-3 bg-green-500/10 border border-green-500/30 rounded">
                    <h4 className="text-sm font-semibold text-green-400 mb-2">Simulation Results</h4>
                    <div className="grid grid-cols-3 gap-3 text-sm">
                      <div>
                        <span className="text-white/40">Status:</span>
                        <Badge className="ml-2 bg-green-500/20 text-green-400">
                          {simulationResults[scenario.id].status}
                        </Badge>
                      </div>
                      <div>
                        <span className="text-white/40">Run ID:</span>
                        <span className="text-white ml-2 font-mono text-xs">
                          {simulationResults[scenario.id].run_id?.slice(0, 8)}...
                        </span>
                      </div>
                      <div>
                        <span className="text-white/40">Final State:</span>
                        <span className="text-orange ml-2">
                          {simulationResults[scenario.id].metrics?.final_state || "N/A"}
                        </span>
                      </div>
                    </div>
                  </div>
                )}
              </div>
            ))}
          </div>
        </Card>

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

      {/* Create Scenario Dialog */}
      <Dialog open={createDialogOpen} onOpenChange={setCreateDialogOpen}>
        <DialogContent className="bg-navy border-blue text-white max-w-lg">
          <DialogHeader>
            <DialogTitle className="text-orange flex items-center gap-2">
              <Wand2 className="h-5 w-5" />
              Create Custom Scenario
            </DialogTitle>
          </DialogHeader>
          
          <div className="space-y-4 py-4">
            <div>
              <label className="text-sm text-white/60 mb-1 block">Scenario Name</label>
              <Input
                value={newScenario.name}
                onChange={(e) => setNewScenario(prev => ({ ...prev, name: e.target.value }))}
                placeholder="e.g., Stress Test, Holiday Rush..."
                className="bg-blue/20 border-blue text-white"
              />
            </div>
            
            <div>
              <label className="text-sm text-white/60 mb-1 block">Description</label>
              <Textarea
                value={newScenario.description}
                onChange={(e) => setNewScenario(prev => ({ ...prev, description: e.target.value }))}
                placeholder="What does this scenario simulate?"
                className="bg-blue/20 border-blue text-white"
                rows={3}
              />
            </div>
            
            <div>
              <label className="text-sm text-white/60 mb-1 block">Duration (steps)</label>
              <Input
                type="number"
                value={newScenario.duration}
                onChange={(e) => setNewScenario(prev => ({ ...prev, duration: parseInt(e.target.value) || 10 }))}
                className="bg-blue/20 border-blue text-white w-32"
              />
            </div>
            
            <div className="bg-blue/10 rounded p-3 text-sm">
              <p className="text-white/60">
                <strong className="text-white">Note:</strong> This creates a basic scenario. 
                For complex scenarios with multiple events, use the chat interface or API.
              </p>
            </div>
          </div>
          
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => setCreateDialogOpen(false)}
              className="border-blue text-white hover:bg-blue/20"
            >
              Cancel
            </Button>
            <Button
              onClick={handleCreateScenario}
              disabled={creatingScenario || !newScenario.name}
              className="bg-orange hover:bg-orange/80 text-white"
            >
              {creatingScenario ? "Creating..." : "Create Scenario"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
