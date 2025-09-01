package pipelines

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"sync"
)

// Serializer handles serialization and deserialization of data
type Serializer interface {
	Serialize(data DataValue) ([]byte, error)
	Deserialize(data []byte, dataType string) (DataValue, error)
}

// JSONSerializer handles JSON serialization with optional compression
type JSONSerializer struct {
	compress bool
}

// NewJSONSerializer creates a new JSON serializer
func NewJSONSerializer(compress bool) *JSONSerializer {
	return &JSONSerializer{compress: compress}
}

// Serialize converts DataValue to bytes
func (js *JSONSerializer) Serialize(data DataValue) ([]byte, error) {
	jsonBytes, err := data.Serialize()
	if err != nil {
		return nil, fmt.Errorf("failed to serialize data: %w", err)
	}

	if !js.compress {
		return jsonBytes, nil
	}

	var buf bytes.Buffer
	gzipWriter := gzip.NewWriter(&buf)
	if _, err := gzipWriter.Write(jsonBytes); err != nil {
		return nil, fmt.Errorf("failed to compress data: %w", err)
	}
	if err := gzipWriter.Close(); err != nil {
		return nil, fmt.Errorf("failed to close gzip writer: %w", err)
	}

	return buf.Bytes(), nil
}

// Deserialize converts bytes to DataValue
func (js *JSONSerializer) Deserialize(data []byte, dataType string) (DataValue, error) {
	var jsonBytes []byte

	if js.compress {
		reader, err := gzip.NewReader(bytes.NewReader(data))
		if err != nil {
			return nil, fmt.Errorf("failed to create gzip reader: %w", err)
		}
		defer reader.Close()

		jsonBytes, err = io.ReadAll(reader)
		if err != nil {
			return nil, fmt.Errorf("failed to decompress data: %w", err)
		}
	} else {
		jsonBytes = data
	}

	var result DataValue
	switch dataType {
	case "json":
		result = NewJSONData(nil)
	case "binary":
		result = NewBinaryData(nil, "")
	case "timeseries":
		result = NewTimeSeriesData()
	case "image":
		result = NewImageData(nil, "", "", 0, 0)
	default:
		return nil, fmt.Errorf("unknown data type: %s", dataType)
	}

	if err := result.Deserialize(jsonBytes); err != nil {
		return nil, fmt.Errorf("failed to deserialize data: %w", err)
	}

	return result, nil
}

// ContextSerializer handles serialization of entire PluginContext
type ContextSerializer struct {
	serializer Serializer
}

// NewContextSerializer creates a new context serializer
func NewContextSerializer(serializer Serializer) *ContextSerializer {
	return &ContextSerializer{serializer: serializer}
}

// SerializeContext serializes an entire PluginContext
func (cs *ContextSerializer) SerializeContext(ctx *PluginContext) ([]byte, error) {
	ctx.mutex.RLock()
	defer ctx.mutex.RUnlock()

	// Create a serializable representation
	serializedData := make(map[string]string)
	metadata := make(map[string]interface{})

	// Serialize each data value
	for key, data := range ctx.data {
		dataBytes, err := cs.serializer.Serialize(data)
		if err != nil {
			return nil, fmt.Errorf("failed to serialize key %s: %w", key, err)
		}
		// Encode binary data as base64 to preserve it through JSON marshaling
		serializedData[key] = base64.StdEncoding.EncodeToString(dataBytes)
	}

	// Copy metadata
	for k, v := range ctx.metadata {
		metadata[k] = v
	}

	// Create final structure
	contextData := map[string]interface{}{
		"data":     serializedData,
		"metadata": metadata,
	}

	return json.Marshal(contextData)
}

// DeserializeContext deserializes into a PluginContext
func (cs *ContextSerializer) DeserializeContext(data []byte) (*PluginContext, error) {
	var contextData map[string]interface{}
	if err := json.Unmarshal(data, &contextData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal context data: %w", err)
	}

	ctx := NewPluginContext()

	// Deserialize data
	if dataMap, ok := contextData["data"].(map[string]interface{}); ok {
		for key, value := range dataMap {
			// Handle base64 encoded strings from JSON unmarshaling
			var dataBytes []byte
			switch v := value.(type) {
			case string:
				// Decode base64 encoded data
				var err error
				dataBytes, err = base64.StdEncoding.DecodeString(v)
				if err != nil {
					return nil, fmt.Errorf("failed to decode base64 data for key %s: %w", key, err)
				}
			default:
				// Skip invalid data
				continue
			}

			if len(dataBytes) > 0 {
				// Determine data type from content
				dataType := "json" // Default fallback
				var temp map[string]interface{}
				if json.Unmarshal(dataBytes, &temp) == nil {
					if _, hasContent := temp["content"]; hasContent {
						if _, hasPoints := temp["points"]; hasPoints {
							dataType = "timeseries"
						} else if _, hasMIMEType := temp["mime_type"]; hasMIMEType {
							if _, hasWidth := temp["width"]; hasWidth {
								dataType = "image"
							} else {
								dataType = "binary"
							}
						}
					}
				}

				dataValue, err := cs.serializer.Deserialize(dataBytes, dataType)
				if err != nil {
					return nil, fmt.Errorf("failed to deserialize key %s: %w", key, err)
				}
				ctx.SetTyped(key, dataValue)
			}
		}
	}

	// Deserialize metadata
	if metadata, ok := contextData["metadata"].(map[string]interface{}); ok {
		for k, v := range metadata {
			ctx.SetMetadata(k, v)
		}
	}

	return ctx, nil
}

// SerializationPool provides object pooling for serializers
type SerializationPool struct {
	pool sync.Pool
}

// NewSerializationPool creates a new serialization pool
func NewSerializationPool() *SerializationPool {
	return &SerializationPool{
		pool: sync.Pool{
			New: func() interface{} {
				return NewJSONSerializer(false)
			},
		},
	}
}

// Get retrieves a serializer from the pool
func (sp *SerializationPool) Get() *JSONSerializer {
	return sp.pool.Get().(*JSONSerializer)
}

// Put returns a serializer to the pool
func (sp *SerializationPool) Put(serializer *JSONSerializer) {
	sp.pool.Put(serializer)
}

// Global serialization instances
var (
	DefaultSerializer         = NewJSONSerializer(false)
	CompressedSerializer      = NewJSONSerializer(true)
	ContextSerializerInstance = NewContextSerializer(DefaultSerializer)
	SerializerPool            = NewSerializationPool()
)
