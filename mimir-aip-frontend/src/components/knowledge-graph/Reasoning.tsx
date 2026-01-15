"use client";

import { useState } from "react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Label } from "@/components/ui/label";
import { Checkbox } from "@/components/ui/checkbox";
import {
  Brain,
  Loader2,
  Info,
  CheckCircle2,
  AlertCircle,
  Download,
} from "lucide-react";
import { toast } from "sonner";

interface InferredTriple {
  subject: string;
  predicate: string;
  object: string;
  rule: string;
  justification?: string;
}

interface ReasoningResult {
  asserted_triples: number;
  inferred_triples: number;
  total_triples: number;
  rules_applied: string[];
  inferences: InferredTriple[];
  execution_time_ms: number;
  statistics: Record<string, number>;
}

const REASONING_RULES = [
  {
    id: "rdfs:subClassOf",
    name: "RDFS SubClass",
    description: "Infer types based on subclass hierarchy",
    category: "RDFS",
  },
  {
    id: "rdfs:domain",
    name: "RDFS Domain",
    description: "Infer types from property domains",
    category: "RDFS",
  },
  {
    id: "rdfs:range",
    name: "RDFS Range",
    description: "Infer types from property ranges",
    category: "RDFS",
  },
  {
    id: "owl:transitiveProperty",
    name: "OWL Transitive",
    description: "Infer transitive relationships",
    category: "OWL",
  },
  {
    id: "owl:symmetricProperty",
    name: "OWL Symmetric",
    description: "Infer symmetric relationships",
    category: "OWL",
  },
  {
    id: "owl:inverseOf",
    name: "OWL Inverse",
    description: "Infer inverse relationships",
    category: "OWL",
  },
];

export function Reasoning() {
  const [selectedRules, setSelectedRules] = useState<string[]>([]);
  const [loading, setLoading] = useState(false);
  const [result, setResult] = useState<ReasoningResult | null>(null);
  const [showInferences, setShowInferences] = useState(false);

  const handleRuleToggle = (ruleId: string) => {
    setSelectedRules((prev) =>
      prev.includes(ruleId)
        ? prev.filter((id) => id !== ruleId)
        : [...prev, ruleId]
    );
  };

  const handleSelectAll = () => {
    if (selectedRules.length === REASONING_RULES.length) {
      setSelectedRules([]);
    } else {
      setSelectedRules(REASONING_RULES.map((r) => r.id));
    }
  };

  const handleRunReasoning = async () => {
    if (selectedRules.length === 0) {
      toast.error("Please select at least one reasoning rule");
      return;
    }

    setLoading(true);
    setResult(null);

    try {
      const response = await fetch("/api/v1/knowledge-graph/reasoning", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          rules: selectedRules,
          max_depth: 10,
        }),
      });

      if (!response.ok) {
        throw new Error(`HTTP ${response.status}`);
      }

      const data = await response.json();
      setResult(data);
      toast.success(
        `Reasoning complete: ${data.inferred_triples} new triples inferred`
      );
    } catch (error) {
      console.error("Reasoning error:", error);
      toast.error("Failed to perform reasoning");
    } finally {
      setLoading(false);
    }
  };

  const handleExportInferences = () => {
    if (!result) return;

    const csv = [
      ["Subject", "Predicate", "Object", "Rule", "Justification"].join(","),
      ...result.inferences.map((inf) =>
        [
          inf.subject,
          inf.predicate,
          inf.object,
          inf.rule,
          inf.justification || "",
        ].join(",")
      ),
    ].join("\n");

    const blob = new Blob([csv], { type: "text/csv" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = `reasoning-inferences-${Date.now()}.csv`;
    a.click();
    URL.revokeObjectURL(url);
    toast.success("Inferences exported");
  };

  const extractLocalName = (uri: string) => {
    const hashIndex = uri.lastIndexOf("#");
    if (hashIndex !== -1) return uri.substring(hashIndex + 1);
    const slashIndex = uri.lastIndexOf("/");
    if (slashIndex !== -1) return uri.substring(slashIndex + 1);
    return uri;
  };

  return (
    <div className="space-y-6">
      {/* Rule Selection */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Brain className="h-5 w-5 text-orange" />
            Reasoning Rules
          </CardTitle>
          <CardDescription>
            Select OWL/RDFS reasoning rules to apply to the knowledge graph
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex items-center justify-between">
            <Button
              variant="outline"
              size="sm"
              onClick={handleSelectAll}
            >
              {selectedRules.length === REASONING_RULES.length
                ? "Deselect All"
                : "Select All"}
            </Button>
            <span className="text-sm text-white/60">
              {selectedRules.length} of {REASONING_RULES.length} selected
            </span>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            {REASONING_RULES.map((rule) => (
              <div
                key={rule.id}
                className="flex items-start space-x-3 p-3 rounded-lg border border-blue/30 hover:border-orange/50 transition-colors"
              >
                <Checkbox
                  id={rule.id}
                  checked={selectedRules.includes(rule.id)}
                  onCheckedChange={() => handleRuleToggle(rule.id)}
                />
                <div className="flex-1">
                  <Label
                    htmlFor={rule.id}
                    className="text-sm font-medium cursor-pointer"
                  >
                    {rule.name}
                  </Label>
                  <p className="text-xs text-white/60 mt-1">
                    {rule.description}
                  </p>
                  <span className="inline-block px-2 py-0.5 mt-2 text-xs rounded bg-blue/30 text-white/80">
                    {rule.category}
                  </span>
                </div>
              </div>
            ))}
          </div>

          <Button
            onClick={handleRunReasoning}
            disabled={loading || selectedRules.length === 0}
            className="w-full"
          >
            {loading ? (
              <>
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                Running Reasoning...
              </>
            ) : (
              <>
                <Brain className="mr-2 h-4 w-4" />
                Run Reasoning
              </>
            )}
          </Button>
        </CardContent>
      </Card>

      {/* Results */}
      {result && (
        <>
          {/* Statistics */}
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <CheckCircle2 className="h-5 w-5 text-green-500" />
                Reasoning Results
              </CardTitle>
              <CardDescription>
                Completed in {result.execution_time_ms}ms
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                <div className="p-4 rounded-lg bg-blue/20 border border-blue/30">
                  <div className="text-2xl font-bold text-white">
                    {result.asserted_triples.toLocaleString()}
                  </div>
                  <div className="text-sm text-white/60">Asserted Triples</div>
                </div>
                <div className="p-4 rounded-lg bg-orange/20 border border-orange/30">
                  <div className="text-2xl font-bold text-white">
                    {result.inferred_triples.toLocaleString()}
                  </div>
                  <div className="text-sm text-white/60">Inferred Triples</div>
                </div>
                <div className="p-4 rounded-lg bg-green-500/20 border border-green-500/30">
                  <div className="text-2xl font-bold text-white">
                    {result.total_triples.toLocaleString()}
                  </div>
                  <div className="text-sm text-white/60">Total Triples</div>
                </div>
              </div>

              {/* Rules Applied */}
              <div className="space-y-2">
                <h4 className="text-sm font-medium flex items-center gap-2">
                  <Info className="h-4 w-4 text-orange" />
                  Rules Applied
                </h4>
                <div className="flex flex-wrap gap-2">
                  {result.rules_applied.map((rule) => (
                    <div
                      key={rule}
                      className="px-3 py-1 rounded-full bg-blue/30 border border-blue/50 text-sm"
                    >
                      {rule}
                      {result.statistics[rule] && (
                        <span className="ml-2 text-orange">
                          ({result.statistics[rule]})
                        </span>
                      )}
                    </div>
                  ))}
                </div>
              </div>
            </CardContent>
          </Card>

          {/* Inferences Table */}
          {result.inferred_triples > 0 && (
            <Card>
              <CardHeader>
                <div className="flex items-center justify-between">
                  <div>
                    <CardTitle>Inferred Triples</CardTitle>
                    <CardDescription>
                      {result.inferences.length} inferences shown
                    </CardDescription>
                  </div>
                  <div className="flex gap-2">
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => setShowInferences(!showInferences)}
                    >
                      {showInferences ? "Hide" : "Show"} Details
                    </Button>
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={handleExportInferences}
                    >
                      <Download className="mr-2 h-4 w-4" />
                      Export CSV
                    </Button>
                  </div>
                </div>
              </CardHeader>
              {showInferences && (
                <CardContent>
                  <div className="overflow-x-auto">
                    <table className="w-full text-sm">
                      <thead>
                        <tr className="border-b border-blue/30">
                          <th className="text-left py-2 px-3 font-medium text-white/80">
                            Subject
                          </th>
                          <th className="text-left py-2 px-3 font-medium text-white/80">
                            Predicate
                          </th>
                          <th className="text-left py-2 px-3 font-medium text-white/80">
                            Object
                          </th>
                          <th className="text-left py-2 px-3 font-medium text-white/80">
                            Rule
                          </th>
                          <th className="text-left py-2 px-3 font-medium text-white/80">
                            Justification
                          </th>
                        </tr>
                      </thead>
                      <tbody>
                        {result.inferences.slice(0, 100).map((inf, idx) => (
                          <tr
                            key={idx}
                            className="border-b border-blue/10 hover:bg-blue/10"
                          >
                            <td className="py-2 px-3 font-mono text-xs">
                              {extractLocalName(inf.subject)}
                            </td>
                            <td className="py-2 px-3 font-mono text-xs text-orange">
                              {inf.predicate}
                            </td>
                            <td className="py-2 px-3 font-mono text-xs">
                              {extractLocalName(inf.object)}
                            </td>
                            <td className="py-2 px-3">
                              <span className="px-2 py-0.5 rounded-full bg-blue/30 text-xs">
                                {inf.rule}
                              </span>
                            </td>
                            <td className="py-2 px-3 text-xs text-white/60">
                              {inf.justification || "-"}
                            </td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                    {result.inferences.length > 100 && (
                      <div className="text-center py-3 text-sm text-white/60">
                        Showing first 100 of {result.inferences.length} inferences
                      </div>
                    )}
                  </div>
                </CardContent>
              )}
            </Card>
          )}

          {result.inferred_triples === 0 && (
            <Card>
              <CardContent className="py-8 text-center">
                <AlertCircle className="h-12 w-12 text-white/40 mx-auto mb-3" />
                <p className="text-white/60">
                  No new triples inferred. The knowledge graph may not contain
                  the necessary axioms for the selected reasoning rules.
                </p>
              </CardContent>
            </Card>
          )}
        </>
      )}
    </div>
  );
}
