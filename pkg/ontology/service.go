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

type OntologyProjectMismatchError struct {
	OntologyID        string
	ExpectedProjectID string
	ActualProjectID   string
}

func (e *OntologyProjectMismatchError) Error() string {
	return fmt.Sprintf("ontology %s belongs to project %s, not %s", e.OntologyID, e.ActualProjectID, e.ExpectedProjectID)
}

type OntologyInUseError struct {
	OntologyID string
	References []string
}

func (e *OntologyInUseError) Error() string {
	return fmt.Sprintf("ontology %s is still referenced by %s", e.OntologyID, strings.Join(e.References, ", "))
}

func validOntologyStatus(status string) bool {
	return status == "draft" || status == "active" || status == "archived"
}

func validateOntologyContent(content string) error {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return fmt.Errorf("content is required")
	}
	if !strings.Contains(trimmed, "@prefix") {
		return fmt.Errorf("ontology content must include Turtle prefixes")
	}
	return nil
}

func (s *Service) ensureProjectExists(projectID string) error {
	if strings.TrimSpace(projectID) == "" {
		return fmt.Errorf("project_id is required")
	}
	if _, err := s.store.GetProject(projectID); err != nil {
		return fmt.Errorf("project not found: %w", err)
	}
	return nil
}

func (s *Service) EnsureProjectExists(projectID string) error {
	return s.ensureProjectExists(projectID)
}

func (s *Service) getOwnedOntology(projectID, ontologyID string) (*models.Ontology, error) {
	if strings.TrimSpace(projectID) == "" {
		return nil, fmt.Errorf("project_id is required")
	}
	ontology, err := s.store.GetOntology(ontologyID)
	if err != nil {
		return nil, fmt.Errorf("ontology not found: %w", err)
	}
	if ontology.ProjectID != projectID {
		return nil, &OntologyProjectMismatchError{OntologyID: ontologyID, ExpectedProjectID: projectID, ActualProjectID: ontology.ProjectID}
	}
	return ontology, nil
}

func (s *Service) GetOwnedOntology(projectID, ontologyID string) (*models.Ontology, error) {
	return s.getOwnedOntology(projectID, ontologyID)
}

// CreateOntology creates a new ontology
func (s *Service) CreateOntology(req *models.OntologyCreateRequest) (*models.Ontology, error) {
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("invalid ontology request: %w", err)
	}
	if err := s.ensureProjectExists(req.ProjectID); err != nil {
		return nil, err
	}
	if !validOntologyStatus(req.Status) {
		return nil, fmt.Errorf("status must be one of: draft, active, archived")
	}
	if err := validateOntologyContent(req.Content); err != nil {
		return nil, err
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

func (s *Service) GetOntologyForProject(projectID, ontologyID string) (*models.Ontology, error) {
	return s.getOwnedOntology(projectID, ontologyID)
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
		if err := validateOntologyContent(*req.Content); err != nil {
			return nil, err
		}
		ontology.Content = *req.Content
	}
	if req.Status != nil {
		if !validOntologyStatus(*req.Status) {
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

func (s *Service) UpdateOntologyForProject(projectID, ontologyID string, req *models.OntologyUpdateRequest) (*models.Ontology, error) {
	if _, err := s.getOwnedOntology(projectID, ontologyID); err != nil {
		return nil, err
	}
	return s.UpdateOntology(ontologyID, req)
}

// DeleteOntology deletes an ontology
func (s *Service) DeleteOntology(ontologyID string) error {
	references, err := s.findOntologyReferences(ontologyID)
	if err != nil {
		return err
	}
	if len(references) > 0 {
		return &OntologyInUseError{OntologyID: ontologyID, References: references}
	}
	if err := s.store.DeleteOntology(ontologyID); err != nil {
		return fmt.Errorf("failed to delete ontology: %w", err)
	}
	log.Printf("Deleted ontology %s", ontologyID)
	return nil
}

func (s *Service) DeleteOntologyForProject(projectID, ontologyID string) error {
	if _, err := s.getOwnedOntology(projectID, ontologyID); err != nil {
		return err
	}
	return s.DeleteOntology(ontologyID)
}

func (s *Service) findOntologyReferences(ontologyID string) ([]string, error) {
	references := make([]string, 0)
	modelList, err := s.store.ListMLModels()
	if err != nil {
		return nil, fmt.Errorf("failed to list ml models for ontology delete: %w", err)
	}
	for _, model := range modelList {
		if model != nil && model.OntologyID == ontologyID {
			references = append(references, fmt.Sprintf("ml model %s", model.ID))
		}
	}
	twins, err := s.store.ListDigitalTwins()
	if err != nil {
		return nil, fmt.Errorf("failed to list digital twins for ontology delete: %w", err)
	}
	for _, twin := range twins {
		if twin != nil && twin.OntologyID == ontologyID {
			references = append(references, fmt.Sprintf("digital twin %s", twin.ID))
		}
	}
	configs, err := s.store.ListStorageConfigs()
	if err != nil {
		return nil, fmt.Errorf("failed to list storage configs for ontology delete: %w", err)
	}
	for _, cfg := range configs {
		if cfg != nil && cfg.OntologyID == ontologyID {
			references = append(references, fmt.Sprintf("storage config %s", cfg.ID))
		}
	}
	return references, nil
}

// OntologyDiff describes the differences between two ontologies.
type OntologyDiff struct {
	AddedClasses      []string `json:"added_classes"`
	RemovedClasses    []string `json:"removed_classes"`
	AddedProperties   []string `json:"added_properties"`
	RemovedProperties []string `json:"removed_properties"`
	HasChanges        bool     `json:"has_changes"`
}

// DiffOntologies computes the symmetric difference in class and property declarations
// between two Turtle ontology strings.
func (s *Service) DiffOntologies(oldContent, newContent string) OntologyDiff {
	oldClasses, oldProps := parseTurtleDeclarations(oldContent)
	newClasses, newProps := parseTurtleDeclarations(newContent)

	diff := OntologyDiff{
		AddedClasses:      setDiff(newClasses, oldClasses),
		RemovedClasses:    setDiff(oldClasses, newClasses),
		AddedProperties:   setDiff(newProps, oldProps),
		RemovedProperties: setDiff(oldProps, newProps),
	}
	diff.HasChanges = len(diff.AddedClasses) > 0 || len(diff.RemovedClasses) > 0 ||
		len(diff.AddedProperties) > 0 || len(diff.RemovedProperties) > 0
	return diff
}

// parseTurtleDeclarations extracts class and property names from Turtle content.
func parseTurtleDeclarations(content string) (classes map[string]bool, properties map[string]bool) {
	classes = make(map[string]bool)
	properties = make(map[string]bool)
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if (strings.Contains(line, "owl:Class") || strings.Contains(line, "owl:class")) &&
			(strings.Contains(line, " a ") || strings.Contains(line, "rdf:type")) {
			name := extractTurtleSubjectOntology(line)
			if name != "" {
				classes[name] = true
			}
		}
		if strings.Contains(line, "owl:DatatypeProperty") || strings.Contains(line, "owl:ObjectProperty") {
			name := extractTurtleSubjectOntology(line)
			if name != "" {
				properties[name] = true
			}
		}
	}
	return
}

// extractTurtleSubjectOntology extracts the local name from the first token of a Turtle triple.
func extractTurtleSubjectOntology(line string) string {
	parts := strings.Fields(line)
	if len(parts) == 0 {
		return ""
	}
	s := parts[0]
	s = strings.TrimPrefix(s, ":")
	if idx := strings.LastIndex(s, ":"); idx >= 0 {
		s = s[idx+1:]
	}
	return strings.Trim(s, "<>.,;")
}

// setDiff returns elements in a that are not in b.
func setDiff(a, b map[string]bool) []string {
	var result []string
	for k := range a {
		if !b[k] {
			result = append(result, k)
		}
	}
	return result
}

// GenerateFromExtraction generates a Turtle ontology from extraction results.
//
// Auto-generated ontologies are maintained as a single active ontology per
// (project, name). Re-generating from new ingestion data updates the existing
// generated ontology in place instead of creating draft copies that require
// manual activation.
func (s *Service) GenerateFromExtraction(projectID, name string, extractionResult *models.ExtractionResult) (*models.Ontology, error) {
	if extractionResult == nil {
		return nil, fmt.Errorf("extraction result cannot be nil")
	}

	turtleContent := generateTurtleFromExtraction(extractionResult)
	desc := fmt.Sprintf("Auto-generated ontology from %d entities", len(extractionResult.Entities))
	if len(extractionResult.CrossSourceLinks) > 0 {
		desc += fmt.Sprintf(", %d cross-source links", len(extractionResult.CrossSourceLinks))
	}

	existingOntologies, err := s.store.ListOntologiesByProject(projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to list existing ontologies: %w", err)
	}

	for _, existing := range existingOntologies {
		if !existing.IsGenerated || existing.Name != name {
			continue
		}

		if existing.Content == turtleContent && existing.Description == desc && existing.Status == "active" {
			return existing, nil
		}

		active := "active"
		updateReq := &models.OntologyUpdateRequest{
			Description: &desc,
			Content:     &turtleContent,
			Status:      &active,
		}
		updated, err := s.UpdateOntology(existing.ID, updateReq)
		if err != nil {
			return nil, fmt.Errorf("failed to update generated ontology: %w", err)
		}
		return updated, nil
	}

	req := &models.OntologyCreateRequest{
		ProjectID:   projectID,
		Name:        name,
		Description: desc,
		Version:     "1.0",
		Content:     turtleContent,
		Status:      "active",
		IsGenerated: true,
	}

	return s.CreateOntology(req)
}

// generateTurtleFromExtraction converts extraction results to Turtle format.
//
// Entity class names are derived from the entity_type attribute set during
// structured extraction (which reflects the column/table name, e.g. "Student"),
// not from the entity value itself (e.g. "Alice Johnson").
//
// Cross-source links discovered by the extraction engine are emitted as
// owl:ObjectProperty declarations with :crossSourceLink "true" so that
// downstream consumers (digital twin sync, SPARQL) can identify join points.
func generateTurtleFromExtraction(result *models.ExtractionResult) string {
	var builder strings.Builder

	// Write prefixes
	builder.WriteString("@prefix : <http://example.org/mimir#> .\n")
	builder.WriteString("@prefix owl: <http://www.w3.org/2002/07/owl#> .\n")
	builder.WriteString("@prefix rdf: <http://www.w3.org/1999/02/22-rdf-syntax-ns#> .\n")
	builder.WriteString("@prefix rdfs: <http://www.w3.org/2000/01/rdf-schema#> .\n")
	builder.WriteString("@prefix xsd: <http://www.w3.org/2001/XMLSchema#> .\n\n")

	// Collect unique entity class names from the entity_type attribute,
	// which is set by structured extraction to the source column/table name.
	// Only fall back to the entity name for unstructured entities whose name
	// looks like a type (no spaces, short enough to be a class name).
	entityTypes := make(map[string]bool)
	for _, entity := range result.Entities {
		t := resolveEntityType(entity)
		if t != "" && t != "Entity" {
			entityTypes[capitalize(t)] = true
		}
	}

	// Also collect entity types from cross-source links (they may reference
	// entity types not present in the extracted entity list).
	for _, link := range result.CrossSourceLinks {
		if link.EntityTypeA != "" {
			entityTypes[capitalize(link.EntityTypeA)] = true
		}
		if link.EntityTypeB != "" {
			entityTypes[capitalize(link.EntityTypeB)] = true
		}
	}

	// Write entity type classes.
	builder.WriteString("# Entity Types (Classes)\n")
	for className := range entityTypes {
		builder.WriteString(fmt.Sprintf(":%s a owl:Class ;\n", className))
		builder.WriteString(fmt.Sprintf("    rdfs:label \"%s\" .\n\n", className))
	}

	// Collect datatype properties (scalar attributes) from entities.
	// Properties are grouped by entity type so that rdfs:domain can be emitted,
	// allowing parseOntologyClasses in the digital twin to correctly associate
	// each property with its owning class.
	//
	// When the same property name appears in multiple entity types, the first
	// entity type encountered takes precedence for domain assignment.
	type propSpec struct {
		xsdType    string
		domainType string // PascalCase entity type, empty if unknown
	}
	propsByName := make(map[string]propSpec) // camelCase propName → spec
	skipAttrs := map[string]bool{
		"entity_type": true, "source_column": true, "occurrence_count": true,
		"doc_frequency": true, "total_occurrences": true, "cap_consistency": true,
		"fields": true,
	}
	for _, entity := range result.Entities {
		et := capitalize(resolveEntityType(entity))
		for attrName, attrValue := range entity.Attributes {
			if skipAttrs[attrName] {
				continue
			}
			propName := toCamelCase(attrName)
			if _, exists := propsByName[propName]; !exists {
				propsByName[propName] = propSpec{
					xsdType:    inferXSDType(attrValue),
					domainType: et,
				}
			}
		}
	}

	builder.WriteString("# Datatype Properties (Attributes)\n")
	for propName, spec := range propsByName {
		builder.WriteString(fmt.Sprintf(":%s a owl:DatatypeProperty ;\n", propName))
		builder.WriteString(fmt.Sprintf("    rdfs:label \"%s\" ;\n", propName))
		if spec.domainType != "" && spec.domainType != "Entity" {
			builder.WriteString(fmt.Sprintf("    rdfs:domain :%s ;\n", spec.domainType))
		}
		builder.WriteString(fmt.Sprintf("    rdfs:range xsd:%s .\n\n", spec.xsdType))
	}

	// Collect object properties from intra-source relationships.
	type relSpec struct{ from, to string }
	relationTypes := make(map[string]relSpec)
	for _, rel := range result.Relationships {
		key := toCamelCase(rel.Relation)
		from := capitalize(resolveEntityType(*rel.Entity1))
		to := capitalize(resolveEntityType(*rel.Entity2))
		if from != "" && to != "" {
			relationTypes[key] = relSpec{from, to}
		}
	}

	// Emit cross-source link ObjectProperties.
	// Each link produces a bidirectional has<EntityTypeB> property on EntityTypeA.
	// The property name encodes both entity types for clarity.
	for _, link := range result.CrossSourceLinks {
		typeA := capitalize(link.EntityTypeA)
		typeB := capitalize(link.EntityTypeB)
		if typeA == "" || typeB == "" {
			continue
		}
		// Produce has<TypeB> on TypeA (and has<TypeA> on TypeB).
		fwdKey := "has" + typeB
		bwdKey := "has" + typeA
		if _, exists := relationTypes[fwdKey]; !exists {
			relationTypes[fwdKey] = relSpec{typeA, typeB}
		}
		if _, exists := relationTypes[bwdKey]; !exists {
			relationTypes[bwdKey] = relSpec{typeB, typeA}
		}
	}

	builder.WriteString("# Object Properties (Relationships)\n")
	for propName, rel := range relationTypes {
		builder.WriteString(fmt.Sprintf(":%s a owl:ObjectProperty ;\n", propName))
		builder.WriteString(fmt.Sprintf("    rdfs:label \"%s\" ;\n", propName))
		builder.WriteString(fmt.Sprintf("    rdfs:domain :%s ;\n", rel.from))
		builder.WriteString(fmt.Sprintf("    rdfs:range :%s .\n\n", rel.to))
	}

	// Annotate cross-source links as identity-key bridges so tools can
	// distinguish them from intra-source structural relationships.
	if len(result.CrossSourceLinks) > 0 {
		builder.WriteString("# Cross-Source Identity Links\n")
		for _, link := range result.CrossSourceLinks {
			typeA := capitalize(link.EntityTypeA)
			typeB := capitalize(link.EntityTypeB)
			propName := toCamelCase(link.ColumnA + "_links_" + link.ColumnB)
			builder.WriteString(fmt.Sprintf(":%s a owl:ObjectProperty ;\n", propName))
			builder.WriteString(fmt.Sprintf("    rdfs:label \"%s\" ;\n", propName))
			builder.WriteString(fmt.Sprintf("    rdfs:domain :%s ;\n", typeA))
			builder.WriteString(fmt.Sprintf("    rdfs:range :%s ;\n", typeB))
			builder.WriteString(fmt.Sprintf("    :crossSourceLink \"true\"^^xsd:boolean ;\n"))
			builder.WriteString(fmt.Sprintf("    :linkConfidence \"%s\"^^xsd:float ;\n", fmt.Sprintf("%.3f", link.Confidence)))
			builder.WriteString(fmt.Sprintf("    :joinColumnA \"%s\" ;\n", link.ColumnA))
			builder.WriteString(fmt.Sprintf("    :joinColumnB \"%s\" .\n\n", link.ColumnB))
		}
	}

	return builder.String()
}

// resolveEntityType extracts a meaningful entity class name from an
// ExtractedEntity, preferring the entity_type attribute set during structured
// extraction over using the raw entity value as a class name.
func resolveEntityType(entity models.ExtractedEntity) string {
	// Structured extraction: entity_type attribute holds the column/table name.
	if et, ok := entity.Attributes["entity_type"].(string); ok && et != "" {
		return et
	}
	// Unstructured extraction: try the first field key the entity appeared in.
	if fields, ok := entity.Attributes["fields"].([]interface{}); ok && len(fields) > 0 {
		if s, ok := fields[0].(string); ok && s != "" {
			return colNameToType(s)
		}
	}
	// Fall back to the entity name only if it looks like a type identifier:
	// no spaces, no digits-only, short enough to be a class name.
	name := strings.TrimSpace(entity.Name)
	if name != "" && !strings.Contains(name, " ") && len(name) <= 50 {
		return name
	}
	return "Entity"
}

// colNameToType converts a snake_case/kebab-case column name to PascalCase,
// matching the logic in pkg/extraction/structured.go so that entity types
// generated from ontology and extraction are consistent.
func colNameToType(col string) string {
	parts := strings.FieldsFunc(col, func(r rune) bool {
		return r == '_' || r == '-' || r == ' '
	})
	var b strings.Builder
	for _, p := range parts {
		runes := []rune(p)
		if len(runes) == 0 {
			continue
		}
		b.WriteRune([]rune(strings.ToUpper(string(runes[0])))[0])
		b.WriteString(string(runes[1:]))
	}
	return b.String()
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
