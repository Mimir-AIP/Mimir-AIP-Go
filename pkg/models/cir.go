package models

import (
	"encoding/json"
	"fmt"
	"time"
)

// CIRVersion is the current version of the CIR schema
const CIRVersion = "1.0"

// SourceType represents the type of data source
type SourceType string

const (
	SourceTypeAPI      SourceType = "api"
	SourceTypeFile     SourceType = "file"
	SourceTypeDatabase SourceType = "database"
	SourceTypeStream   SourceType = "stream"
)

// DataFormat represents the original format of the data
type DataFormat string

const (
	DataFormatCSV    DataFormat = "csv"
	DataFormatJSON   DataFormat = "json"
	DataFormatXML    DataFormat = "xml"
	DataFormatText   DataFormat = "text"
	DataFormatBinary DataFormat = "binary"
)

// CIR represents the Common Internal Representation for all ingested data
// This is the standardized format that bridges raw data ingestion and structured storage
type CIR struct {
	Version  string      `json:"version"`
	Source   CIRSource   `json:"source"`
	Data     interface{} `json:"data"`
	Metadata CIRMetadata `json:"metadata"`
}

// CIRSource contains information about the data source
type CIRSource struct {
	Type       SourceType             `json:"type"`       // e.g., "api", "file", "database", "stream"
	URI        string                 `json:"uri"`        // Source identifier (URL, file path, etc.)
	Timestamp  time.Time              `json:"timestamp"`  // Timestamp of ingestion
	Format     DataFormat             `json:"format"`     // Original format: "csv", "json", "xml", "text", "binary"
	Parameters map[string]interface{} `json:"parameters"` // Optional ingestion parameters
}

// CIRMetadata contains metadata about the data
type CIRMetadata struct {
	Size            int64                  `json:"size"`                       // Data size in bytes
	Encoding        string                 `json:"encoding,omitempty"`         // Character encoding if applicable
	RecordCount     int                    `json:"record_count,omitempty"`     // Number of records/items (for structured data)
	SchemaInference map[string]interface{} `json:"schema_inference,omitempty"` // Optional inferred schema information
	QualityMetrics  map[string]interface{} `json:"quality_metrics,omitempty"`  // Optional data quality indicators
}

// NewCIR creates a new CIR instance with the current version
func NewCIR(sourceType SourceType, sourceURI string, format DataFormat, data interface{}) *CIR {
	return &CIR{
		Version: CIRVersion,
		Source: CIRSource{
			Type:       sourceType,
			URI:        sourceURI,
			Timestamp:  time.Now(),
			Format:     format,
			Parameters: make(map[string]interface{}),
		},
		Data: data,
		Metadata: CIRMetadata{
			Size: int64(len(fmt.Sprintf("%v", data))),
		},
	}
}

// Validate checks if the CIR structure is valid
func (c *CIR) Validate() error {
	if c.Version == "" {
		return fmt.Errorf("CIR version is required")
	}

	if c.Source.Type == "" {
		return fmt.Errorf("CIR source type is required")
	}

	if c.Source.URI == "" {
		return fmt.Errorf("CIR source URI is required")
	}

	if c.Source.Format == "" {
		return fmt.Errorf("CIR source format is required")
	}

	if c.Data == nil {
		return fmt.Errorf("CIR data cannot be nil")
	}

	return nil
}

// ToJSON converts the CIR to JSON
func (c *CIR) ToJSON() ([]byte, error) {
	return json.Marshal(c)
}

// FromJSON parses a CIR from JSON
func FromJSON(data []byte) (*CIR, error) {
	var cir CIR
	if err := json.Unmarshal(data, &cir); err != nil {
		return nil, fmt.Errorf("failed to unmarshal CIR: %w", err)
	}

	if err := cir.Validate(); err != nil {
		return nil, fmt.Errorf("invalid CIR: %w", err)
	}

	return &cir, nil
}

// GetDataAsMap attempts to convert the data to a map[string]interface{}
func (c *CIR) GetDataAsMap() (map[string]interface{}, error) {
	if m, ok := c.Data.(map[string]interface{}); ok {
		return m, nil
	}

	// Try to convert via JSON marshaling/unmarshaling
	data, err := json.Marshal(c.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal data: %w", err)
	}

	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("failed to unmarshal data as map: %w", err)
	}

	return m, nil
}

// GetDataAsArray attempts to convert the data to []interface{}
func (c *CIR) GetDataAsArray() ([]interface{}, error) {
	if arr, ok := c.Data.([]interface{}); ok {
		return arr, nil
	}

	// Try to convert via JSON marshaling/unmarshaling
	data, err := json.Marshal(c.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal data: %w", err)
	}

	var arr []interface{}
	if err := json.Unmarshal(data, &arr); err != nil {
		return nil, fmt.Errorf("failed to unmarshal data as array: %w", err)
	}

	return arr, nil
}

// GetDataAsString returns the data as a string
func (c *CIR) GetDataAsString() string {
	if s, ok := c.Data.(string); ok {
		return s
	}

	data, err := json.Marshal(c.Data)
	if err != nil {
		return fmt.Sprintf("%v", c.Data)
	}

	return string(data)
}

// SetParameter sets a source parameter
func (c *CIR) SetParameter(key string, value interface{}) {
	if c.Source.Parameters == nil {
		c.Source.Parameters = make(map[string]interface{})
	}
	c.Source.Parameters[key] = value
}

// GetParameter gets a source parameter
func (c *CIR) GetParameter(key string) (interface{}, bool) {
	if c.Source.Parameters == nil {
		return nil, false
	}
	val, ok := c.Source.Parameters[key]
	return val, ok
}

// SetSchemaInference sets schema inference information
func (c *CIR) SetSchemaInference(schema map[string]interface{}) {
	c.Metadata.SchemaInference = schema
}

// SetQualityMetrics sets quality metrics
func (c *CIR) SetQualityMetrics(metrics map[string]interface{}) {
	c.Metadata.QualityMetrics = metrics
}

// UpdateSize recalculates and updates the size metadata
func (c *CIR) UpdateSize() {
	data, err := json.Marshal(c.Data)
	if err != nil {
		c.Metadata.Size = int64(len(fmt.Sprintf("%v", c.Data)))
	} else {
		c.Metadata.Size = int64(len(data))
	}
}
