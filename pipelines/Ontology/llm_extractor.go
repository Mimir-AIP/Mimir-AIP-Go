package ontology

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	AI "github.com/Mimir-AIP/Mimir-AIP-Go/pipelines/AI"
)

// LLMExtractor extracts entities from unstructured text using LLM
type LLMExtractor struct {
	config    ExtractionConfig
	llmClient AI.LLMClient
}

// NewLLMExtractor creates a new LLM-based extractor
func NewLLMExtractor(config ExtractionConfig, llmClient AI.LLMClient) *LLMExtractor {
	return &LLMExtractor{
		config:    config,
		llmClient: llmClient,
	}
}

// Extract extracts entities from unstructured data using LLM
func (e *LLMExtractor) Extract(data any, ontology *OntologyContext) (*ExtractionResult, error) {
	text, ok := data.(string)
	if !ok {
		return nil, fmt.Errorf("LLM extractor requires string input")
	}

	if text == "" {
		return &ExtractionResult{
			ExtractionType: ExtractionLLM,
			Confidence:     1.0,
		}, nil
	}

	// Build extraction prompt with ontology context
	prompt := e.buildExtractionPrompt(text, ontology)

	// Call LLM
	ctx := context.Background()
	response, err := e.llmClient.CompleteSimple(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("LLM generation failed: %w", err)
	}

	// Parse LLM response
	result, err := e.parseExtractionResponse(response, ontology)
	if err != nil {
		return nil, fmt.Errorf("failed to parse LLM response: %w", err)
	}

	result.ExtractionType = ExtractionLLM
	return result, nil
}

// GetType returns the extraction type
func (e *LLMExtractor) GetType() ExtractionType {
	return ExtractionLLM
}

// GetSupportedSourceTypes returns supported source types
func (e *LLMExtractor) GetSupportedSourceTypes() []string {
	return []string{SourceTypeText, SourceTypeHTML}
}

// buildExtractionPrompt constructs a prompt for entity extraction
func (e *LLMExtractor) buildExtractionPrompt(text string, ontology *OntologyContext) string {
	var sb strings.Builder

	sb.WriteString("You are an expert in entity extraction and knowledge graph construction.\n\n")
	sb.WriteString("# Task\n")
	sb.WriteString("Extract entities and their relationships from the provided text according to the given ontology.\n\n")

	// Add ontology context
	sb.WriteString("# Ontology Context\n")
	sb.WriteString(fmt.Sprintf("Base URI: %s\n\n", ontology.BaseURI))

	// Add classes
	if len(ontology.Classes) > 0 {
		sb.WriteString("## Available Classes (Entity Types):\n")
		for _, class := range ontology.Classes {
			sb.WriteString(fmt.Sprintf("- %s", class.URI))
			if class.Label != "" {
				sb.WriteString(fmt.Sprintf(" (Label: %s)", class.Label))
			}
			if class.Description != "" {
				sb.WriteString(fmt.Sprintf(" - %s", class.Description))
			}
			sb.WriteString("\n")
		}
		sb.WriteString("\n")
	}

	// Add properties
	if len(ontology.Properties) > 0 {
		sb.WriteString("## Available Properties (Relationships):\n")
		for _, prop := range ontology.Properties {
			sb.WriteString(fmt.Sprintf("- %s", prop.URI))
			if prop.Label != "" {
				sb.WriteString(fmt.Sprintf(" (Label: %s)", prop.Label))
			}
			if prop.Description != "" {
				sb.WriteString(fmt.Sprintf(" - %s", prop.Description))
			}
			if len(prop.Domain) > 0 || len(prop.Range) > 0 {
				sb.WriteString(fmt.Sprintf(" [Domain: %s, Range: %s]", strings.Join(prop.Domain, ", "), strings.Join(prop.Range, ", ")))
			}
			sb.WriteString("\n")
		}
		sb.WriteString("\n")
	}

	// Add input text
	sb.WriteString("# Input Text\n")
	sb.WriteString(text)
	sb.WriteString("\n\n")

	// Add output format instructions
	sb.WriteString("# Output Format\n")
	sb.WriteString("Return a JSON object with the following structure:\n")
	sb.WriteString("{\n")
	sb.WriteString("  \"entities\": [\n")
	sb.WriteString("    {\n")
	sb.WriteString("      \"uri\": \"<base_uri>/entity_<id>\",\n")
	sb.WriteString("      \"type\": \"<class_uri>\",\n")
	sb.WriteString("      \"label\": \"<human_readable_name>\",\n")
	sb.WriteString("      \"confidence\": 0.0-1.0,\n")
	sb.WriteString("      \"properties\": {\n")
	sb.WriteString("        \"<property_uri>\": \"<value>\"\n")
	sb.WriteString("      }\n")
	sb.WriteString("    }\n")
	sb.WriteString("  ],\n")
	sb.WriteString("  \"triples\": [\n")
	sb.WriteString("    {\n")
	sb.WriteString("      \"subject\": \"<entity_uri>\",\n")
	sb.WriteString("      \"predicate\": \"<property_uri>\",\n")
	sb.WriteString("      \"object\": \"<value_or_uri>\",\n")
	sb.WriteString("      \"datatype\": \"<xsd_datatype>\" // optional\n")
	sb.WriteString("    }\n")
	sb.WriteString("  ],\n")
	sb.WriteString("  \"warnings\": [] // optional array of string warnings\n")
	sb.WriteString("}\n\n")

	sb.WriteString("# Important Instructions\n")
	sb.WriteString("- Only extract entities that match the provided ontology classes\n")
	sb.WriteString("- Only use properties defined in the ontology\n")
	sb.WriteString("- Use full URIs for types and properties\n")
	sb.WriteString("- Generate unique entity URIs based on the base URI\n")
	sb.WriteString("- Include a confidence score (0.0-1.0) for each entity\n")
	sb.WriteString("- Always add an rdf:type triple for each entity\n")
	sb.WriteString("- Return ONLY valid JSON, no additional text\n")

	return sb.String()
}

// parseExtractionResponse parses the LLM's JSON response into an ExtractionResult
func (e *LLMExtractor) parseExtractionResponse(response string, ontology *OntologyContext) (*ExtractionResult, error) {
	// Clean up response (remove markdown code blocks if present)
	response = strings.TrimSpace(response)
	if strings.HasPrefix(response, "```json") {
		response = strings.TrimPrefix(response, "```json")
		response = strings.TrimSuffix(response, "```")
		response = strings.TrimSpace(response)
	} else if strings.HasPrefix(response, "```") {
		response = strings.TrimPrefix(response, "```")
		response = strings.TrimSuffix(response, "```")
		response = strings.TrimSpace(response)
	}

	// Parse JSON
	var llmResult struct {
		Entities []struct {
			URI        string         `json:"uri"`
			Type       string         `json:"type"`
			Label      string         `json:"label"`
			Confidence float64        `json:"confidence"`
			Properties map[string]any `json:"properties"`
		} `json:"entities"`
		Triples []struct {
			Subject   string `json:"subject"`
			Predicate string `json:"predicate"`
			Object    string `json:"object"`
			Datatype  string `json:"datatype"`
		} `json:"triples"`
		Warnings []string `json:"warnings"`
	}

	if err := json.Unmarshal([]byte(response), &llmResult); err != nil {
		return nil, fmt.Errorf("failed to parse JSON response: %w (response: %s)", err, response)
	}

	// Convert to ExtractionResult
	result := &ExtractionResult{
		Entities:       make([]Entity, 0, len(llmResult.Entities)),
		Triples:        make([]Triple, 0, len(llmResult.Triples)),
		ExtractionType: ExtractionLLM,
		Warnings:       llmResult.Warnings,
	}

	// Process entities
	totalConfidence := 0.0
	for _, ent := range llmResult.Entities {
		entity := Entity{
			URI:        ent.URI,
			Type:       ent.Type,
			Label:      ent.Label,
			Properties: ent.Properties,
			Confidence: ent.Confidence,
		}

		// Validate entity has required fields
		if entity.URI == "" || entity.Type == "" {
			result.Warnings = append(result.Warnings, fmt.Sprintf("Skipping entity with missing URI or Type: %+v", ent))
			continue
		}

		result.Entities = append(result.Entities, entity)
		totalConfidence += ent.Confidence
	}

	// Process triples
	for _, t := range llmResult.Triples {
		triple := Triple{
			Subject:   t.Subject,
			Predicate: t.Predicate,
			Object:    t.Object,
			Datatype:  t.Datatype,
		}

		// Validate triple
		if triple.Subject == "" || triple.Predicate == "" || triple.Object == "" {
			result.Warnings = append(result.Warnings, fmt.Sprintf("Skipping triple with empty fields: %+v", t))
			continue
		}

		result.Triples = append(result.Triples, triple)
	}

	// Calculate average confidence
	if len(result.Entities) > 0 {
		result.Confidence = totalConfidence / float64(len(result.Entities))
	} else {
		result.Confidence = 0.0
	}

	result.EntitiesExtracted = len(result.Entities)
	result.TriplesGenerated = len(result.Triples)

	return result, nil
}

// ValidateWithOntology validates extracted entities against the ontology
func (e *LLMExtractor) ValidateWithOntology(result *ExtractionResult, ontology *OntologyContext) []string {
	var warnings []string

	// Build quick lookup maps
	validClasses := make(map[string]bool)
	for _, class := range ontology.Classes {
		validClasses[class.URI] = true
	}

	validProperties := make(map[string]bool)
	for _, prop := range ontology.Properties {
		validProperties[prop.URI] = true
	}

	// Validate entities
	for _, entity := range result.Entities {
		// Check if entity type is in ontology
		if !validClasses[entity.Type] {
			warnings = append(warnings, fmt.Sprintf("Entity %s has unknown type: %s", entity.URI, entity.Type))
		}

		// Check if properties are in ontology
		for propURI := range entity.Properties {
			if !validProperties[propURI] {
				warnings = append(warnings, fmt.Sprintf("Entity %s uses unknown property: %s", entity.URI, propURI))
			}
		}
	}

	// Validate triples
	for _, triple := range result.Triples {
		// Check if predicate is in ontology (except rdf:type)
		if triple.Predicate != "http://www.w3.org/1999/02/22-rdf-syntax-ns#type" &&
			!validProperties[triple.Predicate] {
			warnings = append(warnings, fmt.Sprintf("Triple uses unknown predicate: %s", triple.Predicate))
		}
	}

	return warnings
}
