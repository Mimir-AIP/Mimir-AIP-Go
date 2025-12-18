package ml

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Mimir-AIP/Mimir-AIP-Go/pipelines/Ontology"
	"github.com/Mimir-AIP/Mimir-AIP-Go/pipelines/Storage"
)

// OntologyAnalyzer analyzes ontologies to suggest ML capabilities
type OntologyAnalyzer struct {
	Storage *storage.PersistenceBackend
}

// NewOntologyAnalyzer creates a new ontology analyzer
func NewOntologyAnalyzer(store *storage.PersistenceBackend) *OntologyAnalyzer {
	return &OntologyAnalyzer{Storage: store}
}

// MLCapabilities represents the ML capabilities discovered from an ontology
type MLCapabilities struct {
	OntologyID            string                    `json:"ontology_id"`
	RegressionTargets     []MLTarget                `json:"regression_targets"`
	ClassificationTargets []MLTarget                `json:"classification_targets"`
	TimeSeriesMetrics     []TimeSeriesMetric        `json:"time_series_metrics"`
	MonitoringRules       []SuggestedMonitoringRule `json:"monitoring_rules"`
	Summary               string                    `json:"summary"`
	TotalDataPoints       int                       `json:"total_data_points"`
}

// MLTarget represents a potential ML prediction target
type MLTarget struct {
	PropertyURI       string   `json:"property_uri"`
	PropertyLabel     string   `json:"property_label"`
	PropertyType      string   `json:"property_type"`
	DataType          string   `json:"data_type"`
	SuggestedFeatures []string `json:"suggested_features"`
	Confidence        float64  `json:"confidence"`
	Reasoning         string   `json:"reasoning"`
	EstimatedSamples  int      `json:"estimated_samples"`
}

// TimeSeriesMetric represents a temporal property for monitoring
type TimeSeriesMetric struct {
	PropertyURI   string  `json:"property_uri"`
	PropertyLabel string  `json:"property_label"`
	DataType      string  `json:"data_type"`
	Confidence    float64 `json:"confidence"`
	Reasoning     string  `json:"reasoning"`
}

// SuggestedMonitoringRule represents an automatically suggested monitoring rule
type SuggestedMonitoringRule struct {
	RuleName      string                 `json:"rule_name"`
	PropertyURI   string                 `json:"property_uri"`
	PropertyLabel string                 `json:"property_label"`
	RuleType      string                 `json:"rule_type"`
	Condition     map[string]interface{} `json:"condition"`
	Severity      string                 `json:"severity"`
	Reasoning     string                 `json:"reasoning"`
	Confidence    float64                `json:"confidence"`
}

// AnalyzeMLCapabilities analyzes an ontology and returns its ML capabilities
func (oa *OntologyAnalyzer) AnalyzeMLCapabilities(ctx context.Context, ontologyID string) (*MLCapabilities, error) {
	// Fetch all properties for this ontology
	properties, err := oa.getOntologyProperties(ontologyID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch ontology properties: %w", err)
	}

	if len(properties) == 0 {
		return nil, fmt.Errorf("no properties found for ontology %s", ontologyID)
	}

	// Estimate data points (query knowledge graph for count)
	totalDataPoints, err := oa.estimateDataPoints(ontologyID)
	if err != nil {
		totalDataPoints = 0 // Continue even if we can't estimate
	}

	capabilities := &MLCapabilities{
		OntologyID:            ontologyID,
		RegressionTargets:     []MLTarget{},
		ClassificationTargets: []MLTarget{},
		TimeSeriesMetrics:     []TimeSeriesMetric{},
		MonitoringRules:       []SuggestedMonitoringRule{},
		TotalDataPoints:       totalDataPoints,
	}

	// Analyze each property to determine its ML potential
	for _, prop := range properties {
		oa.analyzeProperty(prop, properties, capabilities, totalDataPoints)
	}

	// Generate summary
	capabilities.Summary = oa.generateSummary(capabilities)

	return capabilities, nil
}

// getOntologyProperties retrieves all properties for an ontology from the database
func (oa *OntologyAnalyzer) getOntologyProperties(ontologyID string) ([]ontology.OntologyProperty, error) {
	query := `
		SELECT uri, label, property_type, domain, range, description
		FROM ontology_properties
		WHERE ontology_id = ?
		ORDER BY label
	`

	rows, err := oa.Storage.GetDB().Query(query, ontologyID)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	var properties []ontology.OntologyProperty
	for rows.Next() {
		var prop ontology.OntologyProperty
		var domainStr, rangeStr sql.NullString

		err := rows.Scan(&prop.URI, &prop.Label, &prop.PropertyType, &domainStr, &rangeStr, &prop.Description)
		if err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}

		// Parse JSON arrays for domain and range
		if domainStr.Valid && domainStr.String != "" {
			_ = json.Unmarshal([]byte(domainStr.String), &prop.Domain)
		}
		if rangeStr.Valid && rangeStr.String != "" {
			_ = json.Unmarshal([]byte(rangeStr.String), &prop.Range)
		}

		properties = append(properties, prop)
	}

	return properties, rows.Err()
}

// estimateDataPoints queries the knowledge graph to estimate the number of entities
func (oa *OntologyAnalyzer) estimateDataPoints(ontologyID string) (int, error) {
	// Query the time_series_data table for this ontology (if available)
	query := `
		SELECT COUNT(DISTINCT entity_id)
		FROM time_series_data
		WHERE ontology_id = ?
	`

	var count int
	err := oa.Storage.GetDB().QueryRow(query, ontologyID).Scan(&count)
	if err != nil && err != sql.ErrNoRows {
		return 0, err
	}

	// If no time-series data, try to estimate from the ontology metadata
	if count == 0 {
		// For now, return 0. Later we can query TDB2 for actual triple counts
		return 0, nil
	}

	return count, nil
}

// analyzeProperty analyzes a single property to determine its ML potential
func (oa *OntologyAnalyzer) analyzeProperty(
	prop ontology.OntologyProperty,
	allProperties []ontology.OntologyProperty,
	capabilities *MLCapabilities,
	totalDataPoints int,
) {
	// Skip annotation properties (they're metadata, not data)
	if prop.PropertyType == ontology.PropertyTypeAnnotation {
		return
	}

	// Determine what kind of ML target this could be
	if isNumericRange(prop.Range) {
		// Numeric property - could be regression target or time-series metric
		target := oa.createMLTarget(prop, allProperties, "regression", totalDataPoints)

		// Check if this looks like a time-series metric
		if oa.isTimeSeriesCandidate(prop) {
			tsMetric := TimeSeriesMetric{
				PropertyURI:   prop.URI,
				PropertyLabel: prop.Label,
				DataType:      getDataType(prop.Range),
				Confidence:    calculateTimeSeriesConfidence(prop),
				Reasoning:     fmt.Sprintf("Property '%s' is numeric and appears suitable for time-series monitoring", prop.Label),
			}
			capabilities.TimeSeriesMetrics = append(capabilities.TimeSeriesMetrics, tsMetric)

			// Suggest monitoring rules
			rules := oa.suggestMonitoringRules(prop)
			capabilities.MonitoringRules = append(capabilities.MonitoringRules, rules...)
		}

		capabilities.RegressionTargets = append(capabilities.RegressionTargets, target)

	} else if isCategoricalRange(prop.Range) {
		// Categorical property - could be classification target
		target := oa.createMLTarget(prop, allProperties, "classification", totalDataPoints)
		capabilities.ClassificationTargets = append(capabilities.ClassificationTargets, target)
	}
}

// createMLTarget creates an MLTarget struct for a property
func (oa *OntologyAnalyzer) createMLTarget(
	prop ontology.OntologyProperty,
	allProperties []ontology.OntologyProperty,
	modelType string,
	totalDataPoints int,
) MLTarget {
	// Find potential features (other properties with same domain)
	suggestedFeatures := oa.findSuggestedFeatures(prop, allProperties)

	// Calculate confidence score
	confidence := oa.calculateConfidence(prop, len(suggestedFeatures), totalDataPoints)

	// Generate reasoning
	reasoning := oa.generateReasoning(prop, modelType, len(suggestedFeatures), totalDataPoints)

	return MLTarget{
		PropertyURI:       prop.URI,
		PropertyLabel:     prop.Label,
		PropertyType:      string(prop.PropertyType),
		DataType:          getDataType(prop.Range),
		SuggestedFeatures: suggestedFeatures,
		Confidence:        confidence,
		Reasoning:         reasoning,
		EstimatedSamples:  totalDataPoints,
	}
}

// findSuggestedFeatures finds other properties that could serve as features
func (oa *OntologyAnalyzer) findSuggestedFeatures(
	targetProp ontology.OntologyProperty,
	allProperties []ontology.OntologyProperty,
) []string {
	var features []string

	for _, prop := range allProperties {
		// Skip the target property itself
		if prop.URI == targetProp.URI {
			continue
		}

		// Skip annotation properties
		if prop.PropertyType == ontology.PropertyTypeAnnotation {
			continue
		}

		// Include properties with overlapping domains
		if hasOverlappingDomains(targetProp.Domain, prop.Domain) {
			features = append(features, prop.Label)
		}
	}

	return features
}

// calculateConfidence calculates a confidence score for an ML target
func (oa *OntologyAnalyzer) calculateConfidence(
	prop ontology.OntologyProperty,
	featureCount int,
	dataCount int,
) float64 {
	confidence := 0.5 // Base confidence

	// More data = higher confidence
	if dataCount > 100 {
		confidence += 0.1
	}
	if dataCount > 500 {
		confidence += 0.1
	}
	if dataCount > 1000 {
		confidence += 0.05
	}

	// Good feature availability
	if featureCount >= 3 {
		confidence += 0.1
	}
	if featureCount >= 5 {
		confidence += 0.05
	}

	// Property name suggests common ML target
	if isCommonTargetLabel(prop.Label) {
		confidence += 0.1
	}

	// Cap at 0.95 (never claim 100% certainty)
	if confidence > 0.95 {
		confidence = 0.95
	}

	return confidence
}

// generateReasoning generates a human-readable explanation for the ML suggestion
func (oa *OntologyAnalyzer) generateReasoning(
	prop ontology.OntologyProperty,
	modelType string,
	featureCount int,
	dataCount int,
) string {
	reasons := []string{}

	// Type-based reasoning
	if modelType == "regression" {
		reasons = append(reasons, fmt.Sprintf("'%s' is numeric and can be predicted using regression", prop.Label))
	} else {
		reasons = append(reasons, fmt.Sprintf("'%s' is categorical and can be predicted using classification", prop.Label))
	}

	// Feature-based reasoning
	if featureCount > 0 {
		reasons = append(reasons, fmt.Sprintf("%d related properties available as features", featureCount))
	} else {
		reasons = append(reasons, "limited features available (may need feature engineering)")
	}

	// Data-based reasoning
	if dataCount > 100 {
		reasons = append(reasons, fmt.Sprintf("sufficient data available (~%d samples)", dataCount))
	} else if dataCount > 0 {
		reasons = append(reasons, fmt.Sprintf("limited data available (~%d samples, recommend at least 50)", dataCount))
	} else {
		reasons = append(reasons, "data availability unknown (recommend validation before training)")
	}

	// Common target recognition
	if isCommonTargetLabel(prop.Label) {
		reasons = append(reasons, "commonly predicted in ML applications")
	}

	return strings.Join(reasons, "; ")
}

// isTimeSeriesCandidate determines if a property is suitable for time-series monitoring
func (oa *OntologyAnalyzer) isTimeSeriesCandidate(prop ontology.OntologyProperty) bool {
	// Numeric properties are good candidates for monitoring
	if !isNumericRange(prop.Range) {
		return false
	}

	// Properties with certain labels are commonly monitored
	monitoringKeywords := []string{
		"stock", "level", "quantity", "count", "amount",
		"price", "cost", "value", "revenue", "sales",
		"rate", "speed", "throughput", "latency",
		"usage", "utilization", "capacity",
	}

	lowerLabel := strings.ToLower(prop.Label)
	for _, keyword := range monitoringKeywords {
		if strings.Contains(lowerLabel, keyword) {
			return true
		}
	}

	return false
}

// suggestMonitoringRules suggests monitoring rules for a property
func (oa *OntologyAnalyzer) suggestMonitoringRules(prop ontology.OntologyProperty) []SuggestedMonitoringRule {
	var rules []SuggestedMonitoringRule

	lowerLabel := strings.ToLower(prop.Label)

	// Stock/inventory level rules
	if strings.Contains(lowerLabel, "stock") || strings.Contains(lowerLabel, "inventory") || strings.Contains(lowerLabel, "level") {
		rules = append(rules, SuggestedMonitoringRule{
			RuleName:      fmt.Sprintf("Low %s Alert", prop.Label),
			PropertyURI:   prop.URI,
			PropertyLabel: prop.Label,
			RuleType:      "threshold",
			Condition:     map[string]interface{}{"<": 5},
			Severity:      "high",
			Reasoning:     fmt.Sprintf("Low %s may indicate stockout risk", prop.Label),
			Confidence:    0.8,
		})
	}

	// Price/cost monitoring rules
	if strings.Contains(lowerLabel, "price") || strings.Contains(lowerLabel, "cost") {
		rules = append(rules, SuggestedMonitoringRule{
			RuleName:      fmt.Sprintf("%s Increase Alert", prop.Label),
			PropertyURI:   prop.URI,
			PropertyLabel: prop.Label,
			RuleType:      "trend",
			Condition:     map[string]interface{}{"change_percent": 15, "direction": "increasing"},
			Severity:      "medium",
			Reasoning:     fmt.Sprintf("Large %s increases may require review", prop.Label),
			Confidence:    0.7,
		})
	}

	// Generic anomaly detection for all numeric metrics
	rules = append(rules, SuggestedMonitoringRule{
		RuleName:      fmt.Sprintf("%s Anomaly Detection", prop.Label),
		PropertyURI:   prop.URI,
		PropertyLabel: prop.Label,
		RuleType:      "anomaly",
		Condition:     map[string]interface{}{"z_score": 3},
		Severity:      "medium",
		Reasoning:     fmt.Sprintf("Detect unusual %s values (3+ standard deviations)", prop.Label),
		Confidence:    0.85,
	})

	return rules
}

// generateSummary generates a human-readable summary of capabilities
func (oa *OntologyAnalyzer) generateSummary(capabilities *MLCapabilities) string {
	parts := []string{}

	if len(capabilities.RegressionTargets) > 0 {
		targets := make([]string, 0, len(capabilities.RegressionTargets))
		for _, t := range capabilities.RegressionTargets {
			targets = append(targets, t.PropertyLabel)
		}
		parts = append(parts, fmt.Sprintf("predict %s (regression)", strings.Join(targets, ", ")))
	}

	if len(capabilities.ClassificationTargets) > 0 {
		targets := make([]string, 0, len(capabilities.ClassificationTargets))
		for _, t := range capabilities.ClassificationTargets {
			targets = append(targets, t.PropertyLabel)
		}
		parts = append(parts, fmt.Sprintf("classify %s", strings.Join(targets, ", ")))
	}

	if len(capabilities.TimeSeriesMetrics) > 0 {
		parts = append(parts, fmt.Sprintf("monitor %d metrics", len(capabilities.TimeSeriesMetrics)))
	}

	if len(parts) == 0 {
		return "No ML capabilities detected for this ontology"
	}

	return fmt.Sprintf("I can %s", strings.Join(parts, "; "))
}

// Helper functions

// isNumericRange checks if any of the range values indicate a numeric type
func isNumericRange(ranges []string) bool {
	for _, r := range ranges {
		lowerR := strings.ToLower(r)
		if strings.Contains(lowerR, "xsd:decimal") ||
			strings.Contains(lowerR, "xsd:float") ||
			strings.Contains(lowerR, "xsd:integer") ||
			strings.Contains(lowerR, "xsd:double") ||
			strings.Contains(lowerR, "xsd:int") ||
			strings.Contains(lowerR, "xsd:long") ||
			strings.Contains(lowerR, "xmlschema#decimal") ||
			strings.Contains(lowerR, "xmlschema#float") ||
			strings.Contains(lowerR, "xmlschema#integer") ||
			strings.Contains(lowerR, "xmlschema#double") ||
			strings.Contains(lowerR, "xmlschema#int") ||
			strings.Contains(lowerR, "xmlschema#long") {
			return true
		}
	}
	return false
}

// isCategoricalRange checks if any of the range values indicate a categorical type
func isCategoricalRange(ranges []string) bool {
	for _, r := range ranges {
		lowerR := strings.ToLower(r)
		// Check for XSD string or boolean types
		if strings.Contains(lowerR, "xsd:string") ||
			strings.Contains(lowerR, "xsd:boolean") ||
			strings.Contains(lowerR, "xmlschema#string") ||
			strings.Contains(lowerR, "xmlschema#boolean") {
			return true
		}
		// Check for URIs that are NOT XSD datatypes (these are object properties/classes)
		if (strings.HasPrefix(r, "http://") || strings.HasPrefix(r, "https://")) &&
			!strings.Contains(lowerR, "xmlschema#") {
			return true
		}
	}
	return false
}

// isTemporalRange checks if any of the range values indicate a temporal type
func isTemporalRange(ranges []string) bool {
	for _, r := range ranges {
		if strings.Contains(r, "xsd:dateTime") ||
			strings.Contains(r, "xsd:date") ||
			strings.Contains(r, "xsd:time") {
			return true
		}
	}
	return false
}

// getDataType extracts a human-readable data type from range values
func getDataType(ranges []string) string {
	if isNumericRange(ranges) {
		if len(ranges) > 0 && strings.Contains(ranges[0], "integer") {
			return "integer"
		}
		return "numeric"
	}
	if isCategoricalRange(ranges) {
		if len(ranges) > 0 && strings.Contains(ranges[0], "boolean") {
			return "boolean"
		}
		return "categorical"
	}
	if isTemporalRange(ranges) {
		return "temporal"
	}
	return "unknown"
}

// isCommonTargetLabel checks if a label indicates a commonly predicted property
func isCommonTargetLabel(label string) bool {
	commonTargets := []string{
		"price", "cost", "revenue", "sales", "profit",
		"quantity", "amount", "total", "value",
		"rating", "score", "rank",
		"category", "class", "type", "status",
	}

	lowerLabel := strings.ToLower(label)
	for _, target := range commonTargets {
		if strings.Contains(lowerLabel, target) {
			return true
		}
	}

	return false
}

// hasOverlappingDomains checks if two domain sets have any overlap
func hasOverlappingDomains(domains1, domains2 []string) bool {
	if len(domains1) == 0 || len(domains2) == 0 {
		// If either has no domain restrictions, assume they could overlap
		return true
	}

	for _, d1 := range domains1 {
		for _, d2 := range domains2 {
			if d1 == d2 {
				return true
			}
		}
	}

	return false
}

// calculateTimeSeriesConfidence calculates confidence for time-series monitoring
func calculateTimeSeriesConfidence(prop ontology.OntologyProperty) float64 {
	confidence := 0.6 // Base confidence for numeric properties

	// Boost for monitoring-relevant labels
	lowerLabel := strings.ToLower(prop.Label)
	monitoringKeywords := []string{"stock", "level", "price", "cost", "rate", "usage"}

	for _, keyword := range monitoringKeywords {
		if strings.Contains(lowerLabel, keyword) {
			confidence += 0.2
			break
		}
	}

	if confidence > 0.95 {
		confidence = 0.95
	}

	return confidence
}
