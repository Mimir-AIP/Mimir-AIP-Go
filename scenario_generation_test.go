package main

import (
	"fmt"
	"testing"
	"time"

	"github.com/Mimir-AIP/Mimir-AIP-Go/pipelines/DigitalTwin"
)

// TestGenerateDefaultScenarios tests the scenario generation function
func TestGenerateDefaultScenarios(t *testing.T) {
	// Create a mock digital twin with entities and relationships
	twin := &DigitalTwin.DigitalTwin{
		ID:          "test_twin_123",
		OntologyID:  "test_ontology",
		Name:        "Test Twin",
		Description: "Test digital twin for scenario generation",
		ModelType:   "organization",
		BaseState:   make(map[string]interface{}),
		Entities: []DigitalTwin.TwinEntity{
			{
				URI:   "http://example.org/entity1",
				Type:  "http://example.org/Person",
				Label: "Test Entity 1",
				State: DigitalTwin.EntityState{
					Status:      "active",
					Capacity:    100.0,
					Utilization: 0.5,
					Available:   true,
					Metrics:     make(map[string]float64),
					LastUpdated: time.Now(),
				},
			},
			{
				URI:   "http://example.org/entity2",
				Type:  "http://example.org/Department",
				Label: "Test Entity 2",
				State: DigitalTwin.EntityState{
					Status:      "active",
					Capacity:    100.0,
					Utilization: 0.6,
					Available:   true,
					Metrics:     make(map[string]float64),
					LastUpdated: time.Now(),
				},
			},
			{
				URI:   "http://example.org/entity3",
				Type:  "http://example.org/Resource",
				Label: "Test Entity 3",
				State: DigitalTwin.EntityState{
					Status:      "active",
					Capacity:    100.0,
					Utilization: 0.4,
					Available:   true,
					Metrics:     make(map[string]float64),
					LastUpdated: time.Now(),
				},
			},
		},
		Relationships: []DigitalTwin.TwinRelationship{
			{
				ID:         "rel_1",
				SourceURI:  "http://example.org/entity1",
				TargetURI:  "http://example.org/entity2",
				Type:       "worksIn",
				Strength:   0.8,
				Properties: make(map[string]interface{}),
			},
			{
				ID:         "rel_2",
				SourceURI:  "http://example.org/entity2",
				TargetURI:  "http://example.org/entity3",
				Type:       "uses",
				Strength:   0.7,
				Properties: make(map[string]interface{}),
			},
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Generate scenarios
	scenarios := generateDefaultScenarios(twin)

	// Test: Should generate exactly 3 scenarios
	if len(scenarios) != 3 {
		t.Errorf("Expected 3 scenarios, got %d", len(scenarios))
	}

	// Test: Verify baseline scenario
	baselineScenario := scenarios[0]
	if baselineScenario.Type != "baseline" {
		t.Errorf("First scenario should be baseline, got %s", baselineScenario.Type)
	}
	if baselineScenario.Name != "Baseline Operations" {
		t.Errorf("Expected name 'Baseline Operations', got '%s'", baselineScenario.Name)
	}
	if len(baselineScenario.Events) != 0 {
		t.Errorf("Baseline scenario should have 0 events, got %d", len(baselineScenario.Events))
	}
	if baselineScenario.Duration != 30 {
		t.Errorf("Expected duration 30, got %d", baselineScenario.Duration)
	}

	// Test: Verify data quality scenario
	dataQualityScenario := scenarios[1]
	if dataQualityScenario.Type != "data_quality_issue" {
		t.Errorf("Second scenario should be data_quality_issue, got %s", dataQualityScenario.Type)
	}
	if len(dataQualityScenario.Events) == 0 {
		t.Error("Data quality scenario should have events")
	}
	if dataQualityScenario.Duration != 40 {
		t.Errorf("Expected duration 40, got %d", dataQualityScenario.Duration)
	}
	fmt.Printf("Data Quality Scenario: %d events\n", len(dataQualityScenario.Events))

	// Test: Verify capacity test scenario
	capacityScenario := scenarios[2]
	if capacityScenario.Type != "capacity_test" {
		t.Errorf("Third scenario should be capacity_test, got %s", capacityScenario.Type)
	}
	if len(capacityScenario.Events) == 0 {
		t.Error("Capacity test scenario should have events")
	}
	if capacityScenario.Duration != 50 {
		t.Errorf("Expected duration 50, got %d", capacityScenario.Duration)
	}
	fmt.Printf("Capacity Test Scenario: %d events\n", len(capacityScenario.Events))

	// Test: Verify scenario IDs follow correct pattern
	expectedIDPrefix := fmt.Sprintf("scenario_%s_", twin.ID)
	for _, scenario := range scenarios {
		if len(scenario.ID) < len(expectedIDPrefix) || scenario.ID[:len(expectedIDPrefix)] != expectedIDPrefix {
			t.Errorf("Scenario ID '%s' does not start with expected prefix '%s'", scenario.ID, expectedIDPrefix)
		}
		if scenario.TwinID != twin.ID {
			t.Errorf("Scenario TwinID should be '%s', got '%s'", twin.ID, scenario.TwinID)
		}
	}

	// Test: Verify events target actual entity URIs
	allEntityURIs := make(map[string]bool)
	for _, entity := range twin.Entities {
		allEntityURIs[entity.URI] = true
	}

	for _, scenario := range scenarios {
		for _, event := range scenario.Events {
			if !allEntityURIs[event.TargetURI] {
				t.Errorf("Event targets non-existent entity: %s", event.TargetURI)
			}
		}
	}

	// Test: Verify event propagation rules
	for _, scenario := range scenarios {
		for _, event := range scenario.Events {
			if len(event.Impact.PropagationRules) > 0 {
				fmt.Printf("Event %s has %d propagation rules\n", event.Type, len(event.Impact.PropagationRules))
			}
		}
	}

	fmt.Println("\nScenario Generation Test Summary:")
	fmt.Printf("✓ Generated %d scenarios\n", len(scenarios))
	fmt.Printf("✓ Baseline: %s (duration: %d, events: %d)\n",
		scenarios[0].Name, scenarios[0].Duration, len(scenarios[0].Events))
	fmt.Printf("✓ Data Quality: %s (duration: %d, events: %d)\n",
		scenarios[1].Name, scenarios[1].Duration, len(scenarios[1].Events))
	fmt.Printf("✓ Capacity Test: %s (duration: %d, events: %d)\n",
		scenarios[2].Name, scenarios[2].Duration, len(scenarios[2].Events))
}
