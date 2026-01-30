package Data_Processing

import (
	"context"
	"fmt"
	"sort"

	"github.com/Mimir-AIP/Mimir-AIP-Go/pipelines"
)

// DeduplicatePlugin removes duplicate records according to strategy
type DeduplicatePlugin struct {
	name string
}

func NewDeduplicatePlugin() *DeduplicatePlugin { return &DeduplicatePlugin{name: "deduplicate"} }

func (p *DeduplicatePlugin) ExecuteStep(ctx context.Context, stepConfig pipelines.StepConfig, globalContext *pipelines.PluginContext) (*pipelines.PluginContext, error) {
	cfg := stepConfig.Config
	inputKey, _ := cfg["input"].(string)
	if inputKey == "" {
		return nil, fmt.Errorf("config 'input' required")
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
		return nil, fmt.Errorf("unsupported input shape for key %s", inputKey)
	}

	method, _ := cfg["method"].(string)
	if method == "" {
		method = "exact"
	}

	var unique []map[string]any
	seen := make(map[string]bool)

	switch method {
	case "exact":
		for _, r := range rows {
			// build deterministic key from sorted keys to avoid map order issues
			keys := make([]string, 0, len(r))
			for k := range r {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			key := ""
			for _, k := range keys {
				key += fmt.Sprintf("%s=%v|", k, r[k])
			}
			if !seen[key] {
				seen[key] = true
				unique = append(unique, r)
			}
		}
	case "by_key":
		keysAny, ok := cfg["keys"].([]any)
		if !ok || len(keysAny) == 0 {
			return nil, fmt.Errorf("by_key method requires 'keys' array")
		}
		var keys []string
		for _, k := range keysAny {
			if s, ok := k.(string); ok {
				keys = append(keys, s)
			}
		}
		for _, r := range rows {
			var key string
			for _, k := range keys {
				if v, ok := r[k]; ok {
					key += fmt.Sprintf("|%v", v)
				}
			}
			if key == "" {
				// fallback to full record
				key = fmt.Sprintf("%v", r)
			}
			if !seen[key] {
				seen[key] = true
				unique = append(unique, r)
			}
		}
	default:
		return nil, fmt.Errorf("unsupported dedup method: %s", method)
	}

	out := pipelines.NewPluginContext()
	outKey := stepConfig.Output
	if outKey == "" {
		outKey = inputKey + "_deduped"
	}
	// return row_count as float64 to avoid type assertions differences
	out.Set(outKey, map[string]any{"row_count": float64(len(unique)), "rows": unique})
	return out, nil
}

func (p *DeduplicatePlugin) GetPluginType() string { return "Data_Processing" }
func (p *DeduplicatePlugin) GetPluginName() string { return "deduplicate" }

func (p *DeduplicatePlugin) ValidateConfig(config map[string]any) error {
	if _, ok := config["input"].(string); !ok {
		return fmt.Errorf("config 'input' is required and must be a string")
	}
	return nil
}

func (p *DeduplicatePlugin) GetInputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"input":  map[string]any{"type": "string"},
			"method": map[string]any{"type": "string", "enum": []string{"exact", "by_key"}},
			"keys":   map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
		},
		"required": []string{"input"},
	}
}
