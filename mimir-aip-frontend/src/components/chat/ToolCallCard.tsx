"use client";

import { useState } from "react";
import { ChevronDown, ChevronRight, Wrench, Clock } from "lucide-react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import type { ToolCallInfo } from "@/lib/api";

interface ToolCallCardProps {
  toolCalls: ToolCallInfo[];
}

export function ToolCallCard({ toolCalls }: ToolCallCardProps) {
  const [expandedTools, setExpandedTools] = useState<Set<string>>(new Set());

  const toggleTool = (toolId: string) => {
    const newExpanded = new Set(expandedTools);
    if (newExpanded.has(toolId)) {
      newExpanded.delete(toolId);
    } else {
      newExpanded.add(toolId);
    }
    setExpandedTools(newExpanded);
  };

  const getToolColor = (toolName: string): string => {
    if (toolName.includes("create")) return "bg-green-500/10 text-green-700 border-green-500/20";
    if (toolName.includes("delete") || toolName.includes("remove")) return "bg-red-500/10 text-red-700 border-red-500/20";
    if (toolName.includes("query") || toolName.includes("get") || toolName.includes("list")) return "bg-blue-500/10 text-blue-700 border-blue-500/20";
    if (toolName.includes("run") || toolName.includes("execute")) return "bg-purple-500/10 text-purple-700 border-purple-500/20";
    if (toolName.includes("update") || toolName.includes("modify")) return "bg-yellow-500/10 text-yellow-700 border-yellow-500/20";
    return "bg-gray-500/10 text-gray-700 border-gray-500/20";
  };

  if (toolCalls.length === 0) return null;

  return (
    <div className="space-y-2 my-3">
      {toolCalls.map((tool) => {
        const isExpanded = expandedTools.has(tool.id);
        const colorClass = getToolColor(tool.tool_name);

        return (
          <Card key={tool.id} className={`border ${colorClass}`}>
            <CardHeader className="pb-3">
              <div className="flex items-center justify-between">
                <CardTitle className="text-sm font-medium flex items-center gap-2">
                  <Wrench className="h-4 w-4" />
                  <span>Tool: {tool.tool_name}</span>
                </CardTitle>
                <div className="flex items-center gap-2">
                  <Badge variant="outline" className="text-xs flex items-center gap-1">
                    <Clock className="h-3 w-3" />
                    {tool.duration_ms}ms
                  </Badge>
                  <button
                    onClick={() => toggleTool(tool.id)}
                    className="text-muted-foreground hover:text-foreground transition-colors"
                  >
                    {isExpanded ? (
                      <ChevronDown className="h-4 w-4" />
                    ) : (
                      <ChevronRight className="h-4 w-4" />
                    )}
                  </button>
                </div>
              </div>
            </CardHeader>

            {isExpanded && (
              <CardContent className="pt-0 space-y-3">
                <div>
                  <p className="text-xs font-semibold text-muted-foreground mb-1">Input:</p>
                  <pre className="bg-muted/50 rounded p-2 text-xs overflow-x-auto">
                    {JSON.stringify(tool.input, null, 2)}
                  </pre>
                </div>

                <div>
                  <p className="text-xs font-semibold text-muted-foreground mb-1">Output:</p>
                  <pre className="bg-muted/50 rounded p-2 text-xs overflow-x-auto">
                    {JSON.stringify(tool.output, null, 2)}
                  </pre>
                </div>
              </CardContent>
            )}
          </Card>
        );
      })}
    </div>
  );
}
