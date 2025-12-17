package schema_inference

import (
	"fmt"
	"strings"
	"time"
)

// OntologyGenerator creates OWL ontologies from inferred schemas
type OntologyGenerator struct {
	config OntologyConfig
}

// OntologyConfig configures ontology generation
type OntologyConfig struct {
	BaseURI         string `json:"base_uri"`
	OntologyPrefix  string `json:"ontology_prefix"`
	IncludeMetadata bool   `json:"include_metadata"`
	IncludeComments bool   `json:"include_comments"`
	ClassNaming     string `json:"class_naming"`    // "pascal", "camel", "snake"
	PropertyNaming  string `json:"property_naming"` // "camel", "snake"
}

// Ontology represents a generated OWL ontology
type Ontology struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Version     string                 `json:"version"`
	Format      string                 `json:"format"`
	Content     string                 `json:"content"`
	Classes     []OntologyClass        `json:"classes"`
	Properties  []OntologyProperty     `json:"properties"`
	Metadata    map[string]interface{} `json:"metadata"`
	GeneratedAt time.Time              `json:"generated_at"`
}

// OntologyClass represents an OWL class
type OntologyClass struct {
	Name        string `json:"name"`
	URI         string `json:"uri"`
	Description string `json:"description"`
	SuperClass  string `json:"super_class,omitempty"`
}

// OntologyProperty represents an OWL property
type OntologyProperty struct {
	Name        string `json:"name"`
	URI         string `json:"uri"`
	Domain      string `json:"domain"`
	Range       string `json:"range"`
	Type        string `json:"type"` // "datatype" or "object"
	Description string `json:"description"`
}

// NewOntologyGenerator creates a new ontology generator
func NewOntologyGenerator(config OntologyConfig) *OntologyGenerator {
	if config.BaseURI == "" {
		config.BaseURI = "http://mimir-aip.io/ontology/"
	}
	if config.OntologyPrefix == "" {
		config.OntologyPrefix = "mimir"
	}
	if config.ClassNaming == "" {
		config.ClassNaming = "pascal"
	}
	if config.PropertyNaming == "" {
		config.PropertyNaming = "camel"
	}

	return &OntologyGenerator{
		config: config,
	}
}

// GenerateOntology creates an OWL ontology from a data schema
func (g *OntologyGenerator) GenerateOntology(schema *DataSchema) (*Ontology, error) {
	ontology := &Ontology{
		ID:          fmt.Sprintf("ontology_%d", time.Now().Unix()),
		Name:        schema.Name,
		Description: schema.Description,
		Version:     "1.0.0",
		Format:      "turtle",
		Classes:     []OntologyClass{},
		Properties:  []OntologyProperty{},
		Metadata:    make(map[string]interface{}),
		GeneratedAt: time.Now(),
	}

	// Generate classes from columns
	classes := g.generateClasses(schema)
	ontology.Classes = classes

	// Generate properties from columns
	properties := g.generateProperties(schema, classes)
	ontology.Properties = properties

	// Generate OWL content
	content, err := g.generateOWLContent(ontology)
	if err != nil {
		return nil, fmt.Errorf("failed to generate OWL content: %w", err)
	}
	ontology.Content = content

	// Add metadata
	ontology.Metadata = map[string]interface{}{
		"source_schema":      schema.Name,
		"column_count":       len(schema.Columns),
		"class_count":        len(ontology.Classes),
		"property_count":     len(ontology.Properties),
		"relationship_count": len(schema.Relationships),
	}

	return ontology, nil
}

// generateClasses creates ontology classes from schema columns
func (g *OntologyGenerator) generateClasses(schema *DataSchema) []OntologyClass {
	var classes []OntologyClass

	// Group columns by potential entity types
	entityGroups := g.groupColumnsByEntity(schema.Columns)

	for entityName, columns := range entityGroups {
		className := g.formatClassName(entityName)
		classURI := fmt.Sprintf("%s%s", g.config.BaseURI, strings.ToLower(className))

		class := OntologyClass{
			Name:        className,
			URI:         classURI,
			Description: fmt.Sprintf("Class representing %s entities", entityName),
		}

		// Determine super class based on entity type
		if superClass := g.inferSuperClass(columns); superClass != "" {
			class.SuperClass = superClass
		}

		classes = append(classes, class)
	}

	// If no groups found, create a single class from all columns
	if len(classes) == 0 {
		className := g.formatClassName(schema.Name)
		class := OntologyClass{
			Name:        className,
			URI:         fmt.Sprintf("%s%s", g.config.BaseURI, strings.ToLower(className)),
			Description: fmt.Sprintf("Class representing %s", schema.Name),
		}
		classes = append(classes, class)
	}

	return classes
}

// groupColumnsByEntity groups columns into logical entity types
func (g *OntologyGenerator) groupColumnsByEntity(columns []ColumnSchema) map[string][]ColumnSchema {
	groups := make(map[string][]ColumnSchema)

	// Simple grouping: all columns go into one entity named after the schema
	// In a more sophisticated implementation, this could use clustering algorithms
	// or domain knowledge to identify separate entities

	entityName := "Entity" // Default entity name
	groups[entityName] = columns

	return groups
}

// generateProperties creates ontology properties from schema columns
func (g *OntologyGenerator) generateProperties(schema *DataSchema, classes []OntologyClass) []OntologyProperty {
	var properties []OntologyProperty

	// Use the first class as domain (simplified approach)
	domainClass := ""
	if len(classes) > 0 {
		domainClass = classes[0].URI
	}

	for _, col := range schema.Columns {
		propertyName := g.formatPropertyName(col.Name)
		propertyURI := fmt.Sprintf("%s%s", g.config.BaseURI, propertyName)

		property := OntologyProperty{
			Name:        propertyName,
			URI:         propertyURI,
			Domain:      domainClass,
			Range:       col.OntologyType,
			Type:        "datatype", // Default to datatype property
			Description: col.Description,
		}

		// Special handling for foreign keys (object properties)
		if col.IsForeignKey {
			property.Type = "object"
			// Try to infer the range class from the column name
			if rangeClass := g.inferRangeClass(col.Name, classes); rangeClass != "" {
				property.Range = rangeClass
			}
		}

		properties = append(properties, property)
	}

	// Add properties for relationships
	for _, rel := range schema.Relationships {
		propertyName := g.formatPropertyName(rel.SourceColumn + "_" + rel.TargetColumn)
		propertyURI := fmt.Sprintf("%s%s", g.config.BaseURI, propertyName)

		property := OntologyProperty{
			Name:        propertyName,
			URI:         propertyURI,
			Domain:      domainClass,
			Range:       "xsd:anyURI", // Reference to another entity
			Type:        "object",
			Description: rel.Description,
		}

		properties = append(properties, property)
	}

	return properties
}

// formatClassName formats a class name according to configuration
func (g *OntologyGenerator) formatClassName(name string) string {
	switch g.config.ClassNaming {
	case "pascal":
		return toPascalCase(name)
	case "camel":
		return toCamelCase(name)
	case "snake":
		return toSnakeCase(name)
	default:
		return toPascalCase(name)
	}
}

// formatPropertyName formats a property name according to configuration
func (g *OntologyGenerator) formatPropertyName(name string) string {
	switch g.config.PropertyNaming {
	case "camel":
		return toCamelCase(name)
	case "snake":
		return toSnakeCase(name)
	default:
		return toCamelCase(name)
	}
}

// inferSuperClass attempts to infer a superclass based on column patterns
func (g *OntologyGenerator) inferSuperClass(columns []ColumnSchema) string {
	// Simple heuristics for common entity types
	hasName := false
	hasEmail := false
	hasAddress := false

	for _, col := range columns {
		name := strings.ToLower(col.Name)
		if strings.Contains(name, "name") {
			hasName = true
		}
		if strings.Contains(name, "email") {
			hasEmail = true
		}
		if strings.Contains(name, "address") || strings.Contains(name, "street") {
			hasAddress = true
		}
	}

	if hasName && hasEmail {
		return "http://xmlns.com/foaf/0.1/Person"
	}

	if hasName && hasAddress {
		return "http://schema.org/Organization"
	}

	return "" // No superclass inferred
}

// inferRangeClass attempts to infer the range class for object properties
func (g *OntologyGenerator) inferRangeClass(columnName string, classes []OntologyClass) string {
	// Remove common suffixes
	cleanName := strings.TrimSuffix(strings.ToLower(columnName), "_id")
	cleanName = strings.TrimSuffix(cleanName, "_fk")

	// Look for matching class
	for _, class := range classes {
		if strings.Contains(strings.ToLower(class.Name), cleanName) {
			return class.URI
		}
	}

	return "" // No matching class found
}

// generateOWLContent generates the actual OWL ontology in Turtle format
func (g *OntologyGenerator) generateOWLContent(ontology *Ontology) (string, error) {
	var sb strings.Builder

	// Prefixes
	sb.WriteString("@prefix owl: <http://www.w3.org/2002/07/owl#> .\n")
	sb.WriteString("@prefix rdf: <http://www.w3.org/1999/02/22-rdf-syntax-ns#> .\n")
	sb.WriteString("@prefix rdfs: <http://www.w3.org/2000/01/rdf-schema#> .\n")
	sb.WriteString("@prefix xsd: <http://www.w3.org/2001/XMLSchema#> .\n")
	sb.WriteString(fmt.Sprintf("@prefix %s: <%s> .\n", g.config.OntologyPrefix, g.config.BaseURI))
	sb.WriteString("\n")

	// Ontology declaration
	sb.WriteString(fmt.Sprintf("<%s> a owl:Ontology ;\n", g.config.BaseURI))
	sb.WriteString(fmt.Sprintf("    rdfs:label \"%s\"@en ;\n", ontology.Name))
	sb.WriteString(fmt.Sprintf("    rdfs:comment \"%s\"@en .\n\n", ontology.Description))

	// Classes
	for _, class := range ontology.Classes {
		sb.WriteString(fmt.Sprintf("%s:%s a owl:Class ;\n", g.config.OntologyPrefix, class.Name))
		sb.WriteString(fmt.Sprintf("    rdfs:label \"%s\"@en ;\n", class.Name))
		sb.WriteString(fmt.Sprintf("    rdfs:comment \"%s\"@en", class.Description))

		if class.SuperClass != "" {
			sb.WriteString(" ;\n")
			sb.WriteString(fmt.Sprintf("    rdfs:subClassOf <%s>", class.SuperClass))
		}

		sb.WriteString(" .\n\n")
	}

	// Properties
	for _, prop := range ontology.Properties {
		if prop.Type == "datatype" {
			sb.WriteString(fmt.Sprintf("%s:%s a owl:DatatypeProperty ;\n", g.config.OntologyPrefix, prop.Name))
		} else {
			sb.WriteString(fmt.Sprintf("%s:%s a owl:ObjectProperty ;\n", g.config.OntologyPrefix, prop.Name))
		}

		sb.WriteString(fmt.Sprintf("    rdfs:label \"%s\"@en ;\n", prop.Name))
		sb.WriteString(fmt.Sprintf("    rdfs:comment \"%s\"@en ;\n", prop.Description))
		sb.WriteString(fmt.Sprintf("    rdfs:domain <%s> ;\n", prop.Domain))
		sb.WriteString(fmt.Sprintf("    rdfs:range <%s>", prop.Range))

		sb.WriteString(" .\n\n")
	}

	return sb.String(), nil
}

// Helper functions for name formatting

func toPascalCase(s string) string {
	words := strings.FieldsFunc(s, func(r rune) bool {
		return r == '_' || r == '-' || r == ' '
	})

	for i, word := range words {
		if len(word) > 0 {
			words[i] = strings.ToUpper(word[:1]) + strings.ToLower(word[1:])
		}
	}

	return strings.Join(words, "")
}

func toCamelCase(s string) string {
	pascal := toPascalCase(s)
	if len(pascal) > 0 {
		return strings.ToLower(pascal[:1]) + pascal[1:]
	}
	return pascal
}

func toSnakeCase(s string) string {
	var result []rune
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result = append(result, '_')
		}
		result = append(result, r)
	}
	return strings.ToLower(string(result))
}
