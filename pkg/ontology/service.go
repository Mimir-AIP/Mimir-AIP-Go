package ontology

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mimir-aip/mimir-aip-go/pkg/metadatastore"
	"github.com/mimir-aip/mimir-aip-go/pkg/models"
)

// Service provides ontology management operations
type Service struct {
	store metadatastore.MetadataStore
}

// NewService creates a new ontology service
func NewService(store metadatastore.MetadataStore) *Service {
	return &Service{
		store: store,
	}
}

// CreateOntology creates a new ontology
func (s *Service) CreateOntology(req *models.OntologyCreateRequest) (*models.Ontology, error) {
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("invalid ontology request: %w", err)
	}

	now := time.Now()
	ontology := &models.Ontology{
		ID:          uuid.New().String(),
		ProjectID:   req.ProjectID,
		Name:        req.Name,
		Description: req.Description,
		Version:     req.Version,
		Content:     req.Content,
		Status:      req.Status,
		IsGenerated: req.IsGenerated,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.store.SaveOntology(ontology); err != nil {
		return nil, fmt.Errorf("failed to save ontology: %w", err)
	}

	log.Printf("Created ontology %s for project %s", ontology.ID, ontology.ProjectID)

	return ontology, nil
}

// GetOntology retrieves an ontology by ID
func (s *Service) GetOntology(ontologyID string) (*models.Ontology, error) {
	return s.store.GetOntology(ontologyID)
}

// GetProjectOntologies retrieves all ontologies for a project
func (s *Service) GetProjectOntologies(projectID string) ([]*models.Ontology, error) {
	return s.store.ListOntologiesByProject(projectID)
}

// UpdateOntology updates an ontology
func (s *Service) UpdateOntology(ontologyID string, req *models.OntologyUpdateRequest) (*models.Ontology, error) {
	ontology, err := s.store.GetOntology(ontologyID)
	if err != nil {
		return nil, fmt.Errorf("ontology not found: %w", err)
	}

	// Apply updates
	if req.Name != nil {
		ontology.Name = *req.Name
	}
	if req.Description != nil {
		ontology.Description = *req.Description
	}
	if req.Version != nil {
		ontology.Version = *req.Version
	}
	if req.Content != nil {
		ontology.Content = *req.Content
	}
	if req.Status != nil {
		if *req.Status != "draft" && *req.Status != "active" && *req.Status != "archived" {
			return nil, fmt.Errorf("invalid status: %s", *req.Status)
		}
		ontology.Status = *req.Status
	}

	ontology.UpdatedAt = time.Now()

	if err := s.store.SaveOntology(ontology); err != nil {
		return nil, fmt.Errorf("failed to update ontology: %w", err)
	}

	log.Printf("Updated ontology %s", ontologyID)

	return ontology, nil
}

// DeleteOntology deletes an ontology
func (s *Service) DeleteOntology(ontologyID string) error {
	if err := s.store.DeleteOntology(ontologyID); err != nil {
		return fmt.Errorf("failed to delete ontology: %w", err)
	}

	log.Printf("Deleted ontology %s", ontologyID)

	return nil
}

// GenerateFromExtraction generates a Turtle ontology from extraction results
func (s *Service) GenerateFromExtraction(projectID, name string, extractionResult *models.ExtractionResult) (*models.Ontology, error) {
	if extractionResult == nil {
		return nil, fmt.Errorf("extraction result cannot be nil")
	}

	// Generate Turtle content from extraction result
	turtleContent := generateTurtleFromExtraction(extractionResult)

	// Create ontology
	req := &models.OntologyCreateRequest{
		ProjectID:   projectID,
		Name:        name,
		Description: fmt.Sprintf("Auto-generated ontology from %d entities", len(extractionResult.Entities)),
		Version:     "1.0",
		Content:     turtleContent,
		Status:      "draft",
		IsGenerated: true,
	}

	return s.CreateOntology(req)
}

// generateTurtleFromExtraction converts extraction results to Turtle format
func generateTurtleFromExtraction(result *models.ExtractionResult) string {
	var builder strings.Builder

	// Write prefixes
	builder.WriteString("@prefix : <http://example.org/mimir#> .\n")
	builder.WriteString("@prefix owl: <http://www.w3.org/2002/07/owl#> .\n")
	builder.WriteString("@prefix rdf: <http://www.w3.org/1999/02/22-rdf-syntax-ns#> .\n")
	builder.WriteString("@prefix rdfs: <http://www.w3.org/2000/01/rdf-schema#> .\n")
	builder.WriteString("@prefix xsd: <http://www.w3.org/2001/XMLSchema#> .\n\n")

	// Extract unique entity types from entities
	entityTypes := make(map[string]bool)
	for _, entity := range result.Entities {
		// Use first word of entity name as type (simplified heuristic)
		entityType := getEntityType(entity.Name)
		entityTypes[entityType] = true
	}

	// Write entity type classes
	builder.WriteString("# Entity Types (Classes)\n")
	for entityType := range entityTypes {
		className := capitalize(entityType)
		builder.WriteString(fmt.Sprintf(":%s a owl:Class ;\n", className))
		builder.WriteString(fmt.Sprintf("    rdfs:label \"%s\" .\n\n", className))
	}

	// Extract unique attributes from all entities
	attributes := make(map[string]string) // attribute name -> inferred type

	for _, entity := range result.Entities {
		for attrName, attrValue := range entity.Attributes {
			if _, exists := attributes[attrName]; !exists {
				attributes[attrName] = inferXSDType(attrValue)
			}
		}
	}

	// Write datatype properties (attributes)
	builder.WriteString("# Datatype Properties (Attributes)\n")
	for attrName, xsdType := range attributes {
		propName := toCamelCase(attrName)
		builder.WriteString(fmt.Sprintf(":%s a owl:DatatypeProperty ;\n", propName))
		builder.WriteString(fmt.Sprintf("    rdfs:label \"%s\" ;\n", propName))
		builder.WriteString(fmt.Sprintf("    rdfs:range xsd:%s .\n\n", xsdType))
	}

	// Extract unique relationships
	relationTypes := make(map[string]struct {
		from string
		to   string
	})

	for _, rel := range result.Relationships {
		key := rel.Relation
		relationTypes[key] = struct {
			from string
			to   string
		}{
			from: getEntityType(rel.Entity1.Name),
			to:   getEntityType(rel.Entity2.Name),
		}
	}

	// Write object properties (relationships)
	builder.WriteString("# Object Properties (Relationships)\n")
	for relName, rel := range relationTypes {
		propName := toCamelCase(relName)
		builder.WriteString(fmt.Sprintf(":%s a owl:ObjectProperty ;\n", propName))
		builder.WriteString(fmt.Sprintf("    rdfs:label \"%s\" ;\n", propName))
		builder.WriteString(fmt.Sprintf("    rdfs:domain :%s ;\n", capitalize(rel.from)))
		builder.WriteString(fmt.Sprintf("    rdfs:range :%s .\n\n", capitalize(rel.to)))
	}

	return builder.String()
}

// getEntityType extracts entity type from entity name (simplified heuristic)
func getEntityType(name string) string {
	// Use the entire name as the type for now
	// In a more sophisticated implementation, this would use NLP or pattern matching
	return name
}

// inferXSDType infers XSD type from a value
func inferXSDType(value interface{}) string {
	switch v := value.(type) {
	case int, int32, int64:
		return "int"
	case float32, float64:
		return "float"
	case bool:
		return "boolean"
	case string:
		// Try to detect date/time strings
		if strings.Contains(strings.ToLower(v), "date") || strings.Contains(strings.ToLower(v), "time") {
			return "dateTime"
		}
		return "string"
	default:
		return "string"
	}
}

// capitalize capitalizes the first letter of a string
func capitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// toCamelCase converts a string to camelCase
func toCamelCase(s string) string {
	// Replace underscores and spaces with camelCase
	words := strings.FieldsFunc(s, func(r rune) bool {
		return r == '_' || r == ' ' || r == '-'
	})

	if len(words) == 0 {
		return s
	}

	result := strings.ToLower(words[0])
	for i := 1; i < len(words); i++ {
		result += capitalize(strings.ToLower(words[i]))
	}

	return result
}
