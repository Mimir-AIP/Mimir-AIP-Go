package ontology

// Triple represents an RDF triple
type Triple struct {
	Subject   string `json:"subject"`
	Predicate string `json:"predicate"`
	Object    string `json:"object"`
	Datatype  string `json:"datatype,omitempty"`
	Language  string `json:"language,omitempty"`
}

// Entity represents an extracted entity (in-memory, before DB storage)
type Entity struct {
	URI        string         `json:"uri"`
	Type       string         `json:"type"` // Class URI
	Label      string         `json:"label,omitempty"`
	Properties map[string]any `json:"properties"`
	Confidence float64        `json:"confidence,omitempty"`
	SourceText string         `json:"source_text,omitempty"`
}

// ExtractionResult represents the result of an extraction operation
type ExtractionResult struct {
	Entities          []Entity       `json:"entities"`
	Triples           []Triple       `json:"triples"`
	EntitiesExtracted int            `json:"entities_extracted"`
	TriplesGenerated  int            `json:"triples_generated"`
	Confidence        float64        `json:"confidence"`
	ExtractionType    ExtractionType `json:"extraction_type"`
	Warnings          []string       `json:"warnings,omitempty"`
	Metadata          map[string]any `json:"metadata,omitempty"`
}

// PropertyMapping represents a mapping from a data field to an ontology property
type PropertyMapping struct {
	SourceField string  `json:"source_field"`
	PropertyURI string  `json:"property_uri"`
	Confidence  float64 `json:"confidence"`
	Transform   string  `json:"transform,omitempty"` // e.g., "lowercase", "trim", "parse_date"
}

// ExtractionConfig holds configuration for extraction operations
type ExtractionConfig struct {
	OntologyID     string            `json:"ontology_id"`
	SourceType     string            `json:"source_type"`
	ExtractionType ExtractionType    `json:"extraction_type"`
	Mappings       []PropertyMapping `json:"mappings,omitempty"`
	LLMConfig      map[string]any    `json:"llm_config,omitempty"`
	Options        map[string]any    `json:"options,omitempty"`
}

// OntologyContext provides full ontology details needed for extraction
type OntologyContext struct {
	Metadata   *OntologyMetadata
	BaseURI    string
	Classes    []OntologyClass
	Properties []OntologyProperty
}

// Extractor is the interface that all extractors must implement
type Extractor interface {
	// Extract extracts entities and relationships from data
	Extract(data any, ontology *OntologyContext) (*ExtractionResult, error)

	// GetType returns the extraction type
	GetType() ExtractionType

	// GetSupportedSourceTypes returns the source types this extractor supports
	GetSupportedSourceTypes() []string
}

// SourceType constants
const (
	SourceTypeCSV  = "csv"
	SourceTypeJSON = "json"
	SourceTypeText = "text"
	SourceTypeHTML = "html"
	SourceTypeAPI  = "api"
)
