package digitaltwin

import (
	"fmt"
	"github.com/mimir-aip/mimir-aip-go/pkg/models"
	"sort"
	"strings"
)

func commonKeyFieldNames(entitiesA, entitiesB []*models.Entity) []string {
	sampleA := entitiesA
	if len(sampleA) > 20 {
		sampleA = sampleA[:20]
	}
	sampleB := entitiesB
	if len(sampleB) > 20 {
		sampleB = sampleB[:20]
	}

	keysA := make(map[string]bool)
	for _, e := range sampleA {
		for _, kf := range detectKeyFields(e.Attributes) {
			keysA[kf] = true
		}
	}

	var shared []string
	seenB := make(map[string]bool)
	for _, e := range sampleB {
		for _, kf := range detectKeyFields(e.Attributes) {
			if keysA[kf] && !seenB[kf] {
				seenB[kf] = true
				shared = append(shared, kf)
			}
		}
	}
	return shared
}

// hasRelationship returns true if entity already has a relationship of the
// given type pointing to targetID.
func hasRelationship(e *models.Entity, relType, targetID string) bool {
	for _, r := range e.Relationships {
		if r.Type == relType && r.TargetID == targetID {
			return true
		}
	}
	return false
}

// toCamelCaseRel converts a field name like "student_id" → "StudentId"
// for use in relationship type names.
func toCamelCaseRel(s string) string {
	parts := strings.FieldsFunc(s, func(r rune) bool {
		return r == '_' || r == '-' || r == ' '
	})
	var b strings.Builder
	for _, p := range parts {
		if len(p) == 0 {
			continue
		}
		b.WriteString(strings.ToUpper(p[:1]) + p[1:])
	}
	return b.String()
}

// detectKeyFields is re-exported here for use within the digitaltwin package.
// It returns attribute names that look like stable join keys.
func detectKeyFields(attributes map[string]interface{}) []string {
	keyNameSuffixes := []string{"id", "key", "code", "number", "uuid", "ref", "identifier", "no", "num", "email", "username", "token"}
	var keys []string
	for k := range attributes {
		lower := strings.ToLower(k)
		for _, suffix := range keyNameSuffixes {
			if strings.HasSuffix(lower, suffix) || lower == suffix {
				keys = append(keys, k)
				break
			}
		}
	}
	sort.Strings(keys)
	return keys
}

// keyValue returns a normalised string representation of an attribute value
// for equality comparison across data sources.
func keyValue(v interface{}) string {
	if v == nil {
		return ""
	}
	s := fmt.Sprintf("%v", v)
	if strings.Contains(s, ".") {
		s = strings.TrimRight(s, "0")
		s = strings.TrimRight(s, ".")
	}
	return strings.TrimSpace(s)
}

// inferEntityTypeFromCIR tries to infer the entity type from a CIR record
func inferEntityTypeFromCIR(cir *models.CIR, ontologyClasses []string) string {
	// Check CIR parameter first
	if v, ok := cir.GetParameter("entity_type"); ok {
		if s, ok := v.(string); ok && s != "" {
			return s
		}
	}

	// Match data keys against ontology class property names
	dataMap, err := cir.GetDataAsMap()
	if err != nil {
		return "unknown"
	}

	keys := make([]string, 0, len(dataMap))
	for k := range dataMap {
		keys = append(keys, strings.ToLower(k))
	}

	// Simple heuristic: look for class name hints in data keys
	for _, className := range ontologyClasses {
		lc := strings.ToLower(className)
		for _, k := range keys {
			if k == lc || k == "type" || k == "entity_type" {
				if v, ok := dataMap[k]; ok {
					if s, ok := v.(string); ok && strings.EqualFold(s, className) {
						return className
					}
				}
			}
		}
	}

	// Fall back to URI last path segment
	uri := cir.Source.URI
	if uri != "" {
		parts := strings.Split(strings.TrimRight(uri, "/"), "/")
		if len(parts) > 0 {
			last := parts[len(parts)-1]
			if last != "" {
				return last
			}
		}
	}

	return "default"
}

// StartCacheEviction runs a background goroutine that periodically deletes expired predictions.
