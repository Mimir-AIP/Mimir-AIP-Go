package pipeline

import (
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/mimir-aip/mimir-aip-go/pkg/models"
)

func (p *DefaultPlugin) loadCheckpoint(params map[string]interface{}, ctx *models.PipelineContext) (map[string]interface{}, error) {
	if p.checkpointStore == nil {
		return nil, fmt.Errorf("load_checkpoint: checkpoint store is not available")
	}

	projectID, pipelineID, stepName, scope, err := p.resolveCheckpointIdentity(params, ctx)
	if err != nil {
		return nil, fmt.Errorf("load_checkpoint: %w", err)
	}

	checkpoint, err := p.checkpointStore.GetPipelineCheckpoint(projectID, pipelineID, stepName, scope)
	if err != nil {
		return nil, fmt.Errorf("load_checkpoint: failed to load checkpoint: %w", err)
	}

	defaultValue, err := p.resolveObjectParam(params["default"], ctx, false)
	if err != nil {
		return nil, fmt.Errorf("load_checkpoint: invalid default: %w", err)
	}
	if defaultValue == nil {
		defaultValue = map[string]interface{}{}
	}

	result := map[string]interface{}{
		"exists":     checkpoint != nil,
		"version":    0,
		"checkpoint": defaultValue,
	}
	if checkpoint == nil {
		return result, nil
	}

	result["version"] = checkpoint.Version
	result["checkpoint"] = checkpoint.Checkpoint
	result["updated_at"] = checkpoint.UpdatedAt.UTC().Format(time.RFC3339)
	return result, nil
}

func (p *DefaultPlugin) saveCheckpoint(params map[string]interface{}, ctx *models.PipelineContext) (map[string]interface{}, error) {
	if p.checkpointStore == nil {
		return nil, fmt.Errorf("save_checkpoint: checkpoint store is not available")
	}

	projectID, pipelineID, stepName, scope, err := p.resolveCheckpointIdentity(params, ctx)
	if err != nil {
		return nil, fmt.Errorf("save_checkpoint: %w", err)
	}

	checkpointData, err := p.resolveObjectParam(params["checkpoint"], ctx, true)
	if err != nil {
		return nil, fmt.Errorf("save_checkpoint: invalid checkpoint: %w", err)
	}
	version, err := p.resolveOptionalIntParam(params["version"], ctx)
	if err != nil {
		return nil, fmt.Errorf("save_checkpoint: invalid version: %w", err)
	}

	checkpoint := &models.PipelineCheckpoint{
		ProjectID:  projectID,
		PipelineID: pipelineID,
		StepName:   stepName,
		Scope:      scope,
		Version:    version,
		Checkpoint: checkpointData,
	}
	if err := p.checkpointStore.SavePipelineCheckpoint(checkpoint); err != nil {
		return nil, fmt.Errorf("save_checkpoint: failed to save checkpoint: %w", err)
	}

	return map[string]interface{}{
		"saved":      true,
		"version":    checkpoint.Version,
		"checkpoint": checkpoint.Checkpoint,
		"updated_at": checkpoint.UpdatedAt.UTC().Format(time.RFC3339),
	}, nil
}

func (p *DefaultPlugin) ingestCSVURL(params map[string]interface{}, ctx *models.PipelineContext) (map[string]interface{}, error) {
	url, ok := params["url"].(string)
	if !ok || strings.TrimSpace(url) == "" {
		return nil, fmt.Errorf("ingest_csv_url: url parameter is required")
	}
	url = p.ResolveTemplates(url, ctx)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("ingest_csv_url: failed to create request: %w", err)
	}
	if headers, ok := params["headers"].(map[string]interface{}); ok {
		for key, value := range headers {
			if strValue, ok := value.(string); ok {
				req.Header.Set(key, p.ResolveTemplates(strValue, ctx))
			}
		}
	}

	checkpoint, err := p.parseConnectorCheckpoint(params["checkpoint"], ctx)
	if err != nil {
		return nil, fmt.Errorf("ingest_csv_url: invalid checkpoint: %w", err)
	}
	if checkpoint.ETag != "" {
		req.Header.Set("If-None-Match", checkpoint.ETag)
	}
	if checkpoint.LastModified != "" {
		req.Header.Set("If-Modified-Since", checkpoint.LastModified)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ingest_csv_url: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotModified {
		checkpoint.ETag = resp.Header.Get("ETag")
		checkpoint.LastModified = resp.Header.Get("Last-Modified")
		return map[string]interface{}{
			"items":       []interface{}{},
			"new_count":   0,
			"total_count": 0,
			"checkpoint":  checkpoint.toMap(),
		}, nil
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ingest_csv_url: unexpected status %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("ingest_csv_url: failed to read response: %w", err)
	}

	result, err := p.ingestCSVWithCheckpoint(string(bodyBytes), params, checkpoint)
	if err != nil {
		return nil, fmt.Errorf("ingest_csv_url: %w", err)
	}
	checkpointMap := result["checkpoint"].(map[string]interface{})
	checkpointMap["etag"] = resp.Header.Get("ETag")
	checkpointMap["last_modified"] = resp.Header.Get("Last-Modified")
	return result, nil
}

func (p *DefaultPlugin) querySQL(params map[string]interface{}, ctx *models.PipelineContext) (map[string]interface{}, error) {
	dsn, ok := params["dsn"].(string)
	if !ok || strings.TrimSpace(dsn) == "" {
		return nil, fmt.Errorf("query_sql: dsn parameter is required")
	}
	dsn = p.ResolveTemplates(dsn, ctx)

	query, ok := params["query"].(string)
	if !ok || strings.TrimSpace(query) == "" {
		return nil, fmt.Errorf("query_sql: query parameter is required")
	}
	query = p.ResolveTemplates(query, ctx)

	driver, _ := params["driver"].(string)
	if driver == "" {
		driver = "mysql"
	}

	db, err := sql.Open(driver, dsn)
	if err != nil {
		return nil, fmt.Errorf("query_sql: failed to open database: %w", err)
	}
	defer db.Close()

	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("query_sql: query failed: %w", err)
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("query_sql: failed to read columns: %w", err)
	}

	items := make([]interface{}, 0)
	cursorColumn, _ := params["cursor_column"].(string)
	var nextCursor interface{}
	for rows.Next() {
		values := make([]interface{}, len(columns))
		scans := make([]interface{}, len(columns))
		for i := range values {
			scans[i] = &values[i]
		}
		if err := rows.Scan(scans...); err != nil {
			return nil, fmt.Errorf("query_sql: row scan failed: %w", err)
		}

		row := make(map[string]interface{}, len(columns))
		for i, column := range columns {
			row[column] = normalizeSQLValue(values[i])
		}
		if cursorColumn != "" {
			if cursorVal, exists := row[cursorColumn]; exists {
				nextCursor = cursorVal
			}
		}
		items = append(items, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("query_sql: row iteration failed: %w", err)
	}

	result := map[string]interface{}{
		"items":     items,
		"row_count": len(items),
	}
	if cursorColumn != "" && nextCursor != nil {
		result["next_cursor"] = nextCursor
		result["checkpoint"] = map[string]interface{}{
			"last_cursor":   nextCursor,
			"cursor_column": cursorColumn,
		}
	}
	return result, nil
}

func normalizeSQLValue(value interface{}) interface{} {
	switch typed := value.(type) {
	case nil:
		return nil
	case []byte:
		return string(typed)
	case time.Time:
		return typed.UTC().Format(time.RFC3339)
	default:
		return typed
	}
}

func (p *DefaultPlugin) ingestCSVWithCheckpoint(csvData string, params map[string]interface{}, checkpoint connectorCheckpoint) (map[string]interface{}, error) {
	reader := csv.NewReader(strings.NewReader(csvData))
	if delimiter, ok := params["delimiter"].(string); ok && delimiter != "" {
		reader.Comma = []rune(delimiter)[0]
	}

	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to parse CSV: %w", err)
	}
	if len(records) == 0 {
		return map[string]interface{}{
			"items":       []interface{}{},
			"new_count":   0,
			"total_count": 0,
			"checkpoint":  checkpoint.toMap(),
		}, nil
	}

	hasHeader := true
	if hasHeaderParam, ok := params["has_header"].(bool); ok {
		hasHeader = hasHeaderParam
	}

	headers := make([]string, 0)
	dataStart := 0
	if hasHeader {
		headers = append(headers, records[0]...)
		dataStart = 1
	} else {
		for idx := range records[0] {
			headers = append(headers, fmt.Sprintf("column_%d", idx+1))
		}
	}

	items := make([]interface{}, 0, len(records)-dataStart)
	for rowIdx := dataStart; rowIdx < len(records); rowIdx++ {
		row := records[rowIdx]
		if len(row) != len(headers) {
			continue
		}
		item := make(map[string]interface{}, len(headers))
		for i, header := range headers {
			item[header] = row[i]
		}
		items = append(items, item)
	}

	maxSeen := parseMaxCheckpointItems(params["max_checkpoint_items"])
	newItems, nextCheckpoint := dedupeByCheckpoint(items, checkpoint, maxSeen)
	return map[string]interface{}{
		"items":       newItems,
		"headers":     headers,
		"new_count":   len(newItems),
		"total_count": len(items),
		"checkpoint":  nextCheckpoint.toMap(),
	}, nil
}

func (p *DefaultPlugin) resolveCheckpointIdentity(params map[string]interface{}, ctx *models.PipelineContext) (string, string, string, string, error) {
	projectID := p.resolveStringParam(params["project_id"], ctx)
	if projectID == "" {
		projectID = p.runtimeValue(ctx, "project_id")
	}
	pipelineID := p.resolveStringParam(params["pipeline_id"], ctx)
	if pipelineID == "" {
		pipelineID = p.runtimeValue(ctx, "pipeline_id")
	}
	stepName := p.resolveStringParam(params["step_name"], ctx)
	if stepName == "" {
		stepName = p.runtimeValue(ctx, "current_step")
	}
	scope := p.resolveStringParam(params["scope"], ctx)

	if projectID == "" || pipelineID == "" || stepName == "" {
		return "", "", "", "", fmt.Errorf("project_id, pipeline_id, and step_name must be available")
	}
	return projectID, pipelineID, stepName, scope, nil
}

func (p *DefaultPlugin) runtimeValue(ctx *models.PipelineContext, key string) string {
	value, _ := ctx.GetStepData("_runtime", key)
	return strings.TrimSpace(fmt.Sprintf("%v", value))
}

func (p *DefaultPlugin) resolveStringParam(raw interface{}, ctx *models.PipelineContext) string {
	switch typed := raw.(type) {
	case string:
		return strings.TrimSpace(p.ResolveTemplates(typed, ctx))
	case nil:
		return ""
	default:
		return strings.TrimSpace(fmt.Sprintf("%v", typed))
	}
}

func (p *DefaultPlugin) resolveOptionalIntParam(raw interface{}, ctx *models.PipelineContext) (int, error) {
	switch typed := raw.(type) {
	case nil:
		return 0, nil
	case int:
		return typed, nil
	case float64:
		return int(typed), nil
	case string:
		resolved := strings.TrimSpace(p.ResolveTemplates(typed, ctx))
		if resolved == "" {
			return 0, nil
		}
		value, err := strconv.Atoi(resolved)
		if err != nil {
			return 0, err
		}
		return value, nil
	default:
		return 0, fmt.Errorf("unsupported integer value type %T", raw)
	}
}

func (p *DefaultPlugin) resolveObjectParam(raw interface{}, ctx *models.PipelineContext, required bool) (map[string]interface{}, error) {
	if raw == nil {
		if required {
			return nil, fmt.Errorf("object value is required")
		}
		return nil, nil
	}

	switch typed := raw.(type) {
	case map[string]interface{}:
		return typed, nil
	case string:
		resolved := strings.TrimSpace(p.ResolveTemplates(typed, ctx))
		if resolved == "" {
			if required {
				return nil, fmt.Errorf("object value is required")
			}
			return nil, nil
		}
		var out map[string]interface{}
		if err := json.Unmarshal([]byte(resolved), &out); err != nil {
			return nil, err
		}
		return out, nil
	default:
		bytes, err := json.Marshal(typed)
		if err != nil {
			return nil, err
		}
		var out map[string]interface{}
		if err := json.Unmarshal(bytes, &out); err != nil {
			return nil, err
		}
		return out, nil
	}
}
