package pipeline

import (
	"database/sql"
	"encoding/csv"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/mimir-aip/mimir-aip-go/pkg/models"
)

var sqlIdentifierPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_\.]*$`)

func (p *DefaultPlugin) pollSQLIncremental(params map[string]interface{}, ctx *models.PipelineContext) (map[string]interface{}, error) {
	driver, dsn, err := p.resolveSQLSource(params, ctx)
	if err != nil {
		return nil, fmt.Errorf("poll_sql_incremental: %w", err)
	}
	table := p.resolveStringParam(params["table"], ctx)
	cursorColumn := p.resolveStringParam(params["cursor_column"], ctx)
	if table == "" || cursorColumn == "" {
		return nil, fmt.Errorf("poll_sql_incremental: table and cursor_column are required")
	}
	if !sqlIdentifierPattern.MatchString(table) || !sqlIdentifierPattern.MatchString(cursorColumn) {
		return nil, fmt.Errorf("poll_sql_incremental: table and cursor_column must be simple identifiers")
	}
	checkpoint, err := p.parseConnectorCheckpoint(params["checkpoint"], ctx)
	if err != nil {
		return nil, fmt.Errorf("poll_sql_incremental: invalid checkpoint: %w", err)
	}
	limit, err := p.resolveOptionalIntParam(params["limit"], ctx)
	if err != nil {
		return nil, fmt.Errorf("poll_sql_incremental: invalid limit: %w", err)
	}
	if limit <= 0 {
		limit = 500
	}

	db, err := sql.Open(driver, dsn)
	if err != nil {
		return nil, fmt.Errorf("poll_sql_incremental: failed to open database: %w", err)
	}
	defer db.Close()

	query := fmt.Sprintf("SELECT * FROM %s", table)
	args := make([]interface{}, 0, 1)
	if checkpoint.LastCursor != nil && fmt.Sprintf("%v", checkpoint.LastCursor) != "" {
		query += fmt.Sprintf(" WHERE %s > %s", cursorColumn, sqlPlaceholder(driver, 1))
		args = append(args, checkpoint.LastCursor)
	}
	query += fmt.Sprintf(" ORDER BY %s ASC LIMIT %d", cursorColumn, limit)

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("poll_sql_incremental: query failed: %w", err)
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("poll_sql_incremental: failed to read columns: %w", err)
	}

	items := make([]interface{}, 0)
	lastCursor := checkpoint.LastCursor
	for rows.Next() {
		values := make([]interface{}, len(columns))
		scans := make([]interface{}, len(columns))
		for i := range values {
			scans[i] = &values[i]
		}
		if err := rows.Scan(scans...); err != nil {
			return nil, fmt.Errorf("poll_sql_incremental: row scan failed: %w", err)
		}
		row := make(map[string]interface{}, len(columns)+1)
		for i, column := range columns {
			row[column] = normalizeSQLValue(values[i])
		}
		if cursorVal, exists := row[cursorColumn]; exists {
			lastCursor = cursorVal
		}
		row["_source_table"] = table
		items = append(items, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("poll_sql_incremental: row iteration failed: %w", err)
	}

	checkpoint.LastCursor = lastCursor
	return map[string]interface{}{
		"items":      items,
		"row_count":  len(items),
		"checkpoint": checkpoint.toMap(),
		"source": map[string]interface{}{
			"driver": driver,
			"table":  table,
		},
	}, nil
}

func (p *DefaultPlugin) pollCSVDrop(params map[string]interface{}, ctx *models.PipelineContext) (map[string]interface{}, error) {
	pathGlob := p.resolveStringParam(params["path_glob"], ctx)
	if pathGlob == "" {
		return nil, fmt.Errorf("poll_csv_drop: path_glob parameter is required")
	}
	checkpoint, err := p.parseConnectorCheckpoint(params["checkpoint"], ctx)
	if err != nil {
		return nil, fmt.Errorf("poll_csv_drop: invalid checkpoint: %w", err)
	}
	matches, err := filepath.Glob(pathGlob)
	if err != nil {
		return nil, fmt.Errorf("poll_csv_drop: invalid path_glob: %w", err)
	}
	if len(matches) == 0 {
		return map[string]interface{}{
			"items":       []interface{}{},
			"new_count":   0,
			"total_count": 0,
			"checkpoint":  checkpoint.toMap(),
		}, nil
	}

	type fileCandidate struct {
		path string
		info os.FileInfo
		hash string
	}
	candidates := make([]fileCandidate, 0, len(matches))
	seenFiles := make(map[string]struct{}, len(checkpoint.Seen))
	for _, hash := range checkpoint.Seen {
		seenFiles[hash] = struct{}{}
	}
	for _, match := range matches {
		info, err := os.Stat(match)
		if err != nil || info.IsDir() {
			continue
		}
		hash := hashPayload(map[string]interface{}{
			"path":     match,
			"size":     info.Size(),
			"modified": info.ModTime().UTC().Format(time.RFC3339Nano),
		})
		if _, alreadySeen := seenFiles[hash]; alreadySeen {
			continue
		}
		candidates = append(candidates, fileCandidate{path: match, info: info, hash: hash})
	}
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].info.ModTime().Equal(candidates[j].info.ModTime()) {
			return candidates[i].path < candidates[j].path
		}
		return candidates[i].info.ModTime().Before(candidates[j].info.ModTime())
	})

	items := make([]interface{}, 0)
	orderedSeen := append([]string{}, checkpoint.Seen...)
	for _, candidate := range candidates {
		bytes, err := ioutil.ReadFile(candidate.path)
		if err != nil {
			return nil, fmt.Errorf("poll_csv_drop: failed to read %s: %w", candidate.path, err)
		}
		records, err := csvItemsFromData(string(bytes), params, candidate.path)
		if err != nil {
			return nil, fmt.Errorf("poll_csv_drop: failed to parse %s: %w", candidate.path, err)
		}
		items = append(items, records...)
		orderedSeen = append(orderedSeen, candidate.hash)
	}
	checkpoint.Seen = trimSeenHashes(orderedSeen, parseMaxCheckpointItems(params["max_checkpoint_items"]))
	return map[string]interface{}{
		"items":       items,
		"new_count":   len(items),
		"total_count": len(items),
		"checkpoint":  checkpoint.toMap(),
	}, nil
}

func (p *DefaultPlugin) resolveSQLSource(params map[string]interface{}, ctx *models.PipelineContext) (string, string, error) {
	driver := strings.ToLower(p.resolveStringParam(params["driver"], ctx))
	switch driver {
	case "", "mysql":
		driver = "mysql"
	case "postgres", "postgresql", "pgx":
		driver = "pgx"
	default:
		return "", "", fmt.Errorf("unsupported driver %q", driver)
	}
	dsn := p.resolveStringParam(params["dsn"], ctx)
	if dsn == "" {
		return "", "", fmt.Errorf("dsn parameter is required")
	}
	return driver, dsn, nil
}

func sqlPlaceholder(driver string, index int) string {
	if driver == "pgx" {
		return fmt.Sprintf("$%d", index)
	}
	return "?"
}

func csvItemsFromData(csvData string, params map[string]interface{}, sourceFile string) ([]interface{}, error) {
	reader := csv.NewReader(strings.NewReader(csvData))
	if delimiter, ok := params["delimiter"].(string); ok && delimiter != "" {
		reader.Comma = []rune(delimiter)[0]
	}
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to parse CSV: %w", err)
	}
	if len(records) == 0 {
		return []interface{}{}, nil
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
		item := make(map[string]interface{}, len(headers)+1)
		for idx, header := range headers {
			item[header] = row[idx]
		}
		item["_source_file"] = sourceFile
		items = append(items, item)
	}
	return items, nil
}

func trimSeenHashes(seen []string, maxSeen int) []string {
	if maxSeen <= 0 {
		maxSeen = 200
	}
	if len(seen) <= maxSeen {
		return seen
	}
	return seen[len(seen)-maxSeen:]
}
