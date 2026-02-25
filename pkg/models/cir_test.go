package models_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/mimir-aip/mimir-aip-go/pkg/models"
)

func TestNewCIR(t *testing.T) {
	data := map[string]interface{}{
		"id":   "1",
		"name": "Test",
	}

	cir := models.NewCIR(models.SourceTypeAPI, "https://example.com/test", models.DataFormatJSON, data)

	if cir.Version != models.CIRVersion {
		t.Errorf("Expected version %s, got %s", models.CIRVersion, cir.Version)
	}

	if cir.Source.Type != models.SourceTypeAPI {
		t.Errorf("Expected source type %s, got %s", models.SourceTypeAPI, cir.Source.Type)
	}

	if cir.Source.URI != "https://example.com/test" {
		t.Errorf("Expected URI https://example.com/test, got %s", cir.Source.URI)
	}

	if cir.Source.Format != models.DataFormatJSON {
		t.Errorf("Expected format %s, got %s", models.DataFormatJSON, cir.Source.Format)
	}

	if cir.Data == nil {
		t.Error("Expected data to be set")
	}
}

func TestCIRValidate(t *testing.T) {
	tests := []struct {
		name    string
		cir     *models.CIR
		wantErr bool
	}{
		{
			name: "valid CIR",
			cir: &models.CIR{
				Version: "1.0",
				Source: models.CIRSource{
					Type:      models.SourceTypeAPI,
					URI:       "https://example.com",
					Format:    models.DataFormatJSON,
					Timestamp: time.Now(),
				},
				Data: map[string]interface{}{"test": "data"},
			},
			wantErr: false,
		},
		{
			name: "missing version",
			cir: &models.CIR{
				Source: models.CIRSource{
					Type:   models.SourceTypeAPI,
					URI:    "https://example.com",
					Format: models.DataFormatJSON,
				},
				Data: map[string]interface{}{"test": "data"},
			},
			wantErr: true,
		},
		{
			name: "missing source type",
			cir: &models.CIR{
				Version: "1.0",
				Source: models.CIRSource{
					URI:    "https://example.com",
					Format: models.DataFormatJSON,
				},
				Data: map[string]interface{}{"test": "data"},
			},
			wantErr: true,
		},
		{
			name: "missing URI",
			cir: &models.CIR{
				Version: "1.0",
				Source: models.CIRSource{
					Type:   models.SourceTypeAPI,
					Format: models.DataFormatJSON,
				},
				Data: map[string]interface{}{"test": "data"},
			},
			wantErr: true,
		},
		{
			name: "missing format",
			cir: &models.CIR{
				Version: "1.0",
				Source: models.CIRSource{
					Type: models.SourceTypeAPI,
					URI:  "https://example.com",
				},
				Data: map[string]interface{}{"test": "data"},
			},
			wantErr: true,
		},
		{
			name: "nil data",
			cir: &models.CIR{
				Version: "1.0",
				Source: models.CIRSource{
					Type:   models.SourceTypeAPI,
					URI:    "https://example.com",
					Format: models.DataFormatJSON,
				},
				Data: nil,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cir.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCIRToJSON(t *testing.T) {
	data := map[string]interface{}{
		"id":   "1",
		"name": "Test",
	}

	cir := models.NewCIR(models.SourceTypeAPI, "https://example.com/test", models.DataFormatJSON, data)

	jsonData, err := cir.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON() error = %v", err)
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal(jsonData, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	if decoded["version"] != models.CIRVersion {
		t.Errorf("Expected version %s in JSON", models.CIRVersion)
	}
}

func TestFromJSON(t *testing.T) {
	jsonStr := `{
		"version": "1.0",
		"source": {
			"type": "api",
			"uri": "https://example.com",
			"timestamp": "2024-01-01T00:00:00Z",
			"format": "json",
			"parameters": {}
		},
		"data": {"id": "1", "name": "Test"},
		"metadata": {
			"size": 100,
			"encoding": "utf-8",
			"record_count": 1
		}
	}`

	cir, err := models.FromJSON([]byte(jsonStr))
	if err != nil {
		t.Fatalf("FromJSON() error = %v", err)
	}

	if cir.Version != "1.0" {
		t.Errorf("Expected version 1.0, got %s", cir.Version)
	}

	if cir.Source.Type != models.SourceTypeAPI {
		t.Errorf("Expected source type %s, got %s", models.SourceTypeAPI, cir.Source.Type)
	}
}

func TestCIRGetDataAsMap(t *testing.T) {
	data := map[string]interface{}{
		"id":   "1",
		"name": "Test",
	}

	cir := models.NewCIR(models.SourceTypeAPI, "https://example.com/test", models.DataFormatJSON, data)

	m, err := cir.GetDataAsMap()
	if err != nil {
		t.Fatalf("GetDataAsMap() error = %v", err)
	}

	if m["id"] != "1" {
		t.Errorf("Expected id=1, got %v", m["id"])
	}

	if m["name"] != "Test" {
		t.Errorf("Expected name=Test, got %v", m["name"])
	}
}

func TestCIRGetDataAsArray(t *testing.T) {
	data := []interface{}{
		map[string]interface{}{"id": "1"},
		map[string]interface{}{"id": "2"},
	}

	cir := models.NewCIR(models.SourceTypeAPI, "https://example.com/test", models.DataFormatJSON, data)

	arr, err := cir.GetDataAsArray()
	if err != nil {
		t.Fatalf("GetDataAsArray() error = %v", err)
	}

	if len(arr) != 2 {
		t.Errorf("Expected array length 2, got %d", len(arr))
	}
}

func TestCIRGetDataAsString(t *testing.T) {
	data := "test string data"

	cir := models.NewCIR(models.SourceTypeFile, "/path/to/file.txt", models.DataFormatText, data)

	str := cir.GetDataAsString()
	if str != "test string data" {
		t.Errorf("Expected 'test string data', got '%s'", str)
	}
}

func TestCIRSetGetParameter(t *testing.T) {
	cir := models.NewCIR(models.SourceTypeAPI, "https://example.com", models.DataFormatJSON, map[string]interface{}{})

	cir.SetParameter("entity_type", "Employee")

	value, ok := cir.GetParameter("entity_type")
	if !ok {
		t.Error("Expected parameter to be set")
	}

	if value != "Employee" {
		t.Errorf("Expected entity_type=Employee, got %v", value)
	}

	_, ok = cir.GetParameter("nonexistent")
	if ok {
		t.Error("Expected parameter to not exist")
	}
}

func TestCIRSetSchemaInference(t *testing.T) {
	cir := models.NewCIR(models.SourceTypeAPI, "https://example.com", models.DataFormatJSON, map[string]interface{}{})

	schema := map[string]interface{}{
		"columns": []string{"id", "name"},
		"types":   []string{"string", "string"},
	}

	cir.SetSchemaInference(schema)

	if cir.Metadata.SchemaInference == nil {
		t.Error("Expected schema inference to be set")
	}
}

func TestCIRSetQualityMetrics(t *testing.T) {
	cir := models.NewCIR(models.SourceTypeAPI, "https://example.com", models.DataFormatJSON, map[string]interface{}{})

	metrics := map[string]interface{}{
		"completeness": 0.95,
		"accuracy":     0.98,
	}

	cir.SetQualityMetrics(metrics)

	if cir.Metadata.QualityMetrics == nil {
		t.Error("Expected quality metrics to be set")
	}
}

func TestCIRUpdateSize(t *testing.T) {
	data := map[string]interface{}{
		"id":   "1",
		"name": "Test",
	}

	cir := models.NewCIR(models.SourceTypeAPI, "https://example.com", models.DataFormatJSON, data)

	originalSize := cir.Metadata.Size

	// Update data
	cir.Data = map[string]interface{}{
		"id":          "1",
		"name":        "Test",
		"description": "A much longer description to increase the size",
	}

	cir.UpdateSize()

	if cir.Metadata.Size <= originalSize {
		t.Error("Expected size to increase after adding more data")
	}
}
