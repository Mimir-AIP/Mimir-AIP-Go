"use client";

import { useState } from "react";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";
import { Badge } from "@/components/ui/badge";
import { toast } from "sonner";
import { 
  Sparkles, 
  Send, 
  AlertTriangle, 
  CheckCircle2, 
  TrendingUp, 
  Info,
  Loader2,
  Lightbulb,
  BarChart3
} from "lucide-react";
import { runWhatIfAnalysis, type WhatIfResponse, type WhatIfKeyFinding } from "@/lib/api";

interface WhatIfAnalyzerProps {
  twinId: string;
  twinName: string;
}

const EXAMPLE_QUESTIONS = [
  "What if our largest donor stops funding?",
  "What happens if demand increases by 50%?",
  "What if we lose our primary supplier?",
  "What would happen if staff availability drops by 30%?",
  "What if costs increase by 25%?",
];

export function WhatIfAnalyzer({ twinId, twinName }: WhatIfAnalyzerProps) {
  const [question, setQuestion] = useState("");
  const [loading, setLoading] = useState(false);
  const [response, setResponse] = useState<WhatIfResponse | null>(null);

  async function handleAnalyze() {
    if (!question.trim()) {
      toast.error("Please enter a question");
      return;
    }

    setLoading(true);
    try {
      const result = await runWhatIfAnalysis(twinId, question.trim());
      setResponse(result);
      toast.success("Analysis complete!");
    } catch (err) {
      const message = err instanceof Error ? err.message : "Analysis failed";
      toast.error(message);
    } finally {
      setLoading(false);
    }
  }

  function handleExampleClick(example: string) {
    setQuestion(example);
  }

  function getTypeIcon(type: string) {
    switch (type) {
      case "risk":
        return <AlertTriangle className="h-4 w-4 text-red-500" />;
      case "warning":
        return <AlertTriangle className="h-4 w-4 text-amber-500" />;
      case "impact":
        return <TrendingUp className="h-4 w-4 text-blue-500" />;
      case "opportunity":
        return <CheckCircle2 className="h-4 w-4 text-emerald-500" />;
      default:
        return <Info className="h-4 w-4 text-slate-500" />;
    }
  }

  function getSeverityColor(severity?: string) {
    switch (severity) {
      case "critical":
        return "bg-red-500/10 text-red-700 border-red-500/20";
      case "high":
        return "bg-orange-500/10 text-orange-700 border-orange-500/20";
      case "medium":
        return "bg-amber-500/10 text-amber-700 border-amber-500/20";
      default:
        return "bg-slate-500/10 text-slate-700 border-slate-500/20";
    }
  }

  return (
    <div className="space-y-6">
      {/* Question Input */}
      <Card className="border-2 border-dashed border-primary/20 bg-gradient-to-br from-primary/5 to-transparent">
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Sparkles className="h-5 w-5 text-primary" />
            Ask "What If?"
          </CardTitle>
          <CardDescription>
            Ask a natural language question about {twinName} and get AI-powered analysis
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <Textarea
            placeholder="e.g., What happens if our main supplier becomes unavailable for 2 weeks?"
            value={question}
            onChange={(e) => setQuestion(e.target.value)}
            className="min-h-[100px] text-base"
          />
          
          <div className="flex flex-wrap gap-2">
            <span className="text-sm text-muted-foreground">Try:</span>
            {EXAMPLE_QUESTIONS.slice(0, 3).map((example, i) => (
              <button
                key={i}
                onClick={() => handleExampleClick(example)}
                className="text-xs px-2 py-1 rounded-full bg-secondary hover:bg-secondary/80 transition-colors"
              >
                {example}
              </button>
            ))}
          </div>

          <Button 
            onClick={handleAnalyze} 
            disabled={loading || !question.trim()}
            className="w-full"
          >
            {loading ? (
              <>
                <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                Analyzing...
              </>
            ) : (
              <>
                <Send className="h-4 w-4 mr-2" />
                Analyze Scenario
              </>
            )}
          </Button>
        </CardContent>
      </Card>

      {/* Results */}
      {response && (
        <div className="space-y-4 animate-in fade-in slide-in-from-bottom-4 duration-500">
          {/* Summary Card */}
          <Card>
            <CardHeader>
              <div className="flex items-center justify-between">
                <CardTitle className="text-lg">Analysis Results</CardTitle>
                <div className="flex items-center gap-2">
                  <Badge variant="outline">
                    Confidence: {Math.round(response.confidence * 100)}%
                  </Badge>
                  <Badge variant="secondary">
                    {response.processing_time_ms}ms
                  </Badge>
                </div>
              </div>
              <CardDescription className="text-base font-medium text-foreground">
                "{response.question}"
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              {/* Interpretation */}
              <div className="p-3 rounded-lg bg-muted/50">
                <p className="text-sm text-muted-foreground mb-1">What was simulated:</p>
                <p className="font-medium">{response.interpretation}</p>
              </div>

              {/* Summary */}
              <div className="p-4 rounded-lg bg-gradient-to-br from-primary/10 to-transparent border">
                <p className="text-sm font-medium">{response.summary}</p>
              </div>
            </CardContent>
          </Card>

          {/* Metrics Card */}
          {response.results?.metrics && (
            <Card>
              <CardHeader>
                <CardTitle className="text-lg flex items-center gap-2">
                  <BarChart3 className="h-5 w-5" />
                  Simulation Metrics
                </CardTitle>
              </CardHeader>
              <CardContent>
                <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
                  <div className="text-center p-3 rounded-lg bg-muted/50">
                    <p className="text-2xl font-bold">
                      {Math.round(response.results.metrics.system_stability * 100)}%
                    </p>
                    <p className="text-xs text-muted-foreground">System Stability</p>
                  </div>
                  <div className="text-center p-3 rounded-lg bg-muted/50">
                    <p className="text-2xl font-bold">
                      {Math.round(response.results.metrics.average_utilization * 100)}%
                    </p>
                    <p className="text-xs text-muted-foreground">Avg Utilization</p>
                  </div>
                  <div className="text-center p-3 rounded-lg bg-muted/50">
                    <p className="text-2xl font-bold">
                      {Math.round(response.results.metrics.peak_utilization * 100)}%
                    </p>
                    <p className="text-xs text-muted-foreground">Peak Utilization</p>
                  </div>
                  <div className="text-center p-3 rounded-lg bg-muted/50">
                    <p className="text-2xl font-bold">
                      {response.results.metrics.total_steps}
                    </p>
                    <p className="text-xs text-muted-foreground">Simulation Steps</p>
                  </div>
                </div>
              </CardContent>
            </Card>
          )}

          {/* Key Findings */}
          {response.key_findings.length > 0 && (
            <Card>
              <CardHeader>
                <CardTitle className="text-lg">Key Findings</CardTitle>
              </CardHeader>
              <CardContent>
                <div className="space-y-3">
                  {response.key_findings.map((finding, i) => (
                    <div 
                      key={i}
                      className={`flex items-start gap-3 p-3 rounded-lg border ${getSeverityColor(finding.severity)}`}
                    >
                      {getTypeIcon(finding.type)}
                      <div className="flex-1">
                        <p className="font-medium">{finding.description}</p>
                        {finding.entity && (
                          <p className="text-xs text-muted-foreground mt-1">
                            Affects: {finding.entity}
                          </p>
                        )}
                      </div>
                      {finding.severity && (
                        <Badge variant="outline" className="uppercase text-xs">
                          {finding.severity}
                        </Badge>
                      )}
                    </div>
                  ))}
                </div>
              </CardContent>
            </Card>
          )}

          {/* Recommendations */}
          {response.recommendations.length > 0 && (
            <Card>
              <CardHeader>
                <CardTitle className="text-lg flex items-center gap-2">
                  <Lightbulb className="h-5 w-5 text-amber-500" />
                  Recommendations
                </CardTitle>
              </CardHeader>
              <CardContent>
                <ul className="space-y-2">
                  {response.recommendations.map((rec, i) => (
                    <li key={i} className="flex items-start gap-2">
                      <CheckCircle2 className="h-4 w-4 mt-0.5 text-emerald-500 shrink-0" />
                      <span>{rec}</span>
                    </li>
                  ))}
                </ul>
              </CardContent>
            </Card>
          )}
        </div>
      )}
    </div>
  );
}

