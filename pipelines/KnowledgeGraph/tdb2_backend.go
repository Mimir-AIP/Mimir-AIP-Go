package knowledgegraph

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// TDB2Backend manages communication with Jena Fuseki TDB2 triplestore
type TDB2Backend struct {
	baseURL    string
	httpClient *http.Client
	dataset    string
}

// NewTDB2Backend creates a new TDB2 backend client
func NewTDB2Backend(fusekiURL, dataset string) *TDB2Backend {
	return &TDB2Backend{
		baseURL:    fusekiURL,
		dataset:    dataset,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// Health checks if Fuseki is accessible
func (t *TDB2Backend) Health(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", t.baseURL+"/$/ping", nil)
	if err != nil {
		return fmt.Errorf("failed to create health check request: %w", err)
	}

	resp, err := t.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("fuseki health check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("fuseki health check returned status %d", resp.StatusCode)
	}

	return nil
}

// InsertTriples inserts RDF triples into the knowledge graph
func (t *TDB2Backend) InsertTriples(ctx context.Context, triples []Triple) error {
	if len(triples) == 0 {
		return nil
	}

	// Build SPARQL INSERT DATA query
	var sb strings.Builder
	sb.WriteString("INSERT DATA {\n")

	for _, triple := range triples {
		if triple.Graph != "" {
			sb.WriteString(fmt.Sprintf("  GRAPH <%s> {\n", triple.Graph))
		}
		sb.WriteString(fmt.Sprintf("    <%s> <%s> ", triple.Subject, triple.Predicate))

		// Handle object (could be URI or literal)
		if strings.HasPrefix(triple.Object, "http://") || strings.HasPrefix(triple.Object, "https://") {
			sb.WriteString(fmt.Sprintf("<%s>", triple.Object))
		} else {
			// Escape quotes in literals
			escaped := strings.ReplaceAll(triple.Object, `"`, `\"`)
			sb.WriteString(fmt.Sprintf(`"%s"`, escaped))
		}
		sb.WriteString(" .\n")

		if triple.Graph != "" {
			sb.WriteString("  }\n")
		}
	}
	sb.WriteString("}\n")

	sparqlQuery := sb.String()

	err := t.ExecuteUpdate(ctx, sparqlQuery)
	if err != nil {
		fmt.Printf("ERROR TDB2: ExecuteUpdate failed: %v\n", err)
	}
	return err
}

// ExecuteUpdate executes a SPARQL UPDATE query
func (t *TDB2Backend) ExecuteUpdate(ctx context.Context, updateQuery string) error {
	endpoint := fmt.Sprintf("%s/%s/update", t.baseURL, t.dataset)

	data := url.Values{}
	data.Set("update", updateQuery)

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create update request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := t.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute update: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("update failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// QuerySPARQL executes a SPARQL SELECT query
func (t *TDB2Backend) QuerySPARQL(ctx context.Context, query string) (*QueryResult, error) {
	endpoint := fmt.Sprintf("%s/%s/query", t.baseURL, t.dataset)

	data := url.Values{}
	data.Set("query", query)

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create query request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/sparql-results+json")

	start := time.Now()
	resp, err := t.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("query failed with status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Parse SPARQL JSON results format
	var sparqlResult struct {
		Head struct {
			Vars []string `json:"vars"`
		} `json:"head"`
		Results struct {
			Bindings []map[string]struct {
				Type     string `json:"type"`
				Value    string `json:"value"`
				Datatype string `json:"datatype,omitempty"`
				Lang     string `json:"xml:lang,omitempty"`
			} `json:"bindings"`
		} `json:"results"`
		Boolean *bool `json:"boolean,omitempty"` // For ASK queries
	}

	if err := json.Unmarshal(body, &sparqlResult); err != nil {
		return nil, fmt.Errorf("failed to parse query results: %w", err)
	}

	// Convert to our QueryResult format
	result := &QueryResult{
		Variables: sparqlResult.Head.Vars,
		Bindings:  make([]BindingRow, 0, len(sparqlResult.Results.Bindings)),
		Duration:  time.Since(start),
		Boolean:   sparqlResult.Boolean,
	}

	// Determine query type
	if sparqlResult.Boolean != nil {
		result.QueryType = string(QueryTypeAsk)
	} else if strings.Contains(strings.ToUpper(query), "SELECT") {
		result.QueryType = string(QueryTypeSelect)
	} else if strings.Contains(strings.ToUpper(query), "CONSTRUCT") {
		result.QueryType = string(QueryTypeConstruct)
	} else if strings.Contains(strings.ToUpper(query), "DESCRIBE") {
		result.QueryType = string(QueryTypeDescribe)
	}

	for _, binding := range sparqlResult.Results.Bindings {
		row := make(BindingRow)
		for varName, value := range binding {
			row[varName] = BindingValue{
				Type:     value.Type,
				Value:    value.Value,
				Datatype: value.Datatype,
				Lang:     value.Lang,
			}
		}
		result.Bindings = append(result.Bindings, row)
	}

	return result, nil
}

// Stats retrieves statistics about the knowledge graph
func (t *TDB2Backend) Stats(ctx context.Context) (*GraphStats, error) {
	// Query to count triples
	countQuery := `
		SELECT (COUNT(*) AS ?count) WHERE {
			?s ?p ?o
		}
	`

	result, err := t.QuerySPARQL(ctx, countQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to count triples: %w", err)
	}

	stats := &GraphStats{
		LastUpdated: time.Now(),
	}

	if len(result.Bindings) > 0 {
		if countVal, ok := result.Bindings[0]["count"]; ok {
			fmt.Sscanf(countVal.Value, "%d", &stats.TotalTriples)
		}
	}

	// Query to count unique subjects
	subjectsQuery := `
		SELECT (COUNT(DISTINCT ?s) AS ?count) WHERE {
			?s ?p ?o
		}
	`

	result, err = t.QuerySPARQL(ctx, subjectsQuery)
	if err == nil && len(result.Bindings) > 0 {
		if countVal, ok := result.Bindings[0]["count"]; ok {
			fmt.Sscanf(countVal.Value, "%d", &stats.TotalSubjects)
		}
	}

	// Query to count unique predicates
	predicatesQuery := `
		SELECT (COUNT(DISTINCT ?p) AS ?count) WHERE {
			?s ?p ?o
		}
	`

	result, err = t.QuerySPARQL(ctx, predicatesQuery)
	if err == nil && len(result.Bindings) > 0 {
		if countVal, ok := result.Bindings[0]["count"]; ok {
			fmt.Sscanf(countVal.Value, "%d", &stats.TotalPredicates)
		}
	}

	// Query to list named graphs
	graphsQuery := `
		SELECT DISTINCT ?g WHERE {
			GRAPH ?g { ?s ?p ?o }
		}
	`

	result, err = t.QuerySPARQL(ctx, graphsQuery)
	if err == nil {
		stats.NamedGraphs = make([]string, 0, len(result.Bindings))
		for _, binding := range result.Bindings {
			if graphVal, ok := binding["g"]; ok {
				stats.NamedGraphs = append(stats.NamedGraphs, graphVal.Value)
			}
		}
	}

	return stats, nil
}

// DeleteTriples deletes triples matching a pattern
func (t *TDB2Backend) DeleteTriples(ctx context.Context, subject, predicate, object string) error {
	// Build SPARQL DELETE WHERE query
	var sb strings.Builder
	sb.WriteString("DELETE WHERE {\n")

	// Use variables or concrete values
	if subject == "" {
		sb.WriteString("  ?s")
	} else {
		sb.WriteString(fmt.Sprintf("  <%s>", subject))
	}

	if predicate == "" {
		sb.WriteString(" ?p")
	} else {
		sb.WriteString(fmt.Sprintf(" <%s>", predicate))
	}

	if object == "" {
		sb.WriteString(" ?o")
	} else {
		if strings.HasPrefix(object, "http://") || strings.HasPrefix(object, "https://") {
			sb.WriteString(fmt.Sprintf(" <%s>", object))
		} else {
			escaped := strings.ReplaceAll(object, `"`, `\"`)
			sb.WriteString(fmt.Sprintf(` "%s"`, escaped))
		}
	}

	sb.WriteString(" .\n}\n")

	return t.ExecuteUpdate(ctx, sb.String())
}

// ClearGraph clears all triples from a named graph
func (t *TDB2Backend) ClearGraph(ctx context.Context, graphURI string) error {
	query := fmt.Sprintf("CLEAR GRAPH <%s>", graphURI)
	return t.ExecuteUpdate(ctx, query)
}

// LoadOntology loads an ontology file into a named graph
func (t *TDB2Backend) LoadOntology(ctx context.Context, graphURI, ontologyData, format string) error {
	endpoint := fmt.Sprintf("%s/%s/data", t.baseURL, t.dataset)

	// Add graph parameter
	endpoint += fmt.Sprintf("?graph=%s", url.QueryEscape(graphURI))

	// Determine content type based on format
	contentType := "text/turtle"
	switch format {
	case "rdfxml":
		contentType = "application/rdf+xml"
	case "ntriples":
		contentType = "application/n-triples"
	case "jsonld":
		contentType = "application/ld+json"
	}

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader([]byte(ontologyData)))
	if err != nil {
		return fmt.Errorf("failed to create load request: %w", err)
	}

	req.Header.Set("Content-Type", contentType)

	resp, err := t.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to load ontology: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("load ontology failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// GetSubgraph retrieves a subgraph for visualization
func (t *TDB2Backend) GetSubgraph(ctx context.Context, rootURI string, depth int) (*GraphVisualization, error) {
	if depth <= 0 {
		depth = 1
	}
	if depth > 3 {
		depth = 3 // Limit depth to prevent large queries
	}

	// Query for nodes and edges around the root URI
	query := fmt.Sprintf(`
		SELECT ?s ?p ?o WHERE {
			{
				<%s> ?p ?o .
				BIND(<%s> AS ?s)
			}
			UNION
			{
				?s ?p <%s> .
				BIND(<%s> AS ?o)
			}
		}
		LIMIT 100
	`, rootURI, rootURI, rootURI, rootURI)

	result, err := t.QuerySPARQL(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve subgraph: %w", err)
	}

	viz := &GraphVisualization{
		Nodes: make([]GraphNode, 0),
		Edges: make([]GraphEdge, 0),
	}

	nodeMap := make(map[string]bool)

	for _, binding := range result.Bindings {
		subject := binding["s"].Value
		predicate := binding["p"].Value
		object := binding["o"].Value

		// Add subject node
		if !nodeMap[subject] {
			viz.Nodes = append(viz.Nodes, GraphNode{
				URI:   subject,
				Label: extractLabel(subject),
				Type:  "resource",
			})
			nodeMap[subject] = true
		}

		// Add object node if it's a URI
		if binding["o"].Type == "uri" && !nodeMap[object] {
			viz.Nodes = append(viz.Nodes, GraphNode{
				URI:   object,
				Label: extractLabel(object),
				Type:  "resource",
			})
			nodeMap[object] = true
		}

		// Add edge
		viz.Edges = append(viz.Edges, GraphEdge{
			Source:    subject,
			Target:    object,
			Predicate: predicate,
			Label:     extractLabel(predicate),
		})
	}

	viz.Stats.NodeCount = len(viz.Nodes)
	viz.Stats.EdgeCount = len(viz.Edges)

	return viz, nil
}

// extractLabel extracts a human-readable label from a URI
func extractLabel(uri string) string {
	// Try to extract the fragment or last path component
	if idx := strings.LastIndex(uri, "#"); idx != -1 && idx < len(uri)-1 {
		return uri[idx+1:]
	}
	if idx := strings.LastIndex(uri, "/"); idx != -1 && idx < len(uri)-1 {
		return uri[idx+1:]
	}
	return uri
}

// Close closes the HTTP client (no-op for http.Client)
func (t *TDB2Backend) Close() error {
	return nil
}
