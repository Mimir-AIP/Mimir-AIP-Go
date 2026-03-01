package digitaltwin

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/mimir-aip/mimir-aip-go/pkg/metadatastore"
	"github.com/mimir-aip/mimir-aip-go/pkg/models"
	"github.com/mimir-aip/mimir-aip-go/pkg/ontology"
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

	// Parse query
	tokens := tokenizeSPARQL(query)
	parsedQuery, err := parseSPARQL(tokens)
	if err != nil {
		// Fall back to simple listing on parse failure
		return e.simpleEntityListing(entities, req.Limit), nil
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

// simpleEntityListing returns a simplified result when SPARQL parsing fails
func (e *SPARQLEngine) simpleEntityListing(entities []*models.Entity, limit int) *models.QueryResult {
	rows := make([]map[string]interface{}, 0)
	for _, entity := range entities {
		row := map[string]interface{}{
			"entity_id":   entity.ID,
			"entity_type": entity.Type,
		}
		for k, v := range entity.Attributes {
			row[k] = v
		}
		rows = append(rows, row)
		if limit > 0 && len(rows) >= limit {
			break
		}
	}

	columns := []string{"entity_id", "entity_type"}
	if len(rows) > 0 {
		for k := range rows[0] {
			if k != "entity_id" && k != "entity_type" {
				columns = append(columns, k)
			}
		}
	}

	return &models.QueryResult{
		Columns:  columns,
		Rows:     rows,
		Count:    len(rows),
		Metadata: map[string]interface{}{"query_type": "simple_listing"},
	}
}

// ─── Tokenizer ───────────────────────────────────────────────────────────────

type tokenType int

const (
	tokKeyword   tokenType = iota // SELECT, WHERE, FILTER, ORDER, BY, LIMIT, OFFSET, ASC, DESC, PREFIX, OPTIONAL
	tokVariable                   // ?varName
	tokURI                        // :localName or <full-uri>
	tokLiteral                    // "string"
	tokNumber                     // 42, 3.14
	tokPunct                      // { } ( ) . , ; =
	tokDot                        // .
	tokEOF
)

type token struct {
	typ tokenType
	val string
}

var sparqlKeywords = map[string]bool{
	"SELECT": true, "WHERE": true, "FILTER": true, "ORDER": true, "BY": true,
	"LIMIT": true, "OFFSET": true, "ASC": true, "DESC": true, "PREFIX": true,
	"OPTIONAL": true, "FROM": true, "DISTINCT": true, "A": true,
	"GROUP": true, "HAVING": true,
	"COUNT": true, "SUM": true, "AVG": true, "MIN": true, "MAX": true, "AS": true,
}

func tokenizeSPARQL(query string) []token {
	var tokens []token
	i := 0
	runes := []rune(query)
	n := len(runes)

	for i < n {
		// Skip whitespace
		if runes[i] == ' ' || runes[i] == '\t' || runes[i] == '\n' || runes[i] == '\r' {
			i++
			continue
		}
		// Comment
		if runes[i] == '#' {
			for i < n && runes[i] != '\n' {
				i++
			}
			continue
		}
		// Variable: ?name
		if runes[i] == '?' {
			i++
			start := i
			for i < n && (isAlphaNum(runes[i]) || runes[i] == '_') {
				i++
			}
			tokens = append(tokens, token{tokVariable, string(runes[start:i])})
			continue
		}
		// Prefixed URI: :localName or prefix:local
		if runes[i] == ':' || (i+1 < n && isAlpha(runes[i]) && runes[i+1] == ':') {
			if runes[i] == ':' {
				i++ // skip leading colon
				start := i
				for i < n && (isAlphaNum(runes[i]) || runes[i] == '_' || runes[i] == '-') {
					i++
				}
				tokens = append(tokens, token{tokURI, string(runes[start:i])})
			} else {
				// prefix:local
				start := i
				for i < n && runes[i] != ' ' && runes[i] != '\t' && runes[i] != '\n' && runes[i] != '.' && runes[i] != ';' && runes[i] != ')' && runes[i] != '}' {
					i++
				}
				tokens = append(tokens, token{tokURI, string(runes[start:i])})
			}
			continue
		}
		// Full URI: <uri>
		if runes[i] == '<' {
			i++
			start := i
			for i < n && runes[i] != '>' {
				i++
			}
			uri := string(runes[start:i])
			if i < n {
				i++ // skip >
			}
			tokens = append(tokens, token{tokURI, uri})
			continue
		}
		// String literal: "..."
		if runes[i] == '"' {
			i++
			start := i
			for i < n && runes[i] != '"' {
				if runes[i] == '\\' {
					i++ // skip escape
				}
				i++
			}
			lit := string(runes[start:i])
			if i < n {
				i++ // skip closing "
			}
			tokens = append(tokens, token{tokLiteral, lit})
			continue
		}
		// Number
		if isDigit(runes[i]) || (runes[i] == '-' && i+1 < n && isDigit(runes[i+1])) {
			start := i
			if runes[i] == '-' {
				i++
			}
			for i < n && (isDigit(runes[i]) || runes[i] == '.') {
				i++
			}
			tokens = append(tokens, token{tokNumber, string(runes[start:i])})
			continue
		}
		// Punctuation
		if runes[i] == '{' || runes[i] == '}' || runes[i] == '(' || runes[i] == ')' ||
			runes[i] == ',' || runes[i] == ';' {
			tokens = append(tokens, token{tokPunct, string(runes[i])})
			i++
			continue
		}
		if runes[i] == '.' {
			tokens = append(tokens, token{tokDot, "."})
			i++
			continue
		}
		// Comparison operators
		if runes[i] == '>' || runes[i] == '<' || runes[i] == '=' || runes[i] == '!' {
			op := string(runes[i])
			if i+1 < n && runes[i+1] == '=' {
				op += "="
				i++
			}
			tokens = append(tokens, token{tokPunct, op})
			i++
			continue
		}
		// Keyword or identifier
		if isAlpha(runes[i]) || runes[i] == '_' {
			start := i
			for i < n && (isAlphaNum(runes[i]) || runes[i] == '_') {
				i++
			}
			word := string(runes[start:i])
			upper := strings.ToUpper(word)
			if sparqlKeywords[upper] {
				tokens = append(tokens, token{tokKeyword, upper})
			} else {
				// Could be a bare URI local name after a colon was separate,
				// or a type name in a rdf:type shorthand 'a'
				if upper == "A" {
					tokens = append(tokens, token{tokKeyword, "A"})
				} else {
					tokens = append(tokens, token{tokURI, word})
				}
			}
			continue
		}
		i++ // skip unrecognized char
	}

	tokens = append(tokens, token{tokEOF, ""})
	return tokens
}

func isAlpha(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
}
func isDigit(r rune) bool  { return r >= '0' && r <= '9' }
func isAlphaNum(r rune) bool { return isAlpha(r) || isDigit(r) }

// ─── Parser ───────────────────────────────────────────────────────────────────

// TriplePattern represents a single triple in the WHERE clause
type TriplePattern struct {
	Subject   string   // variable name (without ?) or literal
	Predicate string   // local name (e.g. "age", "a")
	Object    string   // variable name or literal
	IsVar     [3]bool  // which positions are variables
}

// FilterExpr represents a FILTER clause
type FilterExpr struct {
	Variable string
	Operator string      // gt, gte, lt, lte, eq, ne
	Value    interface{} // float64 or string
	IsVar    bool        // comparing to another variable (not used in basic impl)
}

// OrderByClause represents a single ORDER BY term
type OrderByClause struct {
	Variable   string
	Descending bool
}

// AggregateExpr represents a SELECT aggregate expression such as (COUNT(?j) AS ?count).
type AggregateExpr struct {
	Function string // COUNT, SUM, AVG, MIN, MAX
	Variable string // input variable name (without ?), or "*" for COUNT(*)
	Alias    string // output alias variable name (without ?)
}

// SPARQLQuery holds the parsed query components
type SPARQLQuery struct {
	Prefixes   map[string]string
	Variables  []string        // SELECT projected vars (without ?); includes aggregate aliases
	Patterns   []TriplePattern
	Filters    []FilterExpr
	Aggregates []AggregateExpr // Aggregate expressions from SELECT clause
	GroupBy    []string        // GROUP BY variable names (without ?)
	Having     []FilterExpr    // HAVING filters applied after grouping
	OrderBy    []OrderByClause
	Limit      int
	Offset     int
}

type parser struct {
	tokens []token
	pos    int
}

func (p *parser) peek() token {
	if p.pos >= len(p.tokens) {
		return token{tokEOF, ""}
	}
	return p.tokens[p.pos]
}

func (p *parser) next() token {
	t := p.peek()
	if t.typ != tokEOF {
		p.pos++
	}
	return t
}

func (p *parser) expect(typ tokenType, val string) bool {
	t := p.peek()
	if t.typ == typ && (val == "" || strings.EqualFold(t.val, val)) {
		p.pos++
		return true
	}
	return false
}

func parseSPARQL(tokens []token) (*SPARQLQuery, error) {
	p := &parser{tokens: tokens}
	q := &SPARQLQuery{
		Prefixes: make(map[string]string),
	}

	// Parse PREFIX declarations
	for p.peek().typ == tokKeyword && p.peek().val == "PREFIX" {
		p.next()
		// prefix name (may end with :)
		prefixName := ""
		if p.peek().typ == tokURI || p.peek().typ == tokKeyword {
			prefixName = p.next().val
		}
		// URI
		if p.peek().typ == tokURI {
			q.Prefixes[prefixName] = p.next().val
		}
	}

	// SELECT
	if p.peek().typ != tokKeyword || p.peek().val != "SELECT" {
		return nil, fmt.Errorf("expected SELECT")
	}
	p.next()

	// DISTINCT (optional)
	if p.peek().typ == tokKeyword && p.peek().val == "DISTINCT" {
		p.next()
	}

	// Variables or * — also handle aggregate expressions like (COUNT(?j) AS ?count)
	if p.peek().typ == tokPunct && p.peek().val == "*" {
		p.next()
	} else {
		for {
			if p.peek().typ == tokVariable {
				q.Variables = append(q.Variables, p.next().val)
				continue
			}
			// Aggregate expression: (FUNC(?var) AS ?alias) or (FUNC(*) AS ?alias)
			if p.peek().typ == tokPunct && p.peek().val == "(" {
				if agg, ok := parseAggregateExpr(p); ok {
					q.Aggregates = append(q.Aggregates, agg)
					// Alias is also added to Variables so the projection step picks it up.
					q.Variables = append(q.Variables, agg.Alias)
					continue
				}
			}
			break
		}
	}

	// FROM (optional, skip)
	if p.peek().typ == tokKeyword && p.peek().val == "FROM" {
		p.next()
		if p.peek().typ == tokURI {
			p.next()
		}
	}

	// WHERE
	if p.peek().typ == tokKeyword && p.peek().val == "WHERE" {
		p.next()
	}

	// {
	if !p.expect(tokPunct, "{") {
		return nil, fmt.Errorf("expected {")
	}

	// Parse triple patterns and FILTER clauses
	for p.peek().typ != tokPunct || p.peek().val != "}" {
		if p.peek().typ == tokEOF {
			break
		}

		// OPTIONAL block (skip for now, just skip the block)
		if p.peek().typ == tokKeyword && p.peek().val == "OPTIONAL" {
			p.next()
			if p.expect(tokPunct, "{") {
				depth := 1
				for depth > 0 && p.peek().typ != tokEOF {
					t := p.next()
					if t.typ == tokPunct && t.val == "{" {
						depth++
					} else if t.typ == tokPunct && t.val == "}" {
						depth--
					}
				}
			}
			continue
		}

		// FILTER clause
		if p.peek().typ == tokKeyword && p.peek().val == "FILTER" {
			p.next()
			filter := parseFilter(p)
			if filter != nil {
				q.Filters = append(q.Filters, *filter)
			}
			continue
		}

		// Triple pattern: subject predicate object .
		triple, ok := parseTriple(p)
		if !ok {
			break
		}
		q.Patterns = append(q.Patterns, triple)

		// Optional dot separator
		if p.peek().typ == tokDot {
			p.next()
		}
	}

	// }
	if p.peek().typ == tokPunct && p.peek().val == "}" {
		p.next()
	}

	// GROUP BY
	if p.peek().typ == tokKeyword && p.peek().val == "GROUP" {
		p.next()
		if p.peek().typ == tokKeyword && p.peek().val == "BY" {
			p.next()
			for p.peek().typ == tokVariable {
				q.GroupBy = append(q.GroupBy, p.next().val)
			}
		}
	}

	// HAVING
	if p.peek().typ == tokKeyword && p.peek().val == "HAVING" {
		p.next()
		if filter := parseFilter(p); filter != nil {
			q.Having = append(q.Having, *filter)
		}
	}

	// ORDER BY
	if p.peek().typ == tokKeyword && p.peek().val == "ORDER" {
		p.next()
		if p.peek().typ == tokKeyword && p.peek().val == "BY" {
			p.next()
			for {
				desc := false
				if p.peek().typ == tokKeyword && (p.peek().val == "ASC" || p.peek().val == "DESC") {
					if p.peek().val == "DESC" {
						desc = true
					}
					p.next()
					// optional surrounding parens
					if p.peek().typ == tokPunct && p.peek().val == "(" {
						p.next()
					}
				}
				if p.peek().typ == tokVariable {
					v := p.next().val
					// close paren if we opened one
					if p.peek().typ == tokPunct && p.peek().val == ")" {
						p.next()
					}
					q.OrderBy = append(q.OrderBy, OrderByClause{Variable: v, Descending: desc})
				} else {
					break
				}
			}
		}
	}

	// LIMIT
	if p.peek().typ == tokKeyword && p.peek().val == "LIMIT" {
		p.next()
		if p.peek().typ == tokNumber {
			if n, err := strconv.Atoi(p.next().val); err == nil {
				q.Limit = n
			}
		}
	}

	// OFFSET
	if p.peek().typ == tokKeyword && p.peek().val == "OFFSET" {
		p.next()
		if p.peek().typ == tokNumber {
			if n, err := strconv.Atoi(p.next().val); err == nil {
				q.Offset = n
			}
		}
	}

	return q, nil
}

// parseAggregateExpr parses expressions of the form (FUNC(?var) AS ?alias) or (FUNC(*) AS ?alias).
// The opening "(" must already be peeked but not consumed. Returns (expr, true) on success.
func parseAggregateExpr(p *parser) (AggregateExpr, bool) {
	saved := p.pos // allow backtracking on failure
	p.next()       // consume "("

	// Function name: COUNT, SUM, AVG, MIN, MAX
	if p.peek().typ != tokKeyword {
		p.pos = saved
		return AggregateExpr{}, false
	}
	fn := strings.ToUpper(p.next().val)
	switch fn {
	case "COUNT", "SUM", "AVG", "MIN", "MAX":
		// valid
	default:
		p.pos = saved
		return AggregateExpr{}, false
	}

	// Opening paren for argument
	if p.peek().typ != tokPunct || p.peek().val != "(" {
		p.pos = saved
		return AggregateExpr{}, false
	}
	p.next()

	// Argument: ?var or *
	var varName string
	if p.peek().typ == tokVariable {
		varName = p.next().val
	} else if p.peek().typ == tokPunct && p.peek().val == "*" {
		varName = "*"
		p.next()
	} else {
		p.pos = saved
		return AggregateExpr{}, false
	}

	// Closing paren for argument
	if p.peek().typ != tokPunct || p.peek().val != ")" {
		p.pos = saved
		return AggregateExpr{}, false
	}
	p.next()

	// AS keyword
	if p.peek().typ != tokKeyword || p.peek().val != "AS" {
		p.pos = saved
		return AggregateExpr{}, false
	}
	p.next()

	// Alias variable
	if p.peek().typ != tokVariable {
		p.pos = saved
		return AggregateExpr{}, false
	}
	alias := p.next().val

	// Closing outer paren
	if p.peek().typ != tokPunct || p.peek().val != ")" {
		p.pos = saved
		return AggregateExpr{}, false
	}
	p.next()

	return AggregateExpr{Function: fn, Variable: varName, Alias: alias}, true
}

// parseTriple parses a single triple pattern
func parseTriple(p *parser) (TriplePattern, bool) {
	var triple TriplePattern

	// Subject
	s, isVar := parseTerm(p)
	if s == "" {
		return triple, false
	}
	triple.Subject = s
	triple.IsVar[0] = isVar

	// Predicate
	pred, predIsVar := parseTerm(p)
	if pred == "" {
		return triple, false
	}
	// Handle keyword 'a' as rdf:type predicate
	if !predIsVar && (pred == "a" || pred == "A" || pred == "type") {
		pred = "a"
	}
	triple.Predicate = pred
	triple.IsVar[1] = predIsVar

	// Object
	obj, objIsVar := parseTerm(p)
	if obj == "" {
		return triple, false
	}
	triple.Object = obj
	triple.IsVar[2] = objIsVar

	return triple, true
}

// parseTerm extracts a single SPARQL term (variable, URI, or literal) and returns (value, isVariable)
func parseTerm(p *parser) (string, bool) {
	t := p.peek()
	switch t.typ {
	case tokVariable:
		p.next()
		return t.val, true
	case tokURI:
		p.next()
		// Strip prefix if it looks like prefix:local
		parts := strings.SplitN(t.val, ":", 2)
		if len(parts) == 2 {
			return parts[1], false
		}
		return t.val, false
	case tokLiteral:
		p.next()
		return t.val, false
	case tokNumber:
		p.next()
		return t.val, false
	case tokKeyword:
		// 'a' for rdf:type
		if t.val == "A" {
			p.next()
			return "a", false
		}
		return "", false
	default:
		return "", false
	}
}

// parseFilter parses a FILTER(...) expression
func parseFilter(p *parser) *FilterExpr {
	// expect opening paren
	if !p.expect(tokPunct, "(") {
		return nil
	}

	var filter FilterExpr

	// variable
	if p.peek().typ != tokVariable {
		// skip to closing paren
		for p.peek().typ != tokPunct || p.peek().val != ")" {
			if p.peek().typ == tokEOF {
				break
			}
			p.next()
		}
		p.expect(tokPunct, ")")
		return nil
	}
	filter.Variable = p.next().val

	// operator
	if p.peek().typ != tokPunct {
		p.expect(tokPunct, ")")
		return nil
	}
	op := p.next().val
	switch op {
	case ">":
		filter.Operator = "gt"
	case ">=":
		filter.Operator = "gte"
	case "<":
		filter.Operator = "lt"
	case "<=":
		filter.Operator = "lte"
	case "=":
		filter.Operator = "eq"
	case "!=":
		filter.Operator = "ne"
	default:
		filter.Operator = "eq"
	}

	// value
	t := p.peek()
	switch t.typ {
	case tokNumber:
		p.next()
		if f, err := strconv.ParseFloat(t.val, 64); err == nil {
			filter.Value = f
		} else {
			filter.Value = t.val
		}
	case tokLiteral:
		p.next()
		filter.Value = t.val
	case tokVariable:
		p.next()
		filter.IsVar = true
		filter.Value = t.val
	default:
		filter.Value = t.val
		p.next()
	}

	p.expect(tokPunct, ")")
	return &filter
}

// ─── Evaluator ────────────────────────────────────────────────────────────────

type sparqlBinding map[string]interface{}

func cloneBinding(b sparqlBinding) sparqlBinding {
	nb := make(sparqlBinding, len(b))
	for k, v := range b {
		nb[k] = v
	}
	return nb
}

// evaluateSPARQL evaluates the parsed query against the entity set
func evaluateSPARQL(q *SPARQLQuery, entities []*models.Entity) []map[string]interface{} {
	// Build lookup table: entity ID → entity
	entityByID := make(map[string]*models.Entity, len(entities))
	for _, e := range entities {
		entityByID[e.ID] = e
	}

	// Start with one empty binding
	bindings := []sparqlBinding{{}}

	// Apply triple patterns
	for _, pattern := range q.Patterns {
		bindings = applyTriple(pattern, bindings, entities, entityByID)
		if len(bindings) == 0 {
			break
		}
	}

	// Apply FILTER expressions
	filtered := bindings[:0]
	for _, b := range bindings {
		if matchesFilters(q.Filters, b) {
			filtered = append(filtered, b)
		}
	}
	bindings = filtered

	// Apply GROUP BY + aggregates
	if len(q.GroupBy) > 0 || len(q.Aggregates) > 0 {
		bindings = applyGroupBy(q.GroupBy, q.Aggregates, bindings)
		// Apply HAVING filters on the grouped results
		if len(q.Having) > 0 {
			grouped := bindings[:0]
			for _, b := range bindings {
				if matchesFilters(q.Having, b) {
					grouped = append(grouped, b)
				}
			}
			bindings = grouped
		}
	}

	// Apply ORDER BY
	if len(q.OrderBy) > 0 {
		sort.SliceStable(bindings, func(i, j int) bool {
			for _, clause := range q.OrderBy {
				vi := bindingVal(bindings[i], clause.Variable)
				vj := bindingVal(bindings[j], clause.Variable)
				cmp := compareVals(vi, vj)
				if cmp == 0 {
					continue
				}
				if clause.Descending {
					return cmp > 0
				}
				return cmp < 0
			}
			return false
		})
	}

	// Apply OFFSET
	if q.Offset > 0 {
		if q.Offset >= len(bindings) {
			bindings = nil
		} else {
			bindings = bindings[q.Offset:]
		}
	}

	// Apply LIMIT
	if q.Limit > 0 && len(bindings) > q.Limit {
		bindings = bindings[:q.Limit]
	}

	// Project to SELECT variables
	results := make([]map[string]interface{}, 0, len(bindings))
	for _, b := range bindings {
		row := make(map[string]interface{})
		if len(q.Variables) > 0 {
			for _, v := range q.Variables {
				if val, ok := b[v]; ok {
					row[v] = val
				}
			}
		} else {
			for k, v := range b {
				row[k] = v
			}
		}
		results = append(results, row)
	}
	return results
}

// applyTriple extends bindings according to a single triple pattern
func applyTriple(pat TriplePattern, bindings []sparqlBinding, entities []*models.Entity, entityByID map[string]*models.Entity) []sparqlBinding {
	result := make([]sparqlBinding, 0)

	// rdf:type pattern: ?s a :Type  →  bind ?s to entity IDs of matching type
	if pat.Predicate == "a" || pat.Predicate == "type" {
		typeName := pat.Object
		for _, b := range bindings {
			if pat.IsVar[0] {
				if existingID, bound := b[pat.Subject]; bound {
					// Subject already bound – verify type matches
					if ent, ok := entityByID[fmt.Sprintf("%v", existingID)]; ok {
						if ent.Type == typeName {
							result = append(result, b)
						}
					}
				} else {
					// Subject unbound – fan out over all matching entities
					for _, ent := range entities {
						if ent.Type == typeName {
							nb := cloneBinding(b)
							nb[pat.Subject] = ent.ID
							result = append(result, nb)
						}
					}
				}
			}
		}
		return result
	}

	// Attribute or relationship pattern: ?s :predicate ?v  or  ?s :predicate "literal"
	if pat.IsVar[0] {
		for _, b := range bindings {
			if existingID, bound := b[pat.Subject]; bound {
				// Subject is already bound to an entity ID
				ent, ok := entityByID[fmt.Sprintf("%v", existingID)]
				if !ok {
					continue
				}
				attrVal, hasAttr := ent.Attributes[pat.Predicate]
				if hasAttr {
					if pat.IsVar[2] {
						nb := cloneBinding(b)
						nb[pat.Object] = attrVal
						result = append(result, nb)
					} else {
						if fmt.Sprintf("%v", attrVal) == pat.Object {
							result = append(result, b)
						}
					}
				} else {
					// Attribute not found – check entity relationships
					for _, rel := range ent.Relationships {
						if rel.Type != pat.Predicate {
							continue
						}
						target, targetOK := entityByID[rel.TargetID]
						if !targetOK {
							continue
						}
						if pat.IsVar[2] {
							nb := cloneBinding(b)
							nb[pat.Object] = target.ID
							result = append(result, nb)
						} else {
							if target.ID == pat.Object {
								result = append(result, b)
							}
						}
					}
				}
			} else {
				// Subject unbound – fan out over all entities
				for _, ent := range entities {
					attrVal, hasAttr := ent.Attributes[pat.Predicate]
					if hasAttr {
						if pat.IsVar[2] {
							nb := cloneBinding(b)
							nb[pat.Subject] = ent.ID
							nb[pat.Object] = attrVal
							result = append(result, nb)
						} else {
							if fmt.Sprintf("%v", attrVal) == pat.Object {
								nb := cloneBinding(b)
								nb[pat.Subject] = ent.ID
								result = append(result, nb)
							}
						}
					} else {
						// Check relationships
						for _, rel := range ent.Relationships {
							if rel.Type != pat.Predicate {
								continue
							}
							target, targetOK := entityByID[rel.TargetID]
							if !targetOK {
								continue
							}
							if pat.IsVar[2] {
								nb := cloneBinding(b)
								nb[pat.Subject] = ent.ID
								nb[pat.Object] = target.ID
								result = append(result, nb)
							} else {
								if target.ID == pat.Object {
									nb := cloneBinding(b)
									nb[pat.Subject] = ent.ID
									result = append(result, nb)
								}
							}
						}
					}
				}
			}
		}
		return result
	}

	// Subject is a literal – pass through unchanged
	return bindings
}

// applyGroupBy partitions bindings by the GROUP BY variables and computes
// aggregate functions (COUNT, SUM, AVG, MIN, MAX) for each group.
// If groupBy is empty but aggregates are present, all bindings form a single group.
func applyGroupBy(groupBy []string, aggs []AggregateExpr, bindings []sparqlBinding) []sparqlBinding {
	// Build ordered group map to preserve insertion order.
	type group struct {
		rows []sparqlBinding
	}
	groupKeys := []string{}
	groups := map[string]*group{}

	for _, b := range bindings {
		// Build composite key from GROUP BY variable values.
		parts := make([]string, len(groupBy))
		for i, v := range groupBy {
			parts[i] = fmt.Sprintf("%v", b[v])
		}
		key := strings.Join(parts, "\x00")
		if _, exists := groups[key]; !exists {
			groupKeys = append(groupKeys, key)
			groups[key] = &group{}
		}
		groups[key].rows = append(groups[key].rows, b)
	}

	// If no GROUP BY but aggregates exist, treat all rows as one group.
	if len(groupBy) == 0 {
		key := "_all"
		groupKeys = []string{key}
		groups[key] = &group{rows: bindings}
	}

	result := make([]sparqlBinding, 0, len(groups))
	for _, key := range groupKeys {
		g := groups[key]
		row := sparqlBinding{}

		// Copy GROUP BY variable values from the first row of the group.
		if len(g.rows) > 0 {
			for _, v := range groupBy {
				row[v] = g.rows[0][v]
			}
		}

		// Compute aggregates.
		for _, agg := range aggs {
			row[agg.Alias] = computeAggregate(agg, g.rows)
		}

		result = append(result, row)
	}
	return result
}

// computeAggregate calculates a single aggregate function over a set of bindings.
func computeAggregate(agg AggregateExpr, rows []sparqlBinding) interface{} {
	switch agg.Function {
	case "COUNT":
		if agg.Variable == "*" {
			return float64(len(rows))
		}
		count := 0
		for _, b := range rows {
			if _, ok := b[agg.Variable]; ok {
				count++
			}
		}
		return float64(count)

	case "SUM":
		sum := 0.0
		for _, b := range rows {
			if v, ok := b[agg.Variable]; ok {
				if f, err := toFloat(v); err == nil {
					sum += f
				}
			}
		}
		return sum

	case "AVG":
		sum := 0.0
		n := 0
		for _, b := range rows {
			if v, ok := b[agg.Variable]; ok {
				if f, err := toFloat(v); err == nil {
					sum += f
					n++
				}
			}
		}
		if n == 0 {
			return 0.0
		}
		return sum / float64(n)

	case "MIN":
		var minVal *float64
		for _, b := range rows {
			if v, ok := b[agg.Variable]; ok {
				if f, err := toFloat(v); err == nil {
					if minVal == nil || f < *minVal {
						cp := f
						minVal = &cp
					}
				}
			}
		}
		if minVal == nil {
			return nil
		}
		return *minVal

	case "MAX":
		var maxVal *float64
		for _, b := range rows {
			if v, ok := b[agg.Variable]; ok {
				if f, err := toFloat(v); err == nil {
					if maxVal == nil || f > *maxVal {
						cp := f
						maxVal = &cp
					}
				}
			}
		}
		if maxVal == nil {
			return nil
		}
		return *maxVal
	}
	return nil
}

// matchesFilters checks whether a binding satisfies all FILTER expressions
func matchesFilters(filters []FilterExpr, b sparqlBinding) bool {
	for _, f := range filters {
		val, ok := b[f.Variable]
		if !ok {
			return false
		}
		if !evalFilter(val, f.Operator, f.Value) {
			return false
		}
	}
	return true
}

func evalFilter(val interface{}, op string, expected interface{}) bool {
	// Try numeric comparison
	vf, vErr := toFloat(val)
	ef, eErr := toFloat(expected)
	if vErr == nil && eErr == nil {
		switch op {
		case "gt":
			return vf > ef
		case "gte":
			return vf >= ef
		case "lt":
			return vf < ef
		case "lte":
			return vf <= ef
		case "eq":
			return vf == ef
		case "ne":
			return vf != ef
		}
	}
	// String comparison
	vs := fmt.Sprintf("%v", val)
	es := fmt.Sprintf("%v", expected)
	switch op {
	case "eq":
		return vs == es
	case "ne":
		return vs != es
	}
	return false
}

func toFloat(v interface{}) (float64, error) {
	switch x := v.(type) {
	case float64:
		return x, nil
	case float32:
		return float64(x), nil
	case int:
		return float64(x), nil
	case int64:
		return float64(x), nil
	case int32:
		return float64(x), nil
	case string:
		return strconv.ParseFloat(x, 64)
	}
	return 0, fmt.Errorf("not a number")
}

func bindingVal(b sparqlBinding, variable string) interface{} {
	return b[variable]
}

func compareVals(a, b interface{}) int {
	af, aErr := toFloat(a)
	bf, bErr := toFloat(b)
	if aErr == nil && bErr == nil {
		if af < bf {
			return -1
		} else if af > bf {
			return 1
		}
		return 0
	}
	as := fmt.Sprintf("%v", a)
	bs := fmt.Sprintf("%v", b)
	return strings.Compare(as, bs)
}

