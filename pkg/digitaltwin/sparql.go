package digitaltwin

import (
	"fmt"
	"github.com/mimir-aip/mimir-aip-go/pkg/metadatastore"
	"github.com/mimir-aip/mimir-aip-go/pkg/models"
	"github.com/mimir-aip/mimir-aip-go/pkg/ontology"
	"sort"
	"strings"
)

// SPARQLEngine handles SPARQL queries on digital twin data
type SPARQLEngine struct {
	store           metadatastore.MetadataStore
	ontologyService *ontology.Service
}

// NewSPARQLEngine creates a new SPARQL engine
func NewSPARQLEngine(store metadatastore.MetadataStore, ontologyService *ontology.Service) *SPARQLEngine {
	return &SPARQLEngine{
		store:           store,
		ontologyService: ontologyService,
	}
}

// Execute executes a SPARQL SELECT query against the digital twin's entities
func (e *SPARQLEngine) Execute(twin *models.DigitalTwin, req *models.QueryRequest) (*models.QueryResult, error) {
	query := strings.TrimSpace(req.Query)

	if !strings.HasPrefix(strings.ToUpper(query), "SELECT") &&
		!strings.HasPrefix(strings.ToUpper(query), "PREFIX") {
		return nil, fmt.Errorf("only SELECT queries are supported")
	}

	// Get all entities for this digital twin
	entities, err := e.store.ListEntitiesByDigitalTwin(twin.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get entities: %w", err)
	}

	// Parse query. Unsupported or malformed queries must fail explicitly; returning
	// plausible entity listings for invalid queries hides ontology/query mistakes.
	tokens := tokenizeSPARQL(query)
	parsedQuery, err := parseSPARQL(tokens)
	if err != nil {
		return nil, fmt.Errorf("invalid supported SPARQL subset query: %w", err)
	}

	// Override limit from request if provided and no LIMIT in query
	if req.Limit > 0 && parsedQuery.Limit == 0 {
		parsedQuery.Limit = req.Limit
	}
	if req.Offset > 0 && parsedQuery.Offset == 0 {
		parsedQuery.Offset = req.Offset
	}

	// Evaluate the query
	rows := evaluateSPARQL(parsedQuery, entities)

	// Extract columns from SELECT variables (includes aggregate aliases) or first row.
	columns := parsedQuery.Variables
	if len(columns) == 0 && len(rows) > 0 {
		for k := range rows[0] {
			columns = append(columns, k)
		}
		sort.Strings(columns)
	}

	return &models.QueryResult{
		Columns:  columns,
		Rows:     rows,
		Count:    len(rows),
		Metadata: map[string]interface{}{"query_type": "sparql"},
	}, nil
}
