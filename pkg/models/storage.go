package models

import "time"

// ExternalStoragePlugin records metadata about a dynamically installed storage plugin.
// The compiled .so is cached on disk; this record tracks its provenance and status.
type ExternalStoragePlugin struct {
	Name          string    `json:"name"`
	Version       string    `json:"version"`
	Description   string    `json:"description"`
	Author        string    `json:"author"`
	RepositoryURL string    `json:"repository_url"`
	GitCommitHash string    `json:"git_commit_hash"`
	Status        string    `json:"status"` // "active" | "error"
	ErrorMessage  string    `json:"error_message,omitempty"`
	InstalledAt   time.Time `json:"installed_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// ExternalStoragePluginInstallRequest is the body for POST /api/storage-plugins.
type ExternalStoragePluginInstallRequest struct {
	RepositoryURL string `json:"repository_url"`
	GitRef        string `json:"git_ref,omitempty"` // branch / tag / SHA; defaults to "main"
}

// StoragePlugin defines the interface that all storage plugins must implement
// It provides bidirectional translation between CIR format and storage-specific formats
type StoragePlugin interface {
	// Initialize the plugin with configuration
	Initialize(config *PluginConfig) error

	// CreateSchema creates or updates the storage schema based on the ontology definition
	CreateSchema(ontology *OntologyDefinition) error

	// Store CIR data into the storage system
	Store(cir *CIR) (*StorageResult, error)

	// Retrieve data using queries and return as CIR objects
	Retrieve(query *CIRQuery) ([]*CIR, error)

	// Update existing CIR data
	Update(query *CIRQuery, updates *CIRUpdate) (*StorageResult, error)

	// Delete CIR data
	Delete(query *CIRQuery) (*StorageResult, error)

	// GetMetadata returns storage-specific metadata
	GetMetadata() (*StorageMetadata, error)

	// HealthCheck validates connection and storage availability
	HealthCheck() (bool, error)
}

// PluginConfig contains configuration for storage plugins
type PluginConfig struct {
	ConnectionString string                 `json:"connection_string"`
	Credentials      map[string]interface{} `json:"credentials,omitempty"`
	Options          map[string]interface{} `json:"options,omitempty"`
}

// OntologyDefinition defines the structure of entities and relationships
type OntologyDefinition struct {
	Entities      []EntityDefinition       `json:"entities"`
	Relationships []RelationshipDefinition `json:"relationships"`
}

// EntityDefinition defines an entity type
type EntityDefinition struct {
	Name       string                `json:"name"`
	Attributes []AttributeDefinition `json:"attributes"`
	PrimaryKey []string              `json:"primary_key,omitempty"`
}

// AttributeDefinition defines an attribute of an entity
type AttributeDefinition struct {
	Name         string      `json:"name"`
	Type         string      `json:"type"` // string, number, boolean, date, json
	Nullable     bool        `json:"nullable,omitempty"`
	DefaultValue interface{} `json:"default_value,omitempty"`
}

// RelationshipDefinition defines a relationship between entities
type RelationshipDefinition struct {
	Name       string `json:"name"`
	FromEntity string `json:"from_entity"`
	ToEntity   string `json:"to_entity"`
	Type       string `json:"type"` // one-to-one, one-to-many, many-to-many
}

// CIRQuery represents a query for retrieving CIR data
type CIRQuery struct {
	EntityType string          `json:"entity_type,omitempty"`
	Filters    []CIRCondition  `json:"filters,omitempty"`
	OrderBy    []OrderByClause `json:"order_by,omitempty"`
	Limit      int             `json:"limit,omitempty"`
	Offset     int             `json:"offset,omitempty"`
}

// CIRCondition represents a filter condition
type CIRCondition struct {
	Attribute string      `json:"attribute"`
	Operator  string      `json:"operator"` // eq, neq, gt, gte, lt, lte, in, like
	Value     interface{} `json:"value"`
}

// OrderByClause represents sorting criteria
type OrderByClause struct {
	Attribute string `json:"attribute"`
	Direction string `json:"direction"` // asc, desc
}

// CIRUpdate represents updates to apply to CIR data
type CIRUpdate struct {
	Filters []CIRCondition         `json:"filters"`
	Updates map[string]interface{} `json:"updates"`
}

// StorageResult represents the result of a storage operation
type StorageResult struct {
	Success       bool   `json:"success"`
	AffectedItems int    `json:"affected_items,omitempty"`
	Error         string `json:"error,omitempty"`
}

// StorageMetadata provides information about the storage system
type StorageMetadata struct {
	StorageType  string   `json:"storage_type"`
	Version      string   `json:"version"`
	Capabilities []string `json:"capabilities"`
}

type IngestionHealthStatus string

const (
	IngestionHealthHealthy  IngestionHealthStatus = "healthy"
	IngestionHealthWarning  IngestionHealthStatus = "warning"
	IngestionHealthCritical IngestionHealthStatus = "critical"
)

// IngestionHealthSource captures ingestion quality metrics for one storage source.
type IngestionHealthSource struct {
	StorageID         string                `json:"storage_id"`
	PluginType        string                `json:"plugin_type"`
	SampleSize        int                   `json:"sample_size"`
	LastIngestedAt    *time.Time            `json:"last_ingested_at,omitempty"`
	FreshnessScore    float64               `json:"freshness_score"`
	CompletenessScore float64               `json:"completeness_score"`
	SchemaDriftScore  float64               `json:"schema_drift_score"`
	OverallScore      float64               `json:"overall_score"`
	Status            IngestionHealthStatus `json:"status"`
	Findings          []string              `json:"findings,omitempty"`
}

// IngestionHealthReport is a project-level ingestion quality summary.
type IngestionHealthReport struct {
	ProjectID       string                  `json:"project_id"`
	GeneratedAt     time.Time               `json:"generated_at"`
	OverallScore    float64                 `json:"overall_score"`
	Status          IngestionHealthStatus   `json:"status"`
	Sources         []IngestionHealthSource `json:"sources"`
	Recommendations []string                `json:"recommendations,omitempty"`
}

// StorageConfig represents the configuration for a project's storage
type StorageConfig struct {
	ID         string                 `json:"id"`
	ProjectID  string                 `json:"project_id"`
	PluginType string                 `json:"plugin_type"` // filesystem, s3, neo4j, postgres, etc.
	Config     map[string]interface{} `json:"config"`
	OntologyID string                 `json:"ontology_id,omitempty"`
	Active     bool                   `json:"active"`
	CreatedAt  string                 `json:"created_at"`
	UpdatedAt  string                 `json:"updated_at"`
}

// StorageItem represents a stored CIR item
type StorageItem struct {
	ID         string                 `json:"id"`
	ProjectID  string                 `json:"project_id"`
	StorageID  string                 `json:"storage_id"`
	EntityType string                 `json:"entity_type,omitempty"`
	CIRData    *CIR                   `json:"cir_data"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt  string                 `json:"created_at"`
	UpdatedAt  string                 `json:"updated_at"`
}

// StorageQueryRequest represents a request to query stored data
type StorageQueryRequest struct {
	ProjectID string    `json:"project_id"`
	StorageID string    `json:"storage_id"`
	Query     *CIRQuery `json:"query"`
}

// StorageStoreRequest represents a request to store CIR data
type StorageStoreRequest struct {
	ProjectID string `json:"project_id"`
	StorageID string `json:"storage_id"`
	CIRData   *CIR   `json:"cir_data"`
}

// StorageUpdateRequest represents a request to update stored data
type StorageUpdateRequest struct {
	ProjectID string     `json:"project_id"`
	StorageID string     `json:"storage_id"`
	Query     *CIRQuery  `json:"query"`
	Updates   *CIRUpdate `json:"updates"`
}

// StorageDeleteRequest represents a request to delete stored data
type StorageDeleteRequest struct {
	ProjectID string    `json:"project_id"`
	StorageID string    `json:"storage_id"`
	Query     *CIRQuery `json:"query"`
}
