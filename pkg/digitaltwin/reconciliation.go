package digitaltwin

import (
	"fmt"
	"github.com/mimir-aip/mimir-aip-go/pkg/models"
	"time"
)

func mergeEntityAttributes(entity *models.Entity, incoming map[string]interface{}, storageID string, observedAt time.Time, policy *models.TwinReconciliationPolicy) {
	if entity.Attributes == nil {
		entity.Attributes = make(map[string]interface{})
	}
	if entity.ComputedValues == nil {
		entity.ComputedValues = make(map[string]interface{})
	}
	attributeSources := ensureStringMap(entity.ComputedValues, "attribute_sources")
	attributeTimestamps := ensureStringMap(entity.ComputedValues, "attribute_timestamps")
	conflicts := ensureConflictMap(entity.ComputedValues)
	for key, incomingValue := range incoming {
		currentValue, exists := entity.Attributes[key]
		if !exists {
			entity.Attributes[key] = incomingValue
			attributeSources[key] = storageID
			attributeTimestamps[key] = observedAt.Format(time.RFC3339)
			continue
		}
		if valuesEquivalent(currentValue, incomingValue) {
			if _, ok := attributeSources[key]; !ok {
				attributeSources[key] = storageID
			}
			if _, ok := attributeTimestamps[key]; !ok {
				attributeTimestamps[key] = observedAt.Format(time.RFC3339)
			}
			continue
		}
		currentSource := attributeSources[key]
		currentObservedAt, _ := time.Parse(time.RFC3339, attributeTimestamps[key])
		chosenSource, chosenValue := resolveReconciledValue(currentValue, currentSource, currentObservedAt, incomingValue, storageID, observedAt, policy)
		entity.Attributes[key] = chosenValue
		attributeSources[key] = chosenSource
		attributeTimestamps[key] = maxTime(currentObservedAt, observedAt).Format(time.RFC3339)
		conflicts[key] = appendUniqueString(conflicts[key], fmt.Sprintf("%s=%v vs %s=%v", currentSource, currentValue, storageID, incomingValue))
	}
	entity.ComputedValues["attribute_sources"] = stringMapToInterfaceMap(attributeSources)
	entity.ComputedValues["attribute_timestamps"] = stringMapToInterfaceMap(attributeTimestamps)
	entity.ComputedValues["reconciliation_conflicts"] = conflictMapToInterfaceMap(conflicts)
}

func cloneJSONMap(values map[string]interface{}) map[string]interface{} {
	if values == nil {
		return nil
	}
	cloned := make(map[string]interface{}, len(values))
	for key, value := range values {
		cloned[key] = value
	}
	return cloned
}

func resolveReconciledValue(currentValue interface{}, currentSource string, currentObservedAt time.Time, incomingValue interface{}, incomingSource string, incomingObservedAt time.Time, policy *models.TwinReconciliationPolicy) (string, interface{}) {
	if policy == nil {
		policy = &models.TwinReconciliationPolicy{Strategy: "source_priority"}
	}
	switch policy.Strategy {
	case "freshest":
		if incomingObservedAt.After(currentObservedAt) {
			return incomingSource, incomingValue
		}
		return currentSource, currentValue
	case "source_priority", "":
		currentRank := sourcePriorityRank(currentSource, policy.SourcePriority)
		incomingRank := sourcePriorityRank(incomingSource, policy.SourcePriority)
		if incomingRank < currentRank {
			return incomingSource, incomingValue
		}
		if incomingRank == currentRank && incomingObservedAt.After(currentObservedAt) {
			return incomingSource, incomingValue
		}
		return currentSource, currentValue
	default:
		if incomingObservedAt.After(currentObservedAt) {
			return incomingSource, incomingValue
		}
		return currentSource, currentValue
	}
}

func sourcePriorityRank(sourceID string, priorities []string) int {
	for i, candidate := range priorities {
		if candidate == sourceID {
			return i
		}
	}
	return len(priorities) + 1
}

func attributeSourceMap(attrs map[string]interface{}, storageID string) map[string]interface{} {
	out := make(map[string]interface{}, len(attrs))
	for key := range attrs {
		out[key] = storageID
	}
	return out
}

func attributeTimestampMap(attrs map[string]interface{}, observedAt time.Time) map[string]interface{} {
	formatted := observedAt.Format(time.RFC3339)
	out := make(map[string]interface{}, len(attrs))
	for key := range attrs {
		out[key] = formatted
	}
	return out
}

func ensureStringMap(values map[string]interface{}, key string) map[string]string {
	result := make(map[string]string)
	if values == nil {
		return result
	}
	raw, _ := values[key].(map[string]interface{})
	for k, v := range raw {
		result[k] = fmt.Sprintf("%v", v)
	}
	return result
}

func ensureConflictMap(values map[string]interface{}) map[string][]string {
	result := make(map[string][]string)
	if values == nil {
		return result
	}
	raw, _ := values["reconciliation_conflicts"].(map[string]interface{})
	for key, value := range raw {
		if items, ok := value.([]interface{}); ok {
			for _, item := range items {
				result[key] = append(result[key], fmt.Sprintf("%v", item))
			}
		}
	}
	return result
}

func stringMapToInterfaceMap(values map[string]string) map[string]interface{} {
	out := make(map[string]interface{}, len(values))
	for key, value := range values {
		out[key] = value
	}
	return out
}

func conflictMapToInterfaceMap(values map[string][]string) map[string]interface{} {
	out := make(map[string]interface{}, len(values))
	for key, value := range values {
		items := make([]interface{}, 0, len(value))
		for _, item := range value {
			items = append(items, item)
		}
		out[key] = items
	}
	return out
}

func appendUniqueString(items []string, value string) []string {
	for _, item := range items {
		if item == value {
			return items
		}
	}
	return append(items, value)
}

func valuesEquivalent(a, b interface{}) bool {
	return fmt.Sprintf("%v", a) == fmt.Sprintf("%v", b)
}

func maxTime(a, b time.Time) time.Time {
	if b.After(a) {
		return b
	}
	return a
}

// appendSourceID adds storageID to an entity's ComputedValues["source_ids"]
// list if it is not already present.
func appendSourceID(entity *models.Entity, storageID string) {
	if entity.ComputedValues == nil {
		entity.ComputedValues = make(map[string]interface{})
	}
	existing, _ := entity.ComputedValues["source_ids"].([]interface{})
	for _, v := range existing {
		if s, _ := v.(string); s == storageID {
			return // already present
		}
	}
	entity.ComputedValues["source_ids"] = append(existing, storageID)
}

// wireRelationships links entities of different types that share a common
// key-field value, records temporal edge changes, and rewrites the current
