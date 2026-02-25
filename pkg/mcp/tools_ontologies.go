package mcp

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/mimir-aip/mimir-aip-go/pkg/models"
)

func registerOntologyTools(s *server.MCPServer, m *MimirMCPServer) {
	// list_ontologies
	s.AddTool(
		mcp.NewTool("list_ontologies",
			mcp.WithDescription("List ontologies, optionally filtered by project"),
			mcp.WithString("project_id",
				mcp.Description("Filter by project ID; omit to list all ontologies"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			projectID := req.GetString("project_id", "")
			var (
				ontologies []*models.Ontology
				err        error
			)
			if projectID != "" {
				ontologies, err = m.ontologySvc.GetProjectOntologies(projectID)
			} else {
				data, _ := json.Marshal(map[string]string{
					"message": "Provide project_id to list ontologies for a specific project",
				})
				return mcp.NewToolResultText(string(data)), nil
			}
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			data, _ := json.Marshal(ontologies)
			return mcp.NewToolResultText(string(data)), nil
		},
	)

	// get_ontology
	s.AddTool(
		mcp.NewTool("get_ontology",
			mcp.WithDescription("Get details of a specific ontology by ID"),
			mcp.WithString("id",
				mcp.Required(),
				mcp.Description("Ontology ID"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			id := req.GetString("id", "")
			if id == "" {
				return mcp.NewToolResultError("id is required"), nil
			}
			ontology, err := m.ontologySvc.GetOntology(id)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			data, _ := json.Marshal(ontology)
			return mcp.NewToolResultText(string(data)), nil
		},
	)

	// create_ontology
	s.AddTool(
		mcp.NewTool("create_ontology",
			mcp.WithDescription("Create a new ontology for a project"),
			mcp.WithString("project_id",
				mcp.Required(),
				mcp.Description("Project ID"),
			),
			mcp.WithString("name",
				mcp.Required(),
				mcp.Description("Ontology name"),
			),
			mcp.WithString("content",
				mcp.Required(),
				mcp.Description("OWL/Turtle ontology content as a string"),
			),
			mcp.WithString("description",
				mcp.Description("Optional description"),
			),
			mcp.WithString("version",
				mcp.Description("Ontology version (default 1.0.0)"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			projectID := req.GetString("project_id", "")
			name := req.GetString("name", "")
			content := req.GetString("content", "")
			if projectID == "" || name == "" || content == "" {
				return mcp.NewToolResultError("project_id, name, and content are required"), nil
			}
			version := req.GetString("version", "1.0.0")
			createReq := &models.OntologyCreateRequest{
				ProjectID:   projectID,
				Name:        name,
				Content:     content,
				Description: req.GetString("description", ""),
				Version:     version,
				Status:      "active",
			}
			ontology, err := m.ontologySvc.CreateOntology(createReq)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			data, _ := json.Marshal(ontology)
			return mcp.NewToolResultText(string(data)), nil
		},
	)

	// update_ontology
	s.AddTool(
		mcp.NewTool("update_ontology",
			mcp.WithDescription("Update an existing ontology's metadata or content"),
			mcp.WithString("id",
				mcp.Required(),
				mcp.Description("Ontology ID"),
			),
			mcp.WithString("name",
				mcp.Description("New name"),
			),
			mcp.WithString("description",
				mcp.Description("New description"),
			),
			mcp.WithString("version",
				mcp.Description("New version string e.g. 2.0.0"),
			),
			mcp.WithString("content",
				mcp.Description("Replacement OWL/Turtle content"),
			),
			mcp.WithString("status",
				mcp.Description("New status: active or deprecated"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			id := req.GetString("id", "")
			if id == "" {
				return mcp.NewToolResultError("id is required"), nil
			}
			updateReq := &models.OntologyUpdateRequest{}
			if name := req.GetString("name", ""); name != "" {
				updateReq.Name = &name
			}
			if desc := req.GetString("description", ""); desc != "" {
				updateReq.Description = &desc
			}
			if ver := req.GetString("version", ""); ver != "" {
				updateReq.Version = &ver
			}
			if content := req.GetString("content", ""); content != "" {
				updateReq.Content = &content
			}
			if st := req.GetString("status", ""); st != "" {
				updateReq.Status = &st
			}
			ontology, err := m.ontologySvc.UpdateOntology(id, updateReq)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			data, _ := json.Marshal(ontology)
			return mcp.NewToolResultText(string(data)), nil
		},
	)

	// delete_ontology
	s.AddTool(
		mcp.NewTool("delete_ontology",
			mcp.WithDescription("Delete an ontology by ID"),
			mcp.WithString("id",
				mcp.Required(),
				mcp.Description("Ontology ID"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			id := req.GetString("id", "")
			if id == "" {
				return mcp.NewToolResultError("id is required"), nil
			}
			if err := m.ontologySvc.DeleteOntology(id); err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			return mcp.NewToolResultText(`{"success":true}`), nil
		},
	)

	// generate_ontology_from_text
	s.AddTool(
		mcp.NewTool("generate_ontology_from_text",
			mcp.WithDescription("Generate an OWL ontology by extracting entity types and relationships from a text description"),
			mcp.WithString("project_id",
				mcp.Required(),
				mcp.Description("Project ID"),
			),
			mcp.WithString("name",
				mcp.Required(),
				mcp.Description("Name for the generated ontology"),
			),
			mcp.WithString("text",
				mcp.Required(),
				mcp.Description("Domain description text to extract an ontology from"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			projectID := req.GetString("project_id", "")
			name := req.GetString("name", "")
			text := req.GetString("text", "")
			if projectID == "" || name == "" || text == "" {
				return mcp.NewToolResultError("project_id, name, and text are required"), nil
			}

			// Build a synthetic extraction result from the text using
			// simple heuristics (capitalised word sequences → entity candidates).
			extractionResult := extractEntitiesFromText(text)

			ontology, err := m.ontologySvc.GenerateFromExtraction(projectID, name, extractionResult)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			data, _ := json.Marshal(map[string]any{
				"ontology": ontology,
				"extraction_summary": map[string]int{
					"entities_count":      len(extractionResult.Entities),
					"relationships_count": len(extractionResult.Relationships),
				},
			})
			return mcp.NewToolResultText(string(data)), nil
		},
	)

	// extract_from_storage
	s.AddTool(
		mcp.NewTool("extract_from_storage",
			mcp.WithDescription("Extract entities and relationships from one or more storage backends"),
			mcp.WithString("project_id",
				mcp.Required(),
				mcp.Description("Project ID"),
			),
			mcp.WithString("storage_ids",
				mcp.Required(),
				mcp.Description("Comma-separated list of storage config IDs to extract from"),
			),
			mcp.WithString("include_structured",
				mcp.Description("Include structured data extraction (default true)"),
			),
			mcp.WithString("include_unstructured",
				mcp.Description("Include unstructured/text extraction (default true)"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			projectID := req.GetString("project_id", "")
			storageIDsStr := req.GetString("storage_ids", "")
			if projectID == "" || storageIDsStr == "" {
				return mcp.NewToolResultError("project_id and storage_ids are required"), nil
			}
			storageIDs := splitCSV(storageIDsStr)
			includeStructured := req.GetString("include_structured", "true") != "false"
			includeUnstructured := req.GetString("include_unstructured", "true") != "false"
			result, err := m.extractionSvc.ExtractFromStorage(projectID, storageIDs, includeStructured, includeUnstructured)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			data, _ := json.Marshal(result)
			return mcp.NewToolResultText(string(data)), nil
		},
	)
}

// extractEntitiesFromText builds a minimal ExtractionResult from raw text using
// heuristics: capitalised multi-word phrases become entity candidates.
func extractEntitiesFromText(text string) *models.ExtractionResult {
	seen := make(map[string]bool)
	entities := []models.ExtractedEntity{}

	// Split into sentences / clauses and look for capitalised nouns.
	words := strings.Fields(text)
	var phrase []string
	for i, w := range words {
		clean := strings.Trim(w, ".,;:!?()")
		if len(clean) > 0 && clean[0] >= 'A' && clean[0] <= 'Z' {
			phrase = append(phrase, clean)
		} else {
			if len(phrase) > 0 {
				name := strings.Join(phrase, " ")
				if !seen[name] {
					seen[name] = true
					entities = append(entities, models.ExtractedEntity{
						Name:       name,
						Attributes: map[string]any{"source_index": i},
						Source:     "text",
						Confidence: 0.7,
					})
				}
				phrase = nil
			}
		}
	}
	if len(phrase) > 0 {
		name := strings.Join(phrase, " ")
		if !seen[name] {
			entities = append(entities, models.ExtractedEntity{
				Name:       name,
				Attributes: map[string]any{},
				Source:     "text",
				Confidence: 0.7,
			})
		}
	}

	return &models.ExtractionResult{
		Entities:      entities,
		Relationships: []models.ExtractedRelationship{},
		Source:        "text",
	}
}
