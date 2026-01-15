"use client";

import { useEffect, useRef, useState } from "react";
import * as d3 from "d3";

interface GraphNode extends d3.SimulationNodeDatum {
  id: string;
  label: string;
  type: "subject" | "object" | "class" | "property";
  uri: string;
}

interface GraphLink {
  source: string;
  target: string;
  label: string;
  uri: string;
}

interface GraphData {
  nodes: GraphNode[];
  links: GraphLink[];
}

interface GraphVisualizationProps {
  data: GraphData;
  width?: number;
  height?: number;
}

export function GraphVisualization({
  data,
  width = 1200,
  height = 600,
}: GraphVisualizationProps) {
  const svgRef = useRef<SVGSVGElement>(null);
  const [selectedNode, setSelectedNode] = useState<GraphNode | null>(null);
  const [zoom, setZoom] = useState(1);

  useEffect(() => {
    if (!svgRef.current || !data.nodes.length) return;

    // Clear previous content
    d3.select(svgRef.current).selectAll("*").remove();

    // Create SVG container
    const svg = d3
      .select(svgRef.current)
      .attr("width", width)
      .attr("height", height)
      .attr("viewBox", `0 0 ${width} ${height}`)
      .attr("preserveAspectRatio", "xMidYMid meet");

    // Add zoom behavior
    const g = svg.append("g");
    const zoomBehavior = d3
      .zoom<SVGSVGElement, unknown>()
      .scaleExtent([0.1, 10])
      .on("zoom", (event) => {
        g.attr("transform", event.transform);
        setZoom(event.transform.k);
      });

    svg.call(zoomBehavior);

    // Create force simulation
    const simulation = d3
      .forceSimulation<GraphNode>(data.nodes)
      .force(
        "link",
        d3
          .forceLink<GraphNode, GraphLink>(data.links)
          .id((d) => d.id)
          .distance(150)
      )
      .force("charge", d3.forceManyBody().strength(-300))
      .force("center", d3.forceCenter(width / 2, height / 2))
      .force("collision", d3.forceCollide().radius(40));

    // Define arrow markers
    const defs = g.append("defs");
    
    defs
      .append("marker")
      .attr("id", "arrowhead")
      .attr("markerWidth", 10)
      .attr("markerHeight", 10)
      .attr("refX", 25)
      .attr("refY", 5)
      .attr("orient", "auto")
      .append("polygon")
      .attr("points", "0 0, 10 5, 0 10")
      .attr("fill", "#94a3b8");

    // Create links
    const link = g
      .append("g")
      .selectAll("line")
      .data(data.links)
      .enter()
      .append("line")
      .attr("stroke", "#64748b")
      .attr("stroke-width", 2)
      .attr("marker-end", "url(#arrowhead)");

    // Create link labels
    const linkLabel = g
      .append("g")
      .selectAll("text")
      .data(data.links)
      .enter()
      .append("text")
      .attr("class", "link-label")
      .attr("font-size", "10px")
      .attr("fill", "#94a3b8")
      .attr("text-anchor", "middle")
      .text((d) => d.label);

    // Create node groups
    const node = g
      .append("g")
      .selectAll("g")
      .data(data.nodes)
      .enter()
      .append("g")
      .call(
        d3
          .drag<SVGGElement, GraphNode>()
          .on("start", (event, d: GraphNode) => {
            if (!event.active) simulation.alphaTarget(0.3).restart();
            d.fx = d.x;
            d.fy = d.y;
          })
          .on("drag", (event, d: GraphNode) => {
            d.fx = event.x;
            d.fy = event.y;
          })
          .on("end", (event, d: GraphNode) => {
            if (!event.active) simulation.alphaTarget(0);
            d.fx = null;
            d.fy = null;
          })
      );

    // Add circles to nodes
    node
      .append("circle")
      .attr("r", 20)
      .attr("fill", (d) => {
        switch (d.type) {
          case "class":
            return "#3b82f6"; // blue
          case "property":
            return "#a855f7"; // purple
          case "subject":
            return "#10b981"; // green
          case "object":
            return "#f59e0b"; // orange
          default:
            return "#6b7280"; // gray
        }
      })
      .attr("stroke", "#fff")
      .attr("stroke-width", 2)
      .style("cursor", "pointer")
      .on("click", (event, d) => {
        event.stopPropagation();
        setSelectedNode(d);
      });

    // Add labels to nodes
    node
      .append("text")
      .attr("dy", 35)
      .attr("text-anchor", "middle")
      .attr("font-size", "12px")
      .attr("fill", "#e2e8f0")
      .text((d) => d.label.substring(0, 20) + (d.label.length > 20 ? "..." : ""));

    // Add node type indicators
    node
      .append("text")
      .attr("dy", -25)
      .attr("text-anchor", "middle")
      .attr("font-size", "8px")
      .attr("fill", "#94a3b8")
      .text((d) => d.type.toUpperCase());

    // Update positions on tick
    simulation.on("tick", () => {
      link
        .attr("x1", (d: GraphLink) => {
          const source = data.nodes.find((n) => n.id === d.source);
          return source?.x || 0;
        })
        .attr("y1", (d: GraphLink) => {
          const source = data.nodes.find((n) => n.id === d.source);
          return source?.y || 0;
        })
        .attr("x2", (d: GraphLink) => {
          const target = data.nodes.find((n) => n.id === d.target);
          return target?.x || 0;
        })
        .attr("y2", (d: GraphLink) => {
          const target = data.nodes.find((n) => n.id === d.target);
          return target?.y || 0;
        });

      linkLabel
        .attr("x", (d: GraphLink) => {
          const source = data.nodes.find((n) => n.id === d.source);
          const target = data.nodes.find((n) => n.id === d.target);
          return ((source?.x || 0) + (target?.x || 0)) / 2;
        })
        .attr("y", (d: GraphLink) => {
          const source = data.nodes.find((n) => n.id === d.source);
          const target = data.nodes.find((n) => n.id === d.target);
          return ((source?.y || 0) + (target?.y || 0)) / 2;
        });

      node.attr("transform", (d) => {
        return `translate(${d.x || 0},${d.y || 0})`;
      });
    });

    // Click outside to deselect
    svg.on("click", () => setSelectedNode(null));

    return () => {
      simulation.stop();
    };
  }, [data, width, height]);

  const extractLocalName = (uri: string) => {
    const parts = uri.split(/[/#]/);
    return parts[parts.length - 1] || uri;
  };

  return (
    <div className="relative">
      <svg
        ref={svgRef}
        className="border border-gray-600 rounded bg-navy"
        style={{ width: "100%", height: "600px" }}
      />

      {/* Controls */}
      <div className="absolute top-4 right-4 bg-blue border border-gray-600 rounded-lg p-3 shadow-lg">
        <div className="text-xs text-gray-300 mb-2">Zoom: {(zoom * 100).toFixed(0)}%</div>
        <div className="space-y-1 text-xs text-gray-400">
          <div>• Drag nodes to reposition</div>
          <div>• Click node for details</div>
          <div>• Scroll to zoom</div>
        </div>
      </div>

      {/* Legend */}
      <div className="absolute top-4 left-4 bg-blue border border-gray-600 rounded-lg p-3 shadow-lg">
        <div className="text-xs font-semibold text-white mb-2">Legend</div>
        <div className="space-y-1 text-xs">
          <div className="flex items-center gap-2">
            <div className="w-3 h-3 rounded-full bg-blue-500"></div>
            <span className="text-gray-300">Class</span>
          </div>
          <div className="flex items-center gap-2">
            <div className="w-3 h-3 rounded-full bg-purple-500"></div>
            <span className="text-gray-300">Property</span>
          </div>
          <div className="flex items-center gap-2">
            <div className="w-3 h-3 rounded-full bg-green-500"></div>
            <span className="text-gray-300">Subject</span>
          </div>
          <div className="flex items-center gap-2">
            <div className="w-3 h-3 rounded-full bg-orange-500"></div>
            <span className="text-gray-300">Object</span>
          </div>
        </div>
      </div>

      {/* Node Details Panel */}
      {selectedNode && (
        <div className="absolute bottom-4 left-4 right-4 bg-blue border border-gray-600 rounded-lg p-4 shadow-lg max-w-md">
          <div className="flex justify-between items-start mb-2">
            <h3 className="font-semibold text-white">Node Details</h3>
            <button
              onClick={() => setSelectedNode(null)}
              className="text-gray-400 hover:text-white"
            >
              ✕
            </button>
          </div>
          <div className="space-y-2 text-sm">
            <div>
              <span className="text-gray-400">Label:</span>{" "}
              <span className="text-white">{selectedNode.label}</span>
            </div>
            <div>
              <span className="text-gray-400">Type:</span>{" "}
              <span className="text-white capitalize">{selectedNode.type}</span>
            </div>
            <div>
              <span className="text-gray-400">URI:</span>{" "}
              <span className="text-white font-mono text-xs break-all">
                {selectedNode.uri}
              </span>
            </div>
            <div>
              <span className="text-gray-400">Local Name:</span>{" "}
              <span className="text-white">{extractLocalName(selectedNode.uri)}</span>
            </div>
          </div>
        </div>
      )}

      {/* No Data Message */}
      {data.nodes.length === 0 && (
        <div className="absolute inset-0 flex items-center justify-center">
          <div className="text-center text-gray-400">
            <p className="text-lg mb-2">No graph data available</p>
            <p className="text-sm">
              Execute a CONSTRUCT query or SELECT query to visualize relationships
            </p>
          </div>
        </div>
      )}
    </div>
  );
}
