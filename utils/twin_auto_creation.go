package utils

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// TwinAutoCreator handles automatic digital twin creation when models finish training
type TwinAutoCreator struct {
	db     *sql.DB
	logger *Logger
}

// NewTwinAutoCreator creates a new twin auto-creator
func NewTwinAutoCreator(db *sql.DB) *TwinAutoCreator {
	return &TwinAutoCreator{
		db:     db,
		logger: GetLogger(),
	}
}

// HandleModelTrainingCompleted handles model.training.completed events
func (t *TwinAutoCreator) HandleModelTrainingCompleted(event Event) error {
	t.logger.Info("Handling model training completion for twin auto-creation",
		String("event_type", event.Type),
		String("source", event.Source))

	// Extract model info from event payload
	ontologyID, _ := event.Payload["ontology_id"].(string)
	modelID, _ := event.Payload["model_id"].(string)
	modelType, _ := event.Payload["model_type"].(string)
	targetProperty, _ := event.Payload["target_property"].(string)

	if ontologyID == "" || modelID == "" {
		t.logger.Warn("Model training event missing required fields",
			String("ontology_id", ontologyID),
			String("model_id", modelID))
		return nil
	}

	// Check if auto-twin creation is enabled for this ontology
	autoCreate, err := t.shouldAutoCreateTwin(ontologyID)
	if err != nil {
		t.logger.Error("Failed to check auto-twin settings", err,
			String("ontology_id", ontologyID))
		return nil
	}

	if !autoCreate {
		t.logger.Debug("Auto-twin creation not enabled for ontology",
			String("ontology_id", ontologyID))
		return nil
	}

	// Check if a twin already exists for this ontology+model combination
	exists, err := t.twinExistsForModel(ontologyID, modelID)
	if err != nil {
		t.logger.Error("Failed to check existing twins", err)
		return nil
	}

	if exists {
		t.logger.Info("Twin already exists for this model",
			String("ontology_id", ontologyID),
			String("model_id", modelID))
		return nil
	}

	// Create the digital twin
	twin, err := t.createTwinFromModel(ontologyID, modelID, modelType, targetProperty)
	if err != nil {
		t.logger.Error("Failed to auto-create digital twin", err,
			String("ontology_id", ontologyID),
			String("model_id", modelID))
		return fmt.Errorf("auto-twin creation failed: %w", err)
	}

	t.logger.Info("Auto-created digital twin from trained model",
		String("twin_id", twin.ID),
		String("twin_name", twin.Name),
		String("ontology_id", ontologyID),
		String("model_id", modelID))

	// Create default monitoring rules for the twin
	if err := t.createMonitoringRulesForTwin(twin.ID, modelID); err != nil {
		t.logger.Warn("Failed to create monitoring rules for twin",
			String("twin_id", twin.ID),
			String("error", err.Error()))
	}

	// Publish twin.created event
	GetEventBus().Publish(Event{
		Type:   EventTwinCreated,
		Source: "twin-auto-creator",
		Payload: map[string]any{
			"twin_id":         twin.ID,
			"twin_name":       twin.Name,
			"ontology_id":     ontologyID,
			"model_id":        modelID,
			"auto_created":    true,
			"target_property": targetProperty,
		},
	})

	return nil
}

// autoCreatedTwin represents a minimal twin structure for auto-creation
type autoCreatedTwin struct {
	ID          string
	Name        string
	Description string
	OntologyID  string
	ModelID     string
	ModelType   string
}

// shouldAutoCreateTwin checks if automatic twin creation is enabled
func (t *TwinAutoCreator) shouldAutoCreateTwin(ontologyID string) (bool, error) {
	query := `SELECT auto_create_twins FROM ontologies WHERE id = ?`
	var autoCreate sql.NullBool
	err := t.db.QueryRow(query, ontologyID).Scan(&autoCreate)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return autoCreate.Valid && autoCreate.Bool, nil
}

// twinExistsForModel checks if a twin already exists for this model
func (t *TwinAutoCreator) twinExistsForModel(ontologyID, modelID string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM digital_twins WHERE ontology_id = ? AND model_id = ?)`
	var exists bool
	err := t.db.QueryRow(query, ontologyID, modelID).Scan(&exists)
	return exists, err
}

// createTwinFromModel creates a digital twin linked to a trained model
func (t *TwinAutoCreator) createTwinFromModel(ontologyID, modelID, modelType, targetProperty string) (*autoCreatedTwin, error) {
	twin := &autoCreatedTwin{
		ID:          uuid.New().String(),
		Name:        fmt.Sprintf("Auto-Twin for %s", targetProperty),
		Description: fmt.Sprintf("Automatically created from trained %s model targeting %s", modelType, targetProperty),
		OntologyID:  ontologyID,
		ModelID:     modelID,
		ModelType:   modelType,
	}

	// Build initial state from ontology entities
	baseState, err := t.buildBaseStateFromOntology(ontologyID)
	if err != nil {
		t.logger.Warn("Could not build base state from ontology, using empty state",
			String("error", err.Error()))
		baseState = "{}"
	}

	now := time.Now()
	query := `
		INSERT INTO digital_twins (id, ontology_id, model_id, name, description, model_type, base_state, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err = t.db.Exec(query, twin.ID, twin.OntologyID, twin.ModelID, twin.Name, twin.Description, twin.ModelType, baseState, now, now)
	if err != nil {
		return nil, fmt.Errorf("failed to insert twin: %w", err)
	}

	// Auto-configure the twin with scenarios and monitoring
	if err := t.AutoConfigureTwin(twin.ID, ontologyID, modelType); err != nil {
		t.logger.Warn("Auto-configuration partially failed for twin",
			String("twin_id", twin.ID),
			String("error", err.Error()))
	}

	return twin, nil
}

// buildBaseStateFromOntology queries ontology entities to build initial twin state
func (t *TwinAutoCreator) buildBaseStateFromOntology(ontologyID string) (string, error) {
	// Query entity count and types from ontology
	query := `
		SELECT COUNT(*) as entity_count
		FROM ontology_entities
		WHERE ontology_id = ?
	`
	var entityCount int
	err := t.db.QueryRow(query, ontologyID).Scan(&entityCount)
	if err != nil && err != sql.ErrNoRows {
		return "{}", err
	}

	state := map[string]any{
		"entity_count":   entityCount,
		"ontology_id":    ontologyID,
		"initialized_at": time.Now().Format(time.RFC3339),
		"status":         "active",
		"global_metrics": map[string]float64{},
		"entities":       []map[string]any{},
		"relationships":  []map[string]any{},
	}

	stateJSON, err := json.Marshal(state)
	if err != nil {
		return "{}", err
	}

	return string(stateJSON), nil
}

// createMonitoringRulesForTwin sets up default anomaly detection for the twin
func (t *TwinAutoCreator) createMonitoringRulesForTwin(twinID, modelID string) error {
	rules := []struct {
		name      string
		ruleType  string
		threshold float64
		metric    string
	}{
		{"High Deviation Alert", "z_score", 3.0, "prediction_deviation"},
		{"Confidence Drop Alert", "threshold_below", 0.5, "model_confidence"},
		{"Entity Unavailable Alert", "entity_status", 0.0, "entity_availability"},
		{"Utilization Spike Alert", "threshold_above", 0.9, "utilization"},
		{"Data Quality Alert", "data_quality", 0.7, "data_completeness"},
	}

	for _, rule := range rules {
		ruleID := uuid.New().String()
		config, _ := json.Marshal(map[string]any{
			"metric":       rule.metric,
			"threshold":    rule.threshold,
			"model_id":     modelID,
			"auto_created": true,
			"created_from": "auto_twin_creation",
		})

		query := `
			INSERT INTO monitoring_rules (id, twin_id, name, rule_type, config, is_enabled, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?)
		`
		_, err := t.db.Exec(query, ruleID, twinID, rule.name, rule.ruleType, string(config), true, time.Now())
		if err != nil {
			t.logger.Debug("Could not create monitoring rule (table may not exist)",
				String("rule_name", rule.name))
		}
	}

	return nil
}

// AutoConfigureTwin fully configures a digital twin with scenarios, monitoring, and initial state
func (t *TwinAutoCreator) AutoConfigureTwin(twinID, ontologyID, modelType string) error {
	// Generate and save scenarios based on model type
	scenarios := t.generateScenariosForModelType(twinID, modelType)
	if err := t.saveScenarios(twinID, scenarios); err != nil {
		t.logger.Warn("Failed to save auto-generated scenarios",
			String("twin_id", twinID),
			String("error", err.Error()))
	}

	// Create monitoring rules
	if err := t.createMonitoringRulesForTwin(twinID, ""); err != nil {
		t.logger.Warn("Failed to create monitoring rules",
			String("twin_id", twinID),
			String("error", err.Error()))
	}

	t.logger.Info("Auto-configured digital twin",
		String("twin_id", twinID),
		String("model_type", modelType),
		Int("scenario_count", len(scenarios)))

	return nil
}

// generateScenariosForModelType creates model-type specific scenarios
type AutoScenario struct {
	ID          string                   `json:"id"`
	TwinID      string                   `json:"twin_id"`
	Name        string                   `json:"name"`
	Description string                   `json:"description"`
	Type        string                   `json:"scenario_type"`
	Events      []map[string]interface{} `json:"events"`
	Duration    int                      `json:"duration"`
	CreatedAt   time.Time                `json:"created_at"`
}

func (t *TwinAutoCreator) generateScenariosForModelType(twinID, modelType string) []AutoScenario {
	now := time.Now()
	scenarios := []AutoScenario{
		{
			ID:          fmt.Sprintf("scenario_%s_baseline", twinID),
			TwinID:      twinID,
			Name:        "Baseline Operations",
			Description: "Normal operating conditions with no disruptions. Establishes performance baseline.",
			Type:        "baseline",
			Events:      []map[string]interface{}{},
			Duration:    30,
			CreatedAt:   now,
		},
		{
			ID:          fmt.Sprintf("scenario_%s_stress", twinID),
			TwinID:      twinID,
			Name:        "Stress Test",
			Description: "High load conditions testing system limits.",
			Type:        "stress_test",
			Events:      []map[string]interface{}{},
			Duration:    50,
			CreatedAt:   now,
		},
	}

	// Add model-type specific scenarios
	switch modelType {
	case "organization":
		scenarios = append(scenarios, AutoScenario{
			ID:          fmt.Sprintf("scenario_%s_dept_failure", twinID),
			TwinID:      twinID,
			Name:        "Department Failure",
			Description: "Simulates failure of a key department and impact on organization.",
			Type:        "dept_failure",
			Events:      []map[string]interface{}{},
			Duration:    40,
			CreatedAt:   now,
		})
	case "department":
		scenarios = append(scenarios, AutoScenario{
			ID:          fmt.Sprintf("scenario_%s_staff_shortage", twinID),
			TwinID:      twinID,
			Name:        "Staff Shortage",
			Description: "Simulates reduced staffing levels.",
			Type:        "staff_shortage",
			Events:      []map[string]interface{}{},
			Duration:    35,
			CreatedAt:   now,
		})
	case "process":
		scenarios = append(scenarios, AutoScenario{
			ID:          fmt.Sprintf("scenario_%s_bottleneck", twinID),
			TwinID:      twinID,
			Name:        "Process Bottleneck",
			Description: "Identifies bottlenecks in the process flow.",
			Type:        "bottleneck",
			Events:      []map[string]interface{}{},
			Duration:    45,
			CreatedAt:   now,
		})
	case "individual":
		scenarios = append(scenarios, AutoScenario{
			ID:          fmt.Sprintf("scenario_%s_skill_gap", twinID),
			TwinID:      twinID,
			Name:        "Skill Gap Analysis",
			Description: "Analyzes impact of missing skills.",
			Type:        "skill_gap",
			Events:      []map[string]interface{}{},
			Duration:    25,
			CreatedAt:   now,
		})
	default:
		// Generic scenarios for unknown types
		scenarios = append(scenarios, AutoScenario{
			ID:          fmt.Sprintf("scenario_%s_generic", twinID),
			TwinID:      twinID,
			Name:        "Generic Impact Analysis",
			Description: "General purpose scenario for impact analysis.",
			Type:        "generic",
			Events:      []map[string]interface{}{},
			Duration:    30,
			CreatedAt:   now,
		})
	}

	return scenarios
}

func (t *TwinAutoCreator) saveScenarios(twinID string, scenarios []AutoScenario) error {
	for _, scenario := range scenarios {
		eventsJSON, _ := json.Marshal(scenario.Events)
		query := `
			INSERT INTO simulation_scenarios (id, twin_id, name, description, scenario_type, events, duration, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		`
		_, err := t.db.Exec(query, scenario.ID, scenario.TwinID, scenario.Name,
			scenario.Description, scenario.Type, string(eventsJSON), scenario.Duration, scenario.CreatedAt)
		if err != nil {
			t.logger.Debug("Could not save scenario (table may not exist)",
				String("scenario_name", scenario.Name),
				String("error", err.Error()))
		}
	}
	return nil
}

// InitializeTwinAutoCreator sets up automatic twin creation from model training
func InitializeTwinAutoCreator(db *sql.DB) {
	if db == nil {
		GetLogger().Warn("Twin auto-creator not initialized (missing database)")
		return
	}

	creator := NewTwinAutoCreator(db)
	GetEventBus().Subscribe(EventModelTrainingCompleted, creator.HandleModelTrainingCompleted)
	GetLogger().Info("Twin auto-creator initialized - will create twins when models finish training")
}
