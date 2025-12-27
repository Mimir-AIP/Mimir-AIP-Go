"use client";

import { useState } from "react";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { toast } from "sonner";
import { 
  Sparkles,
  Play,
  Save,
  Loader2,
  AlertTriangle,
  Shield,
  TrendingUp,
  Zap,
  RefreshCw,
  CheckCircle2
} from "lucide-react";
import { generateSmartScenarios, type GeneratedScenario } from "@/lib/api";

interface SmartScenariosProps {
  twinId: string;
  twinName: string;
  onScenarioRun?: (scenarioId: string) => void;
}

export function SmartScenarios({ twinId, twinName, onScenarioRun }: SmartScenariosProps) {
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [scenarios, setScenarios] = useState<GeneratedScenario[]>([]);
  const [generated, setGenerated] = useState(false);

  async function handleGenerate() {
    setLoading(true);
    try {
      const result = await generateSmartScenarios(twinId, false);
      setScenarios(result.scenarios);
      setGenerated(true);
      toast.success(`Generated ${result.count} smart scenarios!`);
    } catch (err) {
      const message = err instanceof Error ? err.message : "Failed to generate scenarios";
      toast.error(message);
    } finally {
      setLoading(false);
    }
  }

  async function handleSaveAll() {
    setSaving(true);
    try {
      const result = await generateSmartScenarios(twinId, true);
      toast.success(`Saved ${result.saved_count} scenarios to the twin!`);
    } catch (err) {
      const message = err instanceof Error ? err.message : "Failed to save scenarios";
      toast.error(message);
    } finally {
      setSaving(false);
    }
  }

  function getTypeIcon(type: string) {
    switch (type) {
      case "baseline":
        return <Shield className="h-4 w-4 text-emerald-500" />;
      case "risk_assessment":
        return <AlertTriangle className="h-4 w-4 text-amber-500" />;
      case "demand_surge":
      case "demand_spike":
        return <TrendingUp className="h-4 w-4 text-blue-500" />;
      case "funding_disruption":
      case "supply_disruption":
        return <Zap className="h-4 w-4 text-red-500" />;
      default:
        return <Sparkles className="h-4 w-4 text-purple-500" />;
    }
  }

  function getConfidenceColor(confidence: number) {
    if (confidence >= 0.85) return "text-emerald-600 bg-emerald-500/10";
    if (confidence >= 0.7) return "text-blue-600 bg-blue-500/10";
    return "text-amber-600 bg-amber-500/10";
  }

  return (
    <div className="space-y-6">
      {/* Header Card */}
      <Card className="bg-gradient-to-br from-violet-500/10 via-purple-500/5 to-transparent border-violet-500/20">
        <CardHeader>
          <div className="flex items-center justify-between">
            <div>
              <CardTitle className="flex items-center gap-2">
                <Sparkles className="h-5 w-5 text-violet-500" />
                AI-Generated Scenarios
              </CardTitle>
              <CardDescription className="mt-1">
                Automatically generate relevant simulation scenarios based on your data structure
              </CardDescription>
            </div>
            <div className="flex gap-2">
              {generated && scenarios.length > 0 && (
                <Button variant="outline" onClick={handleSaveAll} disabled={saving}>
                  {saving ? (
                    <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                  ) : (
                    <Save className="h-4 w-4 mr-2" />
                  )}
                  Save All
                </Button>
              )}
              <Button onClick={handleGenerate} disabled={loading}>
                {loading ? (
                  <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                ) : generated ? (
                  <RefreshCw className="h-4 w-4 mr-2" />
                ) : (
                  <Sparkles className="h-4 w-4 mr-2" />
                )}
                {generated ? "Regenerate" : "Generate Scenarios"}
              </Button>
            </div>
          </div>
        </CardHeader>
      </Card>

      {/* Generated Scenarios */}
      {scenarios.length > 0 && (
        <div className="grid gap-4 md:grid-cols-2">
          {scenarios.map((gs, index) => (
            <Card 
              key={gs.scenario.id}
              className="group hover:shadow-lg transition-all duration-300 animate-in fade-in slide-in-from-bottom-2"
              style={{ animationDelay: `${index * 100}ms` }}
            >
              <CardHeader className="pb-2">
                <div className="flex items-start justify-between gap-2">
                  <div className="flex items-center gap-2 flex-1 min-w-0">
                    {getTypeIcon(gs.scenario.type)}
                    <CardTitle className="text-base truncate">
                      {gs.scenario.name}
                    </CardTitle>
                  </div>
                  <Badge 
                    variant="outline" 
                    className={`shrink-0 ${getConfidenceColor(gs.confidence)}`}
                  >
                    {Math.round(gs.confidence * 100)}% match
                  </Badge>
                </div>
              </CardHeader>
              <CardContent className="space-y-3">
                <p className="text-sm text-muted-foreground line-clamp-2">
                  {gs.scenario.description}
                </p>
                
                {gs.explanation && (
                  <div className="p-2 rounded bg-muted/50 text-xs">
                    <span className="font-medium">Why this scenario: </span>
                    {gs.explanation}
                  </div>
                )}

                <div className="flex items-center justify-between text-xs text-muted-foreground">
                  <div className="flex items-center gap-4">
                    <span>
                      <strong>{gs.scenario.events.length}</strong> events
                    </span>
                    <span>
                      <strong>{gs.scenario.duration}</strong> steps
                    </span>
                  </div>
                  <Badge variant="secondary" className="text-xs">
                    {gs.scenario.type.replace(/_/g, " ")}
                  </Badge>
                </div>

                {gs.risk_addressed && (
                  <div className="flex items-center gap-1 text-xs text-amber-600">
                    <AlertTriangle className="h-3 w-3" />
                    Addresses: {gs.risk_addressed}
                  </div>
                )}

                <Button 
                  variant="outline" 
                  size="sm" 
                  className="w-full opacity-0 group-hover:opacity-100 transition-opacity"
                  onClick={() => onScenarioRun?.(gs.scenario.id)}
                >
                  <Play className="h-3 w-3 mr-2" />
                  Run Simulation
                </Button>
              </CardContent>
            </Card>
          ))}
        </div>
      )}

      {/* Empty State */}
      {!loading && !generated && (
        <Card className="border-dashed">
          <CardContent className="py-12 text-center">
            <Sparkles className="h-12 w-12 mx-auto text-muted-foreground mb-4" />
            <h3 className="text-lg font-semibold mb-2">No Scenarios Generated Yet</h3>
            <p className="text-muted-foreground mb-4 max-w-md mx-auto">
              Click "Generate Scenarios" to automatically create relevant simulation scenarios 
              based on your digital twin's data structure and detected patterns.
            </p>
            <div className="flex items-center justify-center gap-2 text-sm text-muted-foreground">
              <CheckCircle2 className="h-4 w-4 text-emerald-500" />
              AI analyzes entity types, relationships, and risks
            </div>
          </CardContent>
        </Card>
      )}
    </div>
  );
}

