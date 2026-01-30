package Data_Processing

import (
	"context"
	"fmt"

	"github.com/Mimir-AIP/Mimir-AIP-Go/pipelines"
)

// ValidatePlugin checks records against rules and returns valid/invalid splits
type ValidatePlugin struct {
	name string
}

func NewValidatePlugin() *ValidatePlugin {
	return &ValidatePlugin{name: "validate"}
}

func (p *ValidatePlugin) ExecuteStep(ctx context.Context, stepConfig pipelines.StepConfig, globalContext *pipelines.PluginContext) (*pipelines.PluginContext, error) {
	cfg := stepConfig.Config
	inputKey, _ := cfg["input"].(string)
	if inputKey == "" {
		return nil, fmt.Errorf("config 'input' is required")
	}

	raw, exists := globalContext.Get(inputKey)
	if !exists {
		return nil, fmt.Errorf("input key not found: %s", inputKey)
	}

	var rows []map[string]any
	switch v := raw.(type) {
	case []map[string]any:
		rows = v
	case []any:
		for _, item := range v {
			if m, ok := item.(map[string]any); ok {
				rows = append(rows, m)
			}
		}
	case map[string]any:
		if r, ok := v["rows"].([]any); ok {
			for _, item := range r {
				if m, ok := item.(map[string]any); ok {
					rows = append(rows, m)
				}
			}
		} else if rv, ok := v["value"]; ok {
			switch vv := rv.(type) {
			case []map[string]any:
				rows = vv
			case []any:
				for _, item := range vv {
					if m, ok := item.(map[string]any); ok {
						rows = append(rows, m)
					}
				}
			default:
				rows = append(rows, v)
			}
		} else {
			rows = append(rows, v)
		}
	default:
		return nil, fmt.Errorf("unsupported input data shape for key %s", inputKey)
	}

	// Parse rules
	rules := cfg["rules"].(map[string]any)
	var required []string
	if r, ok := rules["required"].([]any); ok {
		for _, v := range r {
			if s, ok := v.(string); ok {
				required = append(required, s)
			}
		}
	}

	typesMap := make(map[string]string)
	if t, ok := rules["types"].(map[string]any); ok {
		for k, v := range t {
			if s, ok := v.(string); ok {
				typesMap[k] = s
			}
		}
	}

	ranges := make(map[string]map[string]any)
	if rng, ok := rules["ranges"].(map[string]any); ok {
		for k, v := range rng {
			if m, ok := v.(map[string]any); ok {
				ranges[k] = m
			}
		}
	}

	var valid []map[string]any
	var invalid []map[string]any
	var errorsList []map[string]any

	for idx, row := range rows {
		rowErrors := make([]string, 0)
		// required
		for _, f := range required {
			if _, ok := row[f]; !ok {
				rowErrors = append(rowErrors, fmt.Sprintf("missing required field '%s'", f))
			}
		}
		// types (basic checks)
		for field, typ := range typesMap {
			if val, ok := row[field]; ok {
				switch typ {
				case "number":
					switch val.(type) {
					case int, int64, float32, float64:
					default:
						rowErrors = append(rowErrors, fmt.Sprintf("field '%s' not a number", field))
					}
				case "string":
					if _, ok := val.(string); !ok {
						rowErrors = append(rowErrors, fmt.Sprintf("field '%s' not a string", field))
					}
				case "boolean":
					if _, ok := val.(bool); !ok {
						rowErrors = append(rowErrors, fmt.Sprintf("field '%s' not a boolean", field))
					}
				}
			}
		}
		// ranges (only numeric min/max)
		for field, rng := range ranges {
			if val, ok := row[field]; ok {
				var num float64
				switch t := val.(type) {
				case int:
					num = float64(t)
				case int64:
					num = float64(t)
				case float32:
					num = float64(t)
				case float64:
					num = t
				default:
					rowErrors = append(rowErrors, fmt.Sprintf("field '%s' not numeric for range check", field))
					continue
				}
				if min, ok := rng["min"].(float64); ok {
					if num < min {
						rowErrors = append(rowErrors, fmt.Sprintf("field '%s' < min %v", field, min))
					}
				}
				if max, ok := rng["max"].(float64); ok {
					if num > max {
						rowErrors = append(rowErrors, fmt.Sprintf("field '%s' > max %v", field, max))
					}
				}
			}
		}

		if len(rowErrors) == 0 {
			valid = append(valid, row)
		} else {
			invalid = append(invalid, row)
			errorsList = append(errorsList, map[string]any{"index": idx, "errors": rowErrors})
		}
	}

	stats := map[string]any{"total": len(rows), "valid": len(valid), "invalid": len(invalid)}

	res := map[string]any{
		"valid_records":   valid,
		"invalid_records": invalid,
		"errors":          errorsList,
		"stats":           stats,
	}

	out := pipelines.NewPluginContext()
	outKey := stepConfig.Output
	if outKey == "" {
		outKey = inputKey + "_validated"
	}
	// store as JSONData for consistency
	out.Set(outKey, res)
	return out, nil
}

func (p *ValidatePlugin) GetPluginType() string { return "Data_Processing" }
func (p *ValidatePlugin) GetPluginName() string { return "validate" }

func (p *ValidatePlugin) ValidateConfig(config map[string]any) error {
	if _, ok := config["input"].(string); !ok {
		return fmt.Errorf("config 'input' is required and must be a string")
	}
	// rules optional
	return nil
}

func (p *ValidatePlugin) GetInputSchema() map[string]any {
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"input": map[string]any{"type": "string"},
			"rules": map[string]any{"type": "object"},
		},
		"required": []string{"input"},
	}
	return schema
}
