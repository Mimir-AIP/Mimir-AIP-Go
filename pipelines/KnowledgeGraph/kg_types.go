package knowledgegraph

import (
	"time"
)

// Triple represents an RDF triple (subject, predicate, object)
type Triple struct {
	Subject   string `json:"subject"`
	Predicate string `json:"predicate"`
	Object    string `json:"object"`
	Graph     string `json:"graph,omitempty"` // Named graph (optional)
}

// QueryResult represents the result of a SPARQL query
type QueryResult struct {
	Variables []string      `json:"variables"`
	Bindings  []BindingRow  `json:"bindings"`
	QueryType string        `json:"query_type"`        // SELECT, CONSTRUCT, ASK, DESCRIBE
	Boolean   *bool         `json:"boolean,omitempty"` // For ASK queries
	Duration  time.Duration `json:"duration"`
}

// BindingRow represents a single row of variable bindings
type BindingRow map[string]BindingValue

// BindingValue represents a bound value in a SPARQL result
type BindingValue struct {
	Type     string `json:"type"` // uri, literal, bnode
	Value    string `json:"value"`
	Datatype string `json:"datatype,omitempty"`
	Lang     string `json:"lang,omitempty"`
}

// GraphStats provides statistics about the knowledge graph
type GraphStats struct {
	TotalTriples    int       `json:"total_triples"`
	TotalSubjects   int       `json:"total_subjects"`
	TotalPredicates int       `json:"total_predicates"`
	TotalObjects    int       `json:"total_objects"`
	NamedGraphs     []string  `json:"named_graphs"`
	LastUpdated     time.Time `json:"last_updated"`
	SizeBytes       int64     `json:"size_bytes"`
}

// QueryType represents the type of SPARQL query
type QueryType string

const (
	QueryTypeSelect    QueryType = "SELECT"
	QueryTypeConstruct QueryType = "CONSTRUCT"
	QueryTypeAsk       QueryType = "ASK"
	QueryTypeDescribe  QueryType = "DESCRIBE"
)

// UpdateOperation represents a SPARQL UPDATE operation
type UpdateOperation struct {
	Operation string `json:"operation"` // INSERT, DELETE, CLEAR, etc.
	Query     string `json:"query"`
	Graph     string `json:"graph,omitempty"`
}

// UpdateResult represents the result of a SPARQL UPDATE operation
type UpdateResult struct {
	Success        bool          `json:"success"`
	TriplesAdded   int           `json:"triples_added,omitempty"`
	TriplesRemoved int           `json:"triples_removed,omitempty"`
	Duration       time.Duration `json:"duration"`
	ErrorMessage   string        `json:"error_message,omitempty"`
}

// GraphNode represents a node in the knowledge graph visualization
type GraphNode struct {
	URI        string         `json:"uri"`
	Label      string         `json:"label"`
	Type       string         `json:"type"`
	Properties map[string]any `json:"properties,omitempty"`
}

// GraphEdge represents an edge in the knowledge graph visualization
type GraphEdge struct {
	Source    string `json:"source"`
	Target    string `json:"target"`
	Predicate string `json:"predicate"`
	Label     string `json:"label"`
}

// GraphVisualization represents a subgraph for visualization
type GraphVisualization struct {
	Nodes []GraphNode `json:"nodes"`
	Edges []GraphEdge `json:"edges"`
	Stats struct {
		NodeCount int `json:"node_count"`
		EdgeCount int `json:"edge_count"`
	} `json:"stats"`
}

// ExportFormat represents the export format for graph data
type ExportFormat string

const (
	ExportFormatTurtle   ExportFormat = "turtle"
	ExportFormatRDFXML   ExportFormat = "rdfxml"
	ExportFormatNTriples ExportFormat = "ntriples"
	ExportFormatJSONLD   ExportFormat = "jsonld"
)

// ExportOptions represents options for exporting graph data
type ExportOptions struct {
	Format     ExportFormat `json:"format"`
	Graph      string       `json:"graph,omitempty"`
	Limit      int          `json:"limit,omitempty"`
	Compressed bool         `json:"compressed"`
}
