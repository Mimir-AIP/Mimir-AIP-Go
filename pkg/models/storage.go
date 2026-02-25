package models

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
