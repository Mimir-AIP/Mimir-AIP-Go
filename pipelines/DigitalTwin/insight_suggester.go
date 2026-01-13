package DigitalTwin

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/Mimir-AIP/Mimir-AIP-Go/pipelines/AI"
)

// InsightSuggester proactively identifies insights and suggests what-if questions
type InsightSuggester struct {
	llmClient AI.LLMClient
	analyzer  *OntologyAnalyzer
}

// Insight represents a proactive insight about the system
type Insight struct {
	ID          string    `json:"id"`
	Type        string    `json:"type"` // "risk", "opportunity", "warning", "trend", "question"
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Severity    string    `json:"severity,omitempty"` // "low", "medium", "high", "critical"
	Entities    []string  `json:"entities,omitempty"`
	Actions     []Action  `json:"actions,omitempty"`
	Confidence  float64   `json:"confidence"`
	CreatedAt   time.Time `json:"created_at"`
}

// Action represents a suggested action the user can take
type Action struct {
	Type        string                 `json:"type"` // "simulate", "investigate", "configure"
	Label       string                 `json:"label"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
}

// SuggestedQuestion is a what-if question the system suggests
type SuggestedQuestion struct {
	Question  string   `json:"question"`
	Reason    string   `json:"reason"`
	Relevance float64  `json:"relevance"` // 0-1
	Category  string   `json:"category"`  // "risk", "capacity", "dependency", "optimization"
	RelatedTo []string `json:"related_to,omitempty"`
}

// InsightReport contains all insights for a digital twin
type InsightReport struct {
	TwinID             string              `json:"twin_id"`
	GeneratedAt        time.Time           `json:"generated_at"`
	Insights           []Insight           `json:"insights"`
	SuggestedQuestions []SuggestedQuestion `json:"suggested_questions"`
	RiskScore          float64             `json:"risk_score"`   // Overall risk 0-1
	HealthScore        float64             `json:"health_score"` // Overall health 0-1
	Summary            string              `json:"summary"`
}

// NewInsightSuggester creates a new insight suggester
func NewInsightSuggester(llmClient AI.LLMClient) *InsightSuggester {
	return &InsightSuggester{
		llmClient: llmClient,
		analyzer:  NewOntologyAnalyzer(llmClient),
	}
}

// GenerateInsights analyzes a digital twin and generates proactive insights
func (is *InsightSuggester) GenerateInsights(ctx context.Context, twin *DigitalTwin) (*InsightReport, error) {
	report := &InsightReport{
		TwinID:             twin.ID,
		GeneratedAt:        time.Now(),
		Insights:           []Insight{},
		SuggestedQuestions: []SuggestedQuestion{},
		RiskScore:          0.0,
		HealthScore:        1.0,
	}

	// Analyze ontology structure
	analysis, err := is.analyzer.AnalyzeOntology(ctx, twin.Entities, twin.Relationships)
	if err != nil {
		return nil, fmt.Errorf("ontology analysis failed: %w", err)
	}

	// Generate insights from analysis
	insights := is.generateStructuralInsights(twin, analysis)
	report.Insights = append(report.Insights, insights...)

	// Generate insights from entity states
	stateInsights := is.generateStateInsights(twin)
	report.Insights = append(report.Insights, stateInsights...)

	// Generate suggested what-if questions
	questions := is.generateSuggestedQuestions(twin, analysis)
	report.SuggestedQuestions = questions

	// If LLM available, enhance with AI insights
	if is.llmClient != nil {
		aiInsights, err := is.generateAIInsights(ctx, twin, analysis)
		if err == nil && len(aiInsights) > 0 {
			report.Insights = append(report.Insights, aiInsights...)
		}
	}

	// Sort insights by severity
	is.sortInsights(report)

	// Calculate scores
	report.RiskScore = is.calculateRiskScore(report.Insights)
	report.HealthScore = 1.0 - (report.RiskScore * 0.7)

	// Generate summary
	report.Summary = is.generateSummary(report, analysis)

	return report, nil
}

// generateStructuralInsights creates insights from ontology structure
func (is *InsightSuggester) generateStructuralInsights(twin *DigitalTwin, analysis *OntologyAnalysis) []Insight {
	insights := []Insight{}

	// Insight: Concentration risk
	for _, pattern := range analysis.EntityPatterns {
		if len(pattern.DependedBy) >= 3 {
			insights = append(insights, Insight{
				ID:          fmt.Sprintf("concentration_%s", is.shortName(pattern.EntityType)),
				Type:        "risk",
				Title:       fmt.Sprintf("High Dependency on %s", is.shortName(pattern.EntityType)),
				Description: fmt.Sprintf("%d entity types depend on %s. Failure here would cascade widely.", len(pattern.DependedBy), is.shortName(pattern.EntityType)),
				Severity:    "high",
				Entities:    []string{pattern.EntityType},
				Confidence:  0.9,
				CreatedAt:   time.Now(),
				Actions: []Action{
					{
						Type:        "simulate",
						Label:       "Test Failure Impact",
						Description: fmt.Sprintf("Simulate what happens if %s becomes unavailable", is.shortName(pattern.EntityType)),
						Parameters:  map[string]interface{}{"target_type": pattern.EntityType, "event": "entity.unavailable"},
					},
				},
			})
		}
	}

	// Insight: Single points of failure
	entityCounts := make(map[string]int)
	for _, e := range twin.Entities {
		entityCounts[e.Type]++
	}
	for etype, count := range entityCounts {
		if count == 1 {
			// Check if other entities depend on this
			for _, pattern := range analysis.EntityPatterns {
				if pattern.EntityType == etype && len(pattern.DependedBy) > 0 {
					insights = append(insights, Insight{
						ID:          fmt.Sprintf("spof_%s", is.shortName(etype)),
						Type:        "warning",
						Title:       fmt.Sprintf("Single Point of Failure: %s", is.shortName(etype)),
						Description: fmt.Sprintf("Only one %s entity exists, but %d types depend on it. Consider redundancy.", is.shortName(etype), len(pattern.DependedBy)),
						Severity:    "medium",
						Entities:    []string{etype},
						Confidence:  0.85,
						CreatedAt:   time.Now(),
					})
				}
			}
		}
	}

	// Insight: Orphaned entities (no relationships)
	connectedEntities := make(map[string]bool)
	for _, r := range twin.Relationships {
		connectedEntities[r.SourceURI] = true
		connectedEntities[r.TargetURI] = true
	}
	orphanedCount := 0
	for _, e := range twin.Entities {
		if !connectedEntities[e.URI] {
			orphanedCount++
		}
	}
	if orphanedCount > 0 && float64(orphanedCount)/float64(len(twin.Entities)) > 0.1 {
		insights = append(insights, Insight{
			ID:          "orphaned_entities",
			Type:        "warning",
			Title:       "Disconnected Entities Detected",
			Description: fmt.Sprintf("%d entities (%.0f%%) have no relationships. They may be missing connections or could be cleaned up.", orphanedCount, float64(orphanedCount)/float64(len(twin.Entities))*100),
			Severity:    "low",
			Confidence:  0.8,
			CreatedAt:   time.Now(),
		})
	}

	return insights
}

// generateStateInsights creates insights from entity states
func (is *InsightSuggester) generateStateInsights(twin *DigitalTwin) []Insight {
	insights := []Insight{}

	// Track state metrics
	var totalUtil, totalCap float64
	unavailableCount := 0
	degradedCount := 0
	overloadedCount := 0
	highUtilEntities := []string{}

	for _, entity := range twin.Entities {
		totalUtil += entity.State.Utilization
		totalCap += entity.State.Capacity

		if !entity.State.Available {
			unavailableCount++
		}
		if entity.State.Status == "degraded" {
			degradedCount++
		}
		if entity.State.Utilization > 0.8 {
			overloadedCount++
			highUtilEntities = append(highUtilEntities, entity.Label)
		}
	}

	avgUtil := totalUtil / float64(len(twin.Entities))

	// Insight: High system utilization
	if avgUtil > 0.7 {
		severity := "medium"
		if avgUtil > 0.85 {
			severity = "high"
		}
		insights = append(insights, Insight{
			ID:          "high_utilization",
			Type:        "warning",
			Title:       "High System Utilization",
			Description: fmt.Sprintf("Average system utilization is %.0f%%. Consider capacity expansion.", avgUtil*100),
			Severity:    severity,
			Confidence:  0.95,
			CreatedAt:   time.Now(),
			Actions: []Action{
				{
					Type:        "simulate",
					Label:       "Test Demand Surge",
					Description: "See what happens if demand increases further",
					Parameters:  map[string]interface{}{"event": "demand.surge", "increase_percent": 20},
				},
			},
		})
	}

	// Insight: Entities at capacity
	if overloadedCount > 0 {
		insights = append(insights, Insight{
			ID:          "overloaded_entities",
			Type:        "risk",
			Title:       fmt.Sprintf("%d Entities Near/Over Capacity", overloadedCount),
			Description: fmt.Sprintf("High-utilization entities: %s. They may become bottlenecks.", strings.Join(highUtilEntities[:minInt(3, len(highUtilEntities))], ", ")),
			Severity:    "high",
			Entities:    highUtilEntities,
			Confidence:  0.9,
			CreatedAt:   time.Now(),
		})
	}

	// Insight: Degraded entities
	if degradedCount > 0 {
		insights = append(insights, Insight{
			ID:          "degraded_entities",
			Type:        "warning",
			Title:       fmt.Sprintf("%d Entities in Degraded State", degradedCount),
			Description: "Some entities are not operating at full capacity. Investigate and resolve.",
			Severity:    "medium",
			Confidence:  0.95,
			CreatedAt:   time.Now(),
		})
	}

	// Insight: Unavailable entities
	if unavailableCount > 0 {
		insights = append(insights, Insight{
			ID:          "unavailable_entities",
			Type:        "risk",
			Title:       fmt.Sprintf("%d Entities Currently Unavailable", unavailableCount),
			Description: "Some entities are not available. This may impact dependent systems.",
			Severity:    "critical",
			Confidence:  1.0,
			CreatedAt:   time.Now(),
		})
	}

	return insights
}

// generateSuggestedQuestions creates relevant what-if questions
func (is *InsightSuggester) generateSuggestedQuestions(twin *DigitalTwin, analysis *OntologyAnalysis) []SuggestedQuestion {
	questions := []SuggestedQuestion{}

	// Generate domain-specific questions
	switch analysis.DomainType {
	case "nonprofit":
		questions = append(questions,
			SuggestedQuestion{
				Question:  "What if our largest donor withdraws funding?",
				Reason:    "Testing dependency on major funding sources",
				Relevance: 0.95,
				Category:  "risk",
			},
			SuggestedQuestion{
				Question:  "What happens if we expand to 2 more regions?",
				Reason:    "Planning for growth and capacity needs",
				Relevance: 0.85,
				Category:  "capacity",
			},
			SuggestedQuestion{
				Question:  "How would a 30% increase in beneficiaries affect our programs?",
				Reason:    "Testing scalability of current operations",
				Relevance: 0.9,
				Category:  "capacity",
			},
		)
	case "supply_chain":
		questions = append(questions,
			SuggestedQuestion{
				Question:  "What if our primary supplier has a 2-week outage?",
				Reason:    "Testing supply chain resilience",
				Relevance: 0.95,
				Category:  "risk",
			},
			SuggestedQuestion{
				Question:  "What happens if shipping costs increase by 40%?",
				Reason:    "Understanding cost sensitivity",
				Relevance: 0.8,
				Category:  "optimization",
			},
		)
	case "healthcare":
		questions = append(questions,
			SuggestedQuestion{
				Question:  "What if we lose 20% of nursing staff?",
				Reason:    "Testing staffing resilience",
				Relevance: 0.95,
				Category:  "risk",
			},
			SuggestedQuestion{
				Question:  "How would a disease outbreak affecting 50% more patients impact us?",
				Reason:    "Pandemic/surge capacity planning",
				Relevance: 0.9,
				Category:  "capacity",
			},
		)
	}

	// Generate questions from identified risks
	for _, risk := range analysis.RiskFactors {
		question := fmt.Sprintf("What happens if %s occurs?", strings.ToLower(risk.Description[:minInt(50, len(risk.Description))]))
		questions = append(questions, SuggestedQuestion{
			Question:  question,
			Reason:    fmt.Sprintf("Based on identified risk: %s", risk.Name),
			Relevance: 0.85,
			Category:  "risk",
			RelatedTo: risk.Entities,
		})
	}

	// Generate questions from critical entities
	for _, critical := range analysis.CriticalEntities {
		shortName := is.shortName(critical)
		questions = append(questions,
			SuggestedQuestion{
				Question:  fmt.Sprintf("What if %s capacity is reduced by 50%%?", shortName),
				Reason:    fmt.Sprintf("%s is a critical entity type", shortName),
				Relevance: 0.8,
				Category:  "dependency",
				RelatedTo: []string{critical},
			},
		)
	}

	// Sort by relevance
	sort.Slice(questions, func(i, j int) bool {
		return questions[i].Relevance > questions[j].Relevance
	})

	// Limit to top 10
	if len(questions) > 10 {
		questions = questions[:10]
	}

	return questions
}

// generateAIInsights uses LLM to generate additional insights
func (is *InsightSuggester) generateAIInsights(ctx context.Context, twin *DigitalTwin, analysis *OntologyAnalysis) ([]Insight, error) {
	prompt := fmt.Sprintf(`You are analyzing a digital twin for a %s organization.

SYSTEM SUMMARY:
- %d entities across %d types
- %d relationships
- Identified risks: %s
- Critical entities: %s

Based on this structure, identify 2-3 additional insights that a human analyst might miss.
Focus on:
- Hidden dependencies
- Potential optimization opportunities
- Unusual patterns
- Future risks

Respond in JSON:
{
  "insights": [
    {
      "type": "risk|opportunity|warning|trend",
      "title": "Short title",
      "description": "Detailed description",
      "severity": "low|medium|high",
      "confidence": 0.8
    }
  ]
}`,
		analysis.DomainType,
		len(twin.Entities),
		len(analysis.EntityPatterns),
		len(twin.Relationships),
		is.formatRiskNames(analysis.RiskFactors),
		strings.Join(analysis.CriticalEntities, ", "),
	)

	resp, err := is.llmClient.Complete(ctx, AI.LLMRequest{
		Messages: []AI.LLMMessage{
			{Role: "system", Content: "You are an expert business analyst identifying actionable insights from complex systems."},
			{Role: "user", Content: prompt},
		},
		Temperature: 0.5,
		MaxTokens:   800,
	})
	if err != nil {
		return nil, err
	}

	return is.parseAIInsights(resp.Content)
}

func (is *InsightSuggester) parseAIInsights(content string) ([]Insight, error) {
	insights := []Insight{}

	// Extract JSON
	start := strings.Index(content, "{")
	end := strings.LastIndex(content, "}")
	if start < 0 || end < 0 {
		return insights, nil
	}

	var parsed struct {
		Insights []struct {
			Type        string  `json:"type"`
			Title       string  `json:"title"`
			Description string  `json:"description"`
			Severity    string  `json:"severity"`
			Confidence  float64 `json:"confidence"`
		} `json:"insights"`
	}

	if err := json.Unmarshal([]byte(content[start:end+1]), &parsed); err != nil {
		return insights, nil
	}

	for _, i := range parsed.Insights {
		insights = append(insights, Insight{
			ID:          fmt.Sprintf("ai_%s", strings.ReplaceAll(strings.ToLower(i.Title[:minInt(20, len(i.Title))]), " ", "_")),
			Type:        i.Type,
			Title:       i.Title,
			Description: i.Description,
			Severity:    i.Severity,
			Confidence:  i.Confidence,
			CreatedAt:   time.Now(),
		})
	}

	return insights, nil
}

// sortInsights sorts insights by severity
func (is *InsightSuggester) sortInsights(report *InsightReport) {
	severityOrder := map[string]int{"critical": 0, "high": 1, "medium": 2, "low": 3}
	sort.Slice(report.Insights, func(i, j int) bool {
		return severityOrder[report.Insights[i].Severity] < severityOrder[report.Insights[j].Severity]
	})
}

// calculateRiskScore calculates overall risk score
func (is *InsightSuggester) calculateRiskScore(insights []Insight) float64 {
	if len(insights) == 0 {
		return 0.0
	}

	severityScores := map[string]float64{"critical": 1.0, "high": 0.7, "medium": 0.4, "low": 0.1}
	totalScore := 0.0

	for _, insight := range insights {
		if insight.Type == "risk" || insight.Type == "warning" {
			score := severityScores[insight.Severity]
			totalScore += score * insight.Confidence
		}
	}

	// Normalize to 0-1
	maxPossible := float64(len(insights)) * 1.0
	normalized := totalScore / maxPossible
	if normalized > 1.0 {
		normalized = 1.0
	}

	return normalized
}

// generateSummary creates a human-readable summary
func (is *InsightSuggester) generateSummary(report *InsightReport, analysis *OntologyAnalysis) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Analysis of %s system. ", analysis.DomainType))

	criticalCount := 0
	highCount := 0
	for _, i := range report.Insights {
		if i.Severity == "critical" {
			criticalCount++
		} else if i.Severity == "high" {
			highCount++
		}
	}

	if criticalCount > 0 {
		sb.WriteString(fmt.Sprintf("⚠️ %d critical issues require immediate attention. ", criticalCount))
	}
	if highCount > 0 {
		sb.WriteString(fmt.Sprintf("%d high-priority items identified. ", highCount))
	}

	if report.RiskScore < 0.3 {
		sb.WriteString("Overall system health is good.")
	} else if report.RiskScore < 0.6 {
		sb.WriteString("Moderate risk level - review suggested questions.")
	} else {
		sb.WriteString("High risk level - take action on critical insights.")
	}

	return sb.String()
}

// Helper functions

func (is *InsightSuggester) shortName(fullType string) string {
	if idx := strings.LastIndex(fullType, "#"); idx >= 0 {
		return fullType[idx+1:]
	}
	if idx := strings.LastIndex(fullType, "/"); idx >= 0 {
		return fullType[idx+1:]
	}
	return fullType
}

func (is *InsightSuggester) formatRiskNames(risks []RiskFactor) string {
	if len(risks) == 0 {
		return "None"
	}
	names := make([]string, len(risks))
	for i, r := range risks {
		names[i] = r.Name
	}
	return strings.Join(names, ", ")
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
