package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/Mimir-AIP/Mimir-AIP-Go/pipelines/KnowledgeGraph"
	"github.com/Mimir-AIP/Mimir-AIP-Go/utils"
)

// PathFindingRequest represents a path-finding request
type PathFindingRequest struct {
	Source     string `json:"source"`
	Target     string `json:"target"`
	MaxDepth   int    `json:"max_depth"`
	MaxPaths   int    `json:"max_paths"`
	OntologyID string `json:"ontology_id,omitempty"`
}

// PathNode represents a node in a path
type PathNode struct {
	URI   string `json:"uri"`
	Label string `json:"label,omitempty"`
	Type  string `json:"type"`
}

// PathEdge represents an edge in a path
type PathEdge struct {
	Property string `json:"property"`
	Label    string `json:"label,omitempty"`
}

// Path represents a path between two nodes
type Path struct {
	Nodes  []PathNode `json:"nodes"`
	Edges  []PathEdge `json:"edges"`
	Length int        `json:"length"`
	Weight float64    `json:"weight,omitempty"`
}

// PathFindingResponse represents the response
type PathFindingResponse struct {
	Source          PathNode `json:"source"`
	Target          PathNode `json:"target"`
	Paths           []Path   `json:"paths"`
	ExecutionTimeMS int64    `json:"execution_time_ms"`
	MaxDepth        int      `json:"max_depth"`
}

// handlePathFinding finds paths between two entities in the knowledge graph
func (s *Server) handlePathFinding(w http.ResponseWriter, r *http.Request) {
	if s.tdb2Backend == nil {
		http.Error(w, "Knowledge graph features are not available", http.StatusServiceUnavailable)
		return
	}

	startTime := time.Now()
	logger := utils.GetLogger()

	var req PathFindingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
		return
	}

	// Validate inputs
	if req.Source == "" || req.Target == "" {
		http.Error(w, "Source and target URIs are required", http.StatusBadRequest)
		return
	}

	if req.MaxDepth <= 0 {
		req.MaxDepth = 5
	}
	if req.MaxDepth > 10 {
		req.MaxDepth = 10
	}

	if req.MaxPaths <= 0 {
		req.MaxPaths = 3
	}

	ctx := context.Background()

	// Find paths using SPARQL property paths
	paths, err := s.findPathsSPARQL(ctx, req.Source, req.Target, req.MaxDepth, req.MaxPaths, req.OntologyID)
	if err != nil {
		logger.Error("Failed to find paths", err, utils.Component("path-finding"))
		http.Error(w, fmt.Sprintf("Failed to find paths: %v", err), http.StatusInternalServerError)
		return
	}

	// Get labels for source and target
	sourceLabel := s.getEntityLabel(ctx, req.Source)
	targetLabel := s.getEntityLabel(ctx, req.Target)

	response := PathFindingResponse{
		Source: PathNode{
			URI:   req.Source,
			Type:  "entity",
			Label: sourceLabel,
		},
		Target: PathNode{
			URI:   req.Target,
			Type:  "entity",
			Label: targetLabel,
		},
		Paths:           paths,
		ExecutionTimeMS: time.Since(startTime).Milliseconds(),
		MaxDepth:        req.MaxDepth,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// findPathsSPARQL finds paths between two entities using SPARQL property paths
func (s *Server) findPathsSPARQL(ctx context.Context, source, target string, maxDepth, maxPaths int, ontologyID string) ([]Path, error) {
	logger := utils.GetLogger()
	var allPaths []Path

	// Try different path lengths from 1 to maxDepth
	for depth := 1; depth <= maxDepth && len(allPaths) < maxPaths; depth++ {
		paths, err := s.findPathsAtDepth(ctx, source, target, depth, ontologyID)
		if err != nil {
			logger.Warn("Failed to find paths at depth", utils.Component("path-finding"))
			continue
		}
		allPaths = append(allPaths, paths...)
		if len(allPaths) >= maxPaths {
			allPaths = allPaths[:maxPaths]
			break
		}
	}

	logger.Info("Path finding completed", utils.Component("path-finding"))

	return allPaths, nil
}

// findPathsAtDepth finds paths of a specific length
func (s *Server) findPathsAtDepth(ctx context.Context, source, target string, depth int, ontologyID string) ([]Path, error) {
	// Build GRAPH clause
	var graphClause string
	if ontologyID != "" {
		graphURI := fmt.Sprintf("http://mimir.ai/ontology/%s", ontologyID)
		graphClause = fmt.Sprintf("  GRAPH <%s> {\n", graphURI)
	} else {
		graphClause = "  GRAPH ?g {\n"
	}

	// Build SPARQL query with intermediate nodes
	var intermediateNodes strings.Builder
	var selectVars strings.Builder

	for i := 1; i <= depth; i++ {
		selectVars.WriteString(fmt.Sprintf(" ?p%d", i))
		if i < depth {
			selectVars.WriteString(fmt.Sprintf(" ?n%d", i))
		}

		if i == 1 {
			intermediateNodes.WriteString(fmt.Sprintf("    <%s> ?p%d ?n%d .\n", source, i, i))
		} else if i == depth {
			intermediateNodes.WriteString(fmt.Sprintf("    ?n%d ?p%d <%s> .\n", i-1, i, target))
		} else {
			intermediateNodes.WriteString(fmt.Sprintf("    ?n%d ?p%d ?n%d .\n", i-1, i, i))
		}
	}

	closeGraphClause := ""
	if graphClause != "" {
		closeGraphClause = "  }"
	}

	query := fmt.Sprintf(`
PREFIX rdf: <http://www.w3.org/1999/02/22-rdf-syntax-ns#>
PREFIX rdfs: <http://www.w3.org/2000/01/rdf-schema#>

SELECT DISTINCT%s
WHERE {
%s%s
%s}
LIMIT 10
`, selectVars.String(), graphClause, intermediateNodes.String(), closeGraphClause)

	result, err := s.tdb2Backend.QuerySPARQL(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("SPARQL query failed: %w", err)
	}

	// Parse results into Path structures
	var paths []Path
	for _, binding := range result.Bindings {
		path := s.parsePathFromBinding(ctx, binding, source, target, depth)
		if len(path.Nodes) > 0 {
			paths = append(paths, path)
		}
	}

	return paths, nil
}

// parsePathFromBinding converts SPARQL binding to Path structure
func (s *Server) parsePathFromBinding(ctx context.Context, binding knowledgegraph.BindingRow, source, target string, depth int) Path {
	path := Path{
		Nodes:  []PathNode{},
		Edges:  []PathEdge{},
		Length: depth,
		Weight: 1.0,
	}

	// Add source node
	path.Nodes = append(path.Nodes, PathNode{
		URI:   source,
		Label: s.getEntityLabel(ctx, source),
		Type:  "entity",
	})

	// Add intermediate nodes and edges
	for i := 1; i <= depth; i++ {
		// Add edge
		predKey := fmt.Sprintf("p%d", i)
		if predVal, ok := binding[predKey]; ok {
			path.Edges = append(path.Edges, PathEdge{
				Property: predVal.Value,
				Label:    s.getPropertyLabel(ctx, predVal.Value),
			})
		}

		// Add intermediate node (if not last)
		if i < depth {
			nodeKey := fmt.Sprintf("n%d", i)
			if nodeVal, ok := binding[nodeKey]; ok {
				path.Nodes = append(path.Nodes, PathNode{
					URI:   nodeVal.Value,
					Label: s.getEntityLabel(ctx, nodeVal.Value),
					Type:  "entity",
				})
			}
		}
	}

	// Add target node
	path.Nodes = append(path.Nodes, PathNode{
		URI:   target,
		Label: s.getEntityLabel(ctx, target),
		Type:  "entity",
	})

	return path
}

// getEntityLabel retrieves label for an entity
func (s *Server) getEntityLabel(ctx context.Context, uri string) string {
	query := fmt.Sprintf(`
PREFIX rdfs: <http://www.w3.org/2000/01/rdf-schema#>
PREFIX skos: <http://www.w3.org/2004/02/skos/core#>

SELECT ?label WHERE {
  GRAPH ?g {
    <%s> rdfs:label|skos:prefLabel ?label .
  }
}
LIMIT 1
`, uri)

	result, err := s.tdb2Backend.QuerySPARQL(ctx, query)
	if err == nil && len(result.Bindings) > 0 {
		if labelVal, ok := result.Bindings[0]["label"]; ok {
			return labelVal.Value
		}
	}

	// Fallback to local name
	return extractLocalName(uri)
}

// getPropertyLabel retrieves label for a property
func (s *Server) getPropertyLabel(ctx context.Context, uri string) string {
	return s.getEntityLabel(ctx, uri)
}

// extractLocalName extracts the local name from a URI
func extractLocalName(uri string) string {
	parts := strings.Split(strings.TrimRight(uri, "/#"), "/")
	if len(parts) > 0 {
		localName := parts[len(parts)-1]
		if localName != "" {
			return localName
		}
	}
	parts = strings.Split(uri, "#")
	if len(parts) > 1 {
		return parts[len(parts)-1]
	}
	return uri
}
