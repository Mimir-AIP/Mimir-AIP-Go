"use client";

import { useEffect, useState } from "react";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { toast } from "sonner";
import { 
  Lightbulb,
  AlertTriangle,
  TrendingUp,
  Shield,
  RefreshCw,
  Loader2,
  ChevronRight,
  HelpCircle,
  Zap,
  CheckCircle2
} from "lucide-react";
import { getInsights, type InsightReport, type Insight, type SuggestedQuestion } from "@/lib/api";

interface InsightsPanelProps {
  twinId: string;
  twinName: string;
  onQuestionClick?: (question: string) => void;
}

export function InsightsPanel({ twinId, twinName, onQuestionClick }: InsightsPanelProps) {
  const [loading, setLoading] = useState(false);
  const [report, setReport] = useState<InsightReport | null>(null);
  const [expanded, setExpanded] = useState<string | null>(null);

  useEffect(() => {
    loadInsights();
  }, [twinId]);

  async function loadInsights() {
    setLoading(true);
    try {
      const data = await getInsights(twinId);
      setReport(data);
    } catch (err) {
      const message = err instanceof Error ? err.message : "Failed to load insights";
      toast.error(message);
    } finally {
      setLoading(false);
    }
  }

  function getTypeIcon(type: string) {
    switch (type) {
      case "risk":
        return <AlertTriangle className="h-4 w-4 text-red-500" />;
      case "warning":
        return <AlertTriangle className="h-4 w-4 text-amber-500" />;
      case "opportunity":
        return <TrendingUp className="h-4 w-4 text-emerald-500" />;
      case "trend":
        return <TrendingUp className="h-4 w-4 text-blue-500" />;
      default:
        return <Lightbulb className="h-4 w-4 text-purple-500" />;
    }
  }

  function getSeverityColor(severity?: string) {
    switch (severity) {
      case "critical":
        return "bg-red-500/10 text-red-700 border-red-500/30";
      case "high":
        return "bg-orange-500/10 text-orange-700 border-orange-500/30";
      case "medium":
        return "bg-amber-500/10 text-amber-700 border-amber-500/30";
      default:
        return "bg-slate-500/10 text-slate-700 border-slate-500/30";
    }
  }

  function getHealthColor(score: number) {
    if (score >= 0.8) return "text-emerald-600";
    if (score >= 0.6) return "text-amber-600";
    return "text-red-600";
  }

  function getRiskColor(score: number) {
    if (score <= 0.3) return "text-emerald-600";
    if (score <= 0.6) return "text-amber-600";
    return "text-red-600";
  }

  if (loading && !report) {
    return (
      <Card>
        <CardContent className="py-12 flex items-center justify-center">
          <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
        </CardContent>
      </Card>
    );
  }

  if (!report) {
    return (
      <Card>
        <CardContent className="py-12 text-center">
          <Lightbulb className="h-12 w-12 mx-auto text-muted-foreground mb-4" />
          <p className="text-muted-foreground">No insights available</p>
          <Button onClick={loadInsights} className="mt-4" variant="outline">
            <RefreshCw className="h-4 w-4 mr-2" />
            Generate Insights
          </Button>
        </CardContent>
      </Card>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header with Scores */}
      <Card className="bg-gradient-to-br from-violet-500/10 via-purple-500/5 to-transparent border-violet-500/20">
        <CardHeader>
          <div className="flex items-center justify-between">
            <div>
              <CardTitle className="flex items-center gap-2">
                <Lightbulb className="h-5 w-5 text-violet-500" />
                Proactive Insights
              </CardTitle>
              <CardDescription className="mt-1">
                AI-generated insights for {twinName}
              </CardDescription>
            </div>
            <Button variant="outline" size="sm" onClick={loadInsights} disabled={loading}>
              {loading ? (
                <Loader2 className="h-4 w-4 animate-spin" />
              ) : (
                <RefreshCw className="h-4 w-4" />
              )}
            </Button>
          </div>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-3 gap-4">
            <div className="text-center p-3 rounded-lg bg-background/50">
              <div className={`text-3xl font-bold ${getHealthColor(report.health_score)}`}>
                {Math.round(report.health_score * 100)}%
              </div>
              <div className="text-xs text-muted-foreground">Health Score</div>
            </div>
            <div className="text-center p-3 rounded-lg bg-background/50">
              <div className={`text-3xl font-bold ${getRiskColor(report.risk_score)}`}>
                {Math.round(report.risk_score * 100)}%
              </div>
              <div className="text-xs text-muted-foreground">Risk Score</div>
            </div>
            <div className="text-center p-3 rounded-lg bg-background/50">
              <div className="text-3xl font-bold text-foreground">
                {report.insights.length}
              </div>
              <div className="text-xs text-muted-foreground">Insights Found</div>
            </div>
          </div>
          <p className="mt-4 text-sm text-muted-foreground">{report.summary}</p>
        </CardContent>
      </Card>

      {/* Insights List */}
      {report.insights.length > 0 && (
        <Card>
          <CardHeader>
            <CardTitle className="text-lg">Key Insights</CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            {report.insights.map((insight) => (
              <div
                key={insight.id}
                className={`rounded-lg border p-4 transition-all ${getSeverityColor(insight.severity)} ${expanded === insight.id ? "ring-2 ring-primary/20" : ""}`}
              >
                <div 
                  className="flex items-start gap-3 cursor-pointer"
                  onClick={() => setExpanded(expanded === insight.id ? null : insight.id)}
                >
                  {getTypeIcon(insight.type)}
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-2">
                      <h4 className="font-medium">{insight.title}</h4>
                      {insight.severity && (
                        <Badge variant="outline" className="uppercase text-xs">
                          {insight.severity}
                        </Badge>
                      )}
                    </div>
                    <p className="text-sm text-muted-foreground mt-1 line-clamp-2">
                      {insight.description}
                    </p>
                  </div>
                  <ChevronRight className={`h-4 w-4 transition-transform ${expanded === insight.id ? "rotate-90" : ""}`} />
                </div>
                
                {expanded === insight.id && insight.actions && insight.actions.length > 0 && (
                  <div className="mt-4 pt-3 border-t border-current/10">
                    <p className="text-xs font-medium mb-2">Suggested Actions:</p>
                    <div className="flex flex-wrap gap-2">
                      {insight.actions.map((action, i) => (
                        <Button
                          key={i}
                          variant="secondary"
                          size="sm"
                          onClick={() => {
                            if (action.type === "simulate" && action.parameters) {
                              const question = `What if ${action.parameters.target_type || "this entity"} becomes unavailable?`;
                              onQuestionClick?.(question);
                            }
                            toast.info(`Action: ${action.label}`);
                          }}
                        >
                          <Zap className="h-3 w-3 mr-1" />
                          {action.label}
                        </Button>
                      ))}
                    </div>
                  </div>
                )}
              </div>
            ))}
          </CardContent>
        </Card>
      )}

      {/* Suggested Questions */}
      {report.suggested_questions.length > 0 && (
        <Card>
          <CardHeader>
            <CardTitle className="text-lg flex items-center gap-2">
              <HelpCircle className="h-5 w-5" />
              Suggested What-If Questions
            </CardTitle>
            <CardDescription>
              Click to analyze any of these scenarios
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-2">
            {report.suggested_questions.map((q, i) => (
              <button
                key={i}
                className="w-full text-left p-3 rounded-lg border hover:bg-muted/50 transition-colors group"
                onClick={() => onQuestionClick?.(q.question)}
              >
                <div className="flex items-start gap-3">
                  <Lightbulb className="h-4 w-4 mt-0.5 text-amber-500" />
                  <div className="flex-1 min-w-0">
                    <p className="font-medium group-hover:text-primary transition-colors">
                      {q.question}
                    </p>
                    <p className="text-xs text-muted-foreground mt-1">
                      {q.reason}
                    </p>
                  </div>
                  <div className="flex items-center gap-2">
                    <Badge variant="outline" className="text-xs">
                      {q.category}
                    </Badge>
                    <Badge 
                      variant="secondary" 
                      className={q.relevance > 0.9 ? "bg-emerald-500/10 text-emerald-700" : ""}
                    >
                      {Math.round(q.relevance * 100)}%
                    </Badge>
                  </div>
                </div>
              </button>
            ))}
          </CardContent>
        </Card>
      )}

      {/* Empty state */}
      {report.insights.length === 0 && report.suggested_questions.length === 0 && (
        <Card className="border-dashed">
          <CardContent className="py-12 text-center">
            <CheckCircle2 className="h-12 w-12 mx-auto text-emerald-500 mb-4" />
            <h3 className="text-lg font-semibold mb-2">System Looking Good!</h3>
            <p className="text-muted-foreground">
              No significant issues or risks detected. Keep monitoring for changes.
            </p>
          </CardContent>
        </Card>
      )}
    </div>
  );
}

