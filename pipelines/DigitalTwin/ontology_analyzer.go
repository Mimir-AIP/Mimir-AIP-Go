package DigitalTwin

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Mimir-AIP/Mimir-AIP-Go/pipelines/AI"
)

// OntologyAnalyzer analyzes an ontology to understand domain-specific patterns
// and generate relevant scenarios automatically
type OntologyAnalyzer struct {
	llmClient AI.LLMClient
}

// EntityPattern represents a recognized pattern in the ontology
type EntityPattern struct {
	EntityType   string   `json:"entity_type"`
	PatternType  string   `json:"pattern_type"` // "resource", "actor", "process", "metric", "dependency"
	KeyProperty  string   `json:"key_property"` // The main property to simulate (budget, count, status)
	PropertyType string   `json:"property_type"` // "numeric", "enum", "boolean", "date"
	Importance   float64  `json:"importance"`    // 0-1, based on relationship count
	DependsOn    []string `json:"depends_on"`    // Entity types this depends on
	DependedBy   []string `json:"depended_by"`   // Entity types that depend on this
}

// OntologyAnalysis contains the full analysis of an ontology
type OntologyAnalysis struct {
	DomainType       string                    `json:"domain_type"`        // Inferred domain: "nonprofit", "supply_chain", "healthcare", etc.
	DomainKeywords   []string                  `json:"domain_keywords"`    // Key terms found
	EntityPatterns   []EntityPattern           `json:"entity_patterns"`    // Patterns found per entity type
	CriticalEntities []string                  `json:"critical_entities"`  // Entities with high dependency
	RiskFactors      []RiskFactor              `json:"risk_factors"`       // Identified risks
	SuggestedMetrics []SuggestedMetric         `json:"suggested_metrics"`  // Metrics to track
	RelationshipMap  map[string][]Relationship `json:"relationship_map"`   // How entities connect
}

// RiskFactor represents an identified risk in the system
type RiskFactor struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Severity    string   `json:"severity"` // "low", "medium", "high", "critical"
	Entities    []string `json:"entities"` // Affected entities
	Mitigation  string   `json:"mitigation,omitempty"`
}

// SuggestedMetric is a metric that should be tracked
type SuggestedMetric struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Formula     string   `json:"formula,omitempty"` // How to calculate
	Unit        string   `json:"unit,omitempty"`
	Entities    []string `json:"entities"` // Entities involved
}

// Relationship describes a connection between entities
type Relationship struct {
	TargetType     string  `json:"target_type"`
	RelationType   string  `json:"relation_type"`
	ImpactStrength float64 `json:"impact_strength"` // How much changes propagate
}

// NewOntologyAnalyzer creates a new analyzer
func NewOntologyAnalyzer(llmClient AI.LLMClient) *OntologyAnalyzer {
	return &OntologyAnalyzer{
		llmClient: llmClient,
	}
}

// AnalyzeOntology performs deep analysis of an ontology structure
func (oa *OntologyAnalyzer) AnalyzeOntology(ctx context.Context, entities []TwinEntity, relationships []TwinRelationship) (*OntologyAnalysis, error) {
	// First, do rule-based analysis
	analysis := oa.ruleBasedAnalysis(entities, relationships)

	// Then, if LLM is available, enhance with AI insights
	if oa.llmClient != nil {
		enhanced, err := oa.llmEnhancedAnalysis(ctx, analysis, entities, relationships)
		if err == nil {
			analysis = enhanced
		}
		// If LLM fails, we still have rule-based analysis
	}

	return analysis, nil
}

// ruleBasedAnalysis performs analysis using heuristics (no LLM required)
func (oa *OntologyAnalyzer) ruleBasedAnalysis(entities []TwinEntity, relationships []TwinRelationship) *OntologyAnalysis {
	analysis := &OntologyAnalysis{
		EntityPatterns:   []EntityPattern{},
		CriticalEntities: []string{},
		RiskFactors:      []RiskFactor{},
		SuggestedMetrics: []SuggestedMetric{},
		RelationshipMap:  make(map[string][]Relationship),
	}

	// Count entity types
	entityTypeCounts := make(map[string]int)
	entityTypeExamples := make(map[string][]TwinEntity)
	for _, e := range entities {
		entityTypeCounts[e.Type]++
		entityTypeExamples[e.Type] = append(entityTypeExamples[e.Type], e)
	}

	// Build relationship map and count dependencies
	outgoingCount := make(map[string]int) // How many things this entity type affects
	incomingCount := make(map[string]int) // How many things affect this entity type

	for _, r := range relationships {
		// Find entity types for source and target
		sourceType := oa.getEntityType(r.SourceURI, entities)
		targetType := oa.getEntityType(r.TargetURI, entities)

		if sourceType != "" && targetType != "" {
			outgoingCount[sourceType]++
			incomingCount[targetType]++

			analysis.RelationshipMap[sourceType] = append(analysis.RelationshipMap[sourceType], Relationship{
				TargetType:     targetType,
				RelationType:   r.Type,
				ImpactStrength: r.Strength,
			})
		}
	}

	// Infer domain from entity type names and labels
	analysis.DomainType, analysis.DomainKeywords = oa.inferDomain(entities)

	// Analyze each entity type
	for entityType, count := range entityTypeCounts {
		pattern := EntityPattern{
			EntityType: entityType,
			Importance: oa.calculateImportance(entityType, outgoingCount, incomingCount, count),
		}

		// Infer pattern type from name/relationships
		pattern.PatternType = oa.inferPatternType(entityType, outgoingCount[entityType], incomingCount[entityType])

		// Check for key properties in examples
		if examples, ok := entityTypeExamples[entityType]; ok && len(examples) > 0 {
			pattern.KeyProperty, pattern.PropertyType = oa.inferKeyProperty(examples[0])
		}

		// Track dependencies
		for _, rel := range analysis.RelationshipMap[entityType] {
			pattern.DependsOn = appendUnique(pattern.DependsOn, rel.TargetType)
		}
		for sourceType, rels := range analysis.RelationshipMap {
			for _, rel := range rels {
				if rel.TargetType == entityType {
					pattern.DependedBy = appendUnique(pattern.DependedBy, sourceType)
				}
			}
		}

		analysis.EntityPatterns = append(analysis.EntityPatterns, pattern)

		// Mark as critical if high importance
		if pattern.Importance > 0.7 {
			analysis.CriticalEntities = append(analysis.CriticalEntities, entityType)
		}
	}

	// Generate risk factors based on patterns
	analysis.RiskFactors = oa.identifyRisks(analysis)

	// Generate suggested metrics
	analysis.SuggestedMetrics = oa.suggestMetrics(analysis)

	return analysis
}

// llmEnhancedAnalysis uses LLM to provide deeper insights
func (oa *OntologyAnalyzer) llmEnhancedAnalysis(ctx context.Context, baseAnalysis *OntologyAnalysis, entities []TwinEntity, relationships []TwinRelationship) (*OntologyAnalysis, error) {
	// Build a summary of the ontology for the LLM
	summary := oa.buildOntologySummary(entities, relationships)

	prompt := fmt.Sprintf(`You are analyzing a data ontology for a digital twin simulation system.
Based on the following ontology structure, provide enhanced analysis.

ONTOLOGY SUMMARY:
%s

CURRENT ANALYSIS:
- Inferred Domain: %s
- Critical Entities: %v
- Entity Patterns: %d types identified

Please provide:
1. A more specific domain classification (e.g., "humanitarian_ngo", "retail_supply_chain", "healthcare_provider")
2. Additional risk factors specific to this domain
3. Key metrics that should be tracked for this type of organization
4. Relationships that are critical for impact propagation

Respond in JSON format:
{
  "domain_type": "specific_domain",
  "domain_description": "description",
  "additional_risks": [{"name": "...", "description": "...", "severity": "high/medium/low", "mitigation": "..."}],
  "key_metrics": [{"name": "...", "description": "...", "formula": "...", "unit": "..."}],
  "critical_relationships": [{"from": "EntityType", "to": "EntityType", "why": "explanation"}]
}`, summary, baseAnalysis.DomainType, baseAnalysis.CriticalEntities, len(baseAnalysis.EntityPatterns))

	response, err := oa.llmClient.Complete(ctx, AI.LLMRequest{
		Messages: []AI.LLMMessage{
			{Role: "system", Content: "You are an expert in business process analysis and digital twin modeling. Provide structured JSON responses."},
			{Role: "user", Content: prompt},
		},
		Temperature: 0.3, // Lower temperature for more consistent analysis
		MaxTokens:   1500,
	})
	if err != nil {
		return nil, fmt.Errorf("LLM analysis failed: %w", err)
	}

	// Parse LLM response and merge with base analysis
	enhanced := *baseAnalysis
	oa.mergeLLMInsights(&enhanced, response.Content)

	return &enhanced, nil
}

// buildOntologySummary creates a text summary of the ontology
func (oa *OntologyAnalyzer) buildOntologySummary(entities []TwinEntity, relationships []TwinRelationship) string {
	var sb strings.Builder

	// Count by type
	typeCounts := make(map[string]int)
	typeLabels := make(map[string][]string)
	for _, e := range entities {
		shortType := oa.shortTypeName(e.Type)
		typeCounts[shortType]++
		if len(typeLabels[shortType]) < 3 { // Keep up to 3 examples
			typeLabels[shortType] = append(typeLabels[shortType], e.Label)
		}
	}

	sb.WriteString("Entity Types:\n")
	for typ, count := range typeCounts {
		examples := strings.Join(typeLabels[typ], ", ")
		sb.WriteString(fmt.Sprintf("- %s: %d entities (e.g., %s)\n", typ, count, examples))
	}

	// Summarize relationships
	relCounts := make(map[string]int)
	for _, r := range relationships {
		shortType := oa.shortTypeName(r.Type)
		relCounts[shortType]++
	}

	sb.WriteString("\nRelationship Types:\n")
	for rel, count := range relCounts {
		sb.WriteString(fmt.Sprintf("- %s: %d relationships\n", rel, count))
	}

	return sb.String()
}

// mergeLLMInsights merges LLM response into analysis
func (oa *OntologyAnalyzer) mergeLLMInsights(analysis *OntologyAnalysis, llmResponse string) {
	// Try to extract JSON from response
	jsonStart := strings.Index(llmResponse, "{")
	jsonEnd := strings.LastIndex(llmResponse, "}")
	if jsonStart < 0 || jsonEnd < 0 {
		return
	}

	jsonStr := llmResponse[jsonStart : jsonEnd+1]

	var insights struct {
		DomainType       string `json:"domain_type"`
		DomainDesc       string `json:"domain_description"`
		AdditionalRisks  []struct {
			Name        string `json:"name"`
			Description string `json:"description"`
			Severity    string `json:"severity"`
			Mitigation  string `json:"mitigation"`
		} `json:"additional_risks"`
		KeyMetrics []struct {
			Name        string `json:"name"`
			Description string `json:"description"`
			Formula     string `json:"formula"`
			Unit        string `json:"unit"`
		} `json:"key_metrics"`
	}

	if err := json.Unmarshal([]byte(jsonStr), &insights); err != nil {
		return
	}

	// Update domain
	if insights.DomainType != "" {
		analysis.DomainType = insights.DomainType
	}

	// Add risks
	for _, risk := range insights.AdditionalRisks {
		analysis.RiskFactors = append(analysis.RiskFactors, RiskFactor{
			Name:        risk.Name,
			Description: risk.Description,
			Severity:    risk.Severity,
			Mitigation:  risk.Mitigation,
		})
	}

	// Add metrics
	for _, metric := range insights.KeyMetrics {
		analysis.SuggestedMetrics = append(analysis.SuggestedMetrics, SuggestedMetric{
			Name:        metric.Name,
			Description: metric.Description,
			Formula:     metric.Formula,
			Unit:        metric.Unit,
		})
	}
}

// Helper functions

func (oa *OntologyAnalyzer) getEntityType(uri string, entities []TwinEntity) string {
	for _, e := range entities {
		if e.URI == uri {
			return e.Type
		}
	}
	return ""
}

func (oa *OntologyAnalyzer) shortTypeName(fullType string) string {
	// Extract the last part of a URI
	if idx := strings.LastIndex(fullType, "#"); idx >= 0 {
		return fullType[idx+1:]
	}
	if idx := strings.LastIndex(fullType, "/"); idx >= 0 {
		return fullType[idx+1:]
	}
	return fullType
}

func (oa *OntologyAnalyzer) inferDomain(entities []TwinEntity) (string, []string) {
	keywords := make(map[string]int)

	// Domain keyword patterns
	domainPatterns := map[string][]string{
		//TODO Needs to be either greatly expanded or replaced with a more accurate domain classification.
		"nonprofit":     {"donor", "beneficiary", "grant", "program", "volunteer", "ngo", "charity", "donation", "fund"},
		"supply_chain":  {"supplier", "warehouse", "inventory", "shipment", "order", "product", "logistics", "vendor"},
		"healthcare":    {"patient", "doctor", "treatment", "hospital", "diagnosis", "medication", "clinic", "medical"},
		"retail":        {"customer", "store", "sale", "product", "inventory", "order", "payment", "discount"},
		"manufacturing": {"machine", "production", "assembly", "quality", "defect", "batch", "line", "output"},
		"finance":       {"account", "transaction", "payment", "loan", "credit", "balance", "investment", "risk"},
		"hr":            {"employee", "department", "salary", "hiring", "performance", "leave", "training", "manager"},
	}

	// Scan entities for keywords
	for _, e := range entities {
		text := strings.ToLower(e.Type + " " + e.Label)
		for domain, patterns := range domainPatterns {
			for _, pattern := range patterns {
				if strings.Contains(text, pattern) {
					keywords[domain]++
				}
			}
		}
	}

	// Find best match
	bestDomain := "general"
	bestScore := 0
	foundKeywords := []string{}

	for domain, score := range keywords {
		if score > bestScore {
			bestDomain = domain
			bestScore = score
		}
	}

	// Get keywords for winning domain
	if patterns, ok := domainPatterns[bestDomain]; ok {
		for _, pattern := range patterns {
			for _, e := range entities {
				text := strings.ToLower(e.Type + " " + e.Label)
				if strings.Contains(text, pattern) {
					foundKeywords = appendUnique(foundKeywords, pattern)
					break
				}
			}
		}
	}

	return bestDomain, foundKeywords
}

func (oa *OntologyAnalyzer) calculateImportance(entityType string, outgoing, incoming map[string]int, count int) float64 {
	// Importance = weighted combination of relationship counts and entity count
	out := float64(outgoing[entityType])
	in := float64(incoming[entityType])

	// Entities that affect many others are more critical
	score := (out*0.6 + in*0.4) / 10.0 // Normalize

	// More instances = potentially more important
	countFactor := float64(count) / 100.0
	if countFactor > 0.3 {
		countFactor = 0.3
	}
	score += countFactor

	if score > 1.0 {
		score = 1.0
	}
	return score
}

func (oa *OntologyAnalyzer) inferPatternType(entityType string, outgoing, incoming int) string {
	typeLower := strings.ToLower(entityType)

	// Resource patterns (things that can be depleted/allocated)
	if strings.Contains(typeLower, "fund") || strings.Contains(typeLower, "budget") ||
		strings.Contains(typeLower, "resource") || strings.Contains(typeLower, "inventory") {
		return "resource"
	}

	// Actor patterns (things that do actions)
	if strings.Contains(typeLower, "person") || strings.Contains(typeLower, "user") ||
		strings.Contains(typeLower, "employee") || strings.Contains(typeLower, "donor") ||
		strings.Contains(typeLower, "customer") || strings.Contains(typeLower, "staff") {
		return "actor"
	}

	// Process patterns
	if strings.Contains(typeLower, "process") || strings.Contains(typeLower, "workflow") ||
		strings.Contains(typeLower, "pipeline") || strings.Contains(typeLower, "job") {
		return "process"
	}

	// Metric patterns
	if strings.Contains(typeLower, "metric") || strings.Contains(typeLower, "kpi") ||
		strings.Contains(typeLower, "measure") || strings.Contains(typeLower, "count") {
		return "metric"
	}

	// Infer from relationship pattern
	if outgoing > incoming*2 {
		return "actor" // Things that affect many others
	}
	if incoming > outgoing*2 {
		return "dependency" // Things that are affected by many
	}

	return "entity"
}

func (oa *OntologyAnalyzer) inferKeyProperty(entity TwinEntity) (string, string) {
	// Look for common property names
	propertyPriority := []struct {
		name  string
		ptype string
	}{
		{"budget", "numeric"},
		{"amount", "numeric"},
		{"count", "numeric"},
		{"capacity", "numeric"},
		{"quantity", "numeric"},
		{"status", "enum"},
		{"active", "boolean"},
		{"date", "date"},
		{"deadline", "date"},
	}

	for _, pp := range propertyPriority {
		for propName := range entity.Properties {
			if strings.Contains(strings.ToLower(propName), pp.name) {
				return propName, pp.ptype
			}
		}
	}

	// Default to first property if any
	for propName := range entity.Properties {
		return propName, "unknown"
	}

	return "status", "enum" // Default fallback
}

func (oa *OntologyAnalyzer) identifyRisks(analysis *OntologyAnalysis) []RiskFactor {
	risks := []RiskFactor{}

	// Risk: Single point of failure - entity with many dependents
	for _, pattern := range analysis.EntityPatterns {
		if len(pattern.DependedBy) >= 3 {
			risks = append(risks, RiskFactor{
				Name:        fmt.Sprintf("Concentration Risk: %s", oa.shortTypeName(pattern.EntityType)),
				Description: fmt.Sprintf("Multiple entity types depend on %s. If this fails, %d other types are affected.", oa.shortTypeName(pattern.EntityType), len(pattern.DependedBy)),
				Severity:    "high",
				Entities:    append([]string{pattern.EntityType}, pattern.DependedBy...),
				Mitigation:  "Consider diversifying or adding redundancy",
			})
		}
	}

	// Risk: Resource shortage - entities marked as resources
	for _, pattern := range analysis.EntityPatterns {
		if pattern.PatternType == "resource" {
			risks = append(risks, RiskFactor{
				Name:        fmt.Sprintf("Resource Shortage: %s", oa.shortTypeName(pattern.EntityType)),
				Description: fmt.Sprintf("%s is a resource that could be depleted or constrained", oa.shortTypeName(pattern.EntityType)),
				Severity:    "medium",
				Entities:    []string{pattern.EntityType},
			})
		}
	}

	// Risk: External dependency - actors that affect internal entities
	for _, pattern := range analysis.EntityPatterns {
		if pattern.PatternType == "actor" && len(pattern.DependsOn) > 0 {
			risks = append(risks, RiskFactor{
				Name:        fmt.Sprintf("External Dependency: %s", oa.shortTypeName(pattern.EntityType)),
				Description: fmt.Sprintf("System depends on %s which may be external/uncontrollable", oa.shortTypeName(pattern.EntityType)),
				Severity:    "medium",
				Entities:    []string{pattern.EntityType},
			})
		}
	}

	return risks
}

func (oa *OntologyAnalyzer) suggestMetrics(analysis *OntologyAnalysis) []SuggestedMetric {
	metrics := []SuggestedMetric{}

	// Generate metrics based on domain
	switch analysis.DomainType {
	case "nonprofit":
		metrics = append(metrics,
			SuggestedMetric{Name: "Program Reach", Description: "Number of beneficiaries served", Unit: "people"},
			SuggestedMetric{Name: "Donor Retention", Description: "Percentage of returning donors", Unit: "%"},
			SuggestedMetric{Name: "Fund Utilization", Description: "Funds used vs available", Unit: "%"},
		)
	case "supply_chain":
		metrics = append(metrics,
			SuggestedMetric{Name: "Order Fulfillment Rate", Description: "Orders completed on time", Unit: "%"},
			SuggestedMetric{Name: "Inventory Turnover", Description: "How often inventory is replaced", Unit: "times/year"},
			SuggestedMetric{Name: "Supplier Reliability", Description: "On-time delivery rate from suppliers", Unit: "%"},
		)
	case "healthcare":
		metrics = append(metrics,
			SuggestedMetric{Name: "Patient Wait Time", Description: "Average time to treatment", Unit: "hours"},
			SuggestedMetric{Name: "Bed Utilization", Description: "Percentage of beds in use", Unit: "%"},
			SuggestedMetric{Name: "Treatment Success Rate", Description: "Positive outcomes", Unit: "%"},
		)
	default:
		metrics = append(metrics,
			SuggestedMetric{Name: "System Utilization", Description: "Overall resource usage", Unit: "%"},
			SuggestedMetric{Name: "Process Throughput", Description: "Items processed per period", Unit: "count/day"},
		)
	}

	// Add metrics for critical entities
	for _, critical := range analysis.CriticalEntities {
		shortName := oa.shortTypeName(critical)
		metrics = append(metrics, SuggestedMetric{
			Name:        fmt.Sprintf("%s Health", shortName),
			Description: fmt.Sprintf("Overall health score for %s entities", shortName),
			Entities:    []string{critical},
			Unit:        "0-100",
		})
	}

	return metrics
}

func appendUnique(slice []string, item string) []string {
	for _, s := range slice {
		if s == item {
			return slice
		}
	}
	return append(slice, item)
}

