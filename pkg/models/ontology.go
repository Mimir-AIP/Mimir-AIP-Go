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
