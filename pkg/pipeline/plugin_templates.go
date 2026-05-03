package pipeline

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/mimir-aip/mimir-aip-go/pkg/models"
	"regexp"
	"strings"
	"text/template"
)

// ── Template resolution ───────────────────────────────────────────────────────

// ResolveTemplates resolves {{context.step_name.key}} template variables in a string.
func (p *DefaultPlugin) ResolveTemplates(input string, ctx *models.PipelineContext) string {
	exactPattern := regexp.MustCompile(`^\s*\{\{([^}]+)\}\}\s*$`)
	if matches := exactPattern.FindStringSubmatch(input); len(matches) == 2 {
		expr := strings.TrimSpace(matches[1])
		if value, ok := p.resolveTemplateValue(expr, ctx); ok {
			switch typed := value.(type) {
			case string:
				return typed
			default:
				if bytes, err := json.Marshal(typed); err == nil {
					return string(bytes)
				}
				return fmt.Sprintf("%v", typed)
			}
		}
	}

	pattern := regexp.MustCompile(`\{\{([^}]+)\}\}`)
	return pattern.ReplaceAllStringFunc(input, func(match string) string {
		expr := strings.TrimSpace(match[2 : len(match)-2])
		if value, ok := p.resolveTemplateValue(expr, ctx); ok {
			return fmt.Sprintf("%v", value)
		}
		return match
	})
}

func (p *DefaultPlugin) resolveTemplateValue(expr string, ctx *models.PipelineContext) (interface{}, bool) {
	parts := strings.Split(expr, ".")
	if len(parts) < 2 {
		return nil, false
	}
	if parts[0] != "context" {
		return p.evaluateTemplate(expr, ctx), true
	}
	if len(parts) < 3 {
		return nil, false
	}

	stepName := parts[1]
	key := parts[2]
	value, exists := ctx.GetStepData(stepName, key)
	if !exists {
		return nil, false
	}

	if len(parts) > 3 {
		for i := 3; i < len(parts); i++ {
			if m, ok := value.(map[string]interface{}); ok {
				value, exists = m[parts[i]]
				if !exists {
					return nil, false
				}
				continue
			}
			return nil, false
		}
	}

	return value, true
}

func (p *DefaultPlugin) evaluateTemplate(expr string, ctx *models.PipelineContext) string {
	tmpl, err := template.New("expr").Parse("{{" + expr + "}}")
	if err != nil {
		return ""
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, ctx.Steps); err != nil {
		return ""
	}
	return buf.String()
}

// ── Helpers ───────────────────────────────────────────────────────────────────

// resolveData coerces a raw parameter value into an interface{} suitable for a CIR's Data field.
// JSON strings are decoded; everything else is passed through.
func (p *DefaultPlugin) resolveData(raw interface{}, ctx *models.PipelineContext) (interface{}, error) {
	switch v := raw.(type) {
	case string:
		resolved := p.ResolveTemplates(v, ctx)
		var out interface{}
		if err := json.Unmarshal([]byte(resolved), &out); err == nil {
			return out, nil
		}
		return resolved, nil
	default:
		return v, nil
	}
}

// resolveArray coerces a raw parameter value into []interface{}.
func (p *DefaultPlugin) resolveArray(raw interface{}, ctx *models.PipelineContext) ([]interface{}, error) {
	switch v := raw.(type) {
	case []interface{}:
		return v, nil
	case string:
		resolved := p.ResolveTemplates(v, ctx)
		var items []interface{}
		if err := json.Unmarshal([]byte(resolved), &items); err != nil {
			return nil, fmt.Errorf("items must resolve to a JSON array: %w", err)
		}
		return items, nil
	default:
		// Marshal → unmarshal to normalise (handles []map[string]interface{} etc.)
		b, err := json.Marshal(v)
		if err != nil {
			return nil, fmt.Errorf("items parameter must be an array")
		}
		var items []interface{}
		if err := json.Unmarshal(b, &items); err != nil {
			return nil, fmt.Errorf("items parameter must be an array")
		}
		return items, nil
	}
}
