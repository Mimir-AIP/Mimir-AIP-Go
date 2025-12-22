"use client";

import { useState, useEffect } from "react";
import { Wrench, ChevronDown, ChevronUp, Zap, Loader2 } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { getMCPTools, type MCPTool } from "@/lib/api";

interface MCPToolsPanelProps {
  onToolsLoaded?: (tools: MCPTool[]) => void;
}

export function MCPToolsPanel({ onToolsLoaded }: MCPToolsPanelProps) {
  const [tools, setTools] = useState<MCPTool[]>([]);
  const [isExpanded, setIsExpanded] = useState(false);
  const [expandedTool, setExpandedTool] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Fetch MCP tools on mount
  useEffect(() => {
    async function loadTools() {
      try {
        setIsLoading(true);
        const response = await getMCPTools(true);
        setTools(response.tools);
        if (onToolsLoaded) {
          onToolsLoaded(response.tools);
        }
        setError(null);
      } catch (err) {
        console.error('Failed to load MCP tools:', err);
        setError('Failed to load tools');
      } finally {
        setIsLoading(false);
      }
    }

    loadTools();
  }, [onToolsLoaded]);

  const toggleTool = (toolName: string) => {
    setExpandedTool(expandedTool === toolName ? null : toolName);
  };

  const formatParameterType = (param: any): string => {
    if (param.enum) {
      return `${param.type} (${param.enum.join(' | ')})`;
    }
    return param.type;
  };

  return (
    <div className="border-b bg-muted/10">
      <button
        onClick={() => setIsExpanded(!isExpanded)}
        className="w-full px-4 py-2 flex items-center justify-between hover:bg-muted/20 transition-colors"
      >
        <div className="flex items-center gap-2">
          {isLoading ? (
            <Loader2 className="h-4 w-4 text-muted-foreground animate-spin" />
          ) : (
            <Wrench className="h-4 w-4 text-muted-foreground" />
          )}
          <span className="text-sm font-medium">Available Tools (MCP)</span>
          <Badge variant="secondary" className="text-xs">
            {tools.length}
          </Badge>
        </div>
        {isExpanded ? (
          <ChevronUp className="h-4 w-4 text-muted-foreground" />
        ) : (
          <ChevronDown className="h-4 w-4 text-muted-foreground" />
        )}
      </button>

      {isExpanded && (
        <div className="px-4 pb-3 space-y-2 max-h-96 overflow-y-auto">
          {error ? (
            <p className="text-xs text-destructive">{error}</p>
          ) : isLoading ? (
            <p className="text-xs text-muted-foreground">Loading tools...</p>
          ) : (
            <>
              <p className="text-xs text-muted-foreground mb-2">
                These tools are available to the AI agent via the Model Context Protocol (MCP)
              </p>
              
              {tools.map((tool) => (
                <div
                  key={tool.name}
                  className="border rounded-lg bg-background"
                >
                  <button
                    onClick={() => toggleTool(tool.name)}
                    className="w-full p-2 flex items-start gap-2 hover:bg-muted/20 transition-colors rounded-lg text-left"
                  >
                    <Zap className="h-4 w-4 text-primary mt-0.5 flex-shrink-0" />
                    <div className="flex-1 min-w-0">
                      <div className="flex items-center gap-2 mb-1">
                        <code className="text-xs font-mono font-semibold">{tool.name}</code>
                        {expandedTool === tool.name ? (
                          <ChevronUp className="h-3 w-3 text-muted-foreground flex-shrink-0" />
                        ) : (
                          <ChevronDown className="h-3 w-3 text-muted-foreground flex-shrink-0" />
                        )}
                      </div>
                      <p className="text-xs text-muted-foreground">{tool.description}</p>
                    </div>
                  </button>

                  {expandedTool === tool.name && tool.inputSchema && (
                    <div className="px-4 pb-2 space-y-1 border-t bg-muted/5">
                      <p className="text-xs font-medium text-muted-foreground mt-2 mb-1">Input Schema:</p>
                      
                      {tool.inputSchema.required && tool.inputSchema.required.length > 0 && (
                        <p className="text-xs text-muted-foreground pl-2 mb-2">
                          <span className="font-medium">Required:</span> {tool.inputSchema.required.join(', ')}
                        </p>
                      )}
                      
                      <div className="space-y-1">
                        {Object.entries(tool.inputSchema.properties || {}).map(([paramName, param]) => (
                          <div key={paramName} className="text-xs pl-2 py-1 border-l-2 border-primary/30">
                            <div className="flex items-start gap-2">
                              <span className="font-mono font-medium text-primary">{paramName}</span>
                              <span className="text-muted-foreground">
                                ({formatParameterType(param)})
                              </span>
                              {tool.inputSchema.required?.includes(paramName) && (
                                <span className="text-orange-600 font-bold">*</span>
                              )}
                            </div>
                            <p className="text-muted-foreground pl-2 mt-0.5">{param.description}</p>
                            {param.default !== undefined && (
                              <p className="text-muted-foreground pl-2 mt-0.5">
                                <span className="font-medium">Default:</span> {JSON.stringify(param.default)}
                              </p>
                            )}
                          </div>
                        ))}
                      </div>
                    </div>
                  )}
                </div>
              ))}
            </>
          )}
        </div>
      )}
    </div>
  );
}
