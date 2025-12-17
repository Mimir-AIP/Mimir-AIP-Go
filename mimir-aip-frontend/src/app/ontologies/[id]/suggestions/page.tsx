"use client";

import { useParams, useRouter } from "next/navigation";
import { useEffect, useState } from "react";
import {
  listSuggestions,
  approveSuggestion,
  rejectSuggestion,
  applySuggestion,
  triggerDriftDetection,
  getDriftHistory,
  OntologySuggestion,
  DriftDetection,
} from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { toast } from "sonner";

export default function SuggestionsPage() {
  const params = useParams();
  const router = useRouter();
  const ontologyId = params.id as string;

  const [suggestions, setSuggestions] = useState<OntologySuggestion[]>([]);
  const [driftHistory, setDriftHistory] = useState<DriftDetection[]>([]);
  const [loading, setLoading] = useState(true);
  const [driftLoading, setDriftLoading] = useState(false);
  const [selectedSuggestion, setSelectedSuggestion] = useState<OntologySuggestion | null>(null);
  const [reviewNotes, setReviewNotes] = useState("");
  const [reviewedBy, setReviewedBy] = useState("");
  const [activeTab, setActiveTab] = useState("pending");

  useEffect(() => {
    loadSuggestions();
    loadDriftHistory();
  }, [ontologyId]);

  const loadSuggestions = async () => {
    try {
      const response = await listSuggestions(ontologyId);
      if (response.success) {
        setSuggestions(response.data);
      }
    } catch (error) {
      console.error("Failed to load suggestions:", error);
      toast.error("Failed to load suggestions");
    } finally {
      setLoading(false);
    }
  };

  const loadDriftHistory = async () => {
    try {
      const response = await getDriftHistory(ontologyId);
      if (response.success) {
        setDriftHistory(response.data);
      }
    } catch (error) {
      console.error("Failed to load drift history:", error);
    }
  };

  const handleTriggerDrift = async (source: "knowledge_graph" | "extraction_job" | "data") => {
    setDriftLoading(true);
    try {
      const response = await triggerDriftDetection(ontologyId, { source });
      if (response.success) {
        toast.success(`Drift detection completed: ${response.data.suggestions_generated} suggestions generated`);
        await loadSuggestions();
        await loadDriftHistory();
      }
    } catch (error) {
      console.error("Drift detection failed:", error);
      toast.error("Drift detection failed");
    } finally {
      setDriftLoading(false);
    }
  };

  const handleApprove = async (suggestion: OntologySuggestion) => {
    if (!reviewedBy) {
      toast.error("Please enter your name in the reviewer field");
      return;
    }

    try {
      await approveSuggestion(ontologyId, suggestion.id, {
        reviewed_by: reviewedBy,
        review_notes: reviewNotes,
      });
      toast.success("Suggestion approved");
      setSelectedSuggestion(null);
      setReviewNotes("");
      await loadSuggestions();
    } catch (error) {
      console.error("Failed to approve suggestion:", error);
      toast.error("Failed to approve suggestion");
    }
  };

  const handleReject = async (suggestion: OntologySuggestion) => {
    if (!reviewedBy) {
      toast.error("Please enter your name in the reviewer field");
      return;
    }

    if (!reviewNotes) {
      toast.error("Please provide rejection notes");
      return;
    }

    try {
      await rejectSuggestion(ontologyId, suggestion.id, {
        reviewed_by: reviewedBy,
        review_notes: reviewNotes,
      });
      toast.success("Suggestion rejected");
      setSelectedSuggestion(null);
      setReviewNotes("");
      await loadSuggestions();
    } catch (error) {
      console.error("Failed to reject suggestion:", error);
      toast.error("Failed to reject suggestion");
    }
  };

  const handleApply = async (suggestion: OntologySuggestion) => {
    if (suggestion.status !== "approved") {
      toast.error("Only approved suggestions can be applied");
      return;
    }

    try {
      await applySuggestion(ontologyId, suggestion.id);
      toast.success("Suggestion applied to ontology");
      await loadSuggestions();
    } catch (error) {
      console.error("Failed to apply suggestion:", error);
      toast.error("Failed to apply suggestion");
    }
  };

  const getSuggestionTypeColor = (type: string) => {
    switch (type) {
      case "add_class":
        return "bg-blue-500";
      case "add_property":
        return "bg-green-500";
      case "modify_class":
        return "bg-yellow-500";
      case "modify_property":
        return "bg-orange-500";
      case "deprecate":
        return "bg-red-500";
      default:
        return "bg-gray-500";
    }
  };

  const getRiskLevelColor = (risk: string) => {
    switch (risk) {
      case "low":
        return "bg-green-900/40 text-green-400 border border-green-500";
      case "medium":
        return "bg-yellow-900/40 text-yellow-400 border border-yellow-500";
      case "high":
        return "bg-orange-900/40 text-orange-400 border border-orange-500";
      case "critical":
        return "bg-red-900/40 text-red-400 border border-red-500";
      default:
        return "bg-gray-800 text-gray-400 border border-gray-600";
    }
  };

  const getStatusColor = (status: string) => {
    switch (status) {
      case "pending":
        return "bg-gray-800 text-gray-400 border border-gray-600";
      case "approved":
        return "bg-green-900/40 text-green-400 border border-green-500";
      case "rejected":
        return "bg-red-900/40 text-red-400 border border-red-500";
      case "applied":
        return "bg-blue-900/40 text-blue-400 border border-blue-500";
      default:
        return "bg-gray-800 text-gray-400 border border-gray-600";
    }
  };

  const filteredSuggestions = suggestions.filter((s) => {
    if (activeTab === "all") return true;
    return s.status === activeTab;
  });

  const stats = {
    total: suggestions.length,
    pending: suggestions.filter((s) => s.status === "pending").length,
    approved: suggestions.filter((s) => s.status === "approved").length,
    rejected: suggestions.filter((s) => s.status === "rejected").length,
    applied: suggestions.filter((s) => s.status === "applied").length,
    highRisk: suggestions.filter((s) => s.risk_level === "high" || s.risk_level === "critical").length,
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center h-screen">
        <div className="text-lg">Loading suggestions...</div>
      </div>
    );
  }

  return (
    <div className="container mx-auto p-6 space-y-6">
      {/* Header */}
      <div className="flex justify-between items-center">
        <div>
          <h1 className="text-3xl font-bold">Ontology Suggestions</h1>
          <p className="text-muted-foreground">AI-powered suggestions for improving your ontology</p>
        </div>
        <Button variant="outline" onClick={() => router.back()}>
          Back to Ontology
        </Button>
      </div>

      {/* Stats Cards */}
      <div className="grid grid-cols-6 gap-4">
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium">Total</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{stats.total}</div>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium">Pending</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold text-gray-700">{stats.pending}</div>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium">Approved</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold text-green-700">{stats.approved}</div>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium">Rejected</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold text-red-400">{stats.rejected}</div>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium">Applied</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold text-blue-700">{stats.applied}</div>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium">High Risk</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold text-orange-700">{stats.highRisk}</div>
          </CardContent>
        </Card>
      </div>

      {/* Drift Detection Section */}
      <Card>
        <CardHeader>
          <CardTitle>Drift Detection</CardTitle>
          <CardDescription>Scan for changes in your data that suggest ontology updates</CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex gap-4">
            <Button onClick={() => handleTriggerDrift("knowledge_graph")} disabled={driftLoading}>
              Scan Knowledge Graph
            </Button>
            <Button variant="outline" disabled={driftLoading}>
              From Extraction Job
            </Button>
          </div>

          {/* Recent Drift Detections */}
          {driftHistory.length > 0 && (
            <div className="space-y-2">
              <h3 className="font-semibold text-sm">Recent Drift Detections</h3>
              <div className="space-y-2">
                {driftHistory.slice(0, 5).map((detection) => (
                  <div key={detection.id} className="flex items-center justify-between p-3 border rounded-lg">
                    <div className="space-y-1">
                      <div className="text-sm font-medium">{detection.detection_type}</div>
                      <div className="text-xs text-muted-foreground">
                        {new Date(detection.started_at).toLocaleString()} • {detection.data_source}
                      </div>
                    </div>
                    <div className="flex items-center gap-4">
                      <Badge variant="outline">{detection.suggestions_generated} suggestions</Badge>
                      <Badge className={detection.status === "completed" ? "bg-green-500" : "bg-gray-500"}>
                        {detection.status}
                      </Badge>
                    </div>
                  </div>
                ))}
              </div>
            </div>
          )}
        </CardContent>
      </Card>

      {/* Suggestions List */}
      <Card>
        <CardHeader>
          <CardTitle>Suggestions</CardTitle>
          <CardDescription>Review and manage ontology improvement suggestions</CardDescription>
        </CardHeader>
        <CardContent>
          <Tabs value={activeTab} onValueChange={setActiveTab}>
            <TabsList className="grid w-full grid-cols-5">
              <TabsTrigger value="all">All ({stats.total})</TabsTrigger>
              <TabsTrigger value="pending">Pending ({stats.pending})</TabsTrigger>
              <TabsTrigger value="approved">Approved ({stats.approved})</TabsTrigger>
              <TabsTrigger value="rejected">Rejected ({stats.rejected})</TabsTrigger>
              <TabsTrigger value="applied">Applied ({stats.applied})</TabsTrigger>
            </TabsList>

            <TabsContent value={activeTab} className="space-y-4 mt-4">
              {filteredSuggestions.length === 0 ? (
                <div className="text-center py-8 text-muted-foreground">
                  No suggestions in this category
                </div>
              ) : (
                filteredSuggestions.map((suggestion) => (
                  <Card key={suggestion.id} className="hover:shadow-md transition-shadow">
                    <CardContent className="pt-6">
                      <div className="space-y-4">
                        {/* Header Row */}
                        <div className="flex items-start justify-between">
                          <div className="space-y-2 flex-1">
                            <div className="flex items-center gap-2">
                              <Badge className={getSuggestionTypeColor(suggestion.suggestion_type)}>
                                {suggestion.suggestion_type.replace("_", " ").toUpperCase()}
                              </Badge>
                              <Badge className={getRiskLevelColor(suggestion.risk_level)}>
                                {suggestion.risk_level.toUpperCase()} RISK
                              </Badge>
                              <Badge variant="outline" className={getStatusColor(suggestion.status)}>
                                {suggestion.status.toUpperCase()}
                              </Badge>
                              <span className="text-sm text-muted-foreground">
                                {(suggestion.confidence * 100).toFixed(0)}% confidence
                              </span>
                            </div>
                            
                            {suggestion.entity_uri && (
                              <div className="text-sm font-mono bg-muted p-2 rounded">
                                {suggestion.entity_uri}
                              </div>
                            )}
                          </div>
                        </div>

                        {/* Reasoning */}
                        <div className="space-y-2">
                          <h4 className="font-semibold text-sm">Reasoning:</h4>
                          <p className="text-sm text-muted-foreground">{suggestion.reasoning}</p>
                        </div>

                        {/* Review Info (if reviewed) */}
                        {suggestion.reviewed_at && (
                          <div className="border-t pt-4 space-y-2">
                            <div className="text-xs text-muted-foreground">
                              Reviewed by {suggestion.reviewed_by} on {new Date(suggestion.reviewed_at).toLocaleString()}
                            </div>
                            {suggestion.review_notes && (
                              <div className="text-sm italic">
                                &ldquo;{suggestion.review_notes}&rdquo;
                              </div>
                            )}
                          </div>
                        )}

                        {/* Actions */}
                        <div className="flex items-center justify-between border-t pt-4">
                          <div className="text-xs text-muted-foreground">
                            Created: {new Date(suggestion.created_at).toLocaleString()}
                          </div>
                          
                          <div className="flex gap-2">
                            {suggestion.status === "pending" && (
                              <>
                                <Button
                                  size="sm"
                                  variant="outline"
                                  onClick={() => setSelectedSuggestion(suggestion)}
                                >
                                  Review
                                </Button>
                              </>
                            )}
                            {suggestion.status === "approved" && (
                              <Button
                                size="sm"
                                onClick={() => handleApply(suggestion)}
                                className="bg-green-600 hover:bg-green-700"
                              >
                                Apply to Ontology
                              </Button>
                            )}
                            {suggestion.status === "applied" && (
                              <Badge variant="secondary" className="px-3 py-1">
                                Applied
                              </Badge>
                            )}
                          </div>
                        </div>
                      </div>
                    </CardContent>
                  </Card>
                ))
              )}
            </TabsContent>
          </Tabs>
        </CardContent>
      </Card>

      {/* Review Modal */}
      {selectedSuggestion && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
          <Card className="max-w-2xl w-full max-h-[80vh] overflow-y-auto">
            <CardHeader>
              <CardTitle>Review Suggestion</CardTitle>
              <CardDescription>Approve or reject this ontology change suggestion</CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              {/* Suggestion Details */}
              <div className="space-y-3 p-4 bg-muted rounded-lg">
                <div className="flex items-center gap-2">
                  <Badge className={getSuggestionTypeColor(selectedSuggestion.suggestion_type)}>
                    {selectedSuggestion.suggestion_type.replace("_", " ").toUpperCase()}
                  </Badge>
                  <Badge className={getRiskLevelColor(selectedSuggestion.risk_level)}>
                    {selectedSuggestion.risk_level.toUpperCase()} RISK
                  </Badge>
                </div>
                {selectedSuggestion.entity_uri && (
                  <div className="text-sm font-mono">{selectedSuggestion.entity_uri}</div>
                )}
                <p className="text-sm">{selectedSuggestion.reasoning}</p>
                <div className="text-xs text-muted-foreground">
                  Confidence: {(selectedSuggestion.confidence * 100).toFixed(0)}%
                </div>
              </div>

              {/* Review Form */}
              <div className="space-y-4">
                <div>
                  <label className="text-sm font-medium">Reviewer Name</label>
                  <input
                    type="text"
                    className="w-full p-2 border rounded mt-1"
                    value={reviewedBy}
                    onChange={(e) => setReviewedBy(e.target.value)}
                    placeholder="Your name"
                  />
                </div>
                <div>
                  <label className="text-sm font-medium">Review Notes</label>
                  <textarea
                    className="w-full p-2 border rounded mt-1"
                    rows={3}
                    value={reviewNotes}
                    onChange={(e) => setReviewNotes(e.target.value)}
                    placeholder="Add your comments (required for rejection)"
                  />
                </div>
              </div>

              {/* Action Buttons */}
              <div className="flex gap-2 justify-end pt-4 border-t">
                <Button variant="outline" onClick={() => {
                  setSelectedSuggestion(null);
                  setReviewNotes("");
                }}>
                  Cancel
                </Button>
                <Button
                  variant="destructive"
                  onClick={() => handleReject(selectedSuggestion)}
                >
                  Reject
                </Button>
                <Button
                  className="bg-green-600 hover:bg-green-700"
                  onClick={() => handleApprove(selectedSuggestion)}
                >
                  Approve
                </Button>
              </div>
            </CardContent>
          </Card>
        </div>
      )}
    </div>
  );
}
