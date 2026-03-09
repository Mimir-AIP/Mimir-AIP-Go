package mlmodel

import (
	"testing"

	"github.com/mimir-aip/mimir-aip-go/pkg/models"
)

func benchmarkOntologyContent() string {
	return `@prefix : <http://example.org/bench#> .
@prefix owl: <http://www.w3.org/2002/07/owl#> .

:Machine a owl:Class .
:Sensor a owl:Class .
:Facility a owl:Class .
:Line a owl:Class .

:temperature a owl:DatatypeProperty .
:humidity a owl:DatatypeProperty .
:pressure a owl:DatatypeProperty .
:lineCode a owl:DatatypeProperty .
:facilityCode a owl:DatatypeProperty .
:status a owl:DatatypeProperty .

:belongsToFacility a owl:ObjectProperty .
:locatedOnLine a owl:ObjectProperty .
:connectedTo a owl:ObjectProperty .
`
}

func BenchmarkRecommendationEngineRecommendModelType(b *testing.B) {
	engine := NewRecommendationEngine()
	ontology := &models.Ontology{Content: benchmarkOntologyContent()}
	dataSummary := &models.DataAnalysis{
		Size:            "large",
		RecordCount:     750000,
		HasUnstructured: true,
		FeatureCount:    36,
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		recommendation, err := engine.RecommendModelType(ontology, dataSummary)
		if err != nil {
			b.Fatalf("RecommendModelType failed: %v", err)
		}
		if recommendation.RecommendedType == "" {
			b.Fatal("expected a recommended model type")
		}
	}
}
