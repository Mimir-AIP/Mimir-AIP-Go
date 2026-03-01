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
	Entities          []ExtractedEntity       `json:"entities"`
	Relationships     []ExtractedRelationship `json:"relationships"`
	Attributes        []ExtractedAttribute    `json:"attributes,omitempty"`
	Source            string                  `json:"source"` // "structured" or "unstructured"
	CrossSourceLinks  []CrossSourceLink       `json:"cross_source_links,omitempty"`
}

// ColumnProfile captures the statistical fingerprint of a single column within
// a storage source. It is computed during extraction and used by the cross-source
// link detection algorithm without any domain-specific configuration.
type ColumnProfile struct {
	StorageID        string          `json:"storage_id"`
	EntityType       string          `json:"entity_type"`   // inferred entity type for the table
	ColumnName       string          `json:"column_name"`
	ValueSample      map[string]bool `json:"-"`             // in-memory only; not serialised
	TotalRows        int             `json:"total_rows"`
	UniqueCount      int             `json:"unique_count"`
	CardinalityRatio float64         `json:"cardinality_ratio"` // UniqueCount / TotalRows
	IsNumeric        bool            `json:"is_numeric"`
	IsLikelyKey      bool            `json:"is_likely_key"` // high cardinality or key-like name
}

// CrossSourceLink describes a statistically-discovered bridge between two columns
// from different storage sources — a foreign-key-like join inferred purely from
// value overlap and column name similarity. No domain configuration is required.
type CrossSourceLink struct {
	StorageA         string  `json:"storage_a"`
	ColumnA          string  `json:"column_a"`
	EntityTypeA      string  `json:"entity_type_a"`
	StorageB         string  `json:"storage_b"`
	ColumnB          string  `json:"column_b"`
	EntityTypeB      string  `json:"entity_type_b"`
	Confidence       float64 `json:"confidence"`
	NameSimilarity   float64 `json:"name_similarity"`
	ValueOverlap     float64 `json:"value_overlap"`     // Jaccard index of value sets
	SharedValueCount int     `json:"shared_value_count"`
}

