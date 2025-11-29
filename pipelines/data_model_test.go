package pipelines

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewJSONData(t *testing.T) {
	// Test with nil content
	jsonData := NewJSONData(nil)
	assert.NotNil(t, jsonData)
	assert.NotNil(t, jsonData.Content)
	assert.Equal(t, 0, len(jsonData.Content))

	// Test with content
	content := map[string]interface{}{"key": "value"}
	jsonData = NewJSONData(content)
	assert.Equal(t, content, jsonData.Content)
}

func TestJSONDataInterface(t *testing.T) {
	content := map[string]interface{}{
		"string": "test",
		"number": 42,
		"nested": map[string]interface{}{
			"inner": "value",
		},
	}
	jsonData := NewJSONData(content)

	// Test Type
	assert.Equal(t, "json", jsonData.Type())

	// Test Validate
	err := jsonData.Validate()
	assert.NoError(t, err)

	// Test Validate with nil content
	nilData := &JSONData{Content: nil}
	err = nilData.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "content cannot be nil")

	// Test Serialize
	serialized, err := jsonData.Serialize()
	assert.NoError(t, err)
	assert.NotEmpty(t, serialized)

	// Test Deserialize
	newJsonData := &JSONData{}
	err = newJsonData.Deserialize(serialized)
	assert.NoError(t, err)
	// JSON unmarshaling may convert int to float64, so we need to compare appropriately
	assert.Equal(t, content["string"], newJsonData.Content["string"])
	assert.Equal(t, content["nested"], newJsonData.Content["nested"])
	assert.InDelta(t, content["number"], newJsonData.Content["number"], 0.0001)

	// Test Size
	size := jsonData.Size()
	assert.Greater(t, size, 0)

	// Test Clone
	cloned := jsonData.Clone().(*JSONData)
	assert.Equal(t, jsonData.Content, cloned.Content)
	assert.NotSame(t, jsonData.Content, cloned.Content) // Should be different objects

	// Modify original and verify clone is unaffected
	jsonData.Content["new_key"] = "new_value"
	assert.NotContains(t, cloned.Content, "new_key")
}

func TestNewBinaryData(t *testing.T) {
	content := []byte("test binary data")
	mimeType := "application/octet-stream"

	binaryData := NewBinaryData(content, mimeType)
	assert.Equal(t, content, binaryData.Content)
	assert.Equal(t, mimeType, binaryData.MIMEType)
}

func TestBinaryDataInterface(t *testing.T) {
	content := []byte("test binary data")
	mimeType := "application/pdf"
	binaryData := NewBinaryData(content, mimeType)

	// Test Type
	assert.Equal(t, "binary", binaryData.Type())

	// Test Validate
	err := binaryData.Validate()
	assert.NoError(t, err)

	// Test Validate with nil content
	nilData := &BinaryData{Content: nil}
	err = nilData.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "content cannot be nil")

	// Test Serialize
	serialized, err := binaryData.Serialize()
	assert.NoError(t, err)
	assert.NotEmpty(t, serialized)

	// Test Deserialize
	newData := &BinaryData{}
	err = newData.Deserialize(serialized)
	assert.NoError(t, err)
	assert.Equal(t, binaryData.Content, newData.Content)
	assert.Equal(t, binaryData.MIMEType, newData.MIMEType)

	// Test Size
	size := binaryData.Size()
	assert.Equal(t, len(content)+len(mimeType), size)

	// Test Clone
	cloned := binaryData.Clone().(*BinaryData)
	assert.Equal(t, binaryData.Content, cloned.Content)
	assert.Equal(t, binaryData.MIMEType, cloned.MIMEType)
	assert.NotSame(t, binaryData.Content, cloned.Content) // Should be different slices

	// Modify original and verify clone is unaffected
	binaryData.Content[0] = 'X'
	assert.NotEqual(t, binaryData.Content[0], cloned.Content[0])
}

func TestNewTimeSeriesData(t *testing.T) {
	tsData := NewTimeSeriesData()
	assert.NotNil(t, tsData)
	assert.NotNil(t, tsData.Points)
	assert.NotNil(t, tsData.Metadata)
	assert.Equal(t, 0, len(tsData.Points))
	assert.Equal(t, 0, len(tsData.Metadata))
}

func TestTimeSeriesDataInterface(t *testing.T) {
	tsData := NewTimeSeriesData()

	// Add some test points
	now := time.Now()
	tags := map[string]string{"source": "test"}
	tsData.AddPoint(now, 100.5, tags)
	tsData.AddPoint(now.Add(time.Hour), 200.3, map[string]string{"source": "test2"})

	// Test Type
	assert.Equal(t, "timeseries", tsData.Type())

	// Test Validate
	err := tsData.Validate()
	assert.NoError(t, err)

	// Test Validate with nil points
	nilData := &TimeSeriesData{Points: nil}
	err = nilData.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "points cannot be nil")

	// Test Validate with zero timestamp
	invalidData := &TimeSeriesData{
		Points: []TimePoint{
			{Timestamp: time.Time{}, Value: 100},
		},
	}
	err = invalidData.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid timestamp")

	// Test Serialize
	serialized, err := tsData.Serialize()
	assert.NoError(t, err)
	assert.NotEmpty(t, serialized)

	// Test Deserialize
	newData := &TimeSeriesData{}
	err = newData.Deserialize(serialized)
	assert.NoError(t, err)
	assert.Equal(t, len(tsData.Points), len(newData.Points))
	// Metadata may be nil after deserialization if empty, so handle both cases
	if tsData.Metadata == nil || len(tsData.Metadata) == 0 {
		assert.True(t, newData.Metadata == nil || len(newData.Metadata) == 0)
	} else {
		assert.Equal(t, tsData.Metadata, newData.Metadata)
	}

	// Test Size
	size := tsData.Size()
	assert.Greater(t, size, 0)

	// Test Clone
	cloned := tsData.Clone().(*TimeSeriesData)
	assert.Equal(t, len(tsData.Points), len(cloned.Points))
	assert.Equal(t, tsData.Metadata, cloned.Metadata)
	assert.NotSame(t, tsData.Points, cloned.Points) // Should be different slices

	// Modify original and verify clone is unaffected
	tsData.AddPoint(now.Add(2*time.Hour), 300.0, map[string]string{"new": "tag"})
	assert.Equal(t, 2, len(cloned.Points))
	assert.Equal(t, 3, len(tsData.Points))
}

func TestTimeSeriesDataAddPoint(t *testing.T) {
	tsData := NewTimeSeriesData()
	now := time.Now()

	// Add point with tags
	tags := map[string]string{"source": "test"}
	tsData.AddPoint(now, 100.0, tags)

	assert.Equal(t, 1, len(tsData.Points))
	assert.Equal(t, now, tsData.Points[0].Timestamp)
	assert.Equal(t, 100.0, tsData.Points[0].Value)
	assert.Equal(t, tags, tsData.Points[0].Tags)

	// Add point with nil tags
	tsData.AddPoint(now.Add(time.Hour), 200.0, nil)

	assert.Equal(t, 2, len(tsData.Points))
	assert.NotNil(t, tsData.Points[1].Tags)
	assert.Equal(t, 0, len(tsData.Points[1].Tags))
}

func TestNewImageData(t *testing.T) {
	content := []byte("fake image data")
	mimeType := "image/png"
	format := "png"
	width := 800
	height := 600

	imageData := NewImageData(content, mimeType, format, width, height)

	assert.Equal(t, content, imageData.Content)
	assert.Equal(t, mimeType, imageData.MIMEType)
	assert.Equal(t, format, imageData.Format)
	assert.Equal(t, width, imageData.Width)
	assert.Equal(t, height, imageData.Height)
}

func TestImageDataInterface(t *testing.T) {
	content := []byte("fake image data")
	mimeType := "image/jpeg"
	format := "jpeg"
	width := 1024
	height := 768
	imageData := NewImageData(content, mimeType, format, width, height)

	// Test Type
	assert.Equal(t, "image", imageData.Type())

	// Test Validate
	err := imageData.Validate()
	assert.NoError(t, err)

	// Test Validate with invalid dimensions
	invalidData := NewImageData(content, mimeType, format, 0, height)
	err = invalidData.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid dimensions")

	invalidData = NewImageData(content, mimeType, format, width, -1)
	err = invalidData.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid dimensions")

	// Test Validate with empty format
	invalidData = NewImageData(content, mimeType, "", width, height)
	err = invalidData.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "format cannot be empty")

	// Test Serialize
	serialized, err := imageData.Serialize()
	assert.NoError(t, err)
	assert.NotEmpty(t, serialized)

	// Test Deserialize
	newData := &ImageData{}
	err = newData.Deserialize(serialized)
	assert.NoError(t, err)
	assert.Equal(t, imageData.Content, newData.Content)
	assert.Equal(t, imageData.MIMEType, newData.MIMEType)
	assert.Equal(t, imageData.Format, newData.Format)
	assert.Equal(t, imageData.Width, newData.Width)
	assert.Equal(t, imageData.Height, newData.Height)

	// Test Size
	size := imageData.Size()
	expectedSize := len(content) + len(mimeType) + 12 // width, height, format overhead
	assert.Equal(t, expectedSize, size)

	// Test Clone
	cloned := imageData.Clone().(*ImageData)
	assert.Equal(t, imageData.Content, cloned.Content)
	assert.Equal(t, imageData.MIMEType, cloned.MIMEType)
	assert.Equal(t, imageData.Format, cloned.Format)
	assert.Equal(t, imageData.Width, cloned.Width)
	assert.Equal(t, imageData.Height, cloned.Height)
	assert.NotSame(t, imageData.Content, cloned.Content) // Should be different slices

	// Modify original and verify clone is unaffected
	imageData.Content[0] = 'X'
	assert.NotEqual(t, imageData.Content[0], cloned.Content[0])
}

func TestDeepCopy(t *testing.T) {
	// Test map copy
	originalMap := map[string]interface{}{
		"string": "value",
		"number": 42,
		"nested": map[string]interface{}{
			"inner": "nested_value",
		},
		"array": []interface{}{1, 2, 3},
	}

	copiedMap := deepCopy(originalMap).(map[string]interface{})

	assert.Equal(t, originalMap, copiedMap)
	assert.NotSame(t, originalMap, copiedMap)

	// Modify original and verify copy is unaffected
	originalMap["new_key"] = "new_value"
	assert.NotContains(t, copiedMap, "new_key")

	// Modify nested map in original
	originalMap["nested"].(map[string]interface{})["new_inner"] = "new_nested_value"
	assert.NotContains(t, copiedMap["nested"].(map[string]interface{}), "new_inner")

	// Test array copy
	originalArray := []interface{}{
		"string",
		42,
		map[string]interface{}{"nested": "value"},
	}

	copiedArray := deepCopy(originalArray).([]interface{})

	assert.Equal(t, originalArray, copiedArray)
	assert.NotSame(t, originalArray, copiedArray)

	// Modify original and verify copy is unaffected
	originalArray[0] = "modified"
	assert.Equal(t, "string", copiedArray[0])

	// Test primitive types
	assert.Equal(t, "string", deepCopy("string"))
	assert.Equal(t, 42, deepCopy(42))
	assert.Equal(t, true, deepCopy(true))
	assert.Nil(t, deepCopy(nil))
}

func TestDataValueTypes(t *testing.T) {
	// Test that all data types implement DataValue interface
	var _ DataValue = &JSONData{}
	var _ DataValue = &BinaryData{}
	var _ DataValue = &TimeSeriesData{}
	var _ DataValue = &ImageData{}

	// Test type identification
	jsonData := NewJSONData(map[string]interface{}{})
	binaryData := NewBinaryData([]byte{}, "application/octet-stream")
	tsData := NewTimeSeriesData()
	imageData := NewImageData([]byte{}, "image/png", "png", 100, 100)

	assert.Equal(t, "json", jsonData.Type())
	assert.Equal(t, "binary", binaryData.Type())
	assert.Equal(t, "timeseries", tsData.Type())
	assert.Equal(t, "image", imageData.Type())
}

func TestTimePoint(t *testing.T) {
	now := time.Now()
	tags := map[string]string{"source": "test", "type": "metric"}

	point := TimePoint{
		Timestamp: now,
		Value:     100.5,
		Tags:      tags,
	}

	assert.Equal(t, now, point.Timestamp)
	assert.Equal(t, 100.5, point.Value)
	assert.Equal(t, tags, point.Tags)

	// Test with nil tags
	pointNilTags := TimePoint{
		Timestamp: now,
		Value:     200,
		Tags:      nil,
	}

	assert.Nil(t, pointNilTags.Tags)
}

func TestDataValueSerializationRoundTrip(t *testing.T) {
	// Test JSON data round trip
	jsonContent := map[string]interface{}{
		"string": "test",
		"number": 42,
		"nested": map[string]interface{}{"inner": "value"},
	}
	jsonData := NewJSONData(jsonContent)

	serialized, err := jsonData.Serialize()
	assert.NoError(t, err)

	newJsonData := &JSONData{}
	err = newJsonData.Deserialize(serialized)
	assert.NoError(t, err)
	// JSON unmarshaling may convert int to float64, so we need to compare appropriately
	assert.Equal(t, jsonContent["string"], newJsonData.Content["string"])
	assert.Equal(t, jsonContent["nested"], newJsonData.Content["nested"])
	assert.InDelta(t, jsonContent["number"], newJsonData.Content["number"], 0.0001)

	// Test Binary data round trip
	binaryContent := []byte("binary test data")
	binaryData := NewBinaryData(binaryContent, "application/test")

	serialized, err = binaryData.Serialize()
	assert.NoError(t, err)

	newBinaryData := &BinaryData{}
	err = newBinaryData.Deserialize(serialized)
	assert.NoError(t, err)
	assert.Equal(t, binaryContent, newBinaryData.Content)
	assert.Equal(t, "application/test", newBinaryData.MIMEType)

	// Test TimeSeries data round trip
	tsData := NewTimeSeriesData()
	now := time.Now()
	tsData.AddPoint(now, 100.0, map[string]string{"source": "test"})
	tsData.Metadata["test"] = "metadata"

	serialized, err = tsData.Serialize()
	assert.NoError(t, err)

	newTsData := &TimeSeriesData{}
	err = newTsData.Deserialize(serialized)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(newTsData.Points))
	// Time zone may change during serialization, compare Unix timestamps
	assert.Equal(t, now.Unix(), newTsData.Points[0].Timestamp.Unix())
	assert.Equal(t, 100.0, newTsData.Points[0].Value)
	assert.Equal(t, "metadata", newTsData.Metadata["test"])

	// Test Image data round trip
	imageContent := []byte("fake image")
	imageData := NewImageData(imageContent, "image/png", "png", 800, 600)

	serialized, err = imageData.Serialize()
	assert.NoError(t, err)

	newImageData := &ImageData{}
	err = newImageData.Deserialize(serialized)
	assert.NoError(t, err)
	assert.Equal(t, imageContent, newImageData.Content)
	assert.Equal(t, "image/png", newImageData.MIMEType)
	assert.Equal(t, "png", newImageData.Format)
	assert.Equal(t, 800, newImageData.Width)
	assert.Equal(t, 600, newImageData.Height)
}
