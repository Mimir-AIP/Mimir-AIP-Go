package mcp

import (
	"encoding/json"
	"fmt"
	"strings"
)

func parseJSONMap(raw string) (map[string]any, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, fmt.Errorf("json object is required")
	}
	decoded := map[string]any{}
	if err := json.Unmarshal([]byte(trimmed), &decoded); err != nil {
		return nil, fmt.Errorf("value must be a JSON object: %w", err)
	}
	return decoded, nil
}
