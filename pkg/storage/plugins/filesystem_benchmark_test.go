package plugins

import (
	"fmt"
	"testing"
	"time"

	"github.com/mimir-aip/mimir-aip-go/pkg/models"
)

func benchmarkCIR(i int) *models.CIR {
	cir := models.NewCIR(
		models.SourceTypeAPI,
		fmt.Sprintf("https://bench.local/sensors/%d", i),
		models.DataFormatJSON,
		map[string]interface{}{
			"sensor_id":     i,
			"temperature":   float64(i%40) + 10.0,
			"humidity":      float64(i%80) + 5.0,
			"reading_at":    time.Now().UTC().Format(time.RFC3339),
			"facility_code": fmt.Sprintf("F-%03d", i%100),
		},
	)
	cir.SetParameter("entity_type", "SensorReading")
	return cir
}

func BenchmarkFilesystemStore(b *testing.B) {
	plugin := NewFilesystemPlugin()
	if err := plugin.Initialize(&models.PluginConfig{Options: map[string]interface{}{"base_path": b.TempDir()}}); err != nil {
		b.Fatalf("Initialize failed: %v", err)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result, err := plugin.Store(benchmarkCIR(i))
		if err != nil {
			b.Fatalf("Store failed: %v", err)
		}
		if !result.Success {
			b.Fatal("expected successful store")
		}
	}
}

func BenchmarkFilesystemRetrieve(b *testing.B) {
	plugin := NewFilesystemPlugin()
	if err := plugin.Initialize(&models.PluginConfig{Options: map[string]interface{}{"base_path": b.TempDir()}}); err != nil {
		b.Fatalf("Initialize failed: %v", err)
	}

	for i := 0; i < 3000; i++ {
		if _, err := plugin.Store(benchmarkCIR(i)); err != nil {
			b.Fatalf("prefill store failed: %v", err)
		}
	}

	query := &models.CIRQuery{EntityType: "SensorReading", Limit: 1000}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		results, err := plugin.Retrieve(query)
		if err != nil {
			b.Fatalf("Retrieve failed: %v", err)
		}
		if len(results) == 0 {
			b.Fatal("expected retrieve results")
		}
	}
}
