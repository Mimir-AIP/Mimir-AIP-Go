package mlmodel

import (
	"testing"

	"github.com/mimir-aip/mimir-aip-go/pkg/models"
)

func TestRecommendModelType(t *testing.T) {
	engine := NewRecommendationEngine()

	tests := []struct {
		name                string
		ontologyContent     string
		dataSummary         *models.DataAnalysis
		expectedRecommended models.ModelType
		checkReasoning      bool
	}{
		{
			name: "Small dataset with simple ontology - should recommend Decision Tree",
			ontologyContent: `
@prefix owl: <http://www.w3.org/2002/07/owl#> .
@prefix rdf: <http://www.w3.org/1999/02/22-rdf-syntax-ns#> .
@prefix rdfs: <http://www.w3.org/2000/01/rdf-schema#> .
@prefix xsd: <http://www.w3.org/2001/XMLSchema#> .

:Entity1 a owl:Class .
:Entity2 a owl:Class .

:prop1 a owl:DatatypeProperty ;
    rdfs:range xsd:string .
:prop2 a owl:DatatypeProperty ;
    rdfs:range xsd:int .
			`,
			dataSummary: &models.DataAnalysis{
				Size:            "small",
				RecordCount:     100,
				HasUnstructured: false,
				FeatureCount:    5,
			},
			expectedRecommended: models.ModelTypeDecisionTree,
			checkReasoning:      true,
		},
		{
			name: "Large dataset with high numerical ratio - should recommend Regression or Neural Network",
			ontologyContent: `
@prefix owl: <http://www.w3.org/2002/07/owl#> .
@prefix rdf: <http://www.w3.org/1999/02/22-rdf-syntax-ns#> .
@prefix rdfs: <http://www.w3.org/2000/01/rdf-schema#> .
@prefix xsd: <http://www.w3.org/2001/XMLSchema#> .

:Entity1 a owl:Class .

:prop1 a owl:DatatypeProperty ;
    rdfs:range xsd:float .
:prop2 a owl:DatatypeProperty ;
    rdfs:range xsd:double .
:prop3 a owl:DatatypeProperty ;
    rdfs:range xsd:int .
:prop4 a owl:DatatypeProperty ;
    rdfs:range xsd:float .
			`,
			dataSummary: &models.DataAnalysis{
				Size:            "large",
				RecordCount:     100000,
				HasUnstructured: false,
				FeatureCount:    10,
			},
			expectedRecommended: models.ModelTypeNeuralNetwork, // Large dataset favors NN
			checkReasoning:      true,
		},
		{
			name: "Complex ontology with many relationships - should recommend Random Forest or Neural Network",
			ontologyContent: `
@prefix owl: <http://www.w3.org/2002/07/owl#> .
@prefix rdf: <http://www.w3.org/1999/02/22-rdf-syntax-ns#> .
@prefix rdfs: <http://www.w3.org/2000/01/rdf-schema#> .
@prefix xsd: <http://www.w3.org/2001/XMLSchema#> .

:Entity1 a owl:Class .
:Entity2 a owl:Class .
:Entity3 a owl:Class .
:Entity4 a owl:Class .
:Entity5 a owl:Class .
:Entity6 a owl:Class .
:Entity7 a owl:Class .
:Entity8 a owl:Class .
:Entity9 a owl:Class .
:Entity10 a owl:Class .
:Entity11 a owl:Class .

:rel1 a owl:ObjectProperty .
:rel2 a owl:ObjectProperty .
:rel3 a owl:ObjectProperty .
:rel4 a owl:ObjectProperty .
:rel5 a owl:ObjectProperty .
:rel6 a owl:ObjectProperty .
:rel7 a owl:ObjectProperty .
:rel8 a owl:ObjectProperty .
:rel9 a owl:ObjectProperty .
:rel10 a owl:ObjectProperty .
:rel11 a owl:ObjectProperty .
:rel12 a owl:ObjectProperty .
:rel13 a owl:ObjectProperty .
:rel14 a owl:ObjectProperty .
:rel15 a owl:ObjectProperty .
:rel16 a owl:ObjectProperty .
:rel17 a owl:ObjectProperty .
:rel18 a owl:ObjectProperty .
:rel19 a owl:ObjectProperty .
:rel20 a owl:ObjectProperty .
:rel21 a owl:ObjectProperty .

:prop1 a owl:DatatypeProperty ;
    rdfs:range xsd:string .
:prop2 a owl:DatatypeProperty ;
    rdfs:range xsd:string .
			`,
			dataSummary: &models.DataAnalysis{
				Size:            "medium",
				RecordCount:     5000,
				HasUnstructured: false,
				FeatureCount:    15,
			},
			expectedRecommended: models.ModelTypeRandomForest,
			checkReasoning:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ontology := &models.Ontology{
				Content: tt.ontologyContent,
			}

			recommendation, err := engine.RecommendModelType(ontology, tt.dataSummary)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if recommendation.RecommendedType != tt.expectedRecommended {
				t.Errorf("Expected recommendation %s, got %s", tt.expectedRecommended, recommendation.RecommendedType)
				t.Logf("All scores: %+v", recommendation.AllScores)
				t.Logf("Reasoning: %s", recommendation.Reasoning)
			}

			if tt.checkReasoning && recommendation.Reasoning == "" {
				t.Error("Expected reasoning to be provided")
			}

			if recommendation.Score <= 0 {
				t.Error("Expected score to be greater than 0")
			}

			// Verify ontology analysis
			if recommendation.OntologyAnalysis == nil {
				t.Error("Expected ontology analysis to be provided")
			}

			// Verify all model types have scores
			if len(recommendation.AllScores) != 4 {
				t.Errorf("Expected 4 model type scores, got %d", len(recommendation.AllScores))
			}
		})
	}
}

func TestAnalyzeOntology(t *testing.T) {
	engine := NewRecommendationEngine()

	ontologyContent := `
@prefix owl: <http://www.w3.org/2002/07/owl#> .
@prefix rdf: <http://www.w3.org/1999/02/22-rdf-syntax-ns#> .
@prefix rdfs: <http://www.w3.org/2000/01/rdf-schema#> .
@prefix xsd: <http://www.w3.org/2001/XMLSchema#> .

:Sensor a owl:Class .
:Reading a owl:Class .

:hasReading a owl:ObjectProperty .
:isPartOf a owl:ObjectProperty .

:temperature a owl:DatatypeProperty ;
    rdfs:range xsd:float .
:humidity a owl:DatatypeProperty ;
    rdfs:range xsd:float .
:status a owl:DatatypeProperty ;
    rdfs:range xsd:string .
	`

	ontology := &models.Ontology{
		Content: ontologyContent,
	}

	analysis, err := engine.analyzeOntology(ontology)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if analysis.NumEntities != 2 {
		t.Errorf("Expected 2 entities, got %d", analysis.NumEntities)
	}

	if analysis.NumAttributes != 3 {
		t.Errorf("Expected 3 datatype properties, got %d", analysis.NumAttributes)
	}

	if analysis.NumRelationships != 2 {
		t.Errorf("Expected 2 object properties, got %d", analysis.NumRelationships)
	}

	// Should have 2 numerical (temperature, humidity) and 1 categorical (status)
	if analysis.NumericalRatio < 0.6 || analysis.NumericalRatio > 0.7 {
		t.Errorf("Expected numerical ratio around 0.66, got %.2f", analysis.NumericalRatio)
	}

	if analysis.Complexity != "low" {
		t.Errorf("Expected low complexity, got %s", analysis.Complexity)
	}
}
