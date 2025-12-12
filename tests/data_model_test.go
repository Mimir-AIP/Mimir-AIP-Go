package tests

import (
	"bytes"
	"fmt"
	"testing"
	"time"

	"github.com/Mimir-AIP/Mimir-AIP-Go/pipelines"
)

func TestJSONData(t *testing.T) {
	// Test creation
	data := pipelines.NewJSONData(map[string]any{
		"name":  "test",
		"value": 42,
	})

	if data.Type() != "json" {
		t.Errorf("Expected type 'json', got '%s'", data.Type())
	}

	// Test validation
	if err := data.Validate(); err != nil {
		t.Errorf("Validation failed: %v", err)
	}

	// Test serialization
	serialized, err := data.Serialize()
	if err != nil {
		t.Errorf("Serialization failed: %v", err)
	}

	// Test deserialization
	newData := pipelines.NewJSONData(nil)
	if err := newData.Deserialize(serialized); err != nil {
		t.Errorf("Deserialization failed: %v", err)
	}

	// Verify data integrity
	if newData.Content["name"] != "test" {
		t.Errorf("Expected name 'test', got '%v'", newData.Content["name"])
	}
	// JSON unmarshaling converts numbers to float64
	if value, ok := newData.Content["value"].(float64); !ok || value != 42.0 {
		t.Errorf("Expected value 42.0 (float64), got '%v'", newData.Content["value"])
	}
}

func TestBinaryData(t *testing.T) {
	originalData := []byte("Hello, World!")

	// Test creation
	data := pipelines.NewBinaryData(originalData, "text/plain")

	if data.Type() != "binary" {
		t.Errorf("Expected type 'binary', got '%s'", data.Type())
	}

	// Test validation
	if err := data.Validate(); err != nil {
		t.Errorf("Validation failed: %v", err)
	}

	// Test serialization
	serialized, err := data.Serialize()
	if err != nil {
		t.Errorf("Serialization failed: %v", err)
	}

	// Test deserialization
	newData := pipelines.NewBinaryData(nil, "")
	if err := newData.Deserialize(serialized); err != nil {
		t.Errorf("Deserialization failed: %v", err)
	}

	// Verify data integrity
	if !bytes.Equal(newData.Content, originalData) {
		t.Errorf("Data mismatch after serialization/deserialization")
	}
	if newData.MIMEType != "text/plain" {
		t.Errorf("Expected MIME type 'text/plain', got '%s'", newData.MIMEType)
	}
}

func TestTimeSeriesData(t *testing.T) {
	// Test creation
	data := pipelines.NewTimeSeriesData()

	if data.Type() != "timeseries" {
		t.Errorf("Expected type 'timeseries', got '%s'", data.Type())
	}

	// Add test points
	baseTime := time.Now()
	data.AddPoint(baseTime, 10.5, map[string]string{"sensor": "temp1"})
	data.AddPoint(baseTime.Add(time.Minute), 11.2, map[string]string{"sensor": "temp1"})

	// Test validation
	if err := data.Validate(); err != nil {
		t.Errorf("Validation failed: %v", err)
	}

	// Test serialization
	serialized, err := data.Serialize()
	if err != nil {
		t.Errorf("Serialization failed: %v", err)
	}

	// Test deserialization
	newData := pipelines.NewTimeSeriesData()
	if err := newData.Deserialize(serialized); err != nil {
		t.Errorf("Deserialization failed: %v", err)
	}

	// Verify data integrity
	if len(newData.Points) != 2 {
		t.Errorf("Expected 2 points, got %d", len(newData.Points))
	}
	if newData.Points[0].Value != 10.5 {
		t.Errorf("Expected first point value 10.5, got %v", newData.Points[0].Value)
	}
}

func TestImageData(t *testing.T) {
	// Create simple test image data
	imageBytes := []byte("fake_image_data")

	// Test creation
	data := pipelines.NewImageData(imageBytes, "image/png", "png", 100, 100)

	if data.Type() != "image" {
		t.Errorf("Expected type 'image', got '%s'", data.Type())
	}

	// Test validation
	if err := data.Validate(); err != nil {
		t.Errorf("Validation failed: %v", err)
	}

	// Test serialization
	serialized, err := data.Serialize()
	if err != nil {
		t.Errorf("Serialization failed: %v", err)
	}

	// Test deserialization
	newData := pipelines.NewImageData(nil, "", "", 0, 0)
	if err := newData.Deserialize(serialized); err != nil {
		t.Errorf("Deserialization failed: %v", err)
	}

	// Verify data integrity
	if !bytes.Equal(newData.Content, imageBytes) {
		t.Errorf("Image data mismatch after serialization/deserialization")
	}
	if newData.Width != 100 || newData.Height != 100 {
		t.Errorf("Expected dimensions 100x100, got %dx%d", newData.Width, newData.Height)
	}
	if newData.Format != "png" {
		t.Errorf("Expected format 'png', got '%s'", newData.Format)
	}
}

func TestPluginContext(t *testing.T) {
	ctx := pipelines.NewPluginContext()

	// Test setting and getting data
	jsonData := pipelines.NewJSONData(map[string]any{"key": "value"})
	ctx.SetTyped("test_json", jsonData)

	retrieved, exists := ctx.GetTyped("test_json")
	if !exists {
		t.Error("Expected data to exist")
	}

	if retrievedData, ok := retrieved.(*pipelines.JSONData); ok {
		if retrievedData.Content["key"] != "value" {
			t.Errorf("Expected value 'value', got '%v'", retrievedData.Content["key"])
		}
	} else {
		t.Error("Expected JSONData type")
	}

	// Test generic get
	value, exists := ctx.Get("test_json")
	if !exists {
		t.Error("Expected data to exist")
	}
	if valueMap, ok := value.(map[string]any); ok {
		if valueMap["key"] != "value" {
			t.Errorf("Expected value 'value', got '%v'", valueMap["key"])
		}
	} else {
		t.Error("Expected map[string]any type")
	}

	// Test metadata
	ctx.SetMetadata("version", "1.0")
	metadata, exists := ctx.GetMetadata("version")
	if !exists {
		t.Error("Expected metadata to exist")
	}
	if metadata != "1.0" {
		t.Errorf("Expected version '1.0', got '%v'", metadata)
	}

	// Test size and keys
	if ctx.Size() != 1 {
		t.Errorf("Expected size 1, got %d", ctx.Size())
	}

	keys := ctx.Keys()
	if len(keys) != 1 || keys[0] != "test_json" {
		t.Errorf("Expected keys ['test_json'], got %v", keys)
	}
}

func TestSerialization(t *testing.T) {
	// Test JSON serializer
	serializer := pipelines.NewJSONSerializer(false)
	data := pipelines.NewJSONData(map[string]any{"test": "data"})

	serialized, err := serializer.Serialize(data)
	if err != nil {
		t.Errorf("Serialization failed: %v", err)
	}

	deserialized, err := serializer.Deserialize(serialized, "json")
	if err != nil {
		t.Errorf("Deserialization failed: %v", err)
	}

	if jsonData, ok := deserialized.(*pipelines.JSONData); ok {
		if jsonData.Content["test"] != "data" {
			t.Errorf("Expected 'data', got '%v'", jsonData.Content["test"])
		}
	} else {
		t.Error("Expected JSONData type")
	}
}

func TestCompressedSerialization(t *testing.T) {
	// Test compressed serializer
	serializer := pipelines.NewJSONSerializer(true)
	data := pipelines.NewJSONData(map[string]any{"test": "data", "number": 42})

	serialized, err := serializer.Serialize(data)
	if err != nil {
		t.Errorf("Serialization failed: %v", err)
	}

	deserialized, err := serializer.Deserialize(serialized, "json")
	if err != nil {
		t.Errorf("Deserialization failed: %v", err)
	}

	if jsonData, ok := deserialized.(*pipelines.JSONData); ok {
		if jsonData.Content["test"] != "data" {
			t.Errorf("Expected 'data', got '%v'", jsonData.Content["test"])
		}
		// JSON unmarshaling converts numbers to float64
		if value, ok := jsonData.Content["number"].(float64); !ok || value != 42.0 {
			t.Errorf("Expected 42.0 (float64), got '%v'", jsonData.Content["number"])
		}
	} else {
		t.Error("Expected JSONData type")
	}
}

func TestContextSerialization(t *testing.T) {
	// Create a context with multiple data types
	ctx := pipelines.NewPluginContext()

	// Add JSON data
	jsonData := pipelines.NewJSONData(map[string]any{"user": "test", "active": true})
	ctx.SetTyped("user_data", jsonData)

	// Add binary data
	binaryData := pipelines.NewBinaryData([]byte("binary content"), "application/octet-stream")
	ctx.SetTyped("binary_data", binaryData)

	// Add metadata
	ctx.SetMetadata("version", "2.0")
	ctx.SetMetadata("environment", "test")

	// Serialize context
	serialized, err := pipelines.ContextSerializerInstance.SerializeContext(ctx)
	if err != nil {
		t.Errorf("Context serialization failed: %v", err)
	}

	// Deserialize context
	deserialized, err := pipelines.ContextSerializerInstance.DeserializeContext(serialized)
	if err != nil {
		t.Errorf("Context deserialization failed: %v", err)
	}

	// Verify data integrity
	if deserialized.Size() != 2 {
		t.Errorf("Expected size 2, got %d", deserialized.Size())
	}

	// Check JSON data
	userData, exists := deserialized.GetTyped("user_data")
	if !exists {
		t.Error("Expected user_data to exist")
	}
	if jsonUserData, ok := userData.(*pipelines.JSONData); ok {
		if jsonUserData.Content["user"] != "test" {
			t.Errorf("Expected user 'test', got '%v'", jsonUserData.Content["user"])
		}
	}

	// Check binary data
	binData, exists := deserialized.GetTyped("binary_data")
	if !exists {
		t.Error("Expected binary_data to exist")
	}
	if binaryUserData, ok := binData.(*pipelines.BinaryData); ok {
		if !bytes.Equal(binaryUserData.Content, []byte("binary content")) {
			t.Error("Binary content mismatch")
		}
	}

	// Check metadata
	version, exists := deserialized.GetMetadata("version")
	if !exists || version != "2.0" {
		t.Errorf("Expected version '2.0', got '%v'", version)
	}
}

func TestDataCache(t *testing.T) {
	cache := pipelines.NewDataCache(10, time.Minute)

	// Test cache miss
	_, exists := cache.Get("nonexistent")
	if exists {
		t.Error("Expected cache miss")
	}

	// Test cache put and get
	data := pipelines.NewJSONData(map[string]any{"cached": true})
	cache.Put("test_key", data)

	retrieved, exists := cache.Get("test_key")
	if !exists {
		t.Error("Expected cache hit")
	}

	if jsonData, ok := retrieved.(*pipelines.JSONData); ok {
		if jsonData.Content["cached"] != true {
			t.Errorf("Expected cached=true, got '%v'", jsonData.Content["cached"])
		}
	}

	// Test cache size
	if cache.Size() != 1 {
		t.Errorf("Expected cache size 1, got %d", cache.Size())
	}

	// Test cache clear
	cache.Clear()
	if cache.Size() != 0 {
		t.Errorf("Expected cache size 0 after clear, got %d", cache.Size())
	}
}

func TestMemoryManager(t *testing.T) {
	mm := pipelines.NewMemoryManager(1024) // 1KB limit

	// Test allocation
	if !mm.Allocate("key1", 500) {
		t.Error("Expected allocation to succeed")
	}

	if mm.GetTotalMemory() != 500 {
		t.Errorf("Expected total memory 500, got %d", mm.GetTotalMemory())
	}

	// Test over-allocation
	if mm.Allocate("key2", 600) {
		t.Error("Expected allocation to fail (over limit)")
	}

	// Test deallocation
	mm.Deallocate("key1")
	if mm.GetTotalMemory() != 0 {
		t.Errorf("Expected total memory 0 after deallocation, got %d", mm.GetTotalMemory())
	}

	// Test memory usage percentage
	mm.Allocate("key3", 256)
	usage := mm.GetMemoryUsage()
	expectedUsage := float64(256) / 1024 * 100
	if usage != expectedUsage {
		t.Errorf("Expected usage %f%%, got %f%%", expectedUsage, usage)
	}
}

func BenchmarkJSONDataSerialization(b *testing.B) {
	data := pipelines.NewJSONData(map[string]any{
		"id":       12345,
		"name":     "Benchmark Test",
		"active":   true,
		"tags":     []string{"test", "benchmark", "performance"},
		"metadata": map[string]any{"version": "1.0", "env": "test"},
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := data.Serialize()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkPluginContextOperations(b *testing.B) {
	ctx := pipelines.NewPluginContext()

	b.Run("Set", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			data := pipelines.NewJSONData(map[string]any{"value": i})
			ctx.SetTyped(fmt.Sprintf("key_%d", i%100), data)
		}
	})

	b.Run("Get", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			ctx.GetTyped(fmt.Sprintf("key_%d", i%100))
		}
	})
}
