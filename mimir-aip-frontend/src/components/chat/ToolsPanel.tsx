"use client";

import { useState } from "react";
import { Wrench, ChevronDown, ChevronUp, Zap } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";

interface Tool {
  name: string;
  description: string;
  parameters?: Array<{
    name: string;
    type: string;
    description: string;
    required: boolean;
  }>;
}

interface ToolsPanelProps {
  tools?: Tool[];
}

// Default tools available in Digital Twin context
const DEFAULT_TOOLS: Tool[] = [
  {
    name: "create_scenario",
    description: "Create a new simulation scenario for the digital twin",
    parameters: [
      { name: "scenario_type", type: "string", description: "Type of scenario (supply_disruption, demand_spike, etc.)", required: true },
      { name: "severity", type: "string", description: "Severity level (low, medium, high)", required: true },
      { name: "duration_days", type: "number", description: "Duration in days", required: true },
    ],
  },
  {
    name: "run_simulation",
    description: "Execute a simulation run with the specified parameters",
    parameters: [
      { name: "simulation_type", type: "string", description: "Type of simulation (monte_carlo, deterministic)", required: true },
      { name: "iterations", type: "number", description: "Number of iterations", required: false },
      { name: "time_horizon", type: "number", description: "Simulation time horizon in days", required: true },
    ],
  },
  {
    name: "query_ontology",
    description: "Query the knowledge graph ontology using SPARQL",
    parameters: [
      { name: "query_type", type: "string", description: "Type of query (sparql, natural_language)", required: true },
      { name: "query", type: "string", description: "Query text or SPARQL", required: true },
    ],
  },
  {
    name: "train_model",
    description: "Train a machine learning model on the twin's data",
    parameters: [
      { name: "model_type", type: "string", description: "Model type (decision_tree, random_forest, neural_network)", required: true },
      { name: "target", type: "string", description: "Target variable", required: true },
      { name: "features", type: "array", description: "Feature columns", required: true },
    ],
  },
  {
    name: "analyze_data",
    description: "Perform data analysis and profiling",
    parameters: [
      { name: "dataset", type: "string", description: "Dataset name", required: true },
      { name: "analysis_type", type: "string", description: "Type of analysis (profiling, correlation, outliers)", required: true },
    ],
  },
  {
    name: "create_pipeline",
    description: "Create a data processing pipeline",
    parameters: [
      { name: "pipeline_name", type: "string", description: "Name for the pipeline", required: true },
      { name: "source_type", type: "string", description: "Data source type (csv, api, database)", required: true },
      { name: "schedule", type: "string", description: "Schedule (hourly, daily, weekly)", required: false },
    ],
  },
];

export function ToolsPanel({ tools = DEFAULT_TOOLS }: ToolsPanelProps) {
  const [isExpanded, setIsExpanded] = useState(false);
  const [expandedTool, setExpandedTool] = useState<string | null>(null);

  const toggleTool = (toolName: string) => {
    setExpandedTool(expandedTool === toolName ? null : toolName);
  };

  return (
    <div className="border-b bg-muted/10">
      <button
        onClick={() => setIsExpanded(!isExpanded)}
        className="w-full px-4 py-2 flex items-center justify-between hover:bg-muted/20 transition-colors"
      >
        <div className="flex items-center gap-2">
          <Wrench className="h-4 w-4 text-muted-foreground" />
          <span className="text-sm font-medium">Available Tools</span>
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
        <div className="px-4 pb-3 space-y-2 max-h-64 overflow-y-auto">
          <p className="text-xs text-muted-foreground mb-2">
            Ask the agent to use these tools by mentioning them in your message
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

              {expandedTool === tool.name && tool.parameters && tool.parameters.length > 0 && (
                <div className="px-4 pb-2 space-y-1 border-t bg-muted/5">
                  <p className="text-xs font-medium text-muted-foreground mt-2 mb-1">Parameters:</p>
                  {tool.parameters.map((param) => (
                    <div key={param.name} className="text-xs pl-2">
                      <span className="font-mono font-medium">{param.name}</span>
                      <span className="text-muted-foreground">
                        {' '}({param.type}
                        {param.required && <span className="text-orange-600">*</span>})
                      </span>
                      <p className="text-muted-foreground pl-2">{param.description}</p>
                    </div>
                  ))}
                </div>
              )}
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
