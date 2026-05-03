package digitaltwin

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
)

// ─── Parser ───────────────────────────────────────────────────────────────────

// TriplePattern represents a single triple in the WHERE clause
type TriplePattern struct {
	Subject   string  // variable name (without ?) or literal
	Predicate string  // local name (e.g. "age", "a")
	Object    string  // variable name or literal
	IsVar     [3]bool // which positions are variables
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
	Variables  []string // SELECT projected vars (without ?); includes aggregate aliases
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

var sparqlBindingPool = sync.Pool{
	New: func() interface{} {
		return make(sparqlBinding)
	},
}

type bindingArena struct {
	allocated []sparqlBinding
}
