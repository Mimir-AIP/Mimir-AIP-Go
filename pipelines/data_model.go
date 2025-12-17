package pipelines

import (
	"encoding/json"
	"fmt"
	"reflect"
	"time"
)

// DataValue represents a typed data value that can be stored in PluginContext
type DataValue interface {
	// Type returns the data type identifier
	Type() string

	// Validate checks if the data is valid
	Validate() error

	// Serialize converts the data to bytes
	Serialize() ([]byte, error)

	// Deserialize populates the data from bytes
	Deserialize([]byte) error

	// Size returns the approximate memory size in bytes
	Size() int

	// Clone creates a deep copy of the data
	Clone() DataValue
}

// JSONData represents structured JSON data
type JSONData struct {
	Content map[string]any `json:"content"`
}

// NewJSONData creates a new JSONData instance
func NewJSONData(content map[string]any) *JSONData {
	if content == nil {
		content = make(map[string]any)
	}
	return &JSONData{Content: content}
}

// Type returns "json"
func (j *JSONData) Type() string { return "json" }

// Validate checks if the JSON content is valid
func (j *JSONData) Validate() error {
	if j.Content == nil {
		return fmt.Errorf("content cannot be nil")
	}
	return nil
}

// Serialize converts to JSON bytes
func (j *JSONData) Serialize() ([]byte, error) {
	return json.Marshal(j)
}

// Deserialize populates from JSON bytes
func (j *JSONData) Deserialize(data []byte) error {
	return json.Unmarshal(data, j)
}

// Size returns approximate memory size
func (j *JSONData) Size() int {
	data, _ := json.Marshal(j.Content)
	return len(data)
}

// Clone creates a deep copy
func (j *JSONData) Clone() DataValue {
	newContent := make(map[string]any)
	for k, v := range j.Content {
		newContent[k] = deepCopy(v)
	}
	return NewJSONData(newContent)
}

// BinaryData represents binary data with metadata
type BinaryData struct {
	Content  []byte `json:"content"`
	MIMEType string `json:"mime_type"`
}

// NewBinaryData creates a new BinaryData instance
func NewBinaryData(content []byte, mimeType string) *BinaryData {
	return &BinaryData{
		Content:  content,
		MIMEType: mimeType,
	}
}

// Type returns "binary"
func (b *BinaryData) Type() string { return "binary" }

// Validate checks if the binary data is valid
func (b *BinaryData) Validate() error {
	if b.Content == nil {
		return fmt.Errorf("content cannot be nil")
	}
	return nil
}

// Serialize converts to bytes
func (b *BinaryData) Serialize() ([]byte, error) {
	return json.Marshal(b)
}

// Deserialize populates from bytes
func (b *BinaryData) Deserialize(data []byte) error {
	return json.Unmarshal(data, b)
}

// Size returns memory size
func (b *BinaryData) Size() int {
	return len(b.Content) + len(b.MIMEType)
}

// Clone creates a copy
func (b *BinaryData) Clone() DataValue {
	newContent := make([]byte, len(b.Content))
	copy(newContent, b.Content)
	return NewBinaryData(newContent, b.MIMEType)
}

// TimePoint represents a single point in time series data
type TimePoint struct {
	Timestamp time.Time         `json:"timestamp"`
	Value     any               `json:"value"`
	Tags      map[string]string `json:"tags,omitempty"`
}

// TimeSeriesData represents time series data
type TimeSeriesData struct {
	Points   []TimePoint    `json:"points"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

// NewTimeSeriesData creates a new TimeSeriesData instance
func NewTimeSeriesData() *TimeSeriesData {
	return &TimeSeriesData{
		Points:   make([]TimePoint, 0),
		Metadata: make(map[string]any),
	}
}

// Type returns "timeseries"
func (t *TimeSeriesData) Type() string { return "timeseries" }

// Validate checks if the time series data is valid
func (t *TimeSeriesData) Validate() error {
	if t.Points == nil {
		return fmt.Errorf("points cannot be nil")
	}
	for i, point := range t.Points {
		if point.Timestamp.IsZero() {
			return fmt.Errorf("point %d has invalid timestamp", i)
		}
	}
	return nil
}

// Serialize converts to JSON bytes
func (t *TimeSeriesData) Serialize() ([]byte, error) {
	return json.Marshal(t)
}

// Deserialize populates from JSON bytes
func (t *TimeSeriesData) Deserialize(data []byte) error {
	return json.Unmarshal(data, t)
}

// Size returns approximate memory size
func (t *TimeSeriesData) Size() int {
	data, _ := json.Marshal(t)
	return len(data)
}

// Clone creates a deep copy
func (t *TimeSeriesData) Clone() DataValue {
	newPoints := make([]TimePoint, len(t.Points))
	copy(newPoints, t.Points)

	newMetadata := make(map[string]any)
	for k, v := range t.Metadata {
		newMetadata[k] = deepCopy(v)
	}

	return &TimeSeriesData{
		Points:   newPoints,
		Metadata: newMetadata,
	}
}

// AddPoint adds a new point to the time series
func (t *TimeSeriesData) AddPoint(timestamp time.Time, value any, tags map[string]string) {
	point := TimePoint{
		Timestamp: timestamp,
		Value:     value,
		Tags:      tags,
	}
	if point.Tags == nil {
		point.Tags = make(map[string]string)
	}
	t.Points = append(t.Points, point)
}

// ImageData represents image data with dimensions
type ImageData struct {
	*BinaryData
	Width  int    `json:"width"`
	Height int    `json:"height"`
	Format string `json:"format"`
}

// NewImageData creates a new ImageData instance
func NewImageData(content []byte, mimeType, format string, width, height int) *ImageData {
	return &ImageData{
		BinaryData: NewBinaryData(content, mimeType),
		Width:      width,
		Height:     height,
		Format:     format,
	}
}

// Type returns "image"
func (i *ImageData) Type() string { return "image" }

// Validate checks if the image data is valid
func (i *ImageData) Validate() error {
	if err := i.BinaryData.Validate(); err != nil {
		return err
	}
	if i.Width <= 0 || i.Height <= 0 {
		return fmt.Errorf("invalid dimensions: %dx%d", i.Width, i.Height)
	}
	if i.Format == "" {
		return fmt.Errorf("format cannot be empty")
	}
	return nil
}

// Serialize converts to JSON bytes
func (i *ImageData) Serialize() ([]byte, error) {
	return json.Marshal(i)
}

// Deserialize populates from JSON bytes
func (i *ImageData) Deserialize(data []byte) error {
	return json.Unmarshal(data, i)
}

// Size returns memory size
func (i *ImageData) Size() int {
	return i.BinaryData.Size() + 12 // width, height, format overhead
}

// Clone creates a copy
func (i *ImageData) Clone() DataValue {
	newContent := make([]byte, len(i.Content))
	copy(newContent, i.Content)
	return NewImageData(newContent, i.MIMEType, i.Format, i.Width, i.Height)
}

// deepCopy performs a deep copy of any values
func deepCopy(value any) any {
	if value == nil {
		return nil
	}

	switch v := value.(type) {
	case map[string]any:
		newMap := make(map[string]any)
		for k, val := range v {
			newMap[k] = deepCopy(val)
		}
		return newMap
	case []any:
		newSlice := make([]any, len(v))
		for i, val := range v {
			newSlice[i] = deepCopy(val)
		}
		return newSlice
	default:
		// For primitive types, reflection copy is sufficient
		return reflect.ValueOf(v).Interface()
	}
}
