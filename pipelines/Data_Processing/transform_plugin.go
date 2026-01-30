package Data_Processing

import (
	"context"
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/Mimir-AIP/Mimir-AIP-Go/pipelines"
)

// TransformPlugin implements basic data transformation operations
type TransformPlugin struct {
	name string
}

// NewTransformPlugin creates a new transform plugin
func NewTransformPlugin() *TransformPlugin {
	return &TransformPlugin{name: "transform"}
}

// ExecuteStep executes a transform operation on input data stored in the global context
func (p *TransformPlugin) ExecuteStep(ctx context.Context, stepConfig pipelines.StepConfig, globalContext *pipelines.PluginContext) (*pipelines.PluginContext, error) {
	// Execute transform step
	cfg := stepConfig.Config

	inputKey, _ := cfg["input"].(string)
	if inputKey == "" {
		return nil, fmt.Errorf("config 'input' (context key) is required")
	}

	rawAny, exists := globalContext.Get(inputKey)
	if !exists {
		return nil, fmt.Errorf("input key not found in context: %s", inputKey)
	}

	// Extract rows from common shapes
	var rows []map[string]any

	switch v := rawAny.(type) {
	case map[string]any:
		// common shapes: {"rows": [...]} or {"value": [...]}
		if r, ok := v["rows"].([]any); ok {
			for _, item := range r {
				if m, ok := item.(map[string]any); ok {
					rows = append(rows, m)
				}
			}
		} else if rv, ok := v["value"]; ok {
			// value may be []map[string]any or []any
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
				// treat the map as a single row
				rows = append(rows, v)
			}
		} else {
			// maybe the map itself is a single record with fields -> treat as one row
			rows = append(rows, v)
		}
	case []map[string]any:
		rows = v
	case []any:
		for _, item := range v {
			if m, ok := item.(map[string]any); ok {
				rows = append(rows, m)
			}
		}
	default:
		// try reflection-based detection for slices
		rv := reflect.ValueOf(rawAny)
		if rv.Kind() == reflect.Slice {
			for i := 0; i < rv.Len(); i++ {
				item := rv.Index(i).Interface()
				if m, ok := item.(map[string]any); ok {
					rows = append(rows, m)
				}
			}
		}
	}

	if rows == nil {
		return nil, fmt.Errorf("unsupported input data shape for key %s", inputKey)
	}

	operation, _ := cfg["operation"].(string)
	if operation == "" {
		return nil, fmt.Errorf("config 'operation' is required")
	}

	var outRows []map[string]any

	switch operation {
	case "select":
		fieldsAny, ok := cfg["fields"].([]any)
		if !ok {
			return nil, fmt.Errorf("select operation requires 'fields' array")
		}
		var fields []string
		for _, f := range fieldsAny {
			if s, ok := f.(string); ok {
				fields = append(fields, s)
			}
		}
		for _, row := range rows {
			// ensure we have data
			// (used during tests to diagnose empty-row issues)
			// fmt.Printf("rename: processing row keys=%v\n", reflect.ValueOf(row).MapKeys())
			newRow := make(map[string]any)
			for _, f := range fields {
				if val, ok := row[f]; ok {
					newRow[f] = val
				}
			}
			outRows = append(outRows, newRow)
		}

	case "rename":
		mapping, ok := cfg["mapping"].(map[string]any)
		if !ok {
			return nil, fmt.Errorf("rename operation requires 'mapping' object")
		}
		// debug: ensure rows present
		// fmt.Printf("rename op: input rows count=%d\n", len(rows))
		for _, row := range rows {
			newRow := make(map[string]any)
			for k, v := range row {
				if newNameAny, ok := mapping[k]; ok {
					if newName, ok := newNameAny.(string); ok {
						newRow[newName] = v
						continue
					}
				}
				newRow[k] = v
			}
			outRows = append(outRows, newRow)
		}

	case "filter":
		// Support simple filter specification: field, op, value
		field, ok := cfg["field"].(string)
		if !ok || field == "" {
			// field optional when using expression
			// return error only if no expression provided
			if _, hasExpr := cfg["expression"].(string); !hasExpr {
				return nil, fmt.Errorf("filter operation requires 'field' string or 'expression' string")
			}
		}
		op, _ := cfg["op"].(string)
		if op == "" {
			op = "=="
		}
		rawVal := cfg["value"]
		// evaluate per-row either via expression or via simple field comparison
		for _, row := range rows {
			// if expression provided, evaluate it
			if expr, ok := cfg["expression"].(string); ok && expr != "" {
				okMatch, err := EvaluateExpression(expr, row)
				if err != nil {
					return nil, fmt.Errorf("failed to evaluate expression: %w", err)
				}
				if okMatch {
					outRows = append(outRows, row)
				}
				continue
			}
			v, exists := row[field]
			if !exists {
				continue
			}
			keep := false
			switch op {
			case "==":
				keep = fmt.Sprintf("%v", v) == fmt.Sprintf("%v", rawVal)
			case "!=":
				keep = fmt.Sprintf("%v", v) != fmt.Sprintf("%v", rawVal)
			case ">", "<", ">=", "<=":
				// Try numeric comparison
				var fv, rv float64
				switch t := v.(type) {
				case int:
					fv = float64(t)
				case int64:
					fv = float64(t)
				case float32:
					fv = float64(t)
				case float64:
					fv = t
				case string:
					if parsed, err := strconv.ParseFloat(t, 64); err == nil {
						fv = parsed
					} else {
						// cannot compare
						keep = false
						fv = 0
					}
				default:
					fv = 0
				}
				switch t := rawVal.(type) {
				case int:
					rv = float64(t)
				case int64:
					rv = float64(t)
				case float32:
					rv = float64(t)
				case float64:
					rv = t
				case string:
					if parsed, err := strconv.ParseFloat(t, 64); err == nil {
						rv = parsed
					} else {
						rv = 0
					}
				default:
					rv = 0
				}
				switch op {
				case ">":
					keep = fv > rv
				case "<":
					keep = fv < rv
				case ">=":
					keep = fv >= rv
				case "<=":
					keep = fv <= rv
				}
			default:
				// unknown op, skip
				keep = false
			}
			if keep {
				outRows = append(outRows, row)
			}
		}

	default:
		// additional operations implemented below
		if operation == "aggregate" {
			// group_by: []string, aggregations: [{"field": "revenue", "op": "sum", "as": "total_rev"}, ...]
			groupByAny, _ := cfg["group_by"].([]any)
			var groupBy []string
			for _, g := range groupByAny {
				if s, ok := g.(string); ok {
					groupBy = append(groupBy, s)
				}
			}
			aggsAny, _ := cfg["aggregations"].([]any)
			type aggDef struct{ Field, Op, As string }
			var aggs []aggDef
			for _, a := range aggsAny {
				if m, ok := a.(map[string]any); ok {
					f, _ := m["field"].(string)
					op, _ := m["op"].(string)
					as, _ := m["as"].(string)
					if as == "" && f != "" {
						as = f + "_" + op
					}
					aggs = append(aggs, aggDef{Field: f, Op: op, As: as})
				}
			}
			// state maps
			type aggState struct {
				sum    map[string]float64
				count  map[string]int
				min    map[string]float64
				max    map[string]float64
				minSet map[string]bool
			}
			states := make(map[string]*aggState)
			groupVals := make(map[string]map[string]any)
			for _, r := range rows {
				// build group key
				gk := ""
				vals := make(map[string]any)
				for _, gb := range groupBy {
					if v, ok := r[gb]; ok {
						gk += fmt.Sprintf("%v|", v)
						vals[gb] = v
					} else {
						gk += "|"
						vals[gb] = nil
					}
				}
				if _, ok := states[gk]; !ok {
					states[gk] = &aggState{sum: map[string]float64{}, count: map[string]int{}, min: map[string]float64{}, max: map[string]float64{}, minSet: map[string]bool{}}
					groupVals[gk] = vals
				}
				st := states[gk]
				for _, a := range aggs {
					val, _ := r[a.Field]
					if a.Op == "count" {
						st.count[a.As]++
						continue
					}
					if f, ok := toFloat64OK(val); ok {
						st.sum[a.As] += f
						st.count[a.As]++
						if !st.minSet[a.As] || f < st.min[a.As] {
							st.min[a.As] = f
							st.minSet[a.As] = true
						}
						if !st.minSet[a.As] || f > st.max[a.As] {
							st.max[a.As] = f
						}
					}
				}
			}
			// build output rows
			for gk, st := range states {
				row := map[string]any{}
				if vals, ok := groupVals[gk]; ok {
					for k, v := range vals {
						row[k] = v
					}
				}
				for _, a := range aggs {
					switch a.Op {
					case "sum":
						row[a.As] = st.sum[a.As]
					case "count":
						row[a.As] = st.count[a.As]
					case "avg":
						if st.count[a.As] > 0 {
							row[a.As] = st.sum[a.As] / float64(st.count[a.As])
						} else {
							row[a.As] = 0.0
						}
					case "min":
						row[a.As] = st.min[a.As]
					case "max":
						row[a.As] = st.max[a.As]
					default:
						// unknown op - leave nil
						row[a.As] = nil
					}
				}
				outRows = append(outRows, row)
			}
		} else if operation == "flatten" {
			// flatten nested maps into top-level keys. config: fields optional []string, sep optional string
			fieldsAny, _ := cfg["fields"].([]any)
			var fieldsToFlatten []string
			for _, f := range fieldsAny {
				if s, ok := f.(string); ok {
					fieldsToFlatten = append(fieldsToFlatten, s)
				}
			}
			sep, _ := cfg["sep"].(string)
			if sep == "" {
				sep = "."
			}
			for _, r := range rows {
				newRow := map[string]any{}
				// copy non-flattened fields first
				for k, v := range r {
					newRow[k] = v
				}
				// helper
				var doFlatten func(prefix string, v any)
				doFlatten = func(prefix string, v any) {
					switch vv := v.(type) {
					case map[string]any:
						for kk, vv2 := range vv {
							key := kk
							if prefix != "" {
								key = prefix + sep + kk
							}
							doFlatten(key, vv2)
						}
					default:
						newRow[prefix] = vv
					}
				}
				if len(fieldsToFlatten) == 0 {
					// flatten any nested maps at top level
					for k, v := range r {
						if m, ok := v.(map[string]any); ok {
							// remove original
							delete(newRow, k)
							doFlatten(k, m)
						}
					}
				} else {
					for _, k := range fieldsToFlatten {
						if v, ok := r[k]; ok {
							if m, ok2 := v.(map[string]any); ok2 {
								delete(newRow, k)
								doFlatten(k, m)
							}
						}
					}
				}
				outRows = append(outRows, newRow)
			}
		} else if operation == "sort" {
			// sort by one or more fields. config: keys: [{field, asc (bool)}] or keys: ["field1", "field2"]
			keysAny, _ := cfg["keys"].([]any)
			var keys []struct {
				Field string
				Asc   bool
			}
			for _, k := range keysAny {
				switch t := k.(type) {
				case string:
					keys = append(keys, struct {
						Field string
						Asc   bool
					}{Field: t, Asc: true})
				case map[string]any:
					f, _ := t["field"].(string)
					asc := true
					if a, ok := t["asc"].(bool); ok {
						asc = a
					}
					keys = append(keys, struct {
						Field string
						Asc   bool
					}{Field: f, Asc: asc})
				}
			}
			// copy rows
			outRows = append(outRows, rows...)
			// simple multi-key sort using closure
			less := func(i, j int) bool {
				for _, k := range keys {
					vi, _ := outRows[i][k.Field]
					vj, _ := outRows[j][k.Field]
					// compare as string fallback
					si := fmt.Sprintf("%v", vi)
					sj := fmt.Sprintf("%v", vj)
					if si == sj {
						continue
					}
					if k.Asc {
						return si < sj
					}
					return si > sj
				}
				return false
			}
			sort.SliceStable(outRows, less)
		} else if operation == "map" {
			// field-level map: config: field, expression (not yet full expr engine) or function: uppercase/lowercase
			field, _ := cfg["field"].(string)
			fn, _ := cfg["function"].(string)
			for _, r := range rows {
				newRow := make(map[string]any)
				for k, v := range r {
					newRow[k] = v
				}
				if field != "" {
					if val, ok := newRow[field]; ok {
						switch fn {
						case "uppercase":
							if s, ok := val.(string); ok {
								newRow[field] = strings.ToUpper(s)
							}
						case "lowercase":
							if s, ok := val.(string); ok {
								newRow[field] = strings.ToLower(s)
							}
						}
					}
				}
				outRows = append(outRows, newRow)
			}
		} else if operation == "parse_polymarket" {
			// parse polymarket plugin output (which usually sets {"data": [...]}) into rows
			explode := false
			if e, ok := cfg["explode_outcomes"].(bool); ok {
				explode = e
			}
			var marketsArr []any
			switch v := rawAny.(type) {
			case map[string]any:
				if d, ok := v["data"]; ok {
					if arr, ok2 := d.([]any); ok2 {
						marketsArr = arr
					} else if m, ok2 := d.(map[string]any); ok2 {
						marketsArr = []any{m}
					}
				} else if arr, ok := v["markets"].([]any); ok {
					marketsArr = arr
				}
			case []any:
				marketsArr = v
			}
			for _, mi := range marketsArr {
				if mo, ok := mi.(map[string]any); ok {
					row := map[string]any{}
					row["id"] = mo["id"]
					row["question"] = mo["question"]
					row["category"] = mo["category"]
					row["status"] = mo["status"]
					row["created_at"] = mo["createdAt"]
					row["updated_at"] = mo["updatedAt"]
					row["volume"] = mo["volume"]
					row["total_liquidity"] = mo["totalLiquidity"]
					row["last_close_price"] = mo["lastClosePrice"]
					row["url"] = mo["url"]
					outRows = append(outRows, row)
					if explode {
						if outs, ok := mo["outcomes"].([]any); ok {
							for _, o := range outs {
								if oo, ok := o.(map[string]any); ok {
									orow := map[string]any{"market_id": mo["id"], "outcome_id": oo["id"], "label": oo["label"], "probability": oo["probability"], "price": oo["price"]}
									outRows = append(outRows, orow)
								}
							}
						}
					}
				}
			}
		} else {
			return nil, fmt.Errorf("unsupported transform operation: %s", operation)
		}
	}

	result := map[string]any{
		"operation":  operation,
		"total_rows": len(rows),
		"row_count":  len(outRows),
		"rows":       outRows,
	}

	outCtx := pipelines.NewPluginContext()
	outKey := stepConfig.Output
	if outKey == "" {
		outKey = inputKey + "_transformed"
	}
	outCtx.Set(outKey, result)
	return outCtx, nil
}

func (p *TransformPlugin) GetPluginType() string { return "Data_Processing" }
func (p *TransformPlugin) GetPluginName() string { return "transform" }

func (p *TransformPlugin) ValidateConfig(config map[string]any) error {
	if _, ok := config["input"].(string); !ok {
		return fmt.Errorf("config 'input' is required and must be a string")
	}
	if _, ok := config["operation"].(string); !ok {
		return fmt.Errorf("config 'operation' is required and must be a string")
	}
	return nil
}

func (p *TransformPlugin) GetInputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"input":     map[string]any{"type": "string", "description": "Context key containing input data (rows)"},
			"operation": map[string]any{"type": "string", "description": "Operation to perform", "enum": []string{"select", "rename", "filter"}},
			"fields":    map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			"mapping":   map[string]any{"type": "object", "additionalProperties": map[string]any{"type": "string"}},
			"field":     map[string]any{"type": "string"},
			"op":        map[string]any{"type": "string"},
			"value":     map[string]any{"type": []string{"string", "number", "boolean"}},
		},
		"required": []string{"input", "operation"},
	}
}
