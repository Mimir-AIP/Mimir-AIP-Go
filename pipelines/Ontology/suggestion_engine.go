package ontology

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Mimir-AIP/Mimir-AIP-Go/pipelines"
	"github.com/Mimir-AIP/Mimir-AIP-Go/pipelines/AI"
	knowledgegraph "github.com/Mimir-AIP/Mimir-AIP-Go/pipelines/KnowledgeGraph"
)

// SuggestionEngine manages ontology change suggestions
type SuggestionEngine struct {
	db          *sql.DB
	llmClient   AI.LLMClient
	tdb2Backend *knowledgegraph.TDB2Backend
}

// NewSuggestionEngine creates a new suggestion engine
func NewSuggestionEngine(db *sql.DB, llmClient AI.LLMClient, tdb2Backend *knowledgegraph.TDB2Backend) *SuggestionEngine {
	return &SuggestionEngine{
		db:          db,
		llmClient:   llmClient,
		tdb2Backend: tdb2Backend,
	}
}

// ListSuggestions retrieves suggestions for an ontology
func (s *SuggestionEngine) ListSuggestions(ctx context.Context, ontologyID string, status SuggestionStatus) ([]OntologySuggestion, error) {
	query := `SELECT id, ontology_id, suggestion_type, entity_type, entity_uri, confidence, reasoning, 
	          status, risk_level, created_at, reviewed_at, reviewed_by, review_decision, review_notes 
	          FROM ontology_suggestions WHERE ontology_id = ?`

	args := []interface{}{ontologyID}

	if status != "" {
		query += " AND status = ?"
		args = append(args, status)
	}

	query += " ORDER BY created_at DESC"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query suggestions: %w", err)
	}
	defer rows.Close()

	suggestions := make([]OntologySuggestion, 0)
	for rows.Next() {
		var suggestion OntologySuggestion
		var entityURI, reviewedBy, reviewDecision, reviewNotes sql.NullString
		var reviewedAt sql.NullTime

		err := rows.Scan(
			&suggestion.ID,
			&suggestion.OntologyID,
			&suggestion.SuggestionType,
			&suggestion.EntityType,
			&entityURI,
			&suggestion.Confidence,
			&suggestion.Reasoning,
			&suggestion.Status,
			&suggestion.RiskLevel,
			&suggestion.CreatedAt,
			&reviewedAt,
			&reviewedBy,
			&reviewDecision,
			&reviewNotes,
		)
		if err != nil {
			continue
		}

		if entityURI.Valid {
			suggestion.EntityURI = entityURI.String
		}
		if reviewedAt.Valid {
			suggestion.ReviewedAt = &reviewedAt.Time
		}
		if reviewedBy.Valid {
			suggestion.ReviewedBy = reviewedBy.String
		}
		if reviewDecision.Valid {
			suggestion.ReviewDecision = reviewDecision.String
		}
		if reviewNotes.Valid {
			suggestion.ReviewNotes = reviewNotes.String
		}

		suggestions = append(suggestions, suggestion)
	}

	return suggestions, nil
}

// GetSuggestion retrieves a single suggestion by ID
func (s *SuggestionEngine) GetSuggestion(ctx context.Context, suggestionID int) (*OntologySuggestion, error) {
	query := `SELECT id, ontology_id, suggestion_type, entity_type, entity_uri, confidence, reasoning, 
	          status, risk_level, created_at, reviewed_at, reviewed_by, review_decision, review_notes 
	          FROM ontology_suggestions WHERE id = ?`

	var suggestion OntologySuggestion
	var entityURI, reviewedBy, reviewDecision, reviewNotes sql.NullString
	var reviewedAt sql.NullTime

	err := s.db.QueryRowContext(ctx, query, suggestionID).Scan(
		&suggestion.ID,
		&suggestion.OntologyID,
		&suggestion.SuggestionType,
		&suggestion.EntityType,
		&entityURI,
		&suggestion.Confidence,
		&suggestion.Reasoning,
		&suggestion.Status,
		&suggestion.RiskLevel,
		&suggestion.CreatedAt,
		&reviewedAt,
		&reviewedBy,
		&reviewDecision,
		&reviewNotes,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("suggestion not found")
		}
		return nil, fmt.Errorf("failed to get suggestion: %w", err)
	}

	if entityURI.Valid {
		suggestion.EntityURI = entityURI.String
	}
	if reviewedAt.Valid {
		suggestion.ReviewedAt = &reviewedAt.Time
	}
	if reviewedBy.Valid {
		suggestion.ReviewedBy = reviewedBy.String
	}
	if reviewDecision.Valid {
		suggestion.ReviewDecision = reviewDecision.String
	}
	if reviewNotes.Valid {
		suggestion.ReviewNotes = reviewNotes.String
	}

	return &suggestion, nil
}

// ApproveSuggestion approves a suggestion for implementation
func (s *SuggestionEngine) ApproveSuggestion(ctx context.Context, suggestionID int, reviewedBy, reviewNotes string) error {
	now := time.Now()
	query := `UPDATE ontology_suggestions 
	          SET status = ?, reviewed_at = ?, reviewed_by = ?, review_decision = ?, review_notes = ? 
	          WHERE id = ? AND status = ?`

	result, err := s.db.ExecContext(ctx, query,
		SuggestionApproved, now, reviewedBy, "approved", reviewNotes, suggestionID, SuggestionPending)
	if err != nil {
		return fmt.Errorf("failed to approve suggestion: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("suggestion not found or already reviewed")
	}

	return nil
}

// RejectSuggestion rejects a suggestion
func (s *SuggestionEngine) RejectSuggestion(ctx context.Context, suggestionID int, reviewedBy, reviewNotes string) error {
	now := time.Now()
	query := `UPDATE ontology_suggestions 
	          SET status = ?, reviewed_at = ?, reviewed_by = ?, review_decision = ?, review_notes = ? 
	          WHERE id = ? AND status = ?`

	result, err := s.db.ExecContext(ctx, query,
		SuggestionRejected, now, reviewedBy, "rejected", reviewNotes, suggestionID, SuggestionPending)
	if err != nil {
		return fmt.Errorf("failed to reject suggestion: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("suggestion not found or already reviewed")
	}

	return nil
}

// ApplySuggestion applies an approved suggestion to the ontology
func (s *SuggestionEngine) ApplySuggestion(ctx context.Context, suggestionID int) error {
	// Get suggestion
	suggestion, err := s.GetSuggestion(ctx, suggestionID)
	if err != nil {
		return fmt.Errorf("failed to get suggestion: %w", err)
	}

	if suggestion.Status != SuggestionApproved {
		return fmt.Errorf("suggestion must be approved before applying (current status: %s)", suggestion.Status)
	}

	// Get ontology metadata
	ontology, err := s.getOntology(ctx, suggestion.OntologyID)
	if err != nil {
		return fmt.Errorf("failed to get ontology: %w", err)
	}

	// Apply the suggestion based on type
	switch suggestion.SuggestionType {
	case SuggestionAddClass:
		err = s.applyAddClass(ctx, ontology, suggestion)
	case SuggestionAddProperty:
		err = s.applyAddProperty(ctx, ontology, suggestion)
	case SuggestionModifyClass:
		err = s.applyModifyClass(ctx, ontology, suggestion)
	case SuggestionModifyProperty:
		err = s.applyModifyProperty(ctx, ontology, suggestion)
	case SuggestionDeprecate:
		err = s.applyDeprecate(ctx, ontology, suggestion)
	default:
		return fmt.Errorf("unsupported suggestion type: %s", suggestion.SuggestionType)
	}

	if err != nil {
		return fmt.Errorf("failed to apply suggestion: %w", err)
	}

	// Mark as applied
	query := `UPDATE ontology_suggestions SET status = ? WHERE id = ?`
	_, err = s.db.ExecContext(ctx, query, SuggestionApplied, suggestionID)
	if err != nil {
		return fmt.Errorf("failed to update suggestion status: %w", err)
	}

	return nil
}

// ApplyMultipleSuggestions applies multiple approved suggestions in a batch
func (s *SuggestionEngine) ApplyMultipleSuggestions(ctx context.Context, suggestionIDs []int) ([]int, []error) {
	successful := make([]int, 0)
	errors := make([]error, 0)

	for _, id := range suggestionIDs {
		err := s.ApplySuggestion(ctx, id)
		if err != nil {
			errors = append(errors, fmt.Errorf("suggestion %d: %w", id, err))
		} else {
			successful = append(successful, id)
		}
	}

	return successful, errors
}

// GenerateSuggestionSummary generates a human-readable summary of suggestions
func (s *SuggestionEngine) GenerateSuggestionSummary(ctx context.Context, ontologyID string) (string, error) {
	suggestions, err := s.ListSuggestions(ctx, ontologyID, SuggestionPending)
	if err != nil {
		return "", fmt.Errorf("failed to list suggestions: %w", err)
	}

	if len(suggestions) == 0 {
		return "No pending suggestions", nil
	}

	// Group by type and risk level
	byType := make(map[SuggestionType]int)
	byRisk := make(map[RiskLevel]int)

	for _, sugg := range suggestions {
		byType[sugg.SuggestionType]++
		byRisk[sugg.RiskLevel]++
	}

	summary := fmt.Sprintf("Pending Suggestions: %d\n\n", len(suggestions))
	summary += "By Type:\n"
	for stype, count := range byType {
		summary += fmt.Sprintf("  - %s: %d\n", stype, count)
	}

	summary += "\nBy Risk Level:\n"
	for risk, count := range byRisk {
		summary += fmt.Sprintf("  - %s: %d\n", risk, count)
	}

	return summary, nil
}

// Apply methods for different suggestion types

func (s *SuggestionEngine) applyAddClass(ctx context.Context, ontology *OntologyMetadata, suggestion *OntologySuggestion) error {
	// Insert into ontology_classes table
	query := `INSERT INTO ontology_classes (ontology_id, uri, label, description, parent_uris, created_at) 
	          VALUES (?, ?, ?, ?, ?, ?)`

	label := extractLabel(suggestion.EntityURI)
	parentURIs, _ := json.Marshal([]string{}) // Default: no parents

	_, err := s.db.ExecContext(ctx, query,
		ontology.ID,
		suggestion.EntityURI,
		label,
		suggestion.Reasoning,
		string(parentURIs),
		time.Now())
	if err != nil {
		return fmt.Errorf("failed to insert class: %w", err)
	}

	// Add to TDB2 knowledge graph
	triples := []knowledgegraph.Triple{
		{
			Subject:   suggestion.EntityURI,
			Predicate: "http://www.w3.org/1999/02/22-rdf-syntax-ns#type",
			Object:    "http://www.w3.org/2002/07/owl#Class",
			Graph:     ontology.TDB2Graph,
		},
		{
			Subject:   suggestion.EntityURI,
			Predicate: "http://www.w3.org/2000/01/rdf-schema#label",
			Object:    label,
			Graph:     ontology.TDB2Graph,
		},
	}

	if suggestion.Reasoning != "" {
		triples = append(triples, knowledgegraph.Triple{
			Subject:   suggestion.EntityURI,
			Predicate: "http://www.w3.org/2000/01/rdf-schema#comment",
			Object:    suggestion.Reasoning,
			Graph:     ontology.TDB2Graph,
		})
	}

	return s.tdb2Backend.InsertTriples(ctx, triples)
}

func (s *SuggestionEngine) applyAddProperty(ctx context.Context, ontology *OntologyMetadata, suggestion *OntologySuggestion) error {
	// Insert into ontology_properties table
	query := `INSERT INTO ontology_properties (ontology_id, uri, label, property_type, domain, range, description, created_at) 
	          VALUES (?, ?, ?, ?, ?, ?, ?, ?)`

	label := extractLabel(suggestion.EntityURI)
	domain, _ := json.Marshal([]string{}) // Default: no domain restriction
	range_, _ := json.Marshal([]string{}) // Default: no range restriction

	_, err := s.db.ExecContext(ctx, query,
		ontology.ID,
		suggestion.EntityURI,
		label,
		PropertyTypeDatatype, // Default to datatype property
		string(domain),
		string(range_),
		suggestion.Reasoning,
		time.Now())
	if err != nil {
		return fmt.Errorf("failed to insert property: %w", err)
	}

	// Add to TDB2 knowledge graph
	triples := []knowledgegraph.Triple{
		{
			Subject:   suggestion.EntityURI,
			Predicate: "http://www.w3.org/1999/02/22-rdf-syntax-ns#type",
			Object:    "http://www.w3.org/2002/07/owl#DatatypeProperty",
			Graph:     ontology.TDB2Graph,
		},
		{
			Subject:   suggestion.EntityURI,
			Predicate: "http://www.w3.org/2000/01/rdf-schema#label",
			Object:    label,
			Graph:     ontology.TDB2Graph,
		},
	}

	if suggestion.Reasoning != "" {
		triples = append(triples, knowledgegraph.Triple{
			Subject:   suggestion.EntityURI,
			Predicate: "http://www.w3.org/2000/01/rdf-schema#comment",
			Object:    suggestion.Reasoning,
			Graph:     ontology.TDB2Graph,
		})
	}

	return s.tdb2Backend.InsertTriples(ctx, triples)
}

func (s *SuggestionEngine) applyModifyClass(ctx context.Context, ontology *OntologyMetadata, suggestion *OntologySuggestion) error {
	// Update description in database
	query := `UPDATE ontology_classes SET description = ? WHERE ontology_id = ? AND uri = ?`
	_, err := s.db.ExecContext(ctx, query, suggestion.Reasoning, ontology.ID, suggestion.EntityURI)
	if err != nil {
		return fmt.Errorf("failed to update class: %w", err)
	}

	// Update comment in TDB2
	triples := []knowledgegraph.Triple{
		{
			Subject:   suggestion.EntityURI,
			Predicate: "http://www.w3.org/2000/01/rdf-schema#comment",
			Object:    suggestion.Reasoning,
			Graph:     ontology.TDB2Graph,
		},
	}

	return s.tdb2Backend.InsertTriples(ctx, triples)
}

func (s *SuggestionEngine) applyModifyProperty(ctx context.Context, ontology *OntologyMetadata, suggestion *OntologySuggestion) error {
	// Update description in database
	query := `UPDATE ontology_properties SET description = ? WHERE ontology_id = ? AND uri = ?`
	_, err := s.db.ExecContext(ctx, query, suggestion.Reasoning, ontology.ID, suggestion.EntityURI)
	if err != nil {
		return fmt.Errorf("failed to update property: %w", err)
	}

	// Update comment in TDB2
	triples := []knowledgegraph.Triple{
		{
			Subject:   suggestion.EntityURI,
			Predicate: "http://www.w3.org/2000/01/rdf-schema#comment",
			Object:    suggestion.Reasoning,
			Graph:     ontology.TDB2Graph,
		},
	}

	return s.tdb2Backend.InsertTriples(ctx, triples)
}

func (s *SuggestionEngine) applyDeprecate(ctx context.Context, ontology *OntologyMetadata, suggestion *OntologySuggestion) error {
	// Mark as deprecated in TDB2
	triples := []knowledgegraph.Triple{
		{
			Subject:   suggestion.EntityURI,
			Predicate: "http://www.w3.org/2002/07/owl#deprecated",
			Object:    "http://www.w3.org/2001/XMLSchema#boolean^^true",
			Graph:     ontology.TDB2Graph,
		},
		{
			Subject:   suggestion.EntityURI,
			Predicate: "http://www.w3.org/2000/01/rdf-schema#comment",
			Object:    fmt.Sprintf("Deprecated: %s", suggestion.Reasoning),
			Graph:     ontology.TDB2Graph,
		},
	}

	return s.tdb2Backend.InsertTriples(ctx, triples)
}

// Helper functions

func (s *SuggestionEngine) getOntology(ctx context.Context, ontologyID string) (*OntologyMetadata, error) {
	query := `SELECT id, name, description, version, file_path, tdb2_graph, format, status, created_at, updated_at, created_by, metadata 
	          FROM ontologies WHERE id = ?`
	row := s.db.QueryRowContext(ctx, query, ontologyID)

	var ont OntologyMetadata
	var metadataJSON sql.NullString
	err := row.Scan(&ont.ID, &ont.Name, &ont.Description, &ont.Version, &ont.FilePath, &ont.TDB2Graph,
		&ont.Format, &ont.Status, &ont.CreatedAt, &ont.UpdatedAt, &ont.CreatedBy, &metadataJSON)
	if err != nil {
		return nil, err
	}

	if metadataJSON.Valid {
		json.Unmarshal([]byte(metadataJSON.String), &ont.Metadata)
	}

	return &ont, nil
}

func extractLabel(uri string) string {
	// Extract label from URI (simple heuristic)
	// e.g., "http://example.org/ontology#Product" -> "Product"
	for i := len(uri) - 1; i >= 0; i-- {
		if uri[i] == '#' || uri[i] == '/' {
			return uri[i+1:]
		}
	}
	return uri
}

// SuggestionPlugin implements BasePlugin for suggestion management
type SuggestionPlugin struct {
	engine *SuggestionEngine
}

// NewSuggestionPlugin creates a new suggestion plugin
func NewSuggestionPlugin(db *sql.DB, llmClient AI.LLMClient, tdb2Backend *knowledgegraph.TDB2Backend) *SuggestionPlugin {
	return &SuggestionPlugin{
		engine: NewSuggestionEngine(db, llmClient, tdb2Backend),
	}
}

// ExecuteStep implements BasePlugin.ExecuteStep
func (p *SuggestionPlugin) ExecuteStep(ctx context.Context, stepConfig pipelines.StepConfig, globalContext *pipelines.PluginContext) (*pipelines.PluginContext, error) {
	operation, ok := stepConfig.Config["operation"].(string)
	if !ok {
		return nil, fmt.Errorf("operation not specified in config")
	}

	resultContext := pipelines.NewPluginContext()

	switch operation {
	case "list":
		ontologyID, _ := stepConfig.Config["ontology_id"].(string)
		status := SuggestionStatus("")
		if statusStr, ok := stepConfig.Config["status"].(string); ok {
			status = SuggestionStatus(statusStr)
		}

		suggestions, err := p.engine.ListSuggestions(ctx, ontologyID, status)
		if err != nil {
			return nil, err
		}
		resultContext.Set("suggestions", suggestions)

	case "get":
		suggestionID, ok := stepConfig.Config["suggestion_id"].(int)
		if !ok {
			if idFloat, ok := stepConfig.Config["suggestion_id"].(float64); ok {
				suggestionID = int(idFloat)
			} else {
				return nil, fmt.Errorf("suggestion_id is required")
			}
		}

		suggestion, err := p.engine.GetSuggestion(ctx, suggestionID)
		if err != nil {
			return nil, err
		}
		resultContext.Set("suggestion", suggestion)

	case "approve":
		suggestionID, _ := stepConfig.Config["suggestion_id"].(int)
		reviewedBy, _ := stepConfig.Config["reviewed_by"].(string)
		reviewNotes, _ := stepConfig.Config["review_notes"].(string)

		err := p.engine.ApproveSuggestion(ctx, suggestionID, reviewedBy, reviewNotes)
		if err != nil {
			return nil, err
		}
		resultContext.Set("status", "approved")

	case "reject":
		suggestionID, _ := stepConfig.Config["suggestion_id"].(int)
		reviewedBy, _ := stepConfig.Config["reviewed_by"].(string)
		reviewNotes, _ := stepConfig.Config["review_notes"].(string)

		err := p.engine.RejectSuggestion(ctx, suggestionID, reviewedBy, reviewNotes)
		if err != nil {
			return nil, err
		}
		resultContext.Set("status", "rejected")

	case "apply":
		suggestionID, _ := stepConfig.Config["suggestion_id"].(int)

		err := p.engine.ApplySuggestion(ctx, suggestionID)
		if err != nil {
			return nil, err
		}
		resultContext.Set("status", "applied")

	case "summary":
		ontologyID, _ := stepConfig.Config["ontology_id"].(string)

		summary, err := p.engine.GenerateSuggestionSummary(ctx, ontologyID)
		if err != nil {
			return nil, err
		}
		resultContext.Set("summary", summary)

	default:
		return nil, fmt.Errorf("unknown operation: %s", operation)
	}

	return resultContext, nil
}

// GetPluginType implements BasePlugin.GetPluginType
func (p *SuggestionPlugin) GetPluginType() string {
	return "Ontology"
}

// GetPluginName implements BasePlugin.GetPluginName
func (p *SuggestionPlugin) GetPluginName() string {
	return "suggestions"
}

// ValidateConfig implements BasePlugin.ValidateConfig
func (p *SuggestionPlugin) ValidateConfig(config map[string]any) error {
	operation, ok := config["operation"].(string)
	if !ok {
		return fmt.Errorf("operation is required")
	}

	validOperations := []string{"list", "get", "approve", "reject", "apply", "summary"}
	valid := false
	for _, op := range validOperations {
		if operation == op {
			valid = true
			break
		}
	}

	if !valid {
		return fmt.Errorf("invalid operation: %s", operation)
	}

	return nil
}
