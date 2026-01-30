package DigitalTwin

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"runtime/debug"
	"strings"
	"time"

	"github.com/Mimir-AIP/Mimir-AIP-Go/pipelines/AI"
	"github.com/google/uuid"
)

// WhatIfEngine provides natural language "what-if" analysis
type WhatIfEngine struct {
	llmClient AI.LLMClient
	simEngine *SimulationEngine
	db        *sql.DB // For ML-enhanced simulations
}

// WhatIfQuery represents a natural language question
type WhatIfQuery struct {
	Question   string `json:"question"`
	TwinID     string `json:"twin_id"`
	MaxResults int    `json:"max_results,omitempty"`
}

// WhatIfResponse contains the analysis results
type WhatIfResponse struct {
	Question        string              `json:"question"`
	Interpretation  string              `json:"interpretation"`
	Scenario        *SimulationScenario `json:"scenario"`
	Results         *SimulationRun      `json:"results,omitempty"`
	Summary         string              `json:"summary"`
	KeyFindings     []KeyFinding        `json:"key_findings"`
	Recommendations []string            `json:"recommendations"`
	Confidence      float64             `json:"confidence"`
	ProcessingTime  int64               `json:"processing_time_ms"`
}

// KeyFinding represents an important insight from the analysis
type KeyFinding struct {
	Type        string  `json:"type"`   // "impact", "risk", "opportunity", "warning"
	Entity      string  `json:"entity"` // Which entity is affected
	Description string  `json:"description"`
	Severity    string  `json:"severity,omitempty"`
	Value       float64 `json:"value,omitempty"` // Quantified impact if applicable
}

// NewWhatIfEngine creates a new what-if analysis engine
func NewWhatIfEngine(llmClient AI.LLMClient) *WhatIfEngine {
	return &WhatIfEngine{
		llmClient: llmClient,
	}
}

// NewWhatIfEngineWithDB creates a what-if engine with ML integration
func NewWhatIfEngineWithDB(llmClient AI.LLMClient, db *sql.DB) *WhatIfEngine {
	return &WhatIfEngine{
		llmClient: llmClient,
		db:        db,
	}
}

// AnalyzeQuestion takes a natural language question and runs a simulation
func (wie *WhatIfEngine) AnalyzeQuestion(ctx context.Context, query WhatIfQuery, twin *DigitalTwin) (*WhatIfResponse, error) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("[WHATIF PANIC] %v\n", r)
			fmt.Printf("[WHATIF PANIC] Stack: %s\n", debug.Stack())
		}
	}()

	startTime := time.Now()

	// Validate inputs
	if twin == nil {
		return nil, fmt.Errorf("twin cannot be nil")
	}
	if wie.llmClient == nil {
		return nil, fmt.Errorf("llmClient cannot be nil")
	}
	if len(twin.Entities) == 0 {
		return nil, fmt.Errorf("twin has no entities - cannot run what-if analysis")
	}

	response := &WhatIfResponse{
		Question:    query.Question,
		KeyFindings: []KeyFinding{},
		Confidence:  0.5,
	}

	// Step 1: Interpret the question using LLM
	interpretation, scenario, err := wie.interpretQuestion(ctx, query.Question, twin)
	if err != nil {
		return nil, fmt.Errorf("failed to interpret question: %w", err)
	}
	response.Interpretation = interpretation
	response.Scenario = scenario

	// Step 2: Run the simulation (with ML if available)
	if wie.db != nil {
		wie.simEngine = NewSimulationEngineWithML(twin, wie.db)
	} else {
		wie.simEngine = NewSimulationEngine(twin)
	}
	if wie.simEngine == nil {
		return nil, fmt.Errorf("failed to create simulation engine")
	}
	wie.simEngine.SetMaxSteps(100)
	wie.simEngine.SetSnapshotInterval(5)

	run, err := wie.simEngine.RunSimulation(scenario)
	if err != nil {
		return nil, fmt.Errorf("simulation failed: %w", err)
	}
	response.Results = run

	// Step 3: Analyze results and generate insights
	findings, summary := wie.analyzeResults(ctx, query.Question, twin, run)
	response.KeyFindings = findings
	response.Summary = summary

	// Step 4: Generate recommendations
	response.Recommendations = wie.generateRecommendations(ctx, query.Question, twin, run, findings)

	// Calculate confidence based on data quality
	response.Confidence = wie.calculateConfidence(twin, run)
	response.ProcessingTime = time.Since(startTime).Milliseconds()

	return response, nil
}

// interpretQuestion uses LLM to understand the question and create a scenario
func (wie *WhatIfEngine) interpretQuestion(ctx context.Context, question string, twin *DigitalTwin) (string, *SimulationScenario, error) {
	// Build entity context
	entityContext := wie.buildEntityContext(twin)

	prompt := fmt.Sprintf(`You are analyzing a "what-if" question for a digital twin simulation.

QUESTION: "%s"

AVAILABLE ENTITIES IN THE SYSTEM:
%s

AVAILABLE EVENT TYPES:
- entity.unavailable: Makes an entity unavailable/failed
- resource.decrease: Reduces capacity/resources (specify decrease_percent)
- resource.increase: Increases capacity/resources (specify increase_percent)  
- demand.surge: Increases utilization/demand (specify increase_percent)
- demand.drop: Decreases demand (specify decrease_percent)
- external.disruption: External factor causes disruption
- cost.increase: Costs go up (specify increase_percent)
- staff.shortage: Personnel reduction (specify shortage_percent)

Based on the question, identify:
1. What the user wants to simulate
2. Which entity/entities are involved
3. What type of event should occur
4. What parameters to use

Respond in JSON:
{
  "interpretation": "Plain English explanation of what will be simulated",
  "target_entity_keyword": "keyword to identify target entity",
  "event_type": "one of the event types above",
  "parameters": {
    "key": "value",
    "change_percent": 30
  },
  "timestamp": 5,
  "duration": 40,
  "severity": "low|medium|high|critical"
}`, question, entityContext)

	resp, err := wie.llmClient.Complete(ctx, AI.LLMRequest{
		Messages: []AI.LLMMessage{
			{Role: "system", Content: "You are an expert at understanding business scenario questions and converting them to simulation parameters. Be precise and practical."},
			{Role: "user", Content: prompt},
		},
		Temperature: 0.3,
		MaxTokens:   800,
	})
	if err != nil {
		// Fallback to rule-based interpretation
		return wie.ruleBasedInterpretation(question, twin)
	}

	// Parse LLM response
	return wie.parseLLMInterpretation(resp.Content, question, twin)
}

// ruleBasedInterpretation provides fallback when LLM is unavailable
func (wie *WhatIfEngine) ruleBasedInterpretation(question string, twin *DigitalTwin) (string, *SimulationScenario, error) {
	questionLower := strings.ToLower(question)

	// Pattern matching for common questions
	var eventType string
	var targetKeywords []string
	var changePercent int = 30
	var interpretation string

	switch {
	case strings.Contains(questionLower, "lose") || strings.Contains(questionLower, "loss") || strings.Contains(questionLower, "stops"):
		eventType = "entity.unavailable"
		if strings.Contains(questionLower, "donor") || strings.Contains(questionLower, "funding") {
			targetKeywords = []string{"donor", "funder", "funding"}
			interpretation = "Simulating loss of a funding source"
		} else if strings.Contains(questionLower, "supplier") {
			targetKeywords = []string{"supplier", "vendor"}
			interpretation = "Simulating supplier becoming unavailable"
		} else if strings.Contains(questionLower, "staff") || strings.Contains(questionLower, "employee") {
			targetKeywords = []string{"staff", "employee", "worker"}
			interpretation = "Simulating staff unavailability"
		} else {
			targetKeywords = wie.extractKeywords(question)
			interpretation = "Simulating entity becoming unavailable"
		}

	case strings.Contains(questionLower, "decrease") || strings.Contains(questionLower, "reduce") || strings.Contains(questionLower, "cut"):
		eventType = "resource.decrease"
		targetKeywords = wie.extractKeywords(question)
		changePercent = wie.extractPercentage(question, 30)
		interpretation = fmt.Sprintf("Simulating %d%% decrease in resources", changePercent)

	case strings.Contains(questionLower, "increase") || strings.Contains(questionLower, "surge") || strings.Contains(questionLower, "grow"):
		if strings.Contains(questionLower, "demand") || strings.Contains(questionLower, "need") {
			eventType = "demand.surge"
		} else if strings.Contains(questionLower, "cost") || strings.Contains(questionLower, "price") {
			eventType = "cost.increase"
		} else {
			eventType = "demand.surge"
		}
		targetKeywords = wie.extractKeywords(question)
		changePercent = wie.extractPercentage(question, 50)
		interpretation = fmt.Sprintf("Simulating %d%% increase", changePercent)

	case strings.Contains(questionLower, "shortage"):
		eventType = "resource.decrease"
		targetKeywords = wie.extractKeywords(question)
		changePercent = 40
		interpretation = "Simulating shortage conditions"

	case strings.Contains(questionLower, "fail") || strings.Contains(questionLower, "crash") || strings.Contains(questionLower, "break"):
		eventType = "entity.unavailable"
		targetKeywords = wie.extractKeywords(question)
		interpretation = "Simulating system failure"

	default:
		eventType = "entity.unavailable"
		targetKeywords = wie.extractKeywords(question)
		interpretation = "Simulating disruption scenario based on question"
	}

	// Find target entity
	var targetEntity *TwinEntity
	for _, e := range twin.Entities {
		text := strings.ToLower(e.Type + " " + e.Label)
		for _, kw := range targetKeywords {
			if strings.Contains(text, strings.ToLower(kw)) {
				targetEntity = &e
				break
			}
		}
		if targetEntity != nil {
			break
		}
	}

	// If no entity found, use first entity
	if targetEntity == nil && len(twin.Entities) > 0 {
		targetEntity = &twin.Entities[0]
		interpretation += fmt.Sprintf(" (targeting %s as no specific entity identified)", targetEntity.Label)
	}

	if targetEntity == nil {
		return "", nil, fmt.Errorf("no entities available for simulation")
	}

	// Create scenario
	scenario := &SimulationScenario{
		ID:          fmt.Sprintf("whatif_%s", uuid.New().String()[:8]),
		TwinID:      twin.ID,
		Name:        "What-If Analysis",
		Description: question,
		Type:        "whatif",
		Duration:    40,
		Events: []SimulationEvent{
			{
				ID:        uuid.New().String(),
				Type:      eventType,
				TargetURI: targetEntity.URI,
				Timestamp: 5,
				Parameters: map[string]interface{}{
					"change_percent": changePercent,
				},
				Impact: EventImpact{
					StateChanges: wie.inferStateChanges(eventType, changePercent),
					Severity:     wie.inferSeverity(changePercent),
				},
			},
		},
		CreatedAt: time.Now(),
	}

	return interpretation, scenario, nil
}

// parseLLMInterpretation parses the LLM response into a scenario
func (wie *WhatIfEngine) parseLLMInterpretation(llmResponse, question string, twin *DigitalTwin) (string, *SimulationScenario, error) {
	// Extract JSON
	jsonStart := strings.Index(llmResponse, "{")
	jsonEnd := strings.LastIndex(llmResponse, "}")
	if jsonStart < 0 || jsonEnd < 0 {
		return wie.ruleBasedInterpretation(question, twin)
	}

	jsonStr := llmResponse[jsonStart : jsonEnd+1]

	var parsed struct {
		Interpretation string                 `json:"interpretation"`
		TargetKeyword  string                 `json:"target_entity_keyword"`
		EventType      string                 `json:"event_type"`
		Parameters     map[string]interface{} `json:"parameters"`
		Timestamp      int                    `json:"timestamp"`
		Duration       int                    `json:"duration"`
		Severity       string                 `json:"severity"`
	}

	if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
		return wie.ruleBasedInterpretation(question, twin)
	}

	// Find target entity
	var targetEntity *TwinEntity
	for _, e := range twin.Entities {
		text := strings.ToLower(e.Type + " " + e.Label)
		if strings.Contains(text, strings.ToLower(parsed.TargetKeyword)) {
			targetEntity = &e
			break
		}
	}

	if targetEntity == nil && len(twin.Entities) > 0 {
		targetEntity = &twin.Entities[0]
	}

	if targetEntity == nil {
		return "", nil, fmt.Errorf("no entities available for simulation")
	}

	// Extract change percent for state changes
	changePercent := 30
	if cp, ok := parsed.Parameters["change_percent"].(float64); ok {
		changePercent = int(cp)
	} else if cp, ok := parsed.Parameters["decrease_percent"].(float64); ok {
		changePercent = int(cp)
	} else if cp, ok := parsed.Parameters["increase_percent"].(float64); ok {
		changePercent = int(cp)
	}

	duration := parsed.Duration
	if duration == 0 {
		duration = 40
	}

	scenario := &SimulationScenario{
		ID:          fmt.Sprintf("whatif_%s", uuid.New().String()[:8]),
		TwinID:      twin.ID,
		Name:        "What-If Analysis",
		Description: question,
		Type:        "whatif",
		Duration:    duration,
		Events: []SimulationEvent{
			{
				ID:         uuid.New().String(),
				Type:       parsed.EventType,
				TargetURI:  targetEntity.URI,
				Timestamp:  parsed.Timestamp,
				Parameters: parsed.Parameters,
				Impact: EventImpact{
					StateChanges: wie.inferStateChanges(parsed.EventType, changePercent),
					Severity:     parsed.Severity,
				},
			},
		},
		CreatedAt: time.Now(),
	}

	return parsed.Interpretation, scenario, nil
}

// analyzeResults interprets simulation results into findings
func (wie *WhatIfEngine) analyzeResults(ctx context.Context, question string, twin *DigitalTwin, run *SimulationRun) ([]KeyFinding, string) {
	findings := []KeyFinding{}

	// Analyze metrics
	if run.Metrics.AverageUtilization > 0.8 {
		findings = append(findings, KeyFinding{
			Type:        "warning",
			Description: fmt.Sprintf("System utilization reached %.0f%% - operating near capacity", run.Metrics.AverageUtilization*100),
			Severity:    "high",
			Value:       run.Metrics.AverageUtilization,
		})
	}

	if run.Metrics.PeakUtilization > 1.0 {
		findings = append(findings, KeyFinding{
			Type:        "risk",
			Description: fmt.Sprintf("Peak demand exceeded capacity by %.0f%%", (run.Metrics.PeakUtilization-1.0)*100),
			Severity:    "critical",
			Value:       run.Metrics.PeakUtilization,
		})
	}

	if run.Metrics.SystemStability < 0.7 {
		findings = append(findings, KeyFinding{
			Type:        "risk",
			Description: fmt.Sprintf("System stability dropped to %.0f%% - significant instability detected", run.Metrics.SystemStability*100),
			Severity:    "high",
			Value:       run.Metrics.SystemStability,
		})
	}

	// Analyze event log for propagation
	affectedEntities := make(map[string]bool)
	for _, entry := range run.EventsLog {
		if entry.Success {
			affectedEntities[entry.TargetURI] = true
			for _, propagated := range entry.PropagatedTo {
				affectedEntities[propagated] = true
			}
		}
	}

	if len(affectedEntities) > 1 {
		findings = append(findings, KeyFinding{
			Type:        "impact",
			Description: fmt.Sprintf("Impact propagated to %d entities across the system", len(affectedEntities)),
			Value:       float64(len(affectedEntities)),
		})
	}

	// Generate summary
	summary := wie.generateSummary(question, run, findings)

	return findings, summary
}

// generateSummary creates a human-readable summary
func (wie *WhatIfEngine) generateSummary(question string, run *SimulationRun, findings []KeyFinding) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Simulation completed in %d steps. ", run.Metrics.TotalSteps))

	if run.Metrics.SystemStability >= 0.9 {
		sb.WriteString("The system remained stable throughout the scenario. ")
	} else if run.Metrics.SystemStability >= 0.7 {
		sb.WriteString("The system experienced some instability but recovered. ")
	} else {
		sb.WriteString("The system experienced significant instability. ")
	}

	if run.Metrics.EventsProcessed > 0 {
		sb.WriteString(fmt.Sprintf("%d events were processed. ", run.Metrics.EventsProcessed))
	}

	// Add key insight
	hasHighRisk := false
	for _, f := range findings {
		if f.Severity == "critical" || f.Severity == "high" {
			hasHighRisk = true
			break
		}
	}

	if hasHighRisk {
		sb.WriteString("⚠️ This scenario reveals significant risks that should be addressed.")
	} else {
		sb.WriteString("✅ The system can handle this scenario with current resources.")
	}

	return sb.String()
}

// generateRecommendations creates actionable recommendations
func (wie *WhatIfEngine) generateRecommendations(ctx context.Context, question string, twin *DigitalTwin, run *SimulationRun, findings []KeyFinding) []string {
	recommendations := []string{}

	// Generate recommendations based on findings
	for _, f := range findings {
		switch f.Type {
		case "risk":
			if strings.Contains(f.Description, "capacity") || strings.Contains(f.Description, "exceeded") {
				recommendations = append(recommendations, "Consider increasing capacity or implementing load balancing")
			}
			if strings.Contains(f.Description, "stability") {
				recommendations = append(recommendations, "Review dependencies and add redundancy for critical components")
			}

		case "warning":
			if strings.Contains(f.Description, "utilization") {
				recommendations = append(recommendations, "Monitor utilization levels and plan for capacity expansion")
			}

		case "impact":
			if f.Value > 5 {
				recommendations = append(recommendations, "The impact spreads widely - consider isolating critical systems")
			}
		}
	}

	// Add general recommendations based on metrics
	if run.Metrics.SystemStability < 0.8 {
		recommendations = append(recommendations, "Create contingency plans for this scenario")
	}

	if len(recommendations) == 0 {
		recommendations = append(recommendations, "Current setup appears resilient to this scenario - maintain monitoring")
	}

	return recommendations
}

// calculateConfidence determines how reliable the analysis is
func (wie *WhatIfEngine) calculateConfidence(twin *DigitalTwin, run *SimulationRun) float64 {
	confidence := 0.5

	// More entities = better model
	if len(twin.Entities) > 10 {
		confidence += 0.1
	}
	if len(twin.Entities) > 50 {
		confidence += 0.1
	}

	// More relationships = better understanding of dependencies
	if len(twin.Relationships) > len(twin.Entities) {
		confidence += 0.1
	}

	// Successful simulation increases confidence
	if run.Status == "completed" && run.Error == "" {
		confidence += 0.1
	}

	if confidence > 0.95 {
		confidence = 0.95
	}

	return confidence
}

// Helper functions

func (wie *WhatIfEngine) buildEntityContext(twin *DigitalTwin) string {
	if twin == nil || len(twin.Entities) == 0 {
		return "No entities available"
	}

	// Group by type
	typeCounts := make(map[string]int)
	typeExamples := make(map[string][]string)

	for _, e := range twin.Entities {
		shortType := wie.shortTypeName(e.Type)
		typeCounts[shortType]++
		if len(typeExamples[shortType]) < 3 {
			typeExamples[shortType] = append(typeExamples[shortType], e.Label)
		}
	}

	var sb strings.Builder
	for typ, count := range typeCounts {
		examples := strings.Join(typeExamples[typ], ", ")
		sb.WriteString(fmt.Sprintf("- %s (%d): e.g., %s\n", typ, count, examples))
	}

	return sb.String()
}

func (wie *WhatIfEngine) shortTypeName(fullType string) string {
	if idx := strings.LastIndex(fullType, "#"); idx >= 0 {
		return fullType[idx+1:]
	}
	if idx := strings.LastIndex(fullType, "/"); idx >= 0 {
		return fullType[idx+1:]
	}
	return fullType
}

func (wie *WhatIfEngine) extractKeywords(question string) []string {
	// Common words to skip
	stopWords := map[string]bool{
		"what": true, "if": true, "the": true, "a": true, "an": true,
		"is": true, "are": true, "was": true, "were": true, "be": true,
		"been": true, "being": true, "have": true, "has": true, "had": true,
		"do": true, "does": true, "did": true, "will": true, "would": true,
		"could": true, "should": true, "may": true, "might": true, "must": true,
		"shall": true, "can": true, "need": true, "our": true, "we": true,
		"they": true, "their": true, "there": true, "this": true, "that": true,
		"by": true, "to": true, "of": true, "for": true, "with": true,
		"on": true, "at": true, "from": true, "in": true, "into": true,
		"and": true, "or": true, "but": true, "not": true, "no": true,
		"happens": true, "happen": true, "when": true, "how": true, "why": true,
	}

	words := strings.Fields(strings.ToLower(question))
	keywords := []string{}

	for _, word := range words {
		// Clean punctuation
		word = strings.Trim(word, ".,?!;:'\"")
		if len(word) > 2 && !stopWords[word] {
			keywords = append(keywords, word)
		}
	}

	return keywords
}

func (wie *WhatIfEngine) extractPercentage(question string, defaultVal int) int {
	// Look for patterns like "30%", "by 30", "30 percent"
	words := strings.Fields(question)
	for i, word := range words {
		// Check for N%
		if strings.HasSuffix(word, "%") {
			var val int
			if _, err := fmt.Sscanf(word, "%d%%", &val); err == nil && val > 0 && val <= 100 {
				return val
			}
		}
		// Check for "N percent"
		if i+1 < len(words) && (words[i+1] == "percent" || words[i+1] == "percentage") {
			var val int
			if _, err := fmt.Sscanf(word, "%d", &val); err == nil && val > 0 && val <= 100 {
				return val
			}
		}
	}
	return defaultVal
}

func (wie *WhatIfEngine) inferStateChanges(eventType string, changePercent int) map[string]interface{} {
	change := float64(changePercent) / 100.0

	switch eventType {
	case "entity.unavailable":
		return map[string]interface{}{"available": false, "status": "unavailable"}
	case "resource.decrease":
		return map[string]interface{}{"capacity": 1.0 - change}
	case "resource.increase":
		return map[string]interface{}{"capacity": 1.0 + change}
	case "demand.surge":
		return map[string]interface{}{"utilization": 1.0 + change}
	case "demand.drop":
		return map[string]interface{}{"utilization": 1.0 - change}
	case "external.disruption":
		return map[string]interface{}{"available": false, "status": "disrupted"}
	case "cost.increase":
		return map[string]interface{}{"cost_multiplier": 1.0 + change}
	case "staff.shortage":
		return map[string]interface{}{"capacity": 1.0 - change, "status": "understaffed"}
	default:
		return map[string]interface{}{"status": "affected"}
	}
}

func (wie *WhatIfEngine) inferSeverity(changePercent int) string {
	switch {
	case changePercent >= 80:
		return "critical"
	case changePercent >= 50:
		return "high"
	case changePercent >= 30:
		return "medium"
	default:
		return "low"
	}
}
