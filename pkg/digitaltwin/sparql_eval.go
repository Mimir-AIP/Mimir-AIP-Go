package digitaltwin

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/mimir-aip/mimir-aip-go/pkg/models"
)

func (a *bindingArena) newBinding() sparqlBinding {
	b := sparqlBindingPool.Get().(sparqlBinding)
	for k := range b {
		delete(b, k)
	}
	a.allocated = append(a.allocated, b)
	return b
}

func (a *bindingArena) cloneBinding(src sparqlBinding) sparqlBinding {
	nb := a.newBinding()
	for k, v := range src {
		nb[k] = v
	}
	return nb
}

func (a *bindingArena) release() {
	for _, b := range a.allocated {
		for k := range b {
			delete(b, k)
		}
		sparqlBindingPool.Put(b)
	}
	a.allocated = a.allocated[:0]
}

func bindingID(v interface{}) (string, bool) {
	if id, ok := v.(string); ok {
		return id, true
	}
	id := fmt.Sprintf("%v", v)
	return id, id != ""
}

func bindOrMatch(b sparqlBinding, variable string, value interface{}) bool {
	if existing, ok := b[variable]; ok {
		return evalFilter(existing, "eq", value)
	}
	b[variable] = value
	return true
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
	arena := bindingArena{allocated: make([]sparqlBinding, 0, len(entities))}
	defer arena.release()

	// Apply triple patterns
	for _, pattern := range q.Patterns {
		bindings = applyTriple(pattern, bindings, entities, entityByID, &arena)
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
		row := make(map[string]interface{}, len(q.Variables))
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
func applyTriple(pat TriplePattern, bindings []sparqlBinding, entities []*models.Entity, entityByID map[string]*models.Entity, arena *bindingArena) []sparqlBinding {
	result := make([]sparqlBinding, 0, len(bindings))

	// rdf:type pattern: ?s a :Type  →  bind ?s to entity IDs of matching type
	if pat.Predicate == "a" || pat.Predicate == "type" {
		if !pat.IsVar[0] {
			return result
		}
		typeName := pat.Object
		matchingEntities := make([]*models.Entity, 0, len(entities))
		for _, ent := range entities {
			if ent.Type == typeName {
				matchingEntities = append(matchingEntities, ent)
			}
		}
		result = make([]sparqlBinding, 0, len(bindings)*len(matchingEntities))
		for _, b := range bindings {
			if existingID, bound := b[pat.Subject]; bound {
				id, ok := bindingID(existingID)
				if !ok {
					continue
				}
				if ent, ok := entityByID[id]; ok && ent.Type == typeName {
					result = append(result, b)
				}
				continue
			}

			if len(b) == 0 {
				for _, ent := range matchingEntities {
					nb := arena.newBinding()
					nb[pat.Subject] = ent.ID
					result = append(result, nb)
				}
				continue
			}
			for _, ent := range matchingEntities {
				nb := arena.cloneBinding(b)
				nb[pat.Subject] = ent.ID
				result = append(result, nb)
			}
		}
		return result
	}

	// Attribute or relationship pattern: ?s :predicate ?v  or  ?s :predicate "literal"
	if pat.IsVar[0] {
		for _, b := range bindings {
			if existingID, bound := b[pat.Subject]; bound {
				// Subject is already bound to an entity ID
				id, ok := bindingID(existingID)
				if !ok {
					continue
				}
				ent, ok := entityByID[id]
				if !ok {
					continue
				}
				attrVal, hasAttr := ent.Attributes[pat.Predicate]
				if hasAttr {
					if pat.IsVar[2] {
						if bindOrMatch(b, pat.Object, attrVal) {
							result = append(result, b)
						}
					} else if fmt.Sprintf("%v", attrVal) == pat.Object {
						result = append(result, b)
					}
					continue
				}

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
						nb := arena.cloneBinding(b)
						nb[pat.Object] = target.ID
						result = append(result, nb)
					} else if target.ID == pat.Object {
						result = append(result, b)
					}
				}
				continue
			}

			// Subject unbound – fan out over all entities
			for _, ent := range entities {
				attrVal, hasAttr := ent.Attributes[pat.Predicate]
				if hasAttr {
					if pat.IsVar[2] {
						if len(b) == 0 {
							nb := arena.newBinding()
							nb[pat.Subject] = ent.ID
							nb[pat.Object] = attrVal
							result = append(result, nb)
						} else {
							nb := arena.cloneBinding(b)
							nb[pat.Subject] = ent.ID
							nb[pat.Object] = attrVal
							result = append(result, nb)
						}
					} else if fmt.Sprintf("%v", attrVal) == pat.Object {
						if len(b) == 0 {
							nb := arena.newBinding()
							nb[pat.Subject] = ent.ID
							result = append(result, nb)
						} else {
							nb := arena.cloneBinding(b)
							nb[pat.Subject] = ent.ID
							result = append(result, nb)
						}
					}
					continue
				}

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
						if len(b) == 0 {
							nb := arena.newBinding()
							nb[pat.Subject] = ent.ID
							nb[pat.Object] = target.ID
							result = append(result, nb)
						} else {
							nb := arena.cloneBinding(b)
							nb[pat.Subject] = ent.ID
							nb[pat.Object] = target.ID
							result = append(result, nb)
						}
					} else if target.ID == pat.Object {
						if len(b) == 0 {
							nb := arena.newBinding()
							nb[pat.Subject] = ent.ID
							result = append(result, nb)
						} else {
							nb := arena.cloneBinding(b)
							nb[pat.Subject] = ent.ID
							result = append(result, nb)
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
