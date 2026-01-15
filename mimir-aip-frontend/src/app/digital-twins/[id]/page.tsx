"use client";

import { useEffect, useState } from "react";
import { useParams, useRouter } from "next/navigation";
import Link from "next/link";
import {
  getDigitalTwin,
  listScenarios,
  runSimulation,
  type DigitalTwin,
  type SimulationScenario,
  type SimulationRun,
} from "@/lib/api";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { LoadingSkeleton } from "@/components/LoadingSkeleton";
import { AgentChat } from "@/components/chat/AgentChat";
import { WhatIfAnalyzer } from "@/components/digital-twin/WhatIfAnalyzer";
import { SmartScenarios } from "@/components/digital-twin/SmartScenarios";
import { InsightsPanel } from "@/components/digital-twin/InsightsPanel";
import { toast } from "sonner";
import {
  ArrowLeft,
  Network,
  Database,
  Calendar,
  Play,
  Plus,
  MessageSquare,
  BarChart3,
  Settings,
  Sparkles,
} from "lucide-react";

export default function TwinDetailPage() {
  const params = useParams();
  const router = useRouter();
  const twinId = params.id as string;

  const [twin, setTwin] = useState<DigitalTwin | null>(null);
  const [scenarios, setScenarios] = useState<SimulationScenario[]>([]);
  const [loading, setLoading] = useState(true);
  const [activeTab, setActiveTab] = useState<"overview" | "insights" | "whatif" | "scenarios" | "smart" | "chat">("overview");

  useEffect(() => {
    loadData();
  }, [twinId]);

  async function loadData() {
    try {
      setLoading(true);
      const [twinData, scenariosData] = await Promise.all([
        getDigitalTwin(twinId),
        listScenarios(twinId),
      ]);
      setTwin(twinData);
      setScenarios(scenariosData);
    } catch (err) {
      const message = err instanceof Error ? err.message : "Failed to load twin";
      toast.error(message);
    } finally {
      setLoading(false);
    }
  }

  async function handleRunScenario(scenarioId: string) {
    try {
      toast.info("Starting simulation...");
      const run = await runSimulation(twinId, scenarioId, { snapshot_interval: 5 });
      toast.success("Simulation completed!");
      router.push(`/digital-twins/${twinId}/runs/${run.run_id}`);
    } catch (err) {
      const message = err instanceof Error ? err.message : "Failed to run simulation";
      toast.error(message);
    }
  }

  function formatDate(dateString: string): string {
    return new Date(dateString).toLocaleDateString("en-US", {
      year: "numeric",
      month: "short",
      day: "numeric",
      hour: "2-digit",
      minute: "2-digit",
    });
  }

  if (loading) {
    return (
      <div className="container mx-auto py-8">
        <LoadingSkeleton />
      </div>
    );
  }

  if (!twin) {
    return (
      <div className="container mx-auto py-8">
        <Card>
          <CardContent className="pt-6">
            <p className="text-red-600">Digital twin not found</p>
            <Link href="/digital-twins">
              <Button className="mt-4">Back to Digital Twins</Button>
            </Link>
          </CardContent>
        </Card>
      </div>
    );
  }

  return (
    <div className="container mx-auto py-8">
      {/* Header */}
      <div className="mb-6">
        <Link href="/digital-twins">
          <Button variant="ghost" className="mb-4">
            <ArrowLeft className="h-4 w-4 mr-2" />
            Back to Digital Twins
          </Button>
        </Link>

        <div className="flex items-start justify-between">
          <div>
            <h1 className="text-3xl font-bold flex items-center gap-3">
              <Network className="h-8 w-8" />
              {twin.name}
            </h1>
            <p className="text-muted-foreground mt-2">{twin.description || "No description"}</p>
          </div>
          <Badge variant="secondary" className="text-lg px-4 py-2">
            {twin.model_type}
          </Badge>
        </div>

        {/* Stats */}
        <div className="grid grid-cols-3 gap-4 mt-6">
          <Card>
            <CardContent className="pt-6">
              <div className="text-center">
                <p className="text-3xl font-bold">{twin.entities?.length || 0}</p>
                <p className="text-sm text-muted-foreground">Entities</p>
              </div>
            </CardContent>
          </Card>
          <Card>
            <CardContent className="pt-6">
              <div className="text-center">
                <p className="text-3xl font-bold">{twin.relationships?.length || 0}</p>
                <p className="text-sm text-muted-foreground">Relationships</p>
              </div>
            </CardContent>
          </Card>
          <Card>
            <CardContent className="pt-6">
              <div className="text-center">
                <p className="text-3xl font-bold">{scenarios.length}</p>
                <p className="text-sm text-muted-foreground">Scenarios</p>
              </div>
            </CardContent>
          </Card>
        </div>
      </div>

      {/* Tabs */}
      <div className="border-b mb-6">
        <div className="flex gap-4 overflow-x-auto">
          <button
            onClick={() => setActiveTab("overview")}
            className={`pb-3 px-1 border-b-2 transition-colors whitespace-nowrap ${
              activeTab === "overview"
                ? "border-primary text-primary font-medium"
                : "border-transparent text-muted-foreground hover:text-foreground"
            }`}
          >
            <div className="flex items-center gap-2">
              <BarChart3 className="h-4 w-4" />
              Overview
            </div>
          </button>
          <button
            onClick={() => setActiveTab("insights")}
            className={`pb-3 px-1 border-b-2 transition-colors whitespace-nowrap ${
              activeTab === "insights"
                ? "border-primary text-primary font-medium"
                : "border-transparent text-muted-foreground hover:text-foreground"
            }`}
          >
            <div className="flex items-center gap-2">
              <Sparkles className="h-4 w-4" />
              Insights
            </div>
          </button>
          <button
            onClick={() => setActiveTab("whatif")}
            className={`pb-3 px-1 border-b-2 transition-colors whitespace-nowrap ${
              activeTab === "whatif"
                ? "border-primary text-primary font-medium"
                : "border-transparent text-muted-foreground hover:text-foreground"
            }`}
          >
            <div className="flex items-center gap-2">
              <Sparkles className="h-4 w-4" />
              What-If Analysis
            </div>
          </button>
          <button
            onClick={() => setActiveTab("smart")}
            className={`pb-3 px-1 border-b-2 transition-colors whitespace-nowrap ${
              activeTab === "smart"
                ? "border-primary text-primary font-medium"
                : "border-transparent text-muted-foreground hover:text-foreground"
            }`}
          >
            <div className="flex items-center gap-2">
              <Sparkles className="h-4 w-4" />
              Smart Scenarios
            </div>
          </button>
          <button
            onClick={() => setActiveTab("scenarios")}
            className={`pb-3 px-1 border-b-2 transition-colors whitespace-nowrap ${
              activeTab === "scenarios"
                ? "border-primary text-primary font-medium"
                : "border-transparent text-muted-foreground hover:text-foreground"
            }`}
          >
            <div className="flex items-center gap-2">
              <Settings className="h-4 w-4" />
              Scenarios ({scenarios.length})
            </div>
          </button>
          <button
            onClick={() => setActiveTab("chat")}
            className={`pb-3 px-1 border-b-2 transition-colors whitespace-nowrap ${
              activeTab === "chat"
                ? "border-primary text-primary font-medium"
                : "border-transparent text-muted-foreground hover:text-foreground"
            }`}
          >
            <div className="flex items-center gap-2">
              <MessageSquare className="h-4 w-4" />
              Agent Chat
            </div>
          </button>
        </div>
      </div>

      {/* Tab Content */}
      {activeTab === "overview" && (
        <div className="space-y-6">
          <Card>
            <CardHeader>
              <CardTitle>Twin Information</CardTitle>
            </CardHeader>
            <CardContent>
              <dl className="grid grid-cols-2 gap-4">
                <div>
                  <dt className="text-sm font-medium text-muted-foreground">Ontology ID</dt>
                  <dd className="mt-1 text-sm font-mono">{twin.ontology_id}</dd>
                </div>
                <div>
                  <dt className="text-sm font-medium text-muted-foreground">Created</dt>
                  <dd className="mt-1 text-sm">{formatDate(twin.created_at)}</dd>
                </div>
                <div>
                  <dt className="text-sm font-medium text-muted-foreground">Last Updated</dt>
                  <dd className="mt-1 text-sm">{twin.updated_at ? formatDate(twin.updated_at) : 'N/A'}</dd>
                </div>
                <div>
                  <dt className="text-sm font-medium text-muted-foreground">Model Type</dt>
                  <dd className="mt-1 text-sm">{twin.model_type}</dd>
                </div>
              </dl>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>Entity Types Distribution</CardTitle>
              <CardDescription>Breakdown of entities in this twin</CardDescription>
            </CardHeader>
            <CardContent>
              <div className="space-y-2">
                {Object.entries(
                  (twin.entities || []).reduce((acc, entity) => {
                    acc[entity.type] = (acc[entity.type] || 0) + 1;
                    return acc;
                  }, {} as Record<string, number>)
                ).map(([type, count]) => (
                  <div key={type} className="flex items-center justify-between">
                    <span className="text-sm">{type}</span>
                    <Badge variant="outline">{count}</Badge>
                  </div>
                ))}
              </div>
            </CardContent>
          </Card>
        </div>
      )}

      {activeTab === "insights" && twin && (
        <InsightsPanel 
          twinId={twinId} 
          twinName={twin.name}
          onQuestionClick={(question) => {
            setActiveTab("whatif");
            // The WhatIfAnalyzer will need to accept a default question - for now just switch tabs
          }}
        />
      )}

      {activeTab === "whatif" && twin && (
        <WhatIfAnalyzer twinId={twinId} twinName={twin.name} />
      )}

      {activeTab === "smart" && twin && (
        <SmartScenarios 
          twinId={twinId} 
          twinName={twin.name}
          onScenarioRun={(scenarioId) => handleRunScenario(scenarioId)}
        />
      )}

      {activeTab === "scenarios" && (
        <div className="space-y-6">
          <div className="flex justify-end">
            <Button onClick={() => router.push(`/digital-twins/${twinId}/scenarios/create`)}>
              <Plus className="h-4 w-4 mr-2" />
              Create Scenario
            </Button>
          </div>

          {scenarios.length === 0 ? (
            <Card>
              <CardContent className="pt-6">
                <div className="text-center py-12">
                  <Settings className="h-16 w-16 mx-auto text-muted-foreground mb-4" />
                  <h3 className="text-xl font-semibold mb-2">No Scenarios Yet</h3>
                  <p className="text-muted-foreground mb-6">
                    Create a scenario to simulate what-if situations on this digital twin
                  </p>
                  <Button onClick={() => router.push(`/digital-twins/${twinId}/scenarios/create`)}>
                    <Plus className="h-4 w-4 mr-2" />
                    Create First Scenario
                  </Button>
                </div>
              </CardContent>
            </Card>
          ) : (
            <div className="grid gap-4">
              {scenarios.map((scenario) => (
                <Card key={scenario.id} className="hover:shadow-md transition-shadow">
                  <CardHeader>
                    <div className="flex items-start justify-between">
                      <div>
                        <CardTitle>{scenario.name}</CardTitle>
                        <CardDescription>{scenario.description || "No description"}</CardDescription>
                      </div>
                      <Button
                        size="sm"
                        onClick={() => handleRunScenario(scenario.id)}
                      >
                        <Play className="h-4 w-4 mr-2" />
                        Run
                      </Button>
                    </div>
                  </CardHeader>
                  <CardContent>
                    <div className="grid grid-cols-3 gap-4 text-sm">
                      <div>
                        <p className="text-muted-foreground">Events</p>
                        <p className="font-medium">{scenario.events.length}</p>
                      </div>
                      <div>
                        <p className="text-muted-foreground">Duration</p>
                        <p className="font-medium">{scenario.duration} steps</p>
                      </div>
                      <div>
                        <p className="text-muted-foreground">Type</p>
                        <p className="font-medium">{scenario.scenario_type || "custom"}</p>
                      </div>
                    </div>
                  </CardContent>
                </Card>
              ))}
            </div>
          )}
        </div>
      )}

      {activeTab === "chat" && (
        <AgentChat twinId={twinId} />
      )}
    </div>
  );
}
