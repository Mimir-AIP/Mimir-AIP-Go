package ontology

import (
	"time"
)

// OntologyFormat represents the serialization format of an ontology
type OntologyFormat string

const (
	FormatTurtle   OntologyFormat = "turtle"
	FormatRDFXML   OntologyFormat = "rdfxml"
	FormatNTriples OntologyFormat = "ntriples"
	FormatJSONLD   OntologyFormat = "jsonld"
	FormatOwlXML   OntologyFormat = "owlxml"
)

// OntologyStatus represents the current status of an ontology
type OntologyStatus string

const (
	StatusActive     OntologyStatus = "active"
	StatusDeprecated OntologyStatus = "deprecated"
	StatusDraft      OntologyStatus = "draft"
	StatusArchived   OntologyStatus = "archived"
)

// PropertyType represents the type of an ontology property
type PropertyType string

const (
	PropertyTypeDatatype   PropertyType = "datatype"
	PropertyTypeObject     PropertyType = "object"
	PropertyTypeAnnotation PropertyType = "annotation"
)

// OntologyMetadata contains metadata about an ontology
type OntologyMetadata struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Version     string         `json:"version"`
	FilePath    string         `json:"file_path"`
	TDB2Graph   string         `json:"tdb2_graph"`
	Format      OntologyFormat `json:"format"`
	Status      OntologyStatus `json:"status"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	CreatedBy   string         `json:"created_by,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

// OntologyClass represents a class (concept) in an ontology
type OntologyClass struct {
	URI         string   `json:"uri"`
	Label       string   `json:"label,omitempty"`
	Description string   `json:"description,omitempty"`
	ParentURIs  []string `json:"parent_uris,omitempty"`
	Deprecated  bool     `json:"deprecated,omitempty"`
}

// OntologyProperty represents a property (relationship) in an ontology
type OntologyProperty struct {
	URI          string       `json:"uri"`
	Label        string       `json:"label,omitempty"`
	Description  string       `json:"description,omitempty"`
	PropertyType PropertyType `json:"property_type"`
	Domain       []string     `json:"domain,omitempty"`
	Range        []string     `json:"range,omitempty"`
	Deprecated   bool         `json:"deprecated,omitempty"`
}

// OntologyStats provides statistics about an ontology
type OntologyStats struct {
	OntologyID      string `json:"ontology_id"`
	TotalClasses    int    `json:"total_classes"`
	TotalProperties int    `json:"total_properties"`
	TotalTriples    int    `json:"total_triples"`
	TotalEntities   int    `json:"total_entities"`
}

// ValidationResult represents the result of ontology validation
type ValidationResult struct {
	Valid    bool              `json:"valid"`
	Errors   []ValidationError `json:"errors,omitempty"`
	Warnings []ValidationError `json:"warnings,omitempty"`
}

// ValidationError represents a validation error or warning
type ValidationError struct {
	Severity string `json:"severity"`
	Message  string `json:"message"`
	Location string `json:"location,omitempty"`
	Line     int    `json:"line,omitempty"`
}

// ChangeType represents the type of ontology change
type ChangeType string

const (
	ChangeTypeAddClass       ChangeType = "add_class"
	ChangeTypeRemoveClass    ChangeType = "remove_class"
	ChangeTypeModifyClass    ChangeType = "modify_class"
	ChangeTypeAddProperty    ChangeType = "add_property"
	ChangeTypeRemoveProperty ChangeType = "remove_property"
	ChangeTypeModifyProperty ChangeType = "modify_property"
	ChangeTypeAddAxiom       ChangeType = "add_axiom"
	ChangeTypeRemoveAxiom    ChangeType = "remove_axiom"
)

// OntologyChange represents a change in an ontology
type OntologyChange struct {
	ID          int        `json:"id"`
	VersionID   int        `json:"version_id"`
	ChangeType  ChangeType `json:"change_type"`
	EntityType  string     `json:"entity_type"`
	EntityURI   string     `json:"entity_uri"`
	OldValue    string     `json:"old_value,omitempty"`
	NewValue    string     `json:"new_value,omitempty"`
	Description string     `json:"description,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

// MigrationStrategy represents how to handle ontology updates
type MigrationStrategy string

const (
	MigrationInPlace    MigrationStrategy = "in_place"
	MigrationDualSchema MigrationStrategy = "dual_schema"
	MigrationSnapshot   MigrationStrategy = "snapshot"
)

// OntologyVersion represents a version of an ontology
type OntologyVersion struct {
	ID                int               `json:"id"`
	OntologyID        string            `json:"ontology_id"`
	Version           string            `json:"version"`
	PreviousVersion   string            `json:"previous_version,omitempty"`
	Changelog         string            `json:"changelog,omitempty"`
	MigrationStrategy MigrationStrategy `json:"migration_strategy"`
	CreatedAt         time.Time         `json:"created_at"`
	CreatedBy         string            `json:"created_by,omitempty"`
}

// RiskLevel represents the risk level of a change
type RiskLevel string

const (
	RiskLevelLow      RiskLevel = "low"
	RiskLevelMedium   RiskLevel = "medium"
	RiskLevelHigh     RiskLevel = "high"
	RiskLevelCritical RiskLevel = "critical"
)

// SuggestionType represents the type of ontology suggestion
type SuggestionType string

const (
	SuggestionAddClass       SuggestionType = "add_class"
	SuggestionAddProperty    SuggestionType = "add_property"
	SuggestionModifyClass    SuggestionType = "modify_class"
	SuggestionModifyProperty SuggestionType = "modify_property"
	SuggestionDeprecate      SuggestionType = "deprecate"
)

// SuggestionStatus represents the status of a suggestion
type SuggestionStatus string

const (
	SuggestionPending  SuggestionStatus = "pending"
	SuggestionApproved SuggestionStatus = "approved"
	SuggestionRejected SuggestionStatus = "rejected"
	SuggestionApplied  SuggestionStatus = "applied"
)

// OntologySuggestion represents an AI-generated suggestion for ontology changes
type OntologySuggestion struct {
	ID             int              `json:"id"`
	OntologyID     string           `json:"ontology_id"`
	SuggestionType SuggestionType   `json:"suggestion_type"`
	EntityType     string           `json:"entity_type"`
	EntityURI      string           `json:"entity_uri,omitempty"`
	Confidence     float64          `json:"confidence"`
	Reasoning      string           `json:"reasoning"`
	Status         SuggestionStatus `json:"status"`
	RiskLevel      RiskLevel        `json:"risk_level"`
	CreatedAt      time.Time        `json:"created_at"`
	ReviewedAt     *time.Time       `json:"reviewed_at,omitempty"`
	ReviewedBy     string           `json:"reviewed_by,omitempty"`
	ReviewDecision string           `json:"review_decision,omitempty"`
	ReviewNotes    string           `json:"review_notes,omitempty"`
}

// ExtractionType represents the type of entity extraction
type ExtractionType string

const (
	ExtractionDeterministic ExtractionType = "deterministic"
	ExtractionLLM           ExtractionType = "llm"
	ExtractionHybrid        ExtractionType = "hybrid"
)

// ExtractionJobStatus represents the status of an extraction job
type ExtractionJobStatus string

const (
	ExtractionPending   ExtractionJobStatus = "pending"
	ExtractionRunning   ExtractionJobStatus = "running"
	ExtractionCompleted ExtractionJobStatus = "completed"
	ExtractionFailed    ExtractionJobStatus = "failed"
)

// ExtractionJob represents an entity extraction job
type ExtractionJob struct {
	ID                string              `json:"id"`
	OntologyID        string              `json:"ontology_id"`
	PipelineID        string              `json:"pipeline_id,omitempty"`
	JobName           string              `json:"job_name"`
	Status            ExtractionJobStatus `json:"status"`
	ExtractionType    ExtractionType      `json:"extraction_type"`
	SourceType        string              `json:"source_type"`
	SourcePath        string              `json:"source_path,omitempty"`
	EntitiesExtracted int                 `json:"entities_extracted"`
	TriplesGenerated  int                 `json:"triples_generated"`
	ErrorMessage      string              `json:"error_message,omitempty"`
	StartedAt         *time.Time          `json:"started_at,omitempty"`
	CompletedAt       *time.Time          `json:"completed_at,omitempty"`
	CreatedAt         time.Time           `json:"created_at"`
	Metadata          map[string]any      `json:"metadata,omitempty"`
}

// ExtractedEntity represents an entity extracted from data
type ExtractedEntity struct {
	ID          int            `json:"id"`
	JobID       string         `json:"job_id"`
	EntityURI   string         `json:"entity_uri"`
	EntityType  string         `json:"entity_type"`
	EntityLabel string         `json:"entity_label,omitempty"`
	Confidence  float64        `json:"confidence,omitempty"`
	SourceText  string         `json:"source_text,omitempty"`
	Properties  map[string]any `json:"properties,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
}

// DriftDetectionStatus represents the status of a drift detection run
type DriftDetectionStatus string

const (
	DriftRunning   DriftDetectionStatus = "running"
	DriftCompleted DriftDetectionStatus = "completed"
	DriftFailed    DriftDetectionStatus = "failed"
)

// DriftDetection represents a drift detection run
type DriftDetection struct {
	ID                   int                  `json:"id"`
	OntologyID           string               `json:"ontology_id"`
	DetectionType        string               `json:"detection_type"`
	DataSource           string               `json:"data_source"`
	SuggestionsGenerated int                  `json:"suggestions_generated"`
	Status               DriftDetectionStatus `json:"status"`
	StartedAt            time.Time            `json:"started_at"`
	CompletedAt          *time.Time           `json:"completed_at,omitempty"`
	ErrorMessage         string               `json:"error_message,omitempty"`
}
