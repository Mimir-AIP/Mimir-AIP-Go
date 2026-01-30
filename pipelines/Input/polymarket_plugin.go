package Input

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/Mimir-AIP/Mimir-AIP-Go/pipelines"
)

// PolymarketPlugin fetches markets from Polymarket public API
type PolymarketPlugin struct{}

func NewPolymarketPlugin() *PolymarketPlugin { return &PolymarketPlugin{} }

func (p *PolymarketPlugin) ExecuteStep(ctx context.Context, stepConfig pipelines.StepConfig, globalContext *pipelines.PluginContext) (*pipelines.PluginContext, error) {
	cfg := stepConfig.Config
	category, _ := cfg["category"].(string)
	limit := 50
	if l, ok := cfg["limit"].(int); ok {
		limit = l
	}
	if l64, ok := cfg["limit"].(float64); ok {
		limit = int(l64)
	}

	// default to Gamma API markets endpoint
	base := "https://gamma-api.polymarket.com/markets"
	if b, ok := cfg["base_url"].(string); ok && b != "" {
		base = b
	} else if envb := os.Getenv("POLYMARKET_API_BASE"); envb != "" {
		base = envb
	}
	u, _ := url.Parse(base)
	q := u.Query()
	q.Set("limit", fmt.Sprintf("%d", limit))
	// support offset pagination
	if off, ok := cfg["offset"].(int); ok {
		q.Set("offset", fmt.Sprintf("%d", off))
	} else if offf, ok := cfg["offset"].(float64); ok {
		q.Set("offset", fmt.Sprintf("%d", int(offf)))
	}
	if category != "" {
		q.Set("category", category)
	}
	// Gamma uses closed=true/false to filter
	if closed, ok := cfg["closed"].(bool); ok {
		q.Set("closed", strconv.FormatBool(closed))
	}
	u.RawQuery = q.Encode()

	req, _ := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	req.Header.Set("User-Agent", "Mimir-AIP/1.0")

	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("polymarket request failed: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read polymarket/gamma response: %w", err)
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("polymarket/gamma returned %d: %s", resp.StatusCode, string(bodyBytes))
	}

	// decode into array of markets or object containing markets
	var arr []map[string]any
	if err := json.Unmarshal(bodyBytes, &arr); err != nil {
		// try object
		var obj map[string]any
		if err := json.Unmarshal(bodyBytes, &obj); err != nil {
			return nil, fmt.Errorf("failed to parse polymarket/gamma response: %w", err)
		}
		// try 'markets' key
		if mraw, ok := obj["markets"]; ok {
			if marr, ok := mraw.([]any); ok {
				for _, it := range marr {
					if mm, ok := it.(map[string]any); ok {
						arr = append(arr, mm)
					}
				}
			}
		}
	}

	// normalize markets into rows
	var rows []map[string]any
	for _, mo := range arr {
		row := map[string]any{}
		if v, ok := mo["id"]; ok {
			row["id"] = v
		}
		if v, ok := mo["slug"]; ok {
			row["slug"] = v
		}
		if v, ok := mo["question"]; ok {
			row["question"] = v
		} else if v, ok := mo["title"]; ok {
			row["question"] = v
		}
		if v, ok := mo["category"]; ok {
			row["category"] = v
		}
		if v, ok := mo["endDate"]; ok {
			row["end_date"] = v
		}
		if v, ok := mo["createdAt"]; ok {
			row["created_at"] = v
		}
		if v, ok := mo["updatedAt"]; ok {
			row["updated_at"] = v
		}
		// volume / liquidity
		if v, ok := mo["volumeNum"]; ok {
			row["volume"] = v
		} else if s, ok := mo["volume"].(string); ok {
			if f, err := strconv.ParseFloat(s, 64); err == nil {
				row["volume"] = f
			} else {
				row["volume"] = s
			}
		}
		if v, ok := mo["liquidityNum"]; ok {
			row["liquidity"] = v
		}

		// outcomes may be stringified JSON
		if outs, ok := mo["outcomes"]; ok {
			switch t := outs.(type) {
			case string:
				var parsed []string
				if err := json.Unmarshal([]byte(t), &parsed); err == nil {
					row["outcomes"] = parsed
				} else {
					row["outcomes"] = t
				}
			default:
				row["outcomes"] = t
			}
		}
		if ops, ok := mo["outcomePrices"]; ok {
			switch t := ops.(type) {
			case string:
				var parsed []string
				if err := json.Unmarshal([]byte(t), &parsed); err == nil {
					var farr []float64
					for _, s := range parsed {
						if f, err := strconv.ParseFloat(s, 64); err == nil {
							farr = append(farr, f)
						}
					}
					row["outcome_prices"] = farr
				} else {
					row["outcome_prices"] = t
				}
			default:
				row["outcome_prices"] = t
			}
		}
		if ct, ok := mo["clobTokenIds"]; ok {
			switch t := ct.(type) {
			case string:
				var parsed []string
				if err := json.Unmarshal([]byte(t), &parsed); err == nil {
					row["clob_token_ids"] = parsed
				} else {
					row["clob_token_ids"] = t
				}
			default:
				row["clob_token_ids"] = t
			}
		}
		rows = append(rows, row)
	}

	out := pipelines.NewPluginContext()
	outKey := stepConfig.Output
	if outKey == "" {
		outKey = "polymarket_markets"
	}
	out.Set(outKey, map[string]any{"rows": rows, "fetched_at": time.Now().Format(time.RFC3339)})
	return out, nil
}

func (p *PolymarketPlugin) GetPluginType() string { return "Input" }
func (p *PolymarketPlugin) GetPluginName() string { return "polymarket" }

func (p *PolymarketPlugin) ValidateConfig(config map[string]any) error { return nil }

func (p *PolymarketPlugin) GetInputSchema() map[string]any {
	return map[string]any{"type": "object", "properties": map[string]any{"category": map[string]any{"type": "string"}, "limit": map[string]any{"type": "integer"}}}
}
