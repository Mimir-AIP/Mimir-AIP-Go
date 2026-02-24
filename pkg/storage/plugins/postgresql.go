package plugins

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/mimir-aip/mimir-aip-go/pkg/models"
	"github.com/mimir-aip/mimir-aip-go/pkg/storage"
)

// PostgresPlugin implements the StoragePlugin interface for PostgreSQL storage
type PostgresPlugin struct {
	db          *sql.DB
	initialized bool
}

// NewPostgresPlugin creates a new PostgreSQL storage plugin
func NewPostgresPlugin() *PostgresPlugin {
	return &PostgresPlugin{}
}

// Initialize initializes the PostgreSQL plugin with configuration
func (p *PostgresPlugin) Initialize(config *models.PluginConfig) error {
	dsn := config.ConnectionString
	if dsn == "" {
		return fmt.Errorf("connection string is required for postgresql storage")
	}

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return fmt.Errorf("failed to open postgresql connection: %w", err)
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return fmt.Errorf("failed to ping postgresql: %w", err)
	}

	p.db = db
	p.initialized = true
	return nil
}

// CreateSchema creates tables based on the ontology definition
func (p *PostgresPlugin) CreateSchema(ontology *models.OntologyDefinition) error {
	if !p.initialized {
		return fmt.Errorf("plugin not initialized")
	}

	for _, entity := range ontology.Entities {
		query := fmt.Sprintf(
			`CREATE TABLE IF NOT EXISTS "%s" (id UUID PRIMARY KEY DEFAULT gen_random_uuid(), data JSONB NOT NULL, created_at TIMESTAMPTZ DEFAULT NOW())`,
			entity.Name,
		)
		if _, err := p.db.Exec(query); err != nil {
			return fmt.Errorf("failed to create table %s: %w", entity.Name, err)
		}
	}

	return nil
}

// Store stores CIR data into PostgreSQL
func (p *PostgresPlugin) Store(cir *models.CIR) (*models.StorageResult, error) {
	if !p.initialized {
		return nil, fmt.Errorf("plugin not initialized")
	}

	if err := cir.Validate(); err != nil {
		return nil, fmt.Errorf("invalid CIR: %w", err)
	}

	entityType := p.inferEntityType(cir)
	affectedItems := 0

	// Ensure table exists
	createQuery := fmt.Sprintf(
		`CREATE TABLE IF NOT EXISTS "%s" (id UUID PRIMARY KEY DEFAULT gen_random_uuid(), data JSONB NOT NULL, created_at TIMESTAMPTZ DEFAULT NOW())`,
		entityType,
	)
	if _, err := p.db.Exec(createQuery); err != nil {
		return nil, fmt.Errorf("failed to ensure table exists: %w", err)
	}

	if arr, err := cir.GetDataAsArray(); err == nil {
		for _, item := range arr {
			itemCIR := &models.CIR{
				Version:  cir.Version,
				Source:   cir.Source,
				Data:     item,
				Metadata: cir.Metadata,
			}
			if err := p.insertItem(entityType, itemCIR); err != nil {
				return nil, fmt.Errorf("failed to store item: %w", err)
			}
			affectedItems++
		}
	} else {
		if err := p.insertItem(entityType, cir); err != nil {
			return nil, fmt.Errorf("failed to store item: %w", err)
		}
		affectedItems = 1
	}

	return &models.StorageResult{
		Success:       true,
		AffectedItems: affectedItems,
	}, nil
}

func (p *PostgresPlugin) insertItem(entityType string, cir *models.CIR) error {
	data, err := json.Marshal(cir)
	if err != nil {
		return fmt.Errorf("failed to marshal CIR: %w", err)
	}

	query := fmt.Sprintf(`INSERT INTO "%s" (data) VALUES ($1)`, entityType)
	_, err = p.db.Exec(query, string(data))
	return err
}

// Retrieve retrieves CIR data from PostgreSQL using a query
func (p *PostgresPlugin) Retrieve(query *models.CIRQuery) ([]*models.CIR, error) {
	if !p.initialized {
		return nil, fmt.Errorf("plugin not initialized")
	}

	if err := storage.ValidateCIRQuery(query); err != nil {
		return nil, fmt.Errorf("invalid query: %w", err)
	}

	entityType := query.EntityType
	if entityType == "" {
		entityType = "default"
	}

	// Build WHERE clause and args
	whereClause, args := p.buildWhereClause(query.Filters)

	var sqlQuery string
	if whereClause != "" {
		sqlQuery = fmt.Sprintf(`SELECT data FROM "%s" WHERE %s`, entityType, whereClause)
	} else {
		sqlQuery = fmt.Sprintf(`SELECT data FROM "%s"`, entityType)
	}

	if query.Limit > 0 {
		sqlQuery += fmt.Sprintf(" LIMIT %d", query.Limit)
	}
	if query.Offset > 0 {
		sqlQuery += fmt.Sprintf(" OFFSET %d", query.Offset)
	}

	rows, err := p.db.Query(sqlQuery, args...)
	if err != nil {
		// Table may not exist
		if strings.Contains(err.Error(), "does not exist") || strings.Contains(err.Error(), "undefined") {
			return []*models.CIR{}, nil
		}
		return nil, fmt.Errorf("failed to query postgresql: %w", err)
	}
	defer rows.Close()

	results := make([]*models.CIR, 0)
	for rows.Next() {
		var dataStr string
		if err := rows.Scan(&dataStr); err != nil {
			continue
		}

		var cir models.CIR
		if err := json.Unmarshal([]byte(dataStr), &cir); err != nil {
			continue
		}
		results = append(results, &cir)
	}

	return results, nil
}

// buildWhereClause builds a SQL WHERE clause from CIR conditions
func (p *PostgresPlugin) buildWhereClause(filters []models.CIRCondition) (string, []interface{}) {
	if len(filters) == 0 {
		return "", nil
	}

	clauses := make([]string, 0, len(filters))
	args := make([]interface{}, 0, len(filters))
	argIdx := 1

	for _, f := range filters {
		switch f.Operator {
		case "eq":
			clauses = append(clauses, fmt.Sprintf("data->>'%s' = $%d", f.Attribute, argIdx))
			args = append(args, fmt.Sprintf("%v", f.Value))
			argIdx++
		case "neq":
			clauses = append(clauses, fmt.Sprintf("data->>'%s' != $%d", f.Attribute, argIdx))
			args = append(args, fmt.Sprintf("%v", f.Value))
			argIdx++
		case "gt":
			clauses = append(clauses, fmt.Sprintf("(data->>'%s')::numeric > $%d", f.Attribute, argIdx))
			args = append(args, toFloat(f.Value))
			argIdx++
		case "gte":
			clauses = append(clauses, fmt.Sprintf("(data->>'%s')::numeric >= $%d", f.Attribute, argIdx))
			args = append(args, toFloat(f.Value))
			argIdx++
		case "lt":
			clauses = append(clauses, fmt.Sprintf("(data->>'%s')::numeric < $%d", f.Attribute, argIdx))
			args = append(args, toFloat(f.Value))
			argIdx++
		case "lte":
			clauses = append(clauses, fmt.Sprintf("(data->>'%s')::numeric <= $%d", f.Attribute, argIdx))
			args = append(args, toFloat(f.Value))
			argIdx++
		case "like":
			clauses = append(clauses, fmt.Sprintf("data->>'%s' ILIKE $%d", f.Attribute, argIdx))
			args = append(args, fmt.Sprintf("%%%v%%", f.Value))
			argIdx++
		case "in":
			// Inline the IN values since pgx doesn't support array expansion simply
			if arr, ok := f.Value.([]interface{}); ok {
				placeholders := make([]string, 0, len(arr))
				for _, v := range arr {
					placeholders = append(placeholders, fmt.Sprintf("$%d", argIdx))
					args = append(args, fmt.Sprintf("%v", v))
					argIdx++
				}
				if len(placeholders) > 0 {
					clauses = append(clauses, fmt.Sprintf("data->>'%s' IN (%s)", f.Attribute, strings.Join(placeholders, ", ")))
				}
			}
		}
	}

	return strings.Join(clauses, " AND "), args
}

// Update updates existing CIR data in PostgreSQL
func (p *PostgresPlugin) Update(query *models.CIRQuery, updates *models.CIRUpdate) (*models.StorageResult, error) {
	if !p.initialized {
		return nil, fmt.Errorf("plugin not initialized")
	}

	items, err := p.Retrieve(query)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve items: %w", err)
	}

	entityType := query.EntityType
	if entityType == "" {
		entityType = "default"
	}

	affectedItems := 0

	for _, item := range items {
		dataMap, err := item.GetDataAsMap()
		if err != nil {
			continue
		}

		for key, value := range updates.Updates {
			dataMap[key] = value
		}
		item.Data = dataMap
		item.UpdateSize()

		updatedData, err := json.Marshal(item)
		if err != nil {
			continue
		}

		// We use source URI + timestamp as a heuristic identifier
		whereClause, args := p.buildWhereClause(query.Filters)
		var updateSQL string
		updateArg := append(args, string(updatedData))
		if whereClause != "" {
			updateSQL = fmt.Sprintf(`UPDATE "%s" SET data = $%d WHERE %s`, entityType, len(args)+1, whereClause)
		} else {
			updateSQL = fmt.Sprintf(`UPDATE "%s" SET data = $%d`, entityType, len(args)+1)
		}

		if _, err := p.db.Exec(updateSQL, updateArg...); err != nil {
			continue
		}
		affectedItems++
		break // One bulk update is enough
	}

	return &models.StorageResult{
		Success:       true,
		AffectedItems: affectedItems,
	}, nil
}

// Delete deletes CIR data from PostgreSQL
func (p *PostgresPlugin) Delete(query *models.CIRQuery) (*models.StorageResult, error) {
	if !p.initialized {
		return nil, fmt.Errorf("plugin not initialized")
	}

	entityType := query.EntityType
	if entityType == "" {
		entityType = "default"
	}

	whereClause, args := p.buildWhereClause(query.Filters)

	var deleteSQL string
	if whereClause != "" {
		deleteSQL = fmt.Sprintf(`DELETE FROM "%s" WHERE %s`, entityType, whereClause)
	} else {
		deleteSQL = fmt.Sprintf(`DELETE FROM "%s"`, entityType)
	}

	result, err := p.db.Exec(deleteSQL, args...)
	if err != nil {
		if strings.Contains(err.Error(), "does not exist") || strings.Contains(err.Error(), "undefined") {
			return &models.StorageResult{Success: true, AffectedItems: 0}, nil
		}
		return nil, fmt.Errorf("failed to delete from postgresql: %w", err)
	}

	affected, _ := result.RowsAffected()
	return &models.StorageResult{
		Success:       true,
		AffectedItems: int(affected),
	}, nil
}

// GetMetadata returns metadata about the PostgreSQL storage
func (p *PostgresPlugin) GetMetadata() (*models.StorageMetadata, error) {
	return &models.StorageMetadata{
		StorageType: "postgresql",
		Version:     "1.0.0",
		Capabilities: []string{
			"store",
			"retrieve",
			"update",
			"delete",
			"schema_creation",
			"json_query",
			"transactions",
		},
	}, nil
}

// HealthCheck checks if the PostgreSQL connection is healthy
func (p *PostgresPlugin) HealthCheck() (bool, error) {
	if !p.initialized {
		return false, fmt.Errorf("plugin not initialized")
	}

	if err := p.db.Ping(); err != nil {
		return false, fmt.Errorf("postgresql ping failed: %w", err)
	}

	return true, nil
}

// inferEntityType attempts to infer the entity type from CIR data
func (p *PostgresPlugin) inferEntityType(cir *models.CIR) string {
	if entityType, ok := cir.GetParameter("entity_type"); ok {
		if typeStr, ok := entityType.(string); ok {
			return typeStr
		}
	}

	if dataMap, err := cir.GetDataAsMap(); err == nil {
		if _, hasName := dataMap["name"]; hasName {
			if _, hasDept := dataMap["department"]; hasDept {
				return "Employee"
			}
			if _, hasLoc := dataMap["location"]; hasLoc {
				return "Company"
			}
		}
	}

	return "default"
}

// toFloat converts an interface to float64 for numeric comparisons
func toFloat(v interface{}) float64 {
	switch val := v.(type) {
	case float64:
		return val
	case float32:
		return float64(val)
	case int:
		return float64(val)
	case int64:
		return float64(val)
	case string:
		f, err := strconv.ParseFloat(val, 64)
		if err == nil {
			return f
		}
	}
	return 0
}

