package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/Mimir-AIP/Mimir-AIP-Go/utils"
)

// ReasoningRequest represents a reasoning request
type ReasoningRequest struct {
	OntologyID string   `json:"ontology_id,omitempty"`
	Rules      []string `json:"rules"` // rdfs:subClassOf, rdfs:domain, rdfs:range, owl:inverseOf, owl:transitiveProperty, owl:symmetricProperty
	MaxDepth   int      `json:"max_depth,omitempty"`
}

// ReasoningResult represents the reasoning results
type ReasoningResult struct {
	AssertedTriples int              `json:"asserted_triples"`
	InferredTriples int              `json:"inferred_triples"`
	TotalTriples    int              `json:"total_triples"`
	RulesApplied    []string         `json:"rules_applied"`
	Inferences      []InferredTriple `json:"inferences,omitempty"`
	ExecutionTimeMS int64            `json:"execution_time_ms"`
	Statistics      map[string]int   `json:"statistics"`
}

// InferredTriple represents an inferred triple with its justification
type InferredTriple struct {
	Subject       string `json:"subject"`
	Predicate     string `json:"predicate"`
	Object        string `json:"object"`
	Rule          string `json:"rule"`
	Justification string `json:"justification,omitempty"`
}

// handleReasoning performs OWL/RDFS reasoning on the knowledge graph
func (s *Server) handleReasoning(w http.ResponseWriter, r *http.Request) {
	log := utils.GetLogger()
	ctx := r.Context()
	startTime := time.Now()

	// Parse request
	var req ReasoningRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate request
	if len(req.Rules) == 0 {
		http.Error(w, "At least one reasoning rule must be specified", http.StatusBadRequest)
		return
	}

	// Set defaults
	if req.MaxDepth == 0 {
		req.MaxDepth = 10
	}
	if req.MaxDepth > 100 {
		req.MaxDepth = 100
	}

	log.Info("Starting reasoning", utils.Component("reasoning"))

	// Count asserted triples
	assertedCount, err := s.countTriples(ctx, req.OntologyID)
	if err != nil {
		log.Error("Failed to count asserted triples", err, utils.Component("reasoning"))
		assertedCount = 0
	}

	// Apply reasoning rules
	inferences := []InferredTriple{}
	rulesApplied := []string{}
	statistics := make(map[string]int)

	for _, rule := range req.Rules {
		ruleInferences, err := s.applyReasoningRule(ctx, rule, req.OntologyID, req.MaxDepth)
		if err != nil {
			log.Warn(fmt.Sprintf("Failed to apply rule %s: %v", rule, err))
			continue
		}
		inferences = append(inferences, ruleInferences...)
		rulesApplied = append(rulesApplied, rule)
		statistics[rule] = len(ruleInferences)
	}

	// Count total triples after reasoning
	totalCount := assertedCount + len(inferences)

	// Build response
	result := ReasoningResult{
		AssertedTriples: assertedCount,
		InferredTriples: len(inferences),
		TotalTriples:    totalCount,
		RulesApplied:    rulesApplied,
		Inferences:      inferences,
		ExecutionTimeMS: time.Since(startTime).Milliseconds(),
		Statistics:      statistics,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// countTriples counts the number of triples in the knowledge graph
// If ontologyID is provided, counts only triples in that ontology's graph
func (s *Server) countTriples(ctx context.Context, ontologyID ...string) (int, error) {
	var query string
	if len(ontologyID) > 0 && ontologyID[0] != "" {
		// Count triples in specific ontology graph
		graphURI := fmt.Sprintf("http://mimir.ai/ontology/%s", ontologyID[0])
		query = fmt.Sprintf(`SELECT (COUNT(*) AS ?count) WHERE { GRAPH <%s> { ?s ?p ?o } }`, graphURI)
	} else {
		// Count triples across all named graphs
		query = `SELECT (COUNT(*) AS ?count) WHERE { GRAPH ?g { ?s ?p ?o } }`
	}

	result, err := s.tdb2Backend.QuerySPARQL(ctx, query)
	if err != nil {
		return 0, err
	}

	if len(result.Bindings) > 0 {
		if countVal, ok := result.Bindings[0]["count"]; ok {
			var count int
			fmt.Sscanf(countVal.Value, "%d", &count)
			return count, nil
		}
	}

	return 0, nil
}

// applyReasoningRule applies a specific reasoning rule
func (s *Server) applyReasoningRule(ctx context.Context, rule string, ontologyID string, maxDepth int) ([]InferredTriple, error) {
	log := utils.GetLogger()

	switch rule {
	case "rdfs:subClassOf":
		return s.applySubClassOfReasoning(ctx, ontologyID, maxDepth)
	case "rdfs:domain":
		return s.applyDomainReasoning(ctx, ontologyID)
	case "rdfs:range":
		return s.applyRangeReasoning(ctx, ontologyID)
	case "owl:transitiveProperty":
		return s.applyTransitivePropertyReasoning(ctx, ontologyID, maxDepth)
	case "owl:symmetricProperty":
		return s.applySymmetricPropertyReasoning(ctx, ontologyID)
	case "owl:inverseOf":
		return s.applyInverseOfReasoning(ctx, ontologyID)
	default:
		log.Warn(fmt.Sprintf("Unknown reasoning rule: %s", rule))
		return []InferredTriple{}, nil
	}
}

// applySubClassOfReasoning infers types based on subclass hierarchy
func (s *Server) applySubClassOfReasoning(ctx context.Context, ontologyID string, maxDepth int) ([]InferredTriple, error) {
	// Build GRAPH clause
	var graphClause string
	if ontologyID != "" {
		graphURI := fmt.Sprintf("http://mimir.ai/ontology/%s", ontologyID)
		graphClause = fmt.Sprintf("GRAPH <%s>", graphURI)
	} else {
		graphClause = "GRAPH ?g"
	}

	query := fmt.Sprintf(`
	SELECT ?instance ?subClass ?superClass WHERE {
		%s {
			?instance rdf:type ?subClass .
			?subClass rdfs:subClassOf ?superClass .
			FILTER(?subClass != ?superClass)
		}
	}
	LIMIT 1000
	`, graphClause)

	result, err := s.tdb2Backend.QuerySPARQL(ctx, query)
	if err != nil {
		return nil, err
	}

	inferences := []InferredTriple{}
	for _, binding := range result.Bindings {
		instance := binding["instance"].Value
		superClass := binding["superClass"].Value

		inference := InferredTriple{
			Subject:       instance,
			Predicate:     "rdf:type",
			Object:        superClass,
			Rule:          "rdfs:subClassOf",
			Justification: fmt.Sprintf("Instance of subclass implies instance of superclass"),
		}
		inferences = append(inferences, inference)
	}

	return inferences, nil
}

// applyDomainReasoning infers types based on rdfs:domain
func (s *Server) applyDomainReasoning(ctx context.Context, ontologyID string) ([]InferredTriple, error) {
	// Build GRAPH clause
	var graphClause string
	if ontologyID != "" {
		graphURI := fmt.Sprintf("http://mimir.ai/ontology/%s", ontologyID)
		graphClause = fmt.Sprintf("GRAPH <%s>", graphURI)
	} else {
		graphClause = "GRAPH ?g"
	}

	query := fmt.Sprintf(`
	SELECT ?subject ?property ?domain WHERE {
		%s {
			?subject ?property ?object .
			?property rdfs:domain ?domain .
		}
	}
	LIMIT 1000
	`, graphClause)

	result, err := s.tdb2Backend.QuerySPARQL(ctx, query)
	if err != nil {
		return nil, err
	}

	inferences := []InferredTriple{}
	seen := make(map[string]bool)

	for _, binding := range result.Bindings {
		subject := binding["subject"].Value
		domain := binding["domain"].Value
		key := subject + "|" + domain

		if !seen[key] {
			inference := InferredTriple{
				Subject:       subject,
				Predicate:     "rdf:type",
				Object:        domain,
				Rule:          "rdfs:domain",
				Justification: "Subject of property with domain constraint",
			}
			inferences = append(inferences, inference)
			seen[key] = true
		}
	}

	return inferences, nil
}

// applyRangeReasoning infers types based on rdfs:range
func (s *Server) applyRangeReasoning(ctx context.Context, ontologyID string) ([]InferredTriple, error) {
	// Build GRAPH clause
	var graphClause string
	if ontologyID != "" {
		graphURI := fmt.Sprintf("http://mimir.ai/ontology/%s", ontologyID)
		graphClause = fmt.Sprintf("GRAPH <%s>", graphURI)
	} else {
		graphClause = "GRAPH ?g"
	}

	query := fmt.Sprintf(`
	SELECT ?object ?property ?range WHERE {
		%s {
			?subject ?property ?object .
			?property rdfs:range ?range .
			FILTER(isURI(?object))
		}
	}
	LIMIT 1000
	`, graphClause)

	result, err := s.tdb2Backend.QuerySPARQL(ctx, query)
	if err != nil {
		return nil, err
	}

	inferences := []InferredTriple{}
	seen := make(map[string]bool)

	for _, binding := range result.Bindings {
		object := binding["object"].Value
		rangeType := binding["range"].Value
		key := object + "|" + rangeType

		if !seen[key] {
			inference := InferredTriple{
				Subject:       object,
				Predicate:     "rdf:type",
				Object:        rangeType,
				Rule:          "rdfs:range",
				Justification: "Object of property with range constraint",
			}
			inferences = append(inferences, inference)
			seen[key] = true
		}
	}

	return inferences, nil
}

// applyTransitivePropertyReasoning infers transitive relationships
func (s *Server) applyTransitivePropertyReasoning(ctx context.Context, ontologyID string, maxDepth int) ([]InferredTriple, error) {
	// Build GRAPH clause
	var graphClause string
	if ontologyID != "" {
		graphURI := fmt.Sprintf("http://mimir.ai/ontology/%s", ontologyID)
		graphClause = fmt.Sprintf("GRAPH <%s>", graphURI)
	} else {
		graphClause = "GRAPH ?g"
	}

	// Find all transitive properties
	propsQuery := fmt.Sprintf(`
	SELECT ?property WHERE {
		%s {
			?property rdf:type owl:TransitiveProperty .
		}
	}
	`, graphClause)

	propsResult, err := s.tdb2Backend.QuerySPARQL(ctx, propsQuery)
	if err != nil {
		return nil, err
	}

	if len(propsResult.Bindings) == 0 {
		return []InferredTriple{}, nil
	}

	inferences := []InferredTriple{}

	// For each transitive property, find transitive chains
	for _, propBinding := range propsResult.Bindings {
		property := propBinding["property"].Value
		propertyLocal := extractLocalNameReasoning(property)

		query := fmt.Sprintf(`
		SELECT ?x ?z WHERE {
			%s {
				?x <%s> ?y .
				?y <%s> ?z .
			}
		}
		LIMIT 500
		`, graphClause, property, property)

		result, err := s.tdb2Backend.QuerySPARQL(ctx, query)
		if err != nil {
			continue
		}

		seen := make(map[string]bool)
		for _, binding := range result.Bindings {
			x := binding["x"].Value
			z := binding["z"].Value
			key := x + "|" + property + "|" + z

			if !seen[key] {
				inference := InferredTriple{
					Subject:       x,
					Predicate:     propertyLocal,
					Object:        z,
					Rule:          "owl:transitiveProperty",
					Justification: fmt.Sprintf("Transitive closure of %s", propertyLocal),
				}
				inferences = append(inferences, inference)
				seen[key] = true
			}
		}
	}

	return inferences, nil
}

// applySymmetricPropertyReasoning infers symmetric relationships
func (s *Server) applySymmetricPropertyReasoning(ctx context.Context, ontologyID string) ([]InferredTriple, error) {
	// Build GRAPH clause
	var graphClause string
	if ontologyID != "" {
		graphURI := fmt.Sprintf("http://mimir.ai/ontology/%s", ontologyID)
		graphClause = fmt.Sprintf("GRAPH <%s>", graphURI)
	} else {
		graphClause = "GRAPH ?g"
	}

	// Find all symmetric properties
	propsQuery := fmt.Sprintf(`
	SELECT ?property WHERE {
		%s {
			?property rdf:type owl:SymmetricProperty .
		}
	}
	`, graphClause)

	propsResult, err := s.tdb2Backend.QuerySPARQL(ctx, propsQuery)
	if err != nil {
		return nil, err
	}

	if len(propsResult.Bindings) == 0 {
		return []InferredTriple{}, nil
	}

	inferences := []InferredTriple{}

	// For each symmetric property, infer reverse relationships
	for _, propBinding := range propsResult.Bindings {
		property := propBinding["property"].Value
		propertyLocal := extractLocalNameReasoning(property)

		query := fmt.Sprintf(`
		SELECT ?x ?y WHERE {
			%s {
				?x <%s> ?y .
			}
		}
		LIMIT 500
		`, graphClause, property)

		result, err := s.tdb2Backend.QuerySPARQL(ctx, query)
		if err != nil {
			continue
		}

		for _, binding := range result.Bindings {
			x := binding["x"].Value
			y := binding["y"].Value

			inference := InferredTriple{
				Subject:       y,
				Predicate:     propertyLocal,
				Object:        x,
				Rule:          "owl:symmetricProperty",
				Justification: fmt.Sprintf("Symmetric property %s", propertyLocal),
			}
			inferences = append(inferences, inference)
		}
	}

	return inferences, nil
}

// applyInverseOfReasoning infers inverse relationships
func (s *Server) applyInverseOfReasoning(ctx context.Context, ontologyID string) ([]InferredTriple, error) {
	// Build GRAPH clause
	var graphClause string
	if ontologyID != "" {
		graphURI := fmt.Sprintf("http://mimir.ai/ontology/%s", ontologyID)
		graphClause = fmt.Sprintf("GRAPH <%s>", graphURI)
	} else {
		graphClause = "GRAPH ?g"
	}

	query := fmt.Sprintf(`
	SELECT ?property ?inverseProperty ?x ?y WHERE {
		%s {
			?property owl:inverseOf ?inverseProperty .
			?x ?property ?y .
		}
	}
	LIMIT 500
	`, graphClause)

	result, err := s.tdb2Backend.QuerySPARQL(ctx, query)
	if err != nil {
		return nil, err
	}

	inferences := []InferredTriple{}

	for _, binding := range result.Bindings {
		inverseProp := binding["inverseProperty"].Value
		inversePropLocal := extractLocalName(inverseProp)
		x := binding["x"].Value
		y := binding["y"].Value
		property := binding["property"].Value
		propertyLocal := extractLocalNameReasoning(property)

		inference := InferredTriple{
			Subject:       y,
			Predicate:     inversePropLocal,
			Object:        x,
			Rule:          "owl:inverseOf",
			Justification: fmt.Sprintf("Inverse of %s", propertyLocal),
		}
		inferences = append(inferences, inference)
	}

	return inferences, nil
}

// extractLocalNameReasoning extracts local name from URI
func extractLocalNameReasoning(uri string) string {
	// Try splitting by #
	if idx := strings.LastIndex(uri, "#"); idx != -1 {
		return uri[idx+1:]
	}
	// Try splitting by /
	if idx := strings.LastIndex(uri, "/"); idx != -1 {
		return uri[idx+1:]
	}
	return uri
}
