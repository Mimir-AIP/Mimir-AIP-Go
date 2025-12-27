package Input

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/Mimir-AIP/Mimir-AIP-Go/pipelines"
)

// MySQLPlugin handles MySQL database connections and data extraction
type MySQLPlugin struct{}

// GetPluginType returns the plugin type
func (p *MySQLPlugin) GetPluginType() string {
	return "Input"
}

// GetPluginName returns the plugin name
func (p *MySQLPlugin) GetPluginName() string {
	return "mysql"
}

// ValidateConfig validates MySQL connection configuration
func (p *MySQLPlugin) ValidateConfig(config map[string]any) error {
	if config["host"] == nil {
		return fmt.Errorf("host is required")
	}
	if config["database"] == nil {
		return fmt.Errorf("database is required")
	}
	if config["username"] == nil {
		return fmt.Errorf("username is required")
	}
	if config["query"] == nil {
		return fmt.Errorf("query is required")
	}
	return nil
}

// ExecuteStep connects to MySQL and executes the query
func (p *MySQLPlugin) ExecuteStep(ctx context.Context, stepConfig pipelines.StepConfig, globalContext *pipelines.PluginContext) (*pipelines.PluginContext, error) {
	// Extract configuration with proper type assertions
	host, _ := stepConfig.Config["host"].(string)
	port, _ := stepConfig.Config["port"].(int)
	if port == 0 {
		port = 3306
	}
	database, _ := stepConfig.Config["database"].(string)
	username, _ := stepConfig.Config["username"].(string)
	password, _ := stepConfig.Config["password"].(string)
	query, _ := stepConfig.Config["query"].(string)

	// Build connection string
	connStr := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s", username, password, host, port, database)
	if password == "" {
		connStr = fmt.Sprintf("%s@tcp(%s:%d)/%s", username, host, port, database)
	}

	// Connect to database
	db, err := sql.Open("mysql", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MySQL: %w", err)
	}
	defer db.Close()

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping MySQL: %w", err)
	}

	// Execute query
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}
	defer rows.Close()

	// Get column information
	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("failed to get columns: %w", err)
	}

	// Convert rows to slice of maps
	var results []map[string]interface{}
	for rows.Next() {
		row := make(map[string]interface{})
		columnValues := make([]interface{}, len(columns))
		columnPtrs := make([]interface{}, len(columns))

		for i := range columns {
			columnPtrs[i] = &columnValues[i]
		}

		if err := rows.Scan(columnPtrs...); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		for i, col := range columns {
			row[col] = columnValues[i]
		}

		results = append(results, row)
	}

	// Check for errors from iterating over rows
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	// Create result context with proper method calls
	result := pipelines.NewPluginContext()
	result.Set("connection", map[string]any{
		"host":     host,
		"port":     port,
		"database": database,
		"username": username,
		"query":    query,
	})
	result.Set("query_results", map[string]any{
		"row_count":    len(results),
		"column_count": len(columns),
		"columns":      columns,
		"rows":         results,
		"query":        query,
	})
	result.SetMetadata("source_type", "mysql_database")
	result.SetMetadata("extracted_at", time.Now().Format(time.RFC3339))
	result.SetMetadata("schema", map[string]any{
		"columns":    columns,
		"total_rows": len(results),
	})

	return result, nil
}

// GetInputSchema returns the input schema for the MySQL plugin
func (p *MySQLPlugin) GetInputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"host": map[string]any{
				"type":        "string",
				"description": "MySQL server hostname or IP address",
				"required":    true,
			},
			"port": map[string]any{
				"type":        "integer",
				"description": "MySQL server port (default: 3306)",
				"default":     3306,
			},
			"database": map[string]any{
				"type":        "string",
				"description": "Database name",
				"required":    true,
			},
			"username": map[string]any{
				"type":        "string",
				"description": "MySQL username",
				"required":    true,
			},
			"password": map[string]any{
				"type":        "string",
				"description": "MySQL password (optional if using auth socket)",
				"required":    false,
			},
			"query": map[string]any{
				"type":        "string",
				"description": "SQL query to execute",
				"required":    true,
			},
		},
	}
}
