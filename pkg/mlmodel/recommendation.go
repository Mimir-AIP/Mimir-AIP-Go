package mlmodel

import (
	"fmt"
	"strings"

	"github.com/mimir-aip/mimir-aip-go/pkg/models"
)

// RecommendationEngine analyzes ontologies and data to recommend ML model types
type RecommendationEngine struct{}

// NewRecommendationEngine creates a new recommendation engine
func NewRecommendationEngine() *RecommendationEngine {
	return &RecommendationEngine{}
}

// RecommendModelType analyzes the ontology and data to recommend the best model type
// Implementation follows the pseudocode from Plan/MLModels/MLModelPlan.md
func (re *RecommendationEngine) RecommendModelType(
	ontology *models.Ontology,
	dataSummary *models.DataAnalysis,
) (*models.ModelRecommendation, error) {
	// Parse ontology to extract features
	ontologyAnalysis, err := re.analyzeOntology(ontology)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze ontology: %w", err)
	}

	// Initialize scores for each model type
	scores := map[models.ModelType]int{
		models.ModelTypeDecisionTree:  0,
		models.ModelTypeRandomForest:  0,
		models.ModelTypeRegression:    0,
		models.ModelTypeNeuralNetwork: 0,
	}

	// Scoring rules based on ontology complexity
	if ontologyAnalysis.NumEntities < 10 && ontologyAnalysis.NumRelationships < 20 {
		scores[models.ModelTypeDecisionTree] += 2
	} else if ontologyAnalysis.NumEntities >= 10 && ontologyAnalysis.NumRelationships >= 20 {
		scores[models.ModelTypeRandomForest] += 2
		scores[models.ModelTypeNeuralNetwork] += 1
	}

	// Scoring based on data types
	numericalRatio := ontologyAnalysis.NumericalRatio
	if numericalRatio > 0.7 {
		scores[models.ModelTypeRegression] += 3
		scores[models.ModelTypeNeuralNetwork] += 1
	} else if numericalRatio < 0.3 {
		scores[models.ModelTypeDecisionTree] += 2
		scores[models.ModelTypeRandomForest] += 2
	}

	// Scoring based on data size
	switch dataSummary.Size {
	case "small":
		scores[models.ModelTypeDecisionTree] += 2
	case "medium":
		scores[models.ModelTypeRandomForest] += 2
		scores[models.ModelTypeRegression] += 1
	case "large":
		scores[models.ModelTypeNeuralNetwork] += 3
		scores[models.ModelTypeRandomForest] += 1
	}

	// Scoring for unstructured data
	if dataSummary.HasUnstructured {
		scores[models.ModelTypeNeuralNetwork] += 2
	}

	// Additional scoring for complex relationships
	if ontologyAnalysis.NumRelationships > ontologyAnalysis.NumEntities {
		scores[models.ModelTypeRandomForest] += 1
		scores[models.ModelTypeNeuralNetwork] += 1
	}

	// Find the model with highest score
	recommendedModel := models.ModelTypeDecisionTree
	highestScore := scores[models.ModelTypeDecisionTree]
	for modelType, score := range scores {
		if score > highestScore {
			highestScore = score
			recommendedModel = modelType
		}
	}

	// Handle ties by preferring simpler models
	if scores[models.ModelTypeDecisionTree] == highestScore && recommendedModel != models.ModelTypeDecisionTree {
		recommendedModel = models.ModelTypeDecisionTree
	} else if scores[models.ModelTypeRandomForest] == highestScore &&
		recommendedModel != models.ModelTypeDecisionTree &&
		recommendedModel != models.ModelTypeRandomForest {
		recommendedModel = models.ModelTypeRandomForest
	}

	// Generate reasoning
	reasoning := re.generateReasoning(recommendedModel, ontologyAnalysis, dataSummary, scores)

	return &models.ModelRecommendation{
		RecommendedType:  recommendedModel,
		Score:            highestScore,
		Reasoning:        reasoning,
		AllScores:        scores,
		OntologyAnalysis: ontologyAnalysis,
		DataAnalysis:     dataSummary,
	}, nil
}

// analyzeOntology extracts features from the ontology for recommendation
func (re *RecommendationEngine) analyzeOntology(ontology *models.Ontology) (*models.OntologyAnalysis, error) {
	// Parse the Turtle content to extract classes and properties
	// For now, we'll do a simple text-based parsing
	// In production, this should use a proper RDF/OWL parser

	content := ontology.Content
	lines := strings.Split(content, "\n")

	numEntities := 0
	numDatatypeProps := 0
	numObjectProps := 0
	numericalCount := 0
	categoricalCount := 0
	dataTypes := make(map[string]int)

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Count classes (owl:Class)
		if strings.Contains(line, "a owl:Class") || strings.Contains(line, "rdf:type owl:Class") {
			numEntities++
		}

		// Count datatype properties
		if strings.Contains(line, "a owl:DatatypeProperty") || strings.Contains(line, "rdf:type owl:DatatypeProperty") {
			numDatatypeProps++
		}

		// Count object properties
		if strings.Contains(line, "a owl:ObjectProperty") || strings.Contains(line, "rdf:type owl:ObjectProperty") {
			numObjectProps++
		}

		// Analyze data types for datatype properties
		if strings.Contains(line, "rdfs:range") {
			if strings.Contains(line, "xsd:int") || strings.Contains(line, "xsd:integer") {
				numericalCount++
				dataTypes["integer"]++
			} else if strings.Contains(line, "xsd:float") || strings.Contains(line, "xsd:double") {
				numericalCount++
				dataTypes["float"]++
			} else if strings.Contains(line, "xsd:string") {
				categoricalCount++
				dataTypes["string"]++
			} else if strings.Contains(line, "xsd:boolean") {
				categoricalCount++
				dataTypes["boolean"]++
			} else if strings.Contains(line, "xsd:dateTime") {
				categoricalCount++
				dataTypes["datetime"]++
			}
		}
	}

	// Calculate ratios
	totalDataTypes := numericalCount + categoricalCount
	numericalRatio := 0.0
	categoricalRatio := 0.0
	if totalDataTypes > 0 {
		numericalRatio = float64(numericalCount) / float64(totalDataTypes)
		categoricalRatio = float64(categoricalCount) / float64(totalDataTypes)
	}

	// Determine complexity
	complexity := "low"
	if numEntities >= 10 || numObjectProps >= 20 {
		complexity = "high"
	} else if numEntities >= 5 || numObjectProps >= 10 {
		complexity = "medium"
	}

	return &models.OntologyAnalysis{
		NumEntities:      numEntities,
		NumAttributes:    numDatatypeProps,
		NumRelationships: numObjectProps,
		NumericalRatio:   numericalRatio,
		CategoricalRatio: categoricalRatio,
		DataTypes:        dataTypes,
		Complexity:       complexity,
	}, nil
}

// generateReasoning creates a human-readable explanation for the recommendation
func (re *RecommendationEngine) generateReasoning(
	recommendedModel models.ModelType,
	ontologyAnalysis *models.OntologyAnalysis,
	dataSummary *models.DataAnalysis,
	scores map[models.ModelType]int,
) string {
	var reasons []string

	switch recommendedModel {
	case models.ModelTypeDecisionTree:
		reasons = append(reasons, "Decision Tree recommended based on:")
		if ontologyAnalysis.NumEntities < 10 {
			reasons = append(reasons, fmt.Sprintf("- Simple ontology structure (%d entities)", ontologyAnalysis.NumEntities))
		}
		if dataSummary.Size == "small" {
			reasons = append(reasons, fmt.Sprintf("- Small dataset size (%s)", dataSummary.Size))
		}
		if ontologyAnalysis.NumericalRatio < 0.7 && ontologyAnalysis.NumericalRatio > 0.3 {
			reasons = append(reasons, "- Mixed numerical and categorical features")
		}
		reasons = append(reasons, "- Good interpretability for understanding feature importance")

	case models.ModelTypeRandomForest:
		reasons = append(reasons, "Random Forest recommended based on:")
		if ontologyAnalysis.NumEntities >= 10 {
			reasons = append(reasons, fmt.Sprintf("- Complex ontology structure (%d entities)", ontologyAnalysis.NumEntities))
		}
		if ontologyAnalysis.NumRelationships > ontologyAnalysis.NumEntities {
			reasons = append(reasons, "- High number of relationships between entities")
		}
		if dataSummary.Size == "medium" || dataSummary.Size == "large" {
			reasons = append(reasons, fmt.Sprintf("- Suitable dataset size (%s)", dataSummary.Size))
		}
		if ontologyAnalysis.CategoricalRatio > 0.3 {
			reasons = append(reasons, "- Significant categorical features present")
		}
		reasons = append(reasons, "- Ensemble approach improves accuracy")

	case models.ModelTypeRegression:
		reasons = append(reasons, "Regression Model recommended based on:")
		if ontologyAnalysis.NumericalRatio > 0.7 {
			reasons = append(reasons, fmt.Sprintf("- High proportion of numerical features (%.1f%%)", ontologyAnalysis.NumericalRatio*100))
		}
		reasons = append(reasons, "- Suitable for continuous output prediction")
		if dataSummary.Size == "medium" {
			reasons = append(reasons, "- Medium dataset provides sufficient training data")
		}

	case models.ModelTypeNeuralNetwork:
		reasons = append(reasons, "Neural Network recommended based on:")
		if dataSummary.Size == "large" {
			reasons = append(reasons, fmt.Sprintf("- Large dataset size (%s)", dataSummary.Size))
		}
		if ontologyAnalysis.Complexity == "high" {
			reasons = append(reasons, fmt.Sprintf("- High ontology complexity (%d entities, %d relationships)",
				ontologyAnalysis.NumEntities, ontologyAnalysis.NumRelationships))
		}
		if dataSummary.HasUnstructured {
			reasons = append(reasons, "- Presence of unstructured data components")
		}
		if ontologyAnalysis.NumRelationships > ontologyAnalysis.NumEntities {
			reasons = append(reasons, "- Complex non-linear relationships detected")
		}
		reasons = append(reasons, "- Deep learning can capture complex patterns")
	}

	reasons = append(reasons, fmt.Sprintf("\nRecommendation score: %d", scores[recommendedModel]))

	return strings.Join(reasons, "\n")
}
