package models

import (
	"fmt"
	"time"
)

// DigitalTwin represents a digital twin instance for a project
// It uses a hybrid storage approach: references CIR data + stores deltas
type DigitalTwin struct {
	ID          string                 `json:"id"`
	ProjectID   string                 `json:"project_id"`
	OntologyID  string                 `json:"ontology_id"` // Ontology blueprint
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Status      string                 `json:"status"` // active, syncing, error
	Config      *DigitalTwinConfig     `json:"config,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	LastSyncAt  *time.Time             `json:"last_sync_at,omitempty"`
}

// DigitalTwinConfig holds configuration for a digital twin
type DigitalTwinConfig struct {
	StorageIDs         []string          `json:"storage_ids"`          // CIR data sources to reference
	CacheTTL           int               `json:"cache_ttl"`            // Cache time-to-live in seconds
	AutoSync           bool              `json:"auto_sync"`            // Automatically sync with storage changes
	SyncInterval       int               `json:"sync_interval"`        // Sync interval in seconds
	EnablePredictions  bool              `json:"enable_predictions"`   // Enable ML predictions
	PredictionCacheTTL int               `json:"prediction_cache_ttl"` // Prediction cache TTL
	IndexingStrategy   string            `json:"indexing_strategy"`    // "lazy" or "eager"
	CustomSettings     map[string]string `json:"custom_settings,omitempty"`
}

// Entity represents an entity instance in the digital twin
// It can reference stored data or contain modifications
type Entity struct {
	ID             string                 `json:"id"`
	DigitalTwinID  string                 `json:"digital_twin_id"`
	Type           string                 `json:"type"`                     // Entity type from ontology (e.g., "User", "Product")
	Attributes     map[string]interface{} `json:"attributes"`               // Current attribute values
	SourceDataID   *string                `json:"source_data_id,omitempty"` // Reference to CIR data ID
	IsModified     bool                   `json:"is_modified"`              // True if has delta changes
	Modifications  map[string]interface{} `json:"modifications,omitempty"`  // Delta changes
	Relationships  []*EntityRelationship  `json:"relationships,omitempty"`
	ComputedValues map[string]interface{} `json:"computed_values,omitempty"` // Cached computed values
	CreatedAt      time.Time              `json:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at"`
}

// EntityRelationship represents a relationship between entities
type EntityRelationship struct {
	Type       string                 `json:"type"`                 // Relationship type from ontology
	TargetID   string                 `json:"target_id"`            // Target entity ID
	TargetType string                 `json:"target_type"`          // Target entity type
	Properties map[string]interface{} `json:"properties,omitempty"` // Relationship properties
}

// Scenario represents a what-if scenario with hypothetical modifications
type Scenario struct {
	ID            string                  `json:"id"`
	DigitalTwinID string                  `json:"digital_twin_id"`
	Name          string                  `json:"name"`
	Description   string                  `json:"description,omitempty"`
	BaseState     string                  `json:"base_state"` // "current" or "historical"
	Modifications []*ScenarioModification `json:"modifications"`
	Predictions   []*ScenarioPrediction   `json:"predictions,omitempty"`
	Status        string                  `json:"status"` // "active", "archived"
	CreatedBy     string                  `json:"created_by,omitempty"`
	CreatedAt     time.Time               `json:"created_at"`
	UpdatedAt     time.Time               `json:"updated_at"`
}

// ScenarioModification represents a single modification in a scenario
type ScenarioModification struct {
	EntityType    string      `json:"entity_type"`
	EntityID      string      `json:"entity_id"`
	Attribute     string      `json:"attribute"`
	OriginalValue interface{} `json:"original_value"`
	NewValue      interface{} `json:"new_value"`
	Rationale     string      `json:"rationale,omitempty"`
}

// ScenarioPrediction represents a prediction result within a scenario
type ScenarioPrediction struct {
	ModelID    string                 `json:"model_id"`
	ModelName  string                 `json:"model_name"`
	EntityID   string                 `json:"entity_id,omitempty"`
	EntityType string                 `json:"entity_type,omitempty"`
	Prediction interface{}            `json:"prediction"`
	Confidence float64                `json:"confidence,omitempty"`
	Impact     string                 `json:"impact,omitempty"` // Human-readable impact description
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// Prediction represents an ML model prediction result
type Prediction struct {
	ID             string                 `json:"id"`
	DigitalTwinID  string                 `json:"digital_twin_id"`
	ModelID        string                 `json:"model_id"`
	EntityID       string                 `json:"entity_id,omitempty"`
	EntityType     string                 `json:"entity_type,omitempty"`
	PredictionType string                 `json:"prediction_type"` // "point", "batch", "anomaly"
	Input          map[string]interface{} `json:"input"`
	Output         interface{}            `json:"output"`
	Confidence     float64                `json:"confidence,omitempty"`
	CachedAt       time.Time              `json:"cached_at"`
	ExpiresAt      time.Time              `json:"expires_at"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}

// Action represents a conditional trigger in the digital twin
type Action struct {
	ID            string           `json:"id"`
	DigitalTwinID string           `json:"digital_twin_id"`
	Name          string           `json:"name"`
	Description   string           `json:"description,omitempty"`
	Enabled       bool             `json:"enabled"`
	Condition     *ActionCondition `json:"condition"`
	Trigger       *ActionTrigger   `json:"trigger"`
	LastTriggered *time.Time       `json:"last_triggered,omitempty"`
	TriggerCount  int              `json:"trigger_count"`
	CreatedAt     time.Time        `json:"created_at"`
	UpdatedAt     time.Time        `json:"updated_at"`
}

// ActionCondition defines when an action should trigger
type ActionCondition struct {
	ModelID    string      `json:"model_id,omitempty"`    // ML model to check
	Attribute  string      `json:"attribute,omitempty"`   // Entity attribute to check
	Operator   string      `json:"operator"`              // "gt", "lt", "eq", "gte", "lte", "ne"
	Threshold  interface{} `json:"threshold"`             // Threshold value
	EntityType string      `json:"entity_type,omitempty"` // Entity type to monitor
}

// ActionTrigger defines what happens when condition is met
type ActionTrigger struct {
	PipelineID string                 `json:"pipeline_id"`          // Pipeline to execute
	Parameters map[string]interface{} `json:"parameters,omitempty"` // Pipeline parameters
}

// QueryResult represents the result of a SPARQL query
type QueryResult struct {
	Columns  []string                 `json:"columns"`
	Rows     []map[string]interface{} `json:"rows"`
	Count    int                      `json:"count"`
	Metadata map[string]interface{}   `json:"metadata,omitempty"`
}

// DigitalTwinCreateRequest represents a request to create a digital twin
type DigitalTwinCreateRequest struct {
	ProjectID   string             `json:"project_id"`
	OntologyID  string             `json:"ontology_id"`
	Name        string             `json:"name"`
	Description string             `json:"description,omitempty"`
	Config      *DigitalTwinConfig `json:"config,omitempty"`
}

// Validate checks if the DigitalTwinCreateRequest is valid
func (r *DigitalTwinCreateRequest) Validate() error {
	if r.ProjectID == "" {
		return fmt.Errorf("project_id is required")
	}
	if r.OntologyID == "" {
		return fmt.Errorf("ontology_id is required")
	}
	if r.Name == "" {
		return fmt.Errorf("name is required")
	}
	return nil
}

// DigitalTwinUpdateRequest represents a request to update a digital twin
type DigitalTwinUpdateRequest struct {
	Name        *string            `json:"name,omitempty"`
	Description *string            `json:"description,omitempty"`
	Status      *string            `json:"status,omitempty"`
	Config      *DigitalTwinConfig `json:"config,omitempty"`
}

// QueryRequest represents a SPARQL query request
type QueryRequest struct {
	Query    string                 `json:"query"`
	Bindings map[string]interface{} `json:"bindings,omitempty"` // Variable bindings
	Limit    int                    `json:"limit,omitempty"`
	Offset   int                    `json:"offset,omitempty"`
}

// Validate checks if the QueryRequest is valid
func (r *QueryRequest) Validate() error {
	if r.Query == "" {
		return fmt.Errorf("query is required")
	}
	return nil
}

// PredictionRequest represents a request for ML model prediction
type PredictionRequest struct {
	ModelID    string                 `json:"model_id"`
	EntityID   string                 `json:"entity_id,omitempty"`
	EntityType string                 `json:"entity_type,omitempty"`
	Input      map[string]interface{} `json:"input"`
	UseCache   bool                   `json:"use_cache"`
}

// Validate checks if the PredictionRequest is valid
func (r *PredictionRequest) Validate() error {
	if r.ModelID == "" {
		return fmt.Errorf("model_id is required")
	}
	if r.Input == nil || len(r.Input) == 0 {
		return fmt.Errorf("input is required")
	}
	return nil
}

// BatchPredictionRequest represents a request for batch predictions
type BatchPredictionRequest struct {
	ModelID  string                   `json:"model_id"`
	Inputs   []map[string]interface{} `json:"inputs"`
	UseCache bool                     `json:"use_cache"`
}

// Validate checks if the BatchPredictionRequest is valid
func (r *BatchPredictionRequest) Validate() error {
	if r.ModelID == "" {
		return fmt.Errorf("model_id is required")
	}
	if len(r.Inputs) == 0 {
		return fmt.Errorf("inputs are required")
	}
	return nil
}

// ScenarioCreateRequest represents a request to create a scenario
type ScenarioCreateRequest struct {
	Name           string                  `json:"name"`
	Description    string                  `json:"description,omitempty"`
	BaseState      string                  `json:"base_state,omitempty"` // defaults to "current"
	Modifications  []*ScenarioModification `json:"modifications"`
	RunPredictions bool                    `json:"run_predictions,omitempty"` // Whether to run predictions immediately
}

// Validate checks if the ScenarioCreateRequest is valid
func (r *ScenarioCreateRequest) Validate() error {
	if r.Name == "" {
		return fmt.Errorf("name is required")
	}
	if len(r.Modifications) == 0 {
		return fmt.Errorf("at least one modification is required")
	}
	for _, mod := range r.Modifications {
		if mod.EntityType == "" {
			return fmt.Errorf("entity_type is required for all modifications")
		}
		if mod.EntityID == "" {
			return fmt.Errorf("entity_id is required for all modifications")
		}
		if mod.Attribute == "" {
			return fmt.Errorf("attribute is required for all modifications")
		}
	}
	return nil
}

// ActionCreateRequest represents a request to create an action
type ActionCreateRequest struct {
	Name        string           `json:"name"`
	Description string           `json:"description,omitempty"`
	Enabled     bool             `json:"enabled"`
	Condition   *ActionCondition `json:"condition"`
	Trigger     *ActionTrigger   `json:"trigger"`
}

// Validate checks if the ActionCreateRequest is valid
func (r *ActionCreateRequest) Validate() error {
	if r.Name == "" {
		return fmt.Errorf("name is required")
	}
	if r.Condition == nil {
		return fmt.Errorf("condition is required")
	}
	if r.Condition.Operator == "" {
		return fmt.Errorf("condition operator is required")
	}
	if r.Condition.Threshold == nil {
		return fmt.Errorf("condition threshold is required")
	}
	if r.Trigger == nil {
		return fmt.Errorf("trigger is required")
	}
	if r.Trigger.PipelineID == "" {
		return fmt.Errorf("trigger pipeline_id is required")
	}
	return nil
}

// EntityUpdateRequest represents a request to update an entity
type EntityUpdateRequest struct {
	Attributes map[string]interface{} `json:"attributes"`
}

// Validate checks if the EntityUpdateRequest is valid
func (r *EntityUpdateRequest) Validate() error {
	if len(r.Attributes) == 0 {
		return fmt.Errorf("attributes are required")
	}
	return nil
}
