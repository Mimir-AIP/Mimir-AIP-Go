package ml

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/Mimir-AIP/Mimir-AIP-Go/pipelines/KnowledgeGraph"
	"github.com/Mimir-AIP/Mimir-AIP-Go/pipelines/Ontology"
	"github.com/Mimir-AIP/Mimir-AIP-Go/pipelines/Storage"
)

// KGDataExtractor extracts training data from the knowledge graph
type KGDataExtractor struct {
	Storage  *storage.PersistenceBackend
	KGClient *knowledgegraph.TDB2Backend
}

// NewKGDataExtractor creates a new knowledge graph data extractor
func NewKGDataExtractor(store *storage.PersistenceBackend, kgClient *knowledgegraph.TDB2Backend) *KGDataExtractor {
	return &KGDataExtractor{
		Storage:  store,
		KGClient: kgClient,
	}
}

// TrainingDataset represents ML-ready training data extracted from the knowledge graph
type TrainingDataset struct {
	X               [][]float64         // Feature matrix
	Y               interface{}         // Target values ([]float64 for regression, []string for classification)
	FeatureNames    []string            // Names of features
	TargetName      string              // Name of target variable
	EntityIDs       []string            // Entity URIs for traceability
	ModelType       string              // "regression" or "classification"
	FeatureEncoders map[string]*Encoder // Encoders for categorical features
	TargetEncoder   *Encoder            // Encoder for target (classification only)
	SampleCount     int                 // Number of samples
	FeatureCount    int                 // Number of features
}

// Encoder handles categorical to numeric encoding
type Encoder struct {
	Type           string             // "label" or "onehot"
	Mapping        map[string]float64 // String to numeric mapping
	ReverseMapping map[float64]string // Numeric to string mapping
	UniqueCe       []string           // Unique categorical values
}

// ExtractTrainingData extracts training data from the knowledge graph for a specific target
func (kde *KGDataExtractor) ExtractTrainingData(
	ctx context.Context,
	ontologyID string,
	targetProperty ontology.OntologyProperty,
	featureProperties []ontology.OntologyProperty,
) (*TrainingDataset, error) {
	// Get ontology metadata to find the graph URI
	ontologyMeta, err := kde.Storage.GetOntology(ctx, ontologyID)
	if err != nil {
		return nil, fmt.Errorf("failed to get ontology: %w", err)
	}

	// Build SPARQL query to extract data
	query := kde.buildSPARQLQuery(ontologyMeta.TDB2Graph, targetProperty, featureProperties)

	// Execute query
	result, err := kde.KGClient.QuerySPARQL(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to execute SPARQL query: %w", err)
	}

	if len(result.Bindings) == 0 {
		return nil, fmt.Errorf("no data found in knowledge graph for target property %s", targetProperty.Label)
	}

	// Determine model type based on target property
	modelType := "classification"
	if isNumericRange(targetProperty.Range) {
		modelType = "regression"
	}

	// Convert SPARQL results to ML-ready format
	dataset := &TrainingDataset{
		TargetName:      targetProperty.Label,
		ModelType:       modelType,
		FeatureNames:    make([]string, len(featureProperties)),
		FeatureEncoders: make(map[string]*Encoder),
		EntityIDs:       make([]string, 0, len(result.Bindings)),
	}

	// Extract feature names
	for i, fp := range featureProperties {
		dataset.FeatureNames[i] = fp.Label
	}

	// Process bindings
	rawFeatures := make([][]string, 0, len(result.Bindings))
	rawTargets := make([]string, 0, len(result.Bindings))

	for _, binding := range result.Bindings {
		// Extract entity ID
		if entityVal, ok := binding["entity"]; ok {
			dataset.EntityIDs = append(dataset.EntityIDs, entityVal.Value)
		}

		// Extract features
		featureRow := make([]string, len(featureProperties))
		hasAllFeatures := true

		for i, fp := range featureProperties {
			varName := kde.sanitizeVarName(fp.Label)
			if featureVal, ok := binding[varName]; ok {
				featureRow[i] = featureVal.Value
			} else {
				// Missing value - we'll handle this later
				hasAllFeatures = false
				break
			}
		}

		// Extract target
		targetVarName := kde.sanitizeVarName(targetProperty.Label)
		targetVal, hasTarget := binding[targetVarName]

		// Only include complete rows (for now)
		if hasAllFeatures && hasTarget {
			rawFeatures = append(rawFeatures, featureRow)
			rawTargets = append(rawTargets, targetVal.Value)
		}
	}

	if len(rawFeatures) == 0 {
		return nil, fmt.Errorf("no complete data rows found (all rows had missing values)")
	}

	// Encode features
	X := make([][]float64, len(rawFeatures))
	for i := range X {
		X[i] = make([]float64, len(featureProperties))
	}

	for featureIdx, fp := range featureProperties {
		// Extract column
		column := make([]string, len(rawFeatures))
		for rowIdx := range rawFeatures {
			column[rowIdx] = rawFeatures[rowIdx][featureIdx]
		}

		// Encode based on property type
		var encoded []float64
		var encoder *Encoder

		if isNumericRange(fp.Range) {
			// Numeric feature - parse directly
			encoded = kde.parseNumericColumn(column)
		} else {
			// Categorical feature - encode
			encoded, encoder = kde.encodeCategoricalColumn(column)
			dataset.FeatureEncoders[fp.Label] = encoder
		}

		// Populate feature matrix
		for rowIdx, val := range encoded {
			X[rowIdx][featureIdx] = val
		}
	}

	dataset.X = X

	// Encode target
	if modelType == "regression" {
		// Numeric target
		Y := kde.parseNumericColumn(rawTargets)
		dataset.Y = Y
	} else {
		// Categorical target - encode
		encodedTargets, targetEncoder := kde.encodeCategoricalColumn(rawTargets)
		dataset.TargetEncoder = targetEncoder

		// Convert back to string labels for classifier
		Y := make([]string, len(rawTargets))
		for i, rawTarget := range rawTargets {
			Y[i] = rawTarget
		}
		dataset.Y = Y

		// Also store encoded version for potential use
		_ = encodedTargets // We keep the string version for the classifier
	}

	dataset.SampleCount = len(X)
	dataset.FeatureCount = len(featureProperties)

	return dataset, nil
}

// buildSPARQLQuery constructs a SPARQL SELECT query to extract training data
func (kde *KGDataExtractor) buildSPARQLQuery(
	graphURI string,
	targetProperty ontology.OntologyProperty,
	featureProperties []ontology.OntologyProperty,
) string {
	var sb strings.Builder

	// Build SELECT clause
	sb.WriteString("SELECT ?entity")

	for _, fp := range featureProperties {
		varName := kde.sanitizeVarName(fp.Label)
		sb.WriteString(fmt.Sprintf(" ?%s", varName))
	}

	targetVarName := kde.sanitizeVarName(targetProperty.Label)
	sb.WriteString(fmt.Sprintf(" ?%s", targetVarName))
	sb.WriteString("\n")

	// Build WHERE clause
	sb.WriteString(fmt.Sprintf("WHERE {\n  GRAPH <%s> {\n", graphURI))

	// Determine the class from domain (use first domain if multiple)
	classURI := ""
	if len(targetProperty.Domain) > 0 {
		classURI = targetProperty.Domain[0]
	} else {
		// Fallback - query any entity that has the target property
		classURI = "?entityType"
	}

	if !strings.HasPrefix(classURI, "?") {
		sb.WriteString(fmt.Sprintf("    ?entity a <%s> .\n", classURI))
	}

	// Add patterns for each feature
	for _, fp := range featureProperties {
		varName := kde.sanitizeVarName(fp.Label)
		sb.WriteString(fmt.Sprintf("    ?entity <%s> ?%s .\n", fp.URI, varName))
	}

	// Add pattern for target
	sb.WriteString(fmt.Sprintf("    ?entity <%s> ?%s .\n", targetProperty.URI, targetVarName))

	sb.WriteString("  }\n}\n")

	// Limit results for reasonable training set size
	sb.WriteString("LIMIT 10000")

	return sb.String()
}

// sanitizeVarName converts a property label to a valid SPARQL variable name
func (kde *KGDataExtractor) sanitizeVarName(label string) string {
	// Replace spaces and special characters with underscores
	sanitized := strings.ReplaceAll(label, " ", "_")
	sanitized = strings.ReplaceAll(sanitized, "-", "_")
	sanitized = strings.ReplaceAll(sanitized, ".", "_")
	sanitized = strings.ReplaceAll(sanitized, "/", "_")
	sanitized = strings.ToLower(sanitized)

	// Ensure it starts with a letter
	if len(sanitized) > 0 && (sanitized[0] >= '0' && sanitized[0] <= '9') {
		sanitized = "v_" + sanitized
	}

	return sanitized
}

// parseNumericColumn parses string values to float64
func (kde *KGDataExtractor) parseNumericColumn(values []string) []float64 {
	result := make([]float64, len(values))

	for i, val := range values {
		// Try to parse as float
		parsed, err := strconv.ParseFloat(val, 64)
		if err != nil {
			// If parsing fails, use 0 (later we can implement better imputation)
			parsed = 0.0
		}
		result[i] = parsed
	}

	return result
}

// encodeCategoricalColumn encodes categorical values to numeric using label encoding
func (kde *KGDataExtractor) encodeCategoricalColumn(values []string) ([]float64, *Encoder) {
	// Find unique values
	uniqueMap := make(map[string]bool)
	for _, val := range values {
		uniqueMap[val] = true
	}

	unique := make([]string, 0, len(uniqueMap))
	for val := range uniqueMap {
		unique = append(unique, val)
	}

	// Create encoding mapping
	mapping := make(map[string]float64)
	reverseMapping := make(map[float64]string)
	for i, val := range unique {
		numericVal := float64(i)
		mapping[val] = numericVal
		reverseMapping[numericVal] = val
	}

	// Encode values
	encoded := make([]float64, len(values))
	for i, val := range values {
		encoded[i] = mapping[val]
	}

	encoder := &Encoder{
		Type:           "label",
		Mapping:        mapping,
		ReverseMapping: reverseMapping,
		UniqueCe:       unique,
	}

	return encoded, encoder
}

// DecodeTarget decodes a numeric prediction back to categorical label
func (e *Encoder) DecodeTarget(value float64) string {
	if label, ok := e.ReverseMapping[value]; ok {
		return label
	}
	// Fallback - find nearest
	for numVal, label := range e.ReverseMapping {
		if numVal == value {
			return label
		}
	}
	return "unknown"
}

// GetClassLabels returns all possible class labels for classification
func (kde *KGDataExtractor) GetClassLabels(dataset *TrainingDataset) []string {
	if dataset.ModelType != "classification" || dataset.TargetEncoder == nil {
		return nil
	}
	return dataset.TargetEncoder.UniqueCe
}

// ValidateDataset performs basic validation on the extracted dataset
func (kde *KGDataExtractor) ValidateDataset(dataset *TrainingDataset) error {
	if dataset.SampleCount == 0 {
		return fmt.Errorf("dataset has no samples")
	}

	if dataset.FeatureCount == 0 {
		return fmt.Errorf("dataset has no features")
	}

	if len(dataset.X) != dataset.SampleCount {
		return fmt.Errorf("X matrix size mismatch: expected %d rows, got %d", dataset.SampleCount, len(dataset.X))
	}

	// Check for minimum sample requirements
	if dataset.ModelType == "regression" && dataset.SampleCount < 30 {
		return fmt.Errorf("insufficient samples for regression: %d (recommend at least 30)", dataset.SampleCount)
	}

	if dataset.ModelType == "classification" && dataset.SampleCount < 50 {
		return fmt.Errorf("insufficient samples for classification: %d (recommend at least 50)", dataset.SampleCount)
	}

	// Check feature-to-sample ratio
	if dataset.SampleCount < dataset.FeatureCount*3 {
		return fmt.Errorf("poor feature-to-sample ratio: %d samples for %d features (recommend at least 3x)",
			dataset.SampleCount, dataset.FeatureCount)
	}

	return nil
}

// GetDatasetSummary returns a human-readable summary of the dataset
func (kde *KGDataExtractor) GetDatasetSummary(dataset *TrainingDataset) string {
	summary := fmt.Sprintf("Dataset for %s (%s):\n", dataset.TargetName, dataset.ModelType)
	summary += fmt.Sprintf("  - %d samples\n", dataset.SampleCount)
	summary += fmt.Sprintf("  - %d features: %s\n", dataset.FeatureCount, strings.Join(dataset.FeatureNames, ", "))

	if dataset.ModelType == "classification" && dataset.TargetEncoder != nil {
		summary += fmt.Sprintf("  - %d classes: %s\n",
			len(dataset.TargetEncoder.UniqueCe),
			strings.Join(dataset.TargetEncoder.UniqueCe, ", "))
	}

	return summary
}
