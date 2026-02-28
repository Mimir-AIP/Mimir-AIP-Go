package models

// ExtractedEntity represents an entity extracted from data
type ExtractedEntity struct {
	Name       string                 `json:"name"`
	Attributes map[string]interface{} `json:"attributes"`
	Source     string                 `json:"source"`            // "structured" or "unstructured"
	Confidence float64                `json:"confidence"`        // 0.0 to 1.0
	Sources    []string               `json:"sources,omitempty"` // Multiple sources if reconciled
}

// ExtractedRelationship represents a relationship between entities
type ExtractedRelationship struct {
	Entity1    *ExtractedEntity `json:"entity1"`
	Entity2    *ExtractedEntity `json:"entity2"`
	Relation   string           `json:"relation"`
	Confidence float64          `json:"confidence"` // 0.0 to 1.0
}

// ExtractedAttribute represents an attribute associated with an entity
type ExtractedAttribute struct {
	Entity     *ExtractedEntity `json:"entity"`
	Attribute  string           `json:"attribute"`
	Type       string           `json:"type"` // "descriptive", "quantitative", etc.
	Confidence float64          `json:"confidence"`
}

// ExtractionResult represents the complete result of entity extraction
type ExtractionResult struct {
	Entities      []ExtractedEntity       `json:"entities"`
	Relationships []ExtractedRelationship `json:"relationships"`
	Attributes    []ExtractedAttribute    `json:"attributes,omitempty"`
	Source        string                  `json:"source"` // "structured" or "unstructured"
}

