package ontology

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"strings"

	"github.com/Mimir-AIP/Mimir-AIP-Go/pipelines"
	AI "github.com/Mimir-AIP/Mimir-AIP-Go/pipelines/AI"
	knowledgegraph "github.com/Mimir-AIP/Mimir-AIP-Go/pipelines/KnowledgeGraph"
)

// NLQueryPlugin translates natural language questions to SPARQL and executes them
type NLQueryPlugin struct {
	db          *sql.DB
	tdb2Backend *knowledgegraph.TDB2Backend
	llmClient   AI.LLMClient
}

// NewNLQueryPlugin creates a new natural language query plugin
func NewNLQueryPlugin(db *sql.DB, tdb2Backend *knowledgegraph.TDB2Backend, llmClient AI.LLMClient) *NLQueryPlugin {
	return &NLQueryPlugin{
		db:          db,
		tdb2Backend: tdb2Backend,
		llmClient:   llmClient,
	}
}

// ExecuteStep implements BasePlugin.ExecuteStep
func (p *NLQueryPlugin) ExecuteStep(ctx context.Context, stepConfig pipelines.StepConfig, globalContext *pipelines.PluginContext) (*pipelines.PluginContext, error) {
	question, ok := stepConfig.Config["question"].(string)
	if !ok || question == "" {
		return nil, fmt.Errorf("question is required")
	}

	ontologyID, _ := stepConfig.Config["ontology_id"].(string)

	// Translate natural language to SPARQL
	sparqlQuery, explanation, err := p.translateToSPARQL(ctx, question, ontologyID)
	if err != nil {
		return nil, fmt.Errorf("failed to translate question to SPARQL: %w", err)
	}

	// Validate generated SPARQL for safety
	if err := p.validateSPARQL(sparqlQuery); err != nil {
		return nil, fmt.Errorf("generated SPARQL failed safety validation: %w", err)
	}

	// Execute SPARQL query
	result, err := p.tdb2Backend.QuerySPARQL(ctx, sparqlQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to execute SPARQL query: %w", err)
	}

	// Return results
	resultContext := pipelines.NewPluginContext()
	resultContext.Set("question", question)
	resultContext.Set("sparql_query", sparqlQuery)
	resultContext.Set("explanation", explanation)
	resultContext.Set("results", result)

	return resultContext, nil
}

// GetPluginType implements BasePlugin.GetPluginType
func (p *NLQueryPlugin) GetPluginType() string {
	return "Ontology"
}

// GetPluginName implements BasePlugin.GetPluginName
func (p *NLQueryPlugin) GetPluginName() string {
	return "query"
}

// ValidateConfig implements BasePlugin.ValidateConfig
func (p *NLQueryPlugin) ValidateConfig(config map[string]any) error {
	if _, ok := config["question"].(string); !ok {
		return fmt.Errorf("question is required")
	}
	return nil
}

// GetInputSchema returns the JSON Schema for agent-friendly ontology queries
func (p *NLQueryPlugin) GetInputSchema() map[string]any {
	return map[string]any{
		"type":        "object",
		"description": "Query the knowledge graph using natural language or SPARQL. Converts natural language questions to SPARQL queries and returns results from the RDF triplestore.",
		"properties": map[string]any{
			"ontology_id": map[string]any{
				"type":        "string",
				"description": "ID of the ontology to query",
			},
			"question": map[string]any{
				"type":        "string",
				"description": "Natural language question to query the knowledge graph",
			},
			"use_nl": map[string]any{
				"type":        "boolean",
				"description": "Use natural language translation (true) or provide raw SPARQL (false)",
				"default":     true,
			},
			"sparql_query": map[string]any{
				"type":        "string",
				"description": "Raw SPARQL query (only if use_nl is false)",
			},
		},
		"required": []string{"ontology_id"},
	}
}

// translateToSPARQL uses LLM to convert natural language to SPARQL
func (p *NLQueryPlugin) translateToSPARQL(ctx context.Context, question string, ontologyID string) (string, string, error) {
	if p.llmClient == nil {
		return "", "", fmt.Errorf("LLM client not configured")
	}

	// Load ontology context if ontology ID is provided
	var ontologyContext *OntologyContext
	if ontologyID != "" {
		context, err := p.loadOntologyContext(ctx, ontologyID)
		if err != nil {
			// Log warning but continue without ontology context
			fmt.Printf("Warning: failed to load ontology context: %v\n", err)
		} else {
			ontologyContext = context
		}
	}

	// Build prompt
	prompt := p.buildTranslationPrompt(question, ontologyContext)

	// Call LLM
	response, err := p.llmClient.CompleteSimple(ctx, prompt)
	if err != nil {
		return "", "", fmt.Errorf("LLM completion failed: %w", err)
	}

	// Parse response
	sparqlQuery, explanation := p.parseTranslationResponse(response)

	return sparqlQuery, explanation, nil
}

// buildTranslationPrompt constructs a prompt for NL to SPARQL translation
func (p *NLQueryPlugin) buildTranslationPrompt(question string, ontologyContext *OntologyContext) string {
	var sb strings.Builder

	sb.WriteString("You are an expert in SPARQL and RDF knowledge graphs.\n\n")
	sb.WriteString("# Task\n")
	sb.WriteString("Translate the following natural language question into a SPARQL query.\n\n")

	// Add ontology context if available
	if ontologyContext != nil {
		sb.WriteString("# Ontology Context\n")
		sb.WriteString(fmt.Sprintf("Base URI: %s\n", ontologyContext.BaseURI))
		sb.WriteString(fmt.Sprintf("Graph: %s\n\n", ontologyContext.Metadata.TDB2Graph))

		// Add classes
		if len(ontologyContext.Classes) > 0 && len(ontologyContext.Classes) <= 50 {
			sb.WriteString("## Available Classes:\n")
			for _, class := range ontologyContext.Classes {
				sb.WriteString(fmt.Sprintf("- <%s>", class.URI))
				if class.Label != "" {
					sb.WriteString(fmt.Sprintf(" (%s)", class.Label))
				}
				sb.WriteString("\n")
			}
			sb.WriteString("\n")
		}

		// Add properties
		if len(ontologyContext.Properties) > 0 && len(ontologyContext.Properties) <= 50 {
			sb.WriteString("## Available Properties:\n")
			for _, prop := range ontologyContext.Properties {
				sb.WriteString(fmt.Sprintf("- <%s>", prop.URI))
				if prop.Label != "" {
					sb.WriteString(fmt.Sprintf(" (%s)", prop.Label))
				}
				sb.WriteString("\n")
			}
			sb.WriteString("\n")
		}
	}

	// Add question
	sb.WriteString("# Question\n")
	sb.WriteString(question)
	sb.WriteString("\n\n")

	// Add output format
	sb.WriteString("# Output Format\n")
	sb.WriteString("Return your response in the following format:\n\n")
	sb.WriteString("SPARQL:\n")
	sb.WriteString("```sparql\n")
	sb.WriteString("<your SPARQL query here>\n")
	sb.WriteString("```\n\n")
	sb.WriteString("EXPLANATION:\n")
	sb.WriteString("<brief explanation of what the query does>\n\n")

	// Add guidelines
	sb.WriteString("# Guidelines\n")
	sb.WriteString("- Generate a valid SPARQL query (SELECT, CONSTRUCT, ASK, or DESCRIBE)\n")
	sb.WriteString("- Use proper SPARQL syntax\n")
	sb.WriteString("- Include appropriate FILTER clauses if needed\n")
	sb.WriteString("- Use LIMIT to prevent excessive results (default: 100)\n")
	sb.WriteString("- Add ORDER BY for better result organization\n")
	if ontologyContext != nil {
		sb.WriteString(fmt.Sprintf("- Query the graph <%s>\n", ontologyContext.Metadata.TDB2Graph))
		sb.WriteString("- Use the classes and properties from the ontology context\n")
	}
	sb.WriteString("- For safety, do NOT use: DROP, CLEAR, INSERT, DELETE, LOAD, CREATE\n")
	sb.WriteString("- Only generate read-only SELECT, CONSTRUCT, ASK, or DESCRIBE queries\n")

	return sb.String()
}

// parseTranslationResponse extracts SPARQL query and explanation from LLM response
func (p *NLQueryPlugin) parseTranslationResponse(response string) (string, string) {
	// Extract SPARQL query from code block
	sparqlPattern := `(?s)SPARQL:.*?` + "```" + `(?:sparql)?\s*\n(.*?)\n\s*` + "```"
	sparqlRegex := regexp.MustCompile(sparqlPattern)
	matches := sparqlRegex.FindStringSubmatch(response)

	var sparqlQuery string
	if len(matches) > 1 {
		sparqlQuery = strings.TrimSpace(matches[1])
	} else {
		// Fallback: try to find any code block
		codeBlockRegex := regexp.MustCompile("(?s)```(?:sparql)?\\s*\\n(.*?)\\n\\s*```")
		matches = codeBlockRegex.FindStringSubmatch(response)
		if len(matches) > 1 {
			sparqlQuery = strings.TrimSpace(matches[1])
		} else {
			// Last resort: use entire response
			sparqlQuery = strings.TrimSpace(response)
		}
	}

	// Extract explanation
	explanationRegex := regexp.MustCompile(`(?s)EXPLANATION:\s*\n(.*?)(?:\n\n|\z)`)
	matches = explanationRegex.FindStringSubmatch(response)

	var explanation string
	if len(matches) > 1 {
		explanation = strings.TrimSpace(matches[1])
	} else {
		explanation = "Query generated from natural language question"
	}

	return sparqlQuery, explanation
}

// validateSPARQL validates SPARQL query for safety (no mutations)
func (p *NLQueryPlugin) validateSPARQL(query string) error {
	upperQuery := strings.ToUpper(query)

	// Check for dangerous operations
	dangerousOperations := []string{
		"DROP",
		"CLEAR",
		"INSERT",
		"DELETE",
		"LOAD",
		"CREATE",
		"ADD",
		"MOVE",
		"COPY",
	}

	for _, op := range dangerousOperations {
		if strings.Contains(upperQuery, op) {
			return fmt.Errorf("query contains forbidden operation: %s", op)
		}
	}

	// Check that it's a read-only query
	allowedQueryTypes := []string{"SELECT", "CONSTRUCT", "ASK", "DESCRIBE"}
	hasValidType := false
	for _, qt := range allowedQueryTypes {
		if strings.Contains(upperQuery, qt) {
			hasValidType = true
			break
		}
	}

	if !hasValidType {
		return fmt.Errorf("query must be a read-only query (SELECT, CONSTRUCT, ASK, or DESCRIBE)")
	}

	// Basic sanity check
	if !strings.Contains(upperQuery, "WHERE") && !strings.Contains(upperQuery, "ASK") {
		return fmt.Errorf("query appears to be malformed (missing WHERE clause)")
	}

	return nil
}

// loadOntologyContext loads ontology context for query translation
func (p *NLQueryPlugin) loadOntologyContext(ctx context.Context, ontologyID string) (*OntologyContext, error) {
	// Load ontology metadata
	metadata, err := p.getOntologyMetadata(ctx, ontologyID)
	if err != nil {
		return nil, fmt.Errorf("failed to load ontology metadata: %w", err)
	}

	// Load classes
	classes, err := p.getOntologyClasses(ctx, ontologyID)
	if err != nil {
		return nil, fmt.Errorf("failed to load ontology classes: %w", err)
	}

	// Load properties
	properties, err := p.getOntologyProperties(ctx, ontologyID)
	if err != nil {
		return nil, fmt.Errorf("failed to load ontology properties: %w", err)
	}

	// Infer base URI
	baseURI := metadata.TDB2Graph
	if baseURI == "" {
		baseURI = fmt.Sprintf("http://mimir-aip.io/ontology/%s", ontologyID)
	}

	return &OntologyContext{
		Metadata:   metadata,
		BaseURI:    baseURI,
		Classes:    classes,
		Properties: properties,
	}, nil
}

func (p *NLQueryPlugin) getOntologyMetadata(ctx context.Context, ontologyID string) (*OntologyMetadata, error) {
	metadata := &OntologyMetadata{}

	query := `
		SELECT id, name, description, version, file_path, tdb2_graph, format, status, created_at, updated_at, created_by
		FROM ontologies 
		WHERE id = ?
	`

	err := p.db.QueryRowContext(ctx, query, ontologyID).Scan(
		&metadata.ID,
		&metadata.Name,
		&metadata.Description,
		&metadata.Version,
		&metadata.FilePath,
		&metadata.TDB2Graph,
		&metadata.Format,
		&metadata.Status,
		&metadata.CreatedAt,
		&metadata.UpdatedAt,
		&metadata.CreatedBy,
	)

	return metadata, err
}

func (p *NLQueryPlugin) getOntologyClasses(ctx context.Context, ontologyID string) ([]OntologyClass, error) {
	query := `
		SELECT class_uri, label, description, deprecated
		FROM ontology_classes 
		WHERE ontology_id = ? AND deprecated = 0
		LIMIT 100
	`

	rows, err := p.db.QueryContext(ctx, query, ontologyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var classes []OntologyClass
	for rows.Next() {
		class := OntologyClass{}
		var label, description sql.NullString

		err := rows.Scan(&class.URI, &label, &description, &class.Deprecated)
		if err != nil {
			return nil, err
		}

		if label.Valid {
			class.Label = label.String
		}
		if description.Valid {
			class.Description = description.String
		}

		classes = append(classes, class)
	}

	return classes, nil
}

func (p *NLQueryPlugin) getOntologyProperties(ctx context.Context, ontologyID string) ([]OntologyProperty, error) {
	query := `
		SELECT property_uri, label, description, property_type, deprecated
		FROM ontology_properties 
		WHERE ontology_id = ? AND deprecated = 0
		LIMIT 100
	`

	rows, err := p.db.QueryContext(ctx, query, ontologyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var properties []OntologyProperty
	for rows.Next() {
		prop := OntologyProperty{}
		var label, description sql.NullString

		err := rows.Scan(&prop.URI, &label, &description, &prop.PropertyType, &prop.Deprecated)
		if err != nil {
			return nil, err
		}

		if label.Valid {
			prop.Label = label.String
		}
		if description.Valid {
			prop.Description = description.String
		}

		properties = append(properties, prop)
	}

	return properties, nil
}
