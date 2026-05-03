package pipeline

import (
	"fmt"
	"github.com/mimir-aip/mimir-aip-go/pkg/models"
)

// ── Control flow ──────────────────────────────────────────────────────────────

func (p *DefaultPlugin) ifElse(params map[string]interface{}, ctx *models.PipelineContext) (map[string]interface{}, error) {
	condition, ok := params["condition"]
	if !ok {
		return nil, fmt.Errorf("condition parameter is required")
	}

	conditionStr := p.ResolveTemplates(fmt.Sprintf("%v", condition), ctx)
	isTrue := conditionStr != "" && conditionStr != "0" && conditionStr != "false" && conditionStr != "null"

	var result string
	if isTrue {
		if ifTrue, ok := params["if_true"].(string); ok {
			result = ifTrue
		}
	} else {
		if ifFalse, ok := params["if_false"].(string); ok {
			result = ifFalse
		}
	}

	return map[string]interface{}{
		"result":    result,
		"condition": isTrue,
	}, nil
}

func (p *DefaultPlugin) setContext(params map[string]interface{}, ctx *models.PipelineContext) (map[string]interface{}, error) {
	key, ok := params["key"].(string)
	if !ok {
		return nil, fmt.Errorf("key parameter is required")
	}

	value, ok := params["value"]
	if !ok {
		return nil, fmt.Errorf("value parameter is required")
	}

	stepName, ok := params["step"].(string)
	if !ok {
		stepName = "_global"
	}

	if strValue, ok := value.(string); ok {
		value = p.ResolveTemplates(strValue, ctx)
	}

	ctx.SetStepData(stepName, key, value)
	return map[string]interface{}{"success": true}, nil
}

func (p *DefaultPlugin) getContext(params map[string]interface{}, ctx *models.PipelineContext) (map[string]interface{}, error) {
	key, ok := params["key"].(string)
	if !ok {
		return nil, fmt.Errorf("key parameter is required")
	}

	stepName, ok := params["step"].(string)
	if !ok {
		stepName = "_global"
	}

	value, exists := ctx.GetStepData(stepName, key)
	return map[string]interface{}{
		"exists": exists,
		"value":  value,
	}, nil
}

func (p *DefaultPlugin) gotoAction(params map[string]interface{}, ctx *models.PipelineContext) (map[string]interface{}, error) {
	target, ok := params["target"].(string)
	if !ok {
		return nil, fmt.Errorf("target parameter is required")
	}
	return map[string]interface{}{
		"goto":   target,
		"action": "goto",
	}, nil
}
