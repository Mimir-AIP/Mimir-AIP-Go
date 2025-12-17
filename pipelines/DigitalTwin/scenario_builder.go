package DigitalTwin

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// ScenarioBuilder helps construct simulation scenarios
type ScenarioBuilder struct {
	twin     *DigitalTwin
	scenario *SimulationScenario
}

// NewScenarioBuilder creates a new scenario builder
func NewScenarioBuilder(twin *DigitalTwin, name string) *ScenarioBuilder {
	return &ScenarioBuilder{
		twin: twin,
		scenario: &SimulationScenario{
			ID:        uuid.New().String(),
			TwinID:    twin.ID,
			Name:      name,
			Events:    []SimulationEvent{},
			CreatedAt: time.Now(),
		},
	}
}

// WithDescription sets the scenario description
func (sb *ScenarioBuilder) WithDescription(desc string) *ScenarioBuilder {
	sb.scenario.Description = desc
	return sb
}

// WithType sets the scenario type
func (sb *ScenarioBuilder) WithType(scenarioType string) *ScenarioBuilder {
	sb.scenario.Type = scenarioType
	return sb
}

// WithDuration sets the simulation duration
func (sb *ScenarioBuilder) WithDuration(steps int) *ScenarioBuilder {
	sb.scenario.Duration = steps
	return sb
}

// AddEvent adds a single event to the scenario
func (sb *ScenarioBuilder) AddEvent(event SimulationEvent) *ScenarioBuilder {
	if event.ID == "" {
		event.ID = GenerateEventID(event.Type, event.Timestamp)
	}
	sb.scenario.Events = append(sb.scenario.Events, event)
	return sb
}

// AddEvents adds multiple events to the scenario
func (sb *ScenarioBuilder) AddEvents(events []SimulationEvent) *ScenarioBuilder {
	for _, event := range events {
		sb.AddEvent(event)
	}
	return sb
}

// Build returns the constructed scenario
func (sb *ScenarioBuilder) Build() *SimulationScenario {
	return sb.scenario
}

// Scenario Template Functions

// ResourceUnavailabilityScenario creates a scenario where a resource becomes unavailable
func ResourceUnavailabilityScenario(twin *DigitalTwin, resourceURI string, startStep int, durationSteps int) *SimulationScenario {
	builder := NewScenarioBuilder(twin, "Resource Unavailability")
	builder.WithType("supply_shock").
		WithDescription(fmt.Sprintf("Resource %s becomes unavailable", resourceURI)).
		WithDuration(startStep + durationSteps + 10)

	// Make resource unavailable
	unavailableEvent := CreateEvent(EventResourceUnavailable, resourceURI, startStep, map[string]interface{}{
		"reason":   "planned_maintenance",
		"duration": durationSteps,
	})
	unavailableEvent.WithSeverity(SeverityHigh).
		WithPropagation("depends_on", 0.7, 1).
		WithPropagation("supplies", 0.5, 2)

	builder.AddEvent(*unavailableEvent)

	// Restore resource
	if durationSteps > 0 {
		availableEvent := CreateEvent(EventResourceAvailable, resourceURI, startStep+durationSteps, map[string]interface{}{
			"reason": "maintenance_completed",
		})
		builder.AddEvent(*availableEvent)
	}

	return builder.Build()
}

// DemandSurgeScenario creates a scenario with sudden demand increase
func DemandSurgeScenario(twin *DigitalTwin, targetURIs []string, surgeFactor float64, startStep int, durationSteps int) *SimulationScenario {
	builder := NewScenarioBuilder(twin, "Demand Surge")
	builder.WithType("demand_surge").
		WithDescription(fmt.Sprintf("Demand surge affecting %d entities", len(targetURIs))).
		WithDuration(startStep + durationSteps + 10)

	// Create surge events for each target
	for _, uri := range targetURIs {
		surgeEvent := CreateEvent(EventDemandSurge, uri, startStep, map[string]interface{}{
			"increase_factor": surgeFactor,
			"reason":          "market_conditions",
		})
		surgeEvent.WithSeverity(SeverityHigh).
			WithPropagation("serves", 0.6, 1).
			WithPropagation("depends_on", 0.4, 2)

		builder.AddEvent(*surgeEvent)
	}

	// Normalize demand after duration
	if durationSteps > 0 {
		for _, uri := range targetURIs {
			normalizeEvent := CreateEvent(EventDemandDrop, uri, startStep+durationSteps, map[string]interface{}{
				"decrease_factor": 1.0 / surgeFactor,
				"reason":          "market_normalization",
			})
			builder.AddEvent(*normalizeEvent)
		}
	}

	return builder.Build()
}

// CapacityReductionScenario simulates reduced capacity across resources
func CapacityReductionScenario(twin *DigitalTwin, resourceURIs []string, reductionFactor float64, startStep int) *SimulationScenario {
	builder := NewScenarioBuilder(twin, "Capacity Reduction")
	builder.WithType("resource_constraint").
		WithDescription(fmt.Sprintf("Capacity reduced by %.0f%% for %d resources", (1-reductionFactor)*100, len(resourceURIs))).
		WithDuration(startStep + 50)

	for _, uri := range resourceURIs {
		event := CreateEvent(EventResourceCapacityChange, uri, startStep, map[string]interface{}{
			"multiplier": reductionFactor,
			"reason":     "budget_constraints",
		})
		event.WithSeverity(SeverityMedium).
			WithPropagation("depends_on", 0.5, 1)

		builder.AddEvent(*event)
	}

	return builder.Build()
}

// CascadingFailureScenario creates a scenario where failures cascade through the system
func CascadingFailureScenario(twin *DigitalTwin, initialFailureURI string, startStep int) *SimulationScenario {
	builder := NewScenarioBuilder(twin, "Cascading Failure")
	builder.WithType("system_failure").
		WithDescription(fmt.Sprintf("Failure starts at %s and cascades", initialFailureURI)).
		WithDuration(startStep + 100)

	// Initial failure with high propagation
	failureEvent := CreateEvent(EventProcessFailure, initialFailureURI, startStep, map[string]interface{}{
		"reason": "hardware_failure",
	})
	failureEvent.WithSeverity(SeverityCritical).
		WithPropagation("depends_on", 0.9, 2).
		WithPropagation("serves", 0.7, 3)

	builder.AddEvent(*failureEvent)

	return builder.Build()
}

// SupplyChainDisruptionScenario simulates external supply chain disruption
func SupplyChainDisruptionScenario(twin *DigitalTwin, supplierURIs []string, startStep int, durationSteps int) *SimulationScenario {
	builder := NewScenarioBuilder(twin, "Supply Chain Disruption")
	builder.WithType("supply_shock").
		WithDescription(fmt.Sprintf("Supply chain disruption affecting %d suppliers", len(supplierURIs))).
		WithDuration(startStep + durationSteps + 20)

	for i, uri := range supplierURIs {
		// Stagger disruptions slightly
		disruptionStart := startStep + (i * 2)

		disruptionEvent := CreateEvent(EventExternalSupplyDisruption, uri, disruptionStart, map[string]interface{}{
			"reason":   "geopolitical_event",
			"duration": durationSteps,
		})
		disruptionEvent.WithSeverity(SeverityCritical).
			WithPropagation("supplies", 0.8, 1).
			WithPropagation("depends_on", 0.6, 2)

		builder.AddEvent(*disruptionEvent)

		// Gradual recovery
		if durationSteps > 0 {
			recoveryEvent := CreateEvent(EventResourceAvailable, uri, disruptionStart+durationSteps, map[string]interface{}{
				"reason": "supply_restored",
			})
			builder.AddEvent(*recoveryEvent)
		}
	}

	return builder.Build()
}

// ProcessOptimizationScenario simulates process improvements
func ProcessOptimizationScenario(twin *DigitalTwin, processURIs []string, efficiencyGain float64, startStep int) *SimulationScenario {
	builder := NewScenarioBuilder(twin, "Process Optimization")
	builder.WithType("process_improvement").
		WithDescription(fmt.Sprintf("Process optimization with %.0f%% efficiency gain", efficiencyGain*100)).
		WithDuration(startStep + 100)

	for _, uri := range processURIs {
		event := CreateEvent(EventProcessOptimization, uri, startStep, map[string]interface{}{
			"efficiency_gain": efficiencyGain,
			"reason":          "technology_upgrade",
		})
		event.WithSeverity(SeverityLow)

		builder.AddEvent(*event)
	}

	return builder.Build()
}

// MarketShiftScenario simulates external market changes
func MarketShiftScenario(twin *DigitalTwin, affectedURIs []string, demandImpact float64, startStep int) *SimulationScenario {
	builder := NewScenarioBuilder(twin, "Market Shift")
	builder.WithType("external_event").
		WithDescription(fmt.Sprintf("Market shift with %.2fx demand impact", demandImpact)).
		WithDuration(startStep + 50)

	for _, uri := range affectedURIs {
		event := CreateEvent(EventExternalMarketShift, uri, startStep, map[string]interface{}{
			"demand_impact": demandImpact,
			"reason":        "market_dynamics",
		})
		event.WithSeverity(SeverityMedium).
			WithPropagation("serves", 0.6, 1)

		builder.AddEvent(*event)
	}

	return builder.Build()
}

// PolicyChangeScenario simulates regulatory or policy changes
func PolicyChangeScenario(twin *DigitalTwin, affectedURIs []string, capacityImpact float64, startStep int) *SimulationScenario {
	builder := NewScenarioBuilder(twin, "Policy Change")
	builder.WithType("policy_change").
		WithDescription(fmt.Sprintf("Policy change affecting %d entities", len(affectedURIs))).
		WithDuration(startStep + 100)

	for _, uri := range affectedURIs {
		event := CreateEvent(EventPolicyConstraintAdd, uri, startStep, map[string]interface{}{
			"capacity_impact": capacityImpact,
			"constraint":      "new_regulation",
		})
		event.WithSeverity(SeverityMedium)

		builder.AddEvent(*event)
	}

	return builder.Build()
}

// CompositeScenario creates a complex scenario with multiple event types
func CompositeScenario(twin *DigitalTwin, name string, description string) *ScenarioBuilder {
	builder := NewScenarioBuilder(twin, name)
	builder.WithDescription(description).
		WithType("composite")
	return builder
}

// Healthcare-specific scenario templates

// HealthcarePandemicScenario simulates pandemic impact on healthcare facility
func HealthcarePandemicScenario(twin *DigitalTwin, departmentURIs []string, patientSurgeFactor float64, startStep int, peakDuration int) *SimulationScenario {
	builder := NewScenarioBuilder(twin, "Pandemic Response")
	builder.WithType("healthcare_emergency").
		WithDescription(fmt.Sprintf("Pandemic with %.2fx patient surge", patientSurgeFactor)).
		WithDuration(startStep + peakDuration + 50)

	// Phase 1: Initial surge
	for _, uri := range departmentURIs {
		surgeEvent := CreateEvent(EventDemandSurge, uri, startStep, map[string]interface{}{
			"increase_factor": patientSurgeFactor,
			"reason":          "pandemic_outbreak",
		})
		surgeEvent.WithSeverity(SeverityCritical).
			WithPropagation("serves", 0.8, 1).
			WithPropagation("depends_on", 0.7, 2)

		builder.AddEvent(*surgeEvent)
	}

	// Phase 2: Supply shortages (medicines, PPE)
	// Simulate supply issues starting slightly after surge
	supplyShortageStep := startStep + 5
	for _, uri := range departmentURIs {
		// Reduce capacity due to supply constraints
		event := CreateEvent(EventPolicyConstraintAdd, uri, supplyShortageStep, map[string]interface{}{
			"capacity_impact": 0.7, // 30% capacity reduction
			"constraint":      "ppe_shortage",
		})
		event.WithSeverity(SeverityHigh)
		builder.AddEvent(*event)
	}

	// Phase 3: Gradual recovery
	recoveryStart := startStep + peakDuration
	for _, uri := range departmentURIs {
		normalizeEvent := CreateEvent(EventDemandDrop, uri, recoveryStart, map[string]interface{}{
			"decrease_factor": 1.0 / patientSurgeFactor,
			"reason":          "pandemic_subsiding",
		})
		builder.AddEvent(*normalizeEvent)
	}

	return builder.Build()
}

// Supply Chain-specific scenario templates

// SupplyChainStressTestScenario creates a comprehensive supply chain stress test
func SupplyChainStressTestScenario(twin *DigitalTwin, primarySupplierURI string, backupSupplierURI string, startStep int) *SimulationScenario {
	builder := NewScenarioBuilder(twin, "Supply Chain Stress Test")
	builder.WithType("stress_test").
		WithDescription("Test supply chain resilience with primary supplier failure").
		WithDuration(startStep + 100)

	// Primary supplier fails
	failureEvent := CreateEvent(EventResourceUnavailable, primarySupplierURI, startStep, map[string]interface{}{
		"reason":   "natural_disaster",
		"duration": 50,
	})
	failureEvent.WithSeverity(SeverityCritical).
		WithPropagation("supplies", 0.9, 1)
	builder.AddEvent(*failureEvent)

	// Backup supplier experiences capacity issues
	if backupSupplierURI != "" {
		capacityEvent := CreateEvent(EventResourceCapacityChange, backupSupplierURI, startStep+5, map[string]interface{}{
			"multiplier": 1.5, // Can only provide 150% of normal (not enough to cover both)
			"reason":     "increased_demand",
		})
		capacityEvent.WithSeverity(SeverityHigh)
		builder.AddEvent(*capacityEvent)
	}

	// Primary supplier returns
	recoveryEvent := CreateEvent(EventResourceAvailable, primarySupplierURI, startStep+50, map[string]interface{}{
		"reason": "operations_restored",
	})
	builder.AddEvent(*recoveryEvent)

	return builder.Build()
}

// Financial-specific scenario templates

// FinancialMarketVolatilityScenario simulates market volatility impact
func FinancialMarketVolatilityScenario(twin *DigitalTwin, tradingSystemURIs []string, startStep int) *SimulationScenario {
	builder := NewScenarioBuilder(twin, "Market Volatility")
	builder.WithType("financial_stress").
		WithDescription("High market volatility increasing system load").
		WithDuration(startStep + 50)

	// Multiple waves of high demand
	for wave := 0; wave < 3; wave++ {
		waveStart := startStep + (wave * 15)

		for _, uri := range tradingSystemURIs {
			surgeEvent := CreateEvent(EventDemandSurge, uri, waveStart, map[string]interface{}{
				"increase_factor": 3.0,
				"reason":          "market_volatility",
			})
			surgeEvent.WithSeverity(SeverityHigh).
				WithPropagation("depends_on", 0.8, 1)

			builder.AddEvent(*surgeEvent)

			// Normalize after 5 steps
			normalizeEvent := CreateEvent(EventDemandDrop, uri, waveStart+5, map[string]interface{}{
				"decrease_factor": 0.33,
				"reason":          "market_stabilization",
			})
			builder.AddEvent(*normalizeEvent)
		}
	}

	return builder.Build()
}

// Template Registry

// ScenarioTemplate registry for discoverability
type TemplateRegistry struct {
	templates map[string]*ScenarioTemplate
}

// NewTemplateRegistry creates a new template registry
func NewTemplateRegistry() *TemplateRegistry {
	return &TemplateRegistry{
		templates: make(map[string]*ScenarioTemplate),
	}
}

// RegisterTemplate adds a template to the registry
func (tr *TemplateRegistry) RegisterTemplate(template *ScenarioTemplate) {
	tr.templates[template.Name] = template
}

// GetTemplate retrieves a template by name
func (tr *TemplateRegistry) GetTemplate(name string) (*ScenarioTemplate, bool) {
	template, exists := tr.templates[name]
	return template, exists
}

// ListTemplates returns all available templates
func (tr *TemplateRegistry) ListTemplates() []*ScenarioTemplate {
	templates := make([]*ScenarioTemplate, 0, len(tr.templates))
	for _, template := range tr.templates {
		templates = append(templates, template)
	}
	return templates
}

// ListTemplatesByCategory returns templates filtered by category
func (tr *TemplateRegistry) ListTemplatesByCategory(category string) []*ScenarioTemplate {
	var templates []*ScenarioTemplate
	for _, template := range tr.templates {
		if template.Category == category {
			templates = append(templates, template)
		}
	}
	return templates
}

// GetDefaultTemplateRegistry returns a registry with common templates
func GetDefaultTemplateRegistry() *TemplateRegistry {
	registry := NewTemplateRegistry()

	// Generic templates
	registry.RegisterTemplate(&ScenarioTemplate{
		Name:        "resource_unavailability",
		Description: "Resource becomes unavailable for a period",
		Category:    "generic",
		Parameters: []TemplateParameter{
			{Name: "resource_uri", Type: "entity_uri", Description: "URI of the resource", Required: true},
			{Name: "start_step", Type: "number", Description: "When unavailability starts", Required: true, Default: 10},
			{Name: "duration_steps", Type: "number", Description: "How long resource is unavailable", Required: true, Default: 20},
		},
	})

	registry.RegisterTemplate(&ScenarioTemplate{
		Name:        "demand_surge",
		Description: "Sudden increase in demand",
		Category:    "generic",
		Parameters: []TemplateParameter{
			{Name: "target_uris", Type: "entity_uri[]", Description: "URIs of affected entities", Required: true},
			{Name: "surge_factor", Type: "number", Description: "Demand multiplier", Required: true, Default: 2.0},
			{Name: "start_step", Type: "number", Description: "When surge starts", Required: true, Default: 10},
			{Name: "duration_steps", Type: "number", Description: "How long surge lasts", Required: false, Default: 30},
		},
	})

	// Healthcare templates
	registry.RegisterTemplate(&ScenarioTemplate{
		Name:        "healthcare_pandemic",
		Description: "Pandemic response scenario with patient surge and supply shortages",
		Category:    "healthcare",
		Parameters: []TemplateParameter{
			{Name: "department_uris", Type: "entity_uri[]", Description: "URIs of hospital departments", Required: true},
			{Name: "patient_surge_factor", Type: "number", Description: "Patient surge multiplier", Required: true, Default: 3.0},
			{Name: "start_step", Type: "number", Description: "When pandemic starts", Required: true, Default: 10},
			{Name: "peak_duration", Type: "number", Description: "Duration of peak impact", Required: true, Default: 40},
		},
	})

	// Supply chain templates
	registry.RegisterTemplate(&ScenarioTemplate{
		Name:        "supply_chain_disruption",
		Description: "Supply chain disruption affecting multiple suppliers",
		Category:    "supply_chain",
		Parameters: []TemplateParameter{
			{Name: "supplier_uris", Type: "entity_uri[]", Description: "URIs of suppliers", Required: true},
			{Name: "start_step", Type: "number", Description: "When disruption starts", Required: true, Default: 10},
			{Name: "duration_steps", Type: "number", Description: "Duration of disruption", Required: true, Default: 50},
		},
	})

	return registry
}
