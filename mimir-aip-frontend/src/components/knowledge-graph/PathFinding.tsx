"use client";

import { useState } from "react";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import { toast } from "sonner";
import {
  GitBranch,
  ArrowRight,
  Loader2,
  Search,
  Route,
  Sparkles
} from "lucide-react";

interface PathNode {
  uri: string;
  label?: string;
  type: string;
}

interface PathEdge {
  property: string;
  label?: string;
}

interface Path {
  nodes: PathNode[];
  edges: PathEdge[];
  length: number;
  weight?: number;
}

interface PathFindingResult {
  source: PathNode;
  target: PathNode;
  paths: Path[];
  execution_time_ms: number;
  max_depth: number;
}

interface PathFindingProps {
  ontologyId?: string;
}

export function PathFinding({ ontologyId }: PathFindingProps) {
  const [sourceUri, setSourceUri] = useState("");
  const [targetUri, setTargetUri] = useState("");
  const [maxDepth, setMaxDepth] = useState(5);
  const [maxPaths, setMaxPaths] = useState(3);
  const [loading, setLoading] = useState(false);
  const [result, setResult] = useState<PathFindingResult | null>(null);

  async function handleFindPaths() {
    if (!sourceUri.trim() || !targetUri.trim()) {
      toast.error("Please enter both source and target URIs");
      return;
    }

    setLoading(true);
    try {
      const response = await fetch("/api/v1/knowledge-graph/path-finding", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          source: sourceUri.trim(),
          target: targetUri.trim(),
          max_depth: maxDepth,
          max_paths: maxPaths,
          ontology_id: ontologyId,
        }),
      });

      if (!response.ok) {
        throw new Error(`HTTP ${response.status}: ${await response.text()}`);
      }

      const data = await response.json();
      setResult(data);
      toast.success(`Found ${data.paths.length} path(s)`);
    } catch (err) {
      const message = err instanceof Error ? err.message : "Failed to find paths";
      toast.error(message);
      setResult(null);
    } finally {
      setLoading(false);
    }
  }

  function getNodeLabel(node: PathNode): string {
    return node.label || node.uri.split(/[/#]/).pop() || node.uri;
  }

  function getPropertyLabel(edge: PathEdge): string {
    return edge.label || edge.property.split(/[/#]/).pop() || edge.property;
  }

  return (
    <div className="space-y-6">
      {/* Input Card */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Route className="h-5 w-5" />
            Find Paths Between Entities
          </CardTitle>
          <CardDescription>
            Discover relationships and connections between two entities in the knowledge graph
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <label className="text-sm font-medium">Source URI</label>
              <Input
                placeholder="e.g., http://example.org/Person123"
                value={sourceUri}
                onChange={(e) => setSourceUri(e.target.value)}
              />
            </div>
            <div className="space-y-2">
              <label className="text-sm font-medium">Target URI</label>
              <Input
                placeholder="e.g., http://example.org/Organization456"
                value={targetUri}
                onChange={(e) => setTargetUri(e.target.value)}
              />
            </div>
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <label className="text-sm font-medium">Max Depth</label>
              <Input
                type="number"
                min="1"
                max="10"
                value={maxDepth}
                onChange={(e) => setMaxDepth(parseInt(e.target.value) || 5)}
              />
            </div>
            <div className="space-y-2">
              <label className="text-sm font-medium">Max Paths</label>
              <Input
                type="number"
                min="1"
                max="10"
                value={maxPaths}
                onChange={(e) => setMaxPaths(parseInt(e.target.value) || 3)}
              />
            </div>
          </div>

          <Button
            onClick={handleFindPaths}
            disabled={loading || !sourceUri.trim() || !targetUri.trim()}
            className="w-full"
          >
            {loading ? (
              <>
                <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                Finding Paths...
              </>
            ) : (
              <>
                <Search className="h-4 w-4 mr-2" />
                Find Paths
              </>
            )}
          </Button>
        </CardContent>
      </Card>

      {/* Results */}
      {result && (
        <div className="space-y-4">
          {/* Summary */}
          <Card>
            <CardContent className="pt-6">
              <div className="flex items-center justify-between">
                <div className="space-y-1">
                  <p className="text-sm text-muted-foreground">Found</p>
                  <p className="text-2xl font-bold">{result.paths.length} path(s)</p>
                </div>
                <div className="space-y-1 text-right">
                  <p className="text-sm text-muted-foreground">Execution Time</p>
                  <p className="text-lg font-medium">{result.execution_time_ms}ms</p>
                </div>
                <div className="space-y-1 text-right">
                  <p className="text-sm text-muted-foreground">Max Depth</p>
                  <p className="text-lg font-medium">{result.max_depth}</p>
                </div>
              </div>
            </CardContent>
          </Card>

          {/* Paths */}
          {result.paths.length === 0 ? (
            <Card>
              <CardContent className="pt-6 text-center py-12">
                <GitBranch className="h-12 w-12 mx-auto text-muted-foreground mb-4" />
                <h3 className="text-lg font-semibold mb-2">No Paths Found</h3>
                <p className="text-muted-foreground">
                  No connection found between these entities within the specified depth.
                  Try increasing the max depth or checking the URIs.
                </p>
              </CardContent>
            </Card>
          ) : (
            result.paths.map((path, index) => (
              <Card key={index} className="animate-in fade-in slide-in-from-bottom-4" style={{ animationDelay: `${index * 100}ms` }}>
                <CardHeader>
                  <div className="flex items-center justify-between">
                    <CardTitle className="text-base">Path #{index + 1}</CardTitle>
                    <div className="flex items-center gap-2">
                      <Badge variant="secondary">
                        {path.length} hop{path.length !== 1 ? 's' : ''}
                      </Badge>
                      {path.weight && (
                        <Badge variant="outline">
                          Weight: {path.weight.toFixed(2)}
                        </Badge>
                      )}
                    </div>
                  </div>
                </CardHeader>
                <CardContent>
                  <div className="flex flex-col gap-3">
                    {path.nodes.map((node, nodeIndex) => (
                      <div key={nodeIndex}>
                        {/* Node */}
                        <div className="flex items-center gap-3 p-3 rounded-lg bg-muted/50">
                          <div className="flex-shrink-0">
                            <div className={`h-8 w-8 rounded-full flex items-center justify-center ${
                              nodeIndex === 0 ? 'bg-blue-500 text-white' :
                              nodeIndex === path.nodes.length - 1 ? 'bg-green-500 text-white' :
                              'bg-purple-500 text-white'
                            }`}>
                              {nodeIndex === 0 ? 'S' : nodeIndex === path.nodes.length - 1 ? 'T' : nodeIndex}
                            </div>
                          </div>
                          <div className="flex-1 min-w-0">
                            <p className="font-medium truncate">{getNodeLabel(node)}</p>
                            <p className="text-xs text-muted-foreground truncate">{node.uri}</p>
                          </div>
                          <Badge variant="outline" className="flex-shrink-0">
                            {node.type}
                          </Badge>
                        </div>

                        {/* Edge */}
                        {nodeIndex < path.edges.length && (
                          <div className="flex items-center gap-2 ml-4 my-2">
                            <ArrowRight className="h-4 w-4 text-muted-foreground" />
                            <span className="text-sm font-medium text-primary">
                              {getPropertyLabel(path.edges[nodeIndex])}
                            </span>
                            <ArrowRight className="h-4 w-4 text-muted-foreground" />
                          </div>
                        )}
                      </div>
                    ))}
                  </div>
                </CardContent>
              </Card>
            ))
          )}
        </div>
      )}
    </div>
  );
}
