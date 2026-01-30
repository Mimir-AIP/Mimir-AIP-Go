package knowledgegraph

import (
	"fmt"
	"strings"
)

// GetOntologyGraphURI returns the named graph URI for an ontology
func GetOntologyGraphURI(ontologyID string) string {
	return fmt.Sprintf("http://mimir.ai/ontology/%s", ontologyID)
}

// WrapQueryInGraph wraps a SPARQL query pattern in a GRAPH clause
// Example:
//
//	pattern := "?s ?p ?o"
//	graphURI := "http://mimir.ai/ontology/ont_123"
//	result := WrapQueryInGraph(pattern, graphURI)
//	// Result: "GRAPH <http://mimir.ai/ontology/ont_123> { ?s ?p ?o }"
func WrapQueryInGraph(pattern string, graphURI string) string {
	return fmt.Sprintf("GRAPH <%s> { %s }", graphURI, pattern)
}

// WrapQueryInAllGraphs wraps a SPARQL query pattern to query all named graphs
// Example:
//
//	pattern := "?s ?p ?o"
//	result := WrapQueryInAllGraphs(pattern)
//	// Result: "GRAPH ?g { ?s ?p ?o }"
func WrapQueryInAllGraphs(pattern string) string {
	return fmt.Sprintf("GRAPH ?g { %s }", pattern)
}

// ValidateSPARQLQuery performs basic validation of SPARQL queries
// Returns error if query appears to be missing GRAPH clause
func ValidateSPARQLQuery(query string) error {
	queryUpper := strings.ToUpper(query)

	// Check if it's a SELECT/CONSTRUCT/ASK query
	isQuery := strings.Contains(queryUpper, "SELECT") ||
		strings.Contains(queryUpper, "CONSTRUCT") ||
		strings.Contains(queryUpper, "ASK")

	if !isQuery {
		return nil // Not a query we need to validate
	}

	// Check if it has a WHERE clause
	hasWhere := strings.Contains(queryUpper, "WHERE")
	if !hasWhere {
		return nil // No WHERE clause, probably fine
	}

	// Check if it has GRAPH clause
	hasGraph := strings.Contains(queryUpper, "GRAPH")
	if !hasGraph {
		return fmt.Errorf("SPARQL query appears to be missing GRAPH clause - " +
			"all queries in Mimir AIP must specify which ontology graph to query. " +
			"Add 'GRAPH <http://mimir.ai/ontology/{ontology_id}> { ... }' or " +
			"'GRAPH ?g { ... }' to query all graphs")
	}

	return nil
}

// QueryOntologyEntities is a helper to query entities from a specific ontology
// This enforces the GRAPH clause pattern and prevents common mistakes
func (t *TDB2Backend) QueryOntologyEntities(ontologyID string, entityType string) (*QueryResult, error) {
	graphURI := GetOntologyGraphURI(ontologyID)

	query := fmt.Sprintf(`
		PREFIX rdfs: <http://www.w3.org/2000/01/rdf-schema#>
		SELECT ?entity ?label WHERE {
			GRAPH <%s> {
				?entity a <%s> .
				OPTIONAL { ?entity rdfs:label ?label }
			}
		}
	`, graphURI, entityType)

	return t.QuerySPARQL(nil, query)
}

// CountTriplesInGraph counts triples in a specific ontology graph
func (t *TDB2Backend) CountTriplesInGraph(ontologyID string) (int, error) {
	graphURI := GetOntologyGraphURI(ontologyID)

	query := fmt.Sprintf(`
		SELECT (COUNT(*) AS ?count) WHERE {
			GRAPH <%s> {
				?s ?p ?o
			}
		}
	`, graphURI)

	result, err := t.QuerySPARQL(nil, query)
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
