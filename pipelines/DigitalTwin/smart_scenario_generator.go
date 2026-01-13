package DigitalTwin

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/Mimir-AIP/Mimir-AIP-Go/pipelines/AI"
	"github.com/google/uuid"
)

// SmartScenarioGenerator creates domain-relevant scenarios using ontology analysis
type SmartScenarioGenerator struct {
	llmClient AI.LLMClient
	analyzer  *OntologyAnalyzer
}

// SmartScenarioTemplate represents a pre-defined scenario pattern for auto-generation
type SmartScenarioTemplate struct {
	ID           string               `json:"id"`
	Name         string               `json:"name"`
	Description  string               `json:"description"`
	DomainTypes  []string             `json:"domain_types"` // Which domains this applies to
	EventPattern []SmartEventTemplate `json:"event_pattern"`
	Parameters   map[string]string    `json:"parameters"` // Customizable parameters
}

// SmartEventTemplate is a template for creating events in auto-generation
type SmartEventTemplate struct {
	Type           string                 `json:"type"`
	TargetPattern  string                 `json:"target_pattern"`  // e.g., "critical_entity", "resource", "actor"
	TimestampRange [2]int                 `json:"timestamp_range"` // [min, max] step
	Parameters     map[string]interface{} `json:"parameters"`
}

// GeneratedScenario is a scenario ready to be saved
type GeneratedScenario struct {
	Scenario    SimulationScenario `json:"scenario"`
	Explanation string             `json:"explanation"`
	RiskAddress string             `json:"risk_addressed,omitempty"`
	Confidence  float64            `json:"confidence"` // How relevant this scenario is (0-1)
}

// NewSmartScenarioGenerator creates a new generator
func NewSmartScenarioGenerator(llmClient AI.LLMClient) *SmartScenarioGenerator {
	return &SmartScenarioGenerator{
		llmClient: llmClient,
		analyzer:  NewOntologyAnalyzer(llmClient),
	}
}

// GenerateScenariosForTwin creates relevant scenarios based on twin's ontology
func (ssg *SmartScenarioGenerator) GenerateScenariosForTwin(ctx context.Context, twin *DigitalTwin) ([]GeneratedScenario, error) {
	// Analyze the ontology first
	analysis, err := ssg.analyzer.AnalyzeOntology(ctx, twin.Entities, twin.Relationships)
	if err != nil {
		return nil, fmt.Errorf("ontology analysis failed: %w", err)
	}

	scenarios := []GeneratedScenario{}

	// 1. Generate baseline scenario (always)
	baseline := ssg.generateBaselineScenario(twin, analysis)
	scenarios = append(scenarios, baseline)

	// 2. Generate risk-based scenarios
	riskScenarios := ssg.generateRiskScenarios(twin, analysis)
	scenarios = append(scenarios, riskScenarios...)

	// 3. Generate domain-specific scenarios
	domainScenarios := ssg.generateDomainScenarios(twin, analysis)
	scenarios = append(scenarios, domainScenarios...)

	// 4. If LLM available, generate custom scenarios
	if ssg.llmClient != nil {
		customScenarios, err := ssg.generateLLMScenarios(ctx, twin, analysis)
		if err == nil {
			scenarios = append(scenarios, customScenarios...)
		}
	}

	return scenarios, nil
}

// generateBaselineScenario creates a no-event baseline for comparison
func (ssg *SmartScenarioGenerator) generateBaselineScenario(twin *DigitalTwin, analysis *OntologyAnalysis) GeneratedScenario {
	return GeneratedScenario{
		Scenario: SimulationScenario{
			ID:          fmt.Sprintf("baseline_%s", twin.ID[:8]),
			TwinID:      twin.ID,
			Name:        "Baseline Operations",
			Description: fmt.Sprintf("Normal %s operations with no disruptions. Use as comparison baseline.", analysis.DomainType),
			Type:        "baseline",
			Events:      []SimulationEvent{},
			Duration:    30,
			CreatedAt:   time.Now(),
		},
		Explanation: "Establishes normal operating conditions for comparison with disruption scenarios",
		Confidence:  1.0,
	}
}

// generateRiskScenarios creates scenarios for each identified risk
func (ssg *SmartScenarioGenerator) generateRiskScenarios(twin *DigitalTwin, analysis *OntologyAnalysis) []GeneratedScenario {
	scenarios := []GeneratedScenario{}

	for _, risk := range analysis.RiskFactors {
		// Find target entities for this risk
		targetEntities := ssg.findEntitiesOfTypes(twin, risk.Entities)
		if len(targetEntities) == 0 {
			continue
		}

		events := []SimulationEvent{}

		switch {
		case strings.Contains(strings.ToLower(risk.Name), "concentration"):
			// Critical entity failure scenario
			events = append(events, SimulationEvent{
				ID:        uuid.New().String(),
				Type:      "entity.unavailable",
				TargetURI: targetEntities[0].URI,
				Timestamp: 5,
				Parameters: map[string]interface{}{
					"reason":   "failure",
					"duration": 20,
				},
				Impact: EventImpact{
					StateChanges: map[string]interface{}{
						"available": false,
						"status":    "failed",
					},
					Severity: risk.Severity,
				},
			})

		case strings.Contains(strings.ToLower(risk.Name), "resource"):
			// Resource shortage scenario
			events = append(events, SimulationEvent{
				ID:        uuid.New().String(),
				Type:      "resource.decrease",
				TargetURI: targetEntities[0].URI,
				Timestamp: 3,
				Parameters: map[string]interface{}{
					"decrease_percent": 50,
					"reason":           "shortage",
				},
				Impact: EventImpact{
					StateChanges: map[string]interface{}{
						"capacity": 0.5,
					},
					Severity: risk.Severity,
				},
			})

		case strings.Contains(strings.ToLower(risk.Name), "dependency"):
			// External dependency failure
			events = append(events, SimulationEvent{
				ID:        uuid.New().String(),
				Type:      "external.disruption",
				TargetURI: targetEntities[0].URI,
				Timestamp: 5,
				Parameters: map[string]interface{}{
					"type":     "withdrawal",
					"recovery": 10,
				},
				Impact: EventImpact{
					StateChanges: map[string]interface{}{
						"available":   false,
						"utilization": 0,
					},
					Severity: risk.Severity,
				},
			})
		}

		if len(events) > 0 {
			scenarios = append(scenarios, GeneratedScenario{
				Scenario: SimulationScenario{
					ID:          fmt.Sprintf("risk_%s_%s", risk.Entities[0][:min(8, len(risk.Entities[0]))], uuid.New().String()[:4]),
					TwinID:      twin.ID,
					Name:        risk.Name,
					Description: risk.Description,
					Type:        "risk_assessment",
					Events:      events,
					Duration:    40,
					CreatedAt:   time.Now(),
				},
				Explanation: fmt.Sprintf("Tests impact of %s. %s", risk.Name, risk.Mitigation),
				RiskAddress: risk.Name,
				Confidence:  0.85,
			})
		}
	}

	return scenarios
}

// generateDomainScenarios creates scenarios specific to the detected domain
func (ssg *SmartScenarioGenerator) generateDomainScenarios(twin *DigitalTwin, analysis *OntologyAnalysis) []GeneratedScenario {
	scenarios := []GeneratedScenario{}

	switch analysis.DomainType {
	case "nonprofit":
		scenarios = append(scenarios, ssg.nonprofitScenarios(twin, analysis)...)
	case "supply_chain":
		scenarios = append(scenarios, ssg.supplyChainScenarios(twin, analysis)...)
	case "healthcare":
		scenarios = append(scenarios, ssg.healthcareScenarios(twin, analysis)...)
	case "retail":
		scenarios = append(scenarios, ssg.retailScenarios(twin, analysis)...)
	default:
		scenarios = append(scenarios, ssg.generalScenarios(twin, analysis)...)
	}

	return scenarios
}

// Domain-specific scenario generators

func (ssg *SmartScenarioGenerator) nonprofitScenarios(twin *DigitalTwin, analysis *OntologyAnalysis) []GeneratedScenario {
	scenarios := []GeneratedScenario{}

	// Find donors and programs
	donors := ssg.findEntitiesByKeyword(twin, []string{"donor", "funder", "grant"})
	programs := ssg.findEntitiesByKeyword(twin, []string{"program", "project", "initiative"})

	if len(donors) > 0 {
		// Major donor loss scenario
		events := []SimulationEvent{
			{
				ID:        uuid.New().String(),
				Type:      "funding.withdrawal",
				TargetURI: donors[0].URI,
				Timestamp: 5,
				Parameters: map[string]interface{}{
					"withdrawal_percent": 100,
					"reason":             "donor_withdrawal",
				},
				Impact: EventImpact{
					StateChanges: map[string]interface{}{
						"available": false,
						"status":    "inactive",
					},
					Severity: "high",
				},
			},
		}

		scenarios = append(scenarios, GeneratedScenario{
			Scenario: SimulationScenario{
				ID:          fmt.Sprintf("donor_loss_%s", uuid.New().String()[:8]),
				TwinID:      twin.ID,
				Name:        "Major Donor Loss",
				Description: fmt.Sprintf("What if %s stops funding? Simulates impact on programs and beneficiaries.", donors[0].Label),
				Type:        "funding_disruption",
				Events:      events,
				Duration:    50,
				CreatedAt:   time.Now(),
			},
			Explanation: "Analyzes cascade effect of losing a major funding source",
			Confidence:  0.9,
		})
	}

	if len(programs) > 0 {
		// Program scaling scenario
		events := []SimulationEvent{
			{
				ID:        uuid.New().String(),
				Type:      "demand.surge",
				TargetURI: programs[0].URI,
				Timestamp: 10,
				Parameters: map[string]interface{}{
					"increase_percent": 50,
					"reason":           "crisis_response",
				},
				Impact: EventImpact{
					StateChanges: map[string]interface{}{
						"utilization": 1.5,
					},
					Severity: "medium",
				},
			},
		}

		scenarios = append(scenarios, GeneratedScenario{
			Scenario: SimulationScenario{
				ID:          fmt.Sprintf("demand_surge_%s", uuid.New().String()[:8]),
				TwinID:      twin.ID,
				Name:        "Crisis Response - Demand Surge",
				Description: "Simulates 50% increase in beneficiary demand due to crisis",
				Type:        "demand_surge",
				Events:      events,
				Duration:    40,
				CreatedAt:   time.Now(),
			},
			Explanation: "Tests capacity to handle sudden increase in service demand",
			Confidence:  0.85,
		})
	}

	return scenarios
}

func (ssg *SmartScenarioGenerator) supplyChainScenarios(twin *DigitalTwin, analysis *OntologyAnalysis) []GeneratedScenario {
	scenarios := []GeneratedScenario{}

	suppliers := ssg.findEntitiesByKeyword(twin, []string{"supplier", "vendor", "source"})
	warehouses := ssg.findEntitiesByKeyword(twin, []string{"warehouse", "inventory", "stock"})

	if len(suppliers) > 0 {
		events := []SimulationEvent{
			{
				ID:        uuid.New().String(),
				Type:      "supply.disruption",
				TargetURI: suppliers[0].URI,
				Timestamp: 5,
				Parameters: map[string]interface{}{
					"duration": 15,
					"reason":   "supplier_failure",
				},
				Impact: EventImpact{
					StateChanges: map[string]interface{}{
						"available": false,
						"capacity":  0,
					},
					Severity: "high",
				},
			},
		}

		scenarios = append(scenarios, GeneratedScenario{
			Scenario: SimulationScenario{
				ID:          fmt.Sprintf("supplier_failure_%s", uuid.New().String()[:8]),
				TwinID:      twin.ID,
				Name:        "Critical Supplier Failure",
				Description: fmt.Sprintf("Simulates %s becoming unavailable for 15 time periods", suppliers[0].Label),
				Type:        "supply_disruption",
				Events:      events,
				Duration:    40,
				CreatedAt:   time.Now(),
			},
			Explanation: "Tests resilience to supplier disruptions",
			Confidence:  0.9,
		})
	}

	if len(warehouses) > 0 {
		events := []SimulationEvent{
			{
				ID:        uuid.New().String(),
				Type:      "capacity.reduction",
				TargetURI: warehouses[0].URI,
				Timestamp: 3,
				Parameters: map[string]interface{}{
					"reduction_percent": 40,
					"reason":            "damage",
				},
				Impact: EventImpact{
					StateChanges: map[string]interface{}{
						"capacity": 0.6,
					},
					Severity: "high",
				},
			},
		}

		scenarios = append(scenarios, GeneratedScenario{
			Scenario: SimulationScenario{
				ID:          fmt.Sprintf("warehouse_damage_%s", uuid.New().String()[:8]),
				TwinID:      twin.ID,
				Name:        "Warehouse Capacity Loss",
				Description: "Simulates 40% capacity reduction in primary warehouse",
				Type:        "infrastructure_damage",
				Events:      events,
				Duration:    30,
				CreatedAt:   time.Now(),
			},
			Explanation: "Tests ability to maintain operations with reduced storage capacity",
			Confidence:  0.85,
		})
	}

	return scenarios
}

func (ssg *SmartScenarioGenerator) healthcareScenarios(twin *DigitalTwin, analysis *OntologyAnalysis) []GeneratedScenario {
	scenarios := []GeneratedScenario{}

	staff := ssg.findEntitiesByKeyword(twin, []string{"doctor", "nurse", "staff", "physician"})
	facilities := ssg.findEntitiesByKeyword(twin, []string{"hospital", "clinic", "ward", "bed"})

	if len(staff) > 0 {
		events := []SimulationEvent{
			{
				ID:        uuid.New().String(),
				Type:      "staff.shortage",
				TargetURI: staff[0].URI,
				Timestamp: 5,
				Parameters: map[string]interface{}{
					"shortage_percent": 30,
					"reason":           "illness",
				},
				Impact: EventImpact{
					StateChanges: map[string]interface{}{
						"capacity": 0.7,
					},
					Severity: "high",
				},
			},
		}

		scenarios = append(scenarios, GeneratedScenario{
			Scenario: SimulationScenario{
				ID:          fmt.Sprintf("staff_shortage_%s", uuid.New().String()[:8]),
				TwinID:      twin.ID,
				Name:        "Staff Shortage Crisis",
				Description: "Simulates 30% reduction in available medical staff",
				Type:        "staffing_crisis",
				Events:      events,
				Duration:    40,
				CreatedAt:   time.Now(),
			},
			Explanation: "Tests patient care capacity with reduced staffing",
			Confidence:  0.9,
		})
	}

	if len(facilities) > 0 {
		events := []SimulationEvent{
			{
				ID:        uuid.New().String(),
				Type:      "demand.surge",
				TargetURI: facilities[0].URI,
				Timestamp: 3,
				Parameters: map[string]interface{}{
					"increase_percent": 80,
					"reason":           "outbreak",
				},
				Impact: EventImpact{
					StateChanges: map[string]interface{}{
						"utilization": 1.8,
					},
					Severity: "critical",
				},
			},
		}

		scenarios = append(scenarios, GeneratedScenario{
			Scenario: SimulationScenario{
				ID:          fmt.Sprintf("patient_surge_%s", uuid.New().String()[:8]),
				TwinID:      twin.ID,
				Name:        "Patient Surge - Outbreak Response",
				Description: "Simulates 80% increase in patient admissions",
				Type:        "demand_surge",
				Events:      events,
				Duration:    50,
				CreatedAt:   time.Now(),
			},
			Explanation: "Tests facility capacity during health crisis",
			Confidence:  0.85,
		})
	}

	return scenarios
}

func (ssg *SmartScenarioGenerator) retailScenarios(twin *DigitalTwin, analysis *OntologyAnalysis) []GeneratedScenario {
	scenarios := []GeneratedScenario{}

	stores := ssg.findEntitiesByKeyword(twin, []string{"store", "outlet", "location"})
	products := ssg.findEntitiesByKeyword(twin, []string{"product", "item", "sku"})

	if len(stores) > 0 && len(products) > 0 {
		events := []SimulationEvent{
			{
				ID:        uuid.New().String(),
				Type:      "demand.surge",
				TargetURI: products[0].URI,
				Timestamp: 5,
				Parameters: map[string]interface{}{
					"increase_percent": 200,
					"reason":           "viral_trend",
				},
				Impact: EventImpact{
					StateChanges: map[string]interface{}{
						"utilization": 3.0,
					},
					Severity: "medium",
				},
			},
		}

		scenarios = append(scenarios, GeneratedScenario{
			Scenario: SimulationScenario{
				ID:          fmt.Sprintf("viral_demand_%s", uuid.New().String()[:8]),
				TwinID:      twin.ID,
				Name:        "Viral Product Demand",
				Description: "Simulates sudden 200% demand increase for a product",
				Type:        "demand_spike",
				Events:      events,
				Duration:    30,
				CreatedAt:   time.Now(),
			},
			Explanation: "Tests inventory and supply chain response to viral demand",
			Confidence:  0.8,
		})
	}

	return scenarios
}

func (ssg *SmartScenarioGenerator) generalScenarios(twin *DigitalTwin, analysis *OntologyAnalysis) []GeneratedScenario {
	scenarios := []GeneratedScenario{}

	// Pick critical entities for general scenarios
	if len(analysis.CriticalEntities) > 0 {
		critical := ssg.findEntitiesOfTypes(twin, analysis.CriticalEntities)
		if len(critical) > 0 {
			events := []SimulationEvent{
				{
					ID:        uuid.New().String(),
					Type:      "entity.degraded",
					TargetURI: critical[0].URI,
					Timestamp: 10,
					Parameters: map[string]interface{}{
						"degradation_percent": 50,
					},
					Impact: EventImpact{
						StateChanges: map[string]interface{}{
							"capacity":    0.5,
							"utilization": 1.5,
							"status":      "degraded",
						},
						Severity: "high",
					},
				},
			}

			scenarios = append(scenarios, GeneratedScenario{
				Scenario: SimulationScenario{
					ID:          fmt.Sprintf("critical_degrade_%s", uuid.New().String()[:8]),
					TwinID:      twin.ID,
					Name:        fmt.Sprintf("Critical Entity Degradation: %s", critical[0].Label),
					Description: "Simulates 50% capacity degradation of a critical system component",
					Type:        "degradation",
					Events:      events,
					Duration:    40,
					CreatedAt:   time.Now(),
				},
				Explanation: "Tests system resilience when critical component operates at reduced capacity",
				Confidence:  0.75,
			})
		}
	}

	return scenarios
}

// generateLLMScenarios uses AI to create custom scenarios
func (ssg *SmartScenarioGenerator) generateLLMScenarios(ctx context.Context, twin *DigitalTwin, analysis *OntologyAnalysis) ([]GeneratedScenario, error) {
	// Build context for LLM
	entitySummary := ssg.buildEntitySummary(twin)

	prompt := fmt.Sprintf(`You are creating simulation scenarios for a digital twin of a %s organization.

SYSTEM OVERVIEW:
%s

IDENTIFIED RISKS:
%s

CURRENT SCENARIOS ALREADY EXIST FOR:
- Baseline operations
- Risk-based scenarios for critical dependencies
- Domain-specific standard scenarios

Please suggest 1-2 ADDITIONAL creative "what-if" scenarios that would provide unique insights.
These should be realistic, relevant scenarios that haven't been covered yet.

Respond in JSON:
{
  "scenarios": [
    {
      "name": "Scenario Name",
      "description": "What this scenario simulates",
      "type": "scenario_type",
      "target_entity_keyword": "keyword to find target entity (e.g., 'staff', 'budget')",
      "event_type": "entity.unavailable | resource.decrease | demand.surge | external.disruption",
      "change_percent": 30,
      "timestamp": 10,
      "explanation": "Why this scenario provides value"
    }
  ]
}`, analysis.DomainType, entitySummary, ssg.formatRisks(analysis.RiskFactors))

	response, err := ssg.llmClient.Complete(ctx, AI.LLMRequest{
		Messages: []AI.LLMMessage{
			{Role: "system", Content: "You are an expert in business simulation and scenario planning. Create practical what-if scenarios."},
			{Role: "user", Content: prompt},
		},
		Temperature: 0.7,
		MaxTokens:   1000,
	})
	if err != nil {
		return nil, err
	}

	// Parse response
	return ssg.parseLLMScenarios(twin, response.Content)
}

// Helper functions

func (ssg *SmartScenarioGenerator) findEntitiesOfTypes(twin *DigitalTwin, types []string) []TwinEntity {
	result := []TwinEntity{}
	for _, e := range twin.Entities {
		for _, t := range types {
			if strings.Contains(strings.ToLower(e.Type), strings.ToLower(t)) {
				result = append(result, e)
				break
			}
		}
	}
	return result
}

func (ssg *SmartScenarioGenerator) findEntitiesByKeyword(twin *DigitalTwin, keywords []string) []TwinEntity {
	result := []TwinEntity{}
	for _, e := range twin.Entities {
		text := strings.ToLower(e.Type + " " + e.Label)
		for _, kw := range keywords {
			if strings.Contains(text, strings.ToLower(kw)) {
				result = append(result, e)
				break
			}
		}
	}
	return result
}

func (ssg *SmartScenarioGenerator) buildEntitySummary(twin *DigitalTwin) string {
	typeCounts := make(map[string]int)
	for _, e := range twin.Entities {
		typeCounts[ssg.analyzer.shortTypeName(e.Type)]++
	}

	var sb strings.Builder
	sb.WriteString("Entities:\n")
	for t, c := range typeCounts {
		sb.WriteString(fmt.Sprintf("- %s: %d\n", t, c))
	}
	sb.WriteString(fmt.Sprintf("\nTotal relationships: %d\n", len(twin.Relationships)))
	return sb.String()
}

func (ssg *SmartScenarioGenerator) formatRisks(risks []RiskFactor) string {
	if len(risks) == 0 {
		return "None identified"
	}
	var sb strings.Builder
	for _, r := range risks {
		sb.WriteString(fmt.Sprintf("- %s (%s): %s\n", r.Name, r.Severity, r.Description))
	}
	return sb.String()
}

func (ssg *SmartScenarioGenerator) parseLLMScenarios(twin *DigitalTwin, llmResponse string) ([]GeneratedScenario, error) {
	// Extract JSON
	jsonStart := strings.Index(llmResponse, "{")
	jsonEnd := strings.LastIndex(llmResponse, "}")
	if jsonStart < 0 || jsonEnd < 0 {
		return []GeneratedScenario{}, nil
	}

	jsonStr := llmResponse[jsonStart : jsonEnd+1]

	var parsed struct {
		Scenarios []struct {
			Name          string `json:"name"`
			Description   string `json:"description"`
			Type          string `json:"type"`
			TargetKeyword string `json:"target_entity_keyword"`
			EventType     string `json:"event_type"`
			ChangePercent int    `json:"change_percent"`
			Timestamp     int    `json:"timestamp"`
			Explanation   string `json:"explanation"`
		} `json:"scenarios"`
	}

	if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
		return []GeneratedScenario{}, nil
	}

	result := []GeneratedScenario{}
	for _, s := range parsed.Scenarios {
		// Find target entity
		targets := ssg.findEntitiesByKeyword(twin, []string{s.TargetKeyword})
		if len(targets) == 0 {
			continue
		}

		events := []SimulationEvent{
			{
				ID:        uuid.New().String(),
				Type:      s.EventType,
				TargetURI: targets[0].URI,
				Timestamp: s.Timestamp,
				Parameters: map[string]interface{}{
					"change_percent": s.ChangePercent,
				},
				Impact: EventImpact{
					StateChanges: ssg.inferStateChanges(s.EventType, s.ChangePercent),
					Severity:     ssg.inferSeverity(s.ChangePercent),
				},
			},
		}

		result = append(result, GeneratedScenario{
			Scenario: SimulationScenario{
				ID:          fmt.Sprintf("llm_%s", uuid.New().String()[:8]),
				TwinID:      twin.ID,
				Name:        s.Name,
				Description: s.Description,
				Type:        s.Type,
				Events:      events,
				Duration:    s.Timestamp + 30,
				CreatedAt:   time.Now(),
			},
			Explanation: s.Explanation,
			Confidence:  0.7, // LLM-generated scenarios have moderate confidence
		})
	}

	return result, nil
}

func (ssg *SmartScenarioGenerator) inferStateChanges(eventType string, changePercent int) map[string]interface{} {
	change := float64(changePercent) / 100.0

	switch eventType {
	case "entity.unavailable":
		return map[string]interface{}{"available": false, "status": "unavailable"}
	case "resource.decrease":
		return map[string]interface{}{"capacity": 1.0 - change}
	case "demand.surge":
		return map[string]interface{}{"utilization": 1.0 + change}
	case "external.disruption":
		return map[string]interface{}{"available": false, "status": "disrupted"}
	default:
		return map[string]interface{}{"capacity": 1.0 - change}
	}
}

func (ssg *SmartScenarioGenerator) inferSeverity(changePercent int) string {
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

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
