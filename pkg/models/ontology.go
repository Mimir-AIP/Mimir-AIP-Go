package models

import (
	"fmt"
	"time"
)

// Ontology represents an ontology definition for a project
// Ontologies are stored in Turtle (.ttl) format following OWL 2 specifications
type Ontology struct {
	ID          string    `json:"id"`
	ProjectID   string    `json:"project_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Version     string    `json:"version"`
	Content     string    `json:"content"`      // Turtle (.ttl) format content
	Status      string    `json:"status"`       // draft, active, archived
	IsGenerated bool      `json:"is_generated"` // true if auto-generated from extraction
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// OntologyCreateRequest represents a request to create a new ontology
type OntologyCreateRequest struct {
	ProjectID   string `json:"project_id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Version     string `json:"version"`
	Content     string `json:"content"`      // Turtle (.ttl) format content
	Status      string `json:"status"`       // draft, active, archived
	IsGenerated bool   `json:"is_generated"` // true if auto-generated
}

// OntologyUpdateRequest represents a request to update an ontology
type OntologyUpdateRequest struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
	Version     *string `json:"version,omitempty"`
	Content     *string `json:"content,omitempty"`
	Status      *string `json:"status,omitempty"`
}

// OntologyValidationRequest validates ontology content without persisting it.
type OntologyValidationRequest struct {
	ProjectID string `json:"project_id,omitempty"`
	Name      string `json:"name,omitempty"`
	Version   string `json:"version,omitempty"`
	Content   string `json:"content"`
}

// OntologyValidationResponse returns canonical compiler output and diagnostics.
type OntologyValidationResponse struct {
	Valid       bool                 `json:"valid"`
	Compiled    *CompiledOntology    `json:"compiled,omitempty"`
	Diagnostics []OntologyDiagnostic `json:"diagnostics,omitempty"`
}

// OntologySearchResult is a ranked semantic term match from a compiled ontology.
type OntologySearchResult struct {
	Term       OntologySearchTerm `json:"term"`
	Score      float64            `json:"score"`
	Match      string             `json:"match"`
	Diagnostic string             `json:"diagnostic,omitempty"`
}

// OntologyRetrieveRequest retrieves stored CIR records through compiled ontology semantics.
type OntologyRetrieveRequest struct {
	ProjectID  string                   `json:"project_id"`
	ClassID    string                   `json:"class"`
	Properties []string                 `json:"properties,omitempty"`
	Filters    []OntologyRetrieveFilter `json:"filters,omitempty"`
	StorageIDs []string                 `json:"storage_ids,omitempty"`
	Limit      int                      `json:"limit,omitempty"`
}

// OntologyRetrieveFilter filters ontology-backed CIR properties.
type OntologyRetrieveFilter struct {
	Property string      `json:"property"`
	Operator string      `json:"operator"`
	Value    interface{} `json:"value"`
}

// OntologyRetrieveResponse returns CIR records projected through ontology terms.
type OntologyRetrieveResponse struct {
	OntologyID  string                   `json:"ontology_id"`
	ContentHash string                   `json:"content_hash"`
	Results     []OntologyRetrieveResult `json:"results"`
	Count       int                      `json:"count"`
}

// OntologyRetrieveResult is one ontology-backed stored CIR match.
type OntologyRetrieveResult struct {
	StorageID  string                     `json:"storage_id"`
	ClassID    string                     `json:"class_id"`
	Properties map[string]interface{}     `json:"properties"`
	Source     CIRSource                  `json:"source"`
	Violations []SemanticMappingViolation `json:"violations,omitempty"`
}

// OntologyAggregateRequest computes semantic aggregations over ontology-mapped CIR data.
type OntologyAggregateRequest struct {
	ProjectID  string                   `json:"project_id"`
	ClassID    string                   `json:"class"`
	Metric     OntologyAggregateMetric  `json:"metric"`
	GroupBy    []string                 `json:"group_by,omitempty"`
	Filters    []OntologyRetrieveFilter `json:"filters,omitempty"`
	StorageIDs []string                 `json:"storage_ids,omitempty"`
	Limit      int                      `json:"limit,omitempty"`
}

// OntologyAggregateMetric identifies the ontology property and function to aggregate.
type OntologyAggregateMetric struct {
	Property string `json:"property"`
	Function string `json:"function"` // count | sum | avg | min | max
}

// OntologyAggregateResponse returns semantic aggregation output with ontology provenance.
type OntologyAggregateResponse struct {
	OntologyID  string                   `json:"ontology_id"`
	ContentHash string                   `json:"content_hash"`
	ClassID     string                   `json:"class_id"`
	PropertyID  string                   `json:"property_id"`
	Function    string                   `json:"function"`
	Groups      []OntologyAggregateGroup `json:"groups"`
	Count       int                      `json:"count"`
}

// OntologyAggregateGroup is one semantic aggregation bucket.
type OntologyAggregateGroup struct {
	Key   map[string]interface{} `json:"key"`
	Value float64                `json:"value"`
	Count int                    `json:"count"`
}

// Validate checks if the Ontology is valid
func (o *Ontology) Validate() error {
	if o.ProjectID == "" {
		return fmt.Errorf("project_id is required")
	}
	if o.Name == "" {
		return fmt.Errorf("name is required")
	}
	if o.Content == "" {
		return fmt.Errorf("content is required")
	}
	if o.Status != "draft" && o.Status != "active" && o.Status != "archived" {
		return fmt.Errorf("status must be one of: draft, active, archived")
	}
	return nil
}

// Validate checks if the OntologyCreateRequest is valid
func (r *OntologyCreateRequest) Validate() error {
	if r.ProjectID == "" {
		return fmt.Errorf("project_id is required")
	}
	if r.Name == "" {
		return fmt.Errorf("name is required")
	}
	if r.Content == "" {
		return fmt.Errorf("content is required")
	}
	if r.Status == "" {
		r.Status = "draft" // Default status
	}
	if r.Status != "draft" && r.Status != "active" && r.Status != "archived" {
		return fmt.Errorf("status must be one of: draft, active, archived")
	}
	if r.Version == "" {
		r.Version = "1.0" // Default version
	}
	return nil
}

// TurtleClass represents an OWL class in the ontology
type TurtleClass struct {
	URI         string   `json:"uri"`
	Label       string   `json:"label"`
	SubClassOf  []string `json:"subclass_of,omitempty"`
	Description string   `json:"description,omitempty"`
}

// TurtleProperty represents an OWL property (datatype or object property)
type TurtleProperty struct {
	URI         string   `json:"uri"`
	Label       string   `json:"label"`
	Type        string   `json:"type"` // DatatypeProperty or ObjectProperty
	Domain      []string `json:"domain,omitempty"`
	Range       []string `json:"range,omitempty"`
	InverseOf   string   `json:"inverse_of,omitempty"`
	Description string   `json:"description,omitempty"`
}

// TurtleIndividual represents an OWL individual (instance)
type TurtleIndividual struct {
	URI        string                 `json:"uri"`
	Type       string                 `json:"type"` // Class URI
	Properties map[string]interface{} `json:"properties,omitempty"`
}

// ParsedOntology represents a parsed Turtle ontology structure
type ParsedOntology struct {
	Prefixes    map[string]string  `json:"prefixes"`
	Classes     []TurtleClass      `json:"classes"`
	Properties  []TurtleProperty   `json:"properties"`
	Individuals []TurtleIndividual `json:"individuals,omitempty"`
}

// OntologyDiagnosticSeverity classifies ontology compiler findings.
type OntologyDiagnosticSeverity string

const (
	OntologyDiagnosticError   OntologyDiagnosticSeverity = "error"
	OntologyDiagnosticWarning OntologyDiagnosticSeverity = "warning"
	OntologyDiagnosticInfo    OntologyDiagnosticSeverity = "info"
)

// OntologyDiagnostic reports a validation, parsing, or semantic compiler finding.
type OntologyDiagnostic struct {
	Severity OntologyDiagnosticSeverity `json:"severity"`
	Code     string                     `json:"code"`
	Message  string                     `json:"message"`
	Line     int                        `json:"line,omitempty"`
	Column   int                        `json:"column,omitempty"`
	Subject  string                     `json:"subject,omitempty"`
}

// CompiledOntology is the canonical semantic representation every ontology-backed subsystem should consume.
type CompiledOntology struct {
	OntologyID  string                     `json:"ontology_id"`
	ProjectID   string                     `json:"project_id"`
	Name        string                     `json:"name"`
	Version     string                     `json:"version"`
	ContentHash string                     `json:"content_hash"`
	Prefixes    map[string]string          `json:"prefixes"`
	Classes     []CompiledOntologyClass    `json:"classes"`
	Properties  []CompiledOntologyProperty `json:"properties"`
	Relations   []CompiledOntologyRelation `json:"relations"`
	SearchTerms []OntologySearchTerm       `json:"search_terms"`
	Diagnostics []OntologyDiagnostic       `json:"diagnostics,omitempty"`
	CompiledAt  time.Time                  `json:"compiled_at"`
}

// CompiledOntologyClass captures the classes available to ingestion, storage, aggregation, twins, and ML.
type CompiledOntologyClass struct {
	ID          string   `json:"id"`
	URI         string   `json:"uri"`
	Name        string   `json:"name"`
	Label       string   `json:"label,omitempty"`
	Description string   `json:"description,omitempty"`
	SubClassOf  []string `json:"subclass_of,omitempty"`
	SearchTerms []string `json:"search_terms,omitempty"`
}

// CompiledOntologyProperty captures datatype and object properties with explicit domain/range semantics.
type CompiledOntologyProperty struct {
	ID          string   `json:"id"`
	URI         string   `json:"uri"`
	Name        string   `json:"name"`
	Label       string   `json:"label,omitempty"`
	Description string   `json:"description,omitempty"`
	Kind        string   `json:"kind"` // datatype | object | annotation | unknown
	Domain      []string `json:"domain,omitempty"`
	Range       []string `json:"range,omitempty"`
	InverseOf   string   `json:"inverse_of,omitempty"`
	SearchTerms []string `json:"search_terms,omitempty"`
}

// CompiledOntologyRelation is a searchable relationship edge derived from object properties.
type CompiledOntologyRelation struct {
	PropertyID string `json:"property_id"`
	Name       string `json:"name"`
	FromClass  string `json:"from_class"`
	ToClass    string `json:"to_class"`
	URI        string `json:"uri"`
}

// OntologySearchTerm makes ontology-backed retrieval straightforward across class/property labels and aliases.
type OntologySearchTerm struct {
	Term       string   `json:"term"`
	Kind       string   `json:"kind"` // class | property | relation
	ID         string   `json:"id"`
	URI        string   `json:"uri"`
	Weight     float64  `json:"weight"`
	Aliases    []string `json:"aliases,omitempty"`
	ClassID    string   `json:"class_id,omitempty"`
	PropertyID string   `json:"property_id,omitempty"`
}

// OntologyExtractionRequest represents a request to extract ontology from CIR data
type OntologyExtractionRequest struct {
	ProjectID           string   `json:"project_id"`
	StorageIDs          []string `json:"storage_ids"`          // Storage configs to extract from
	OntologyName        string   `json:"ontology_name"`        // Name for the generated ontology
	IncludeStructured   bool     `json:"include_structured"`   // Extract from structured data
	IncludeUnstructured bool     `json:"include_unstructured"` // Extract from unstructured data
}

// Validate checks if the OntologyExtractionRequest is valid
func (r *OntologyExtractionRequest) Validate() error {
	if r.ProjectID == "" {
		return fmt.Errorf("project_id is required")
	}
	if len(r.StorageIDs) == 0 {
		return fmt.Errorf("at least one storage_id is required")
	}
	if r.OntologyName == "" {
		return fmt.Errorf("ontology_name is required")
	}
	if !r.IncludeStructured && !r.IncludeUnstructured {
		r.IncludeStructured = true // Default to structured extraction
	}
	return nil
}
