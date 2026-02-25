package plugins

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	_ "github.com/go-sql-driver/mysql"
	"github.com/mimir-aip/mimir-aip-go/pkg/models"
	"github.com/mimir-aip/mimir-aip-go/pkg/storage"
)

// MySQLPlugin implements the StoragePlugin interface for MySQL storage
type MySQLPlugin struct {
	db          *sql.DB
	initialized bool
}

// NewMySQLPlugin creates a new MySQL storage plugin
func NewMySQLPlugin() *MySQLPlugin {
	return &MySQLPlugin{}
}

// Initialize initializes the MySQL plugin with configuration
func (m *MySQLPlugin) Initialize(config *models.PluginConfig) error {
	dsn := config.ConnectionString
	if dsn == "" {
		return fmt.Errorf("connection string is required for mysql storage")
	}

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return fmt.Errorf("failed to open mysql connection: %w", err)
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return fmt.Errorf("failed to ping mysql: %w", err)
	}

	m.db = db
	m.initialized = true
	return nil
}

// CreateSchema creates tables based on the ontology definition
func (m *MySQLPlugin) CreateSchema(ontology *models.OntologyDefinition) error {
	if !m.initialized {
		return fmt.Errorf("plugin not initialized")
	}

	for _, entity := range ontology.Entities {
		query := fmt.Sprintf(
			"CREATE TABLE IF NOT EXISTS `%s` (id CHAR(36) PRIMARY KEY, data JSON NOT NULL, created_at DATETIME DEFAULT CURRENT_TIMESTAMP)",
			entity.Name,
		)
		if _, err := m.db.Exec(query); err != nil {
			return fmt.Errorf("failed to create table %s: %w", entity.Name, err)
		}
	}

	return nil
}

// Store stores CIR data into MySQL
func (m *MySQLPlugin) Store(cir *models.CIR) (*models.StorageResult, error) {
	if !m.initialized {
		return nil, fmt.Errorf("plugin not initialized")
	}

	if err := cir.Validate(); err != nil {
		return nil, fmt.Errorf("invalid CIR: %w", err)
	}

	entityType := m.inferEntityType(cir)
	affectedItems := 0

	// Ensure table exists
	createQuery := fmt.Sprintf(
		"CREATE TABLE IF NOT EXISTS `%s` (id CHAR(36) PRIMARY KEY, data JSON NOT NULL, created_at DATETIME DEFAULT CURRENT_TIMESTAMP)",
		entityType,
	)
	if _, err := m.db.Exec(createQuery); err != nil {
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
			if err := m.insertItem(entityType, itemCIR); err != nil {
				return nil, fmt.Errorf("failed to store item: %w", err)
			}
			affectedItems++
		}
	} else {
		if err := m.insertItem(entityType, cir); err != nil {
			return nil, fmt.Errorf("failed to store item: %w", err)
		}
		affectedItems = 1
	}

	return &models.StorageResult{
		Success:       true,
		AffectedItems: affectedItems,
	}, nil
}

func (m *MySQLPlugin) insertItem(entityType string, cir *models.CIR) error {
	data, err := json.Marshal(cir)
	if err != nil {
		return fmt.Errorf("failed to marshal CIR: %w", err)
	}

	id := uuid.New().String()
	query := fmt.Sprintf("INSERT INTO `%s` (id, data) VALUES (?, ?)", entityType)
	_, err = m.db.Exec(query, id, string(data))
	return err
}

// Retrieve retrieves CIR data from MySQL using a query
func (m *MySQLPlugin) Retrieve(query *models.CIRQuery) ([]*models.CIR, error) {
	if !m.initialized {
		return nil, fmt.Errorf("plugin not initialized")
	}

	if err := storage.ValidateCIRQuery(query); err != nil {
		return nil, fmt.Errorf("invalid query: %w", err)
	}

	entityType := query.EntityType
	if entityType == "" {
		entityType = "default"
	}

	whereClause, args := m.buildWhereClause(query.Filters)

	var sqlQuery string
	if whereClause != "" {
		sqlQuery = fmt.Sprintf("SELECT data FROM `%s` WHERE %s", entityType, whereClause)
	} else {
		sqlQuery = fmt.Sprintf("SELECT data FROM `%s`", entityType)
	}

	if query.Limit > 0 {
		sqlQuery += fmt.Sprintf(" LIMIT %d", query.Limit)
	}
	if query.Offset > 0 {
		sqlQuery += fmt.Sprintf(" OFFSET %d", query.Offset)
	}

	rows, err := m.db.Query(sqlQuery, args...)
	if err != nil {
		if strings.Contains(err.Error(), "doesn't exist") || strings.Contains(err.Error(), "Table") {
			return []*models.CIR{}, nil
		}
		return nil, fmt.Errorf("failed to query mysql: %w", err)
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

// buildWhereClause builds a SQL WHERE clause from CIR conditions for MySQL
func (m *MySQLPlugin) buildWhereClause(filters []models.CIRCondition) (string, []interface{}) {
	if len(filters) == 0 {
		return "", nil
	}

	clauses := make([]string, 0, len(filters))
	args := make([]interface{}, 0, len(filters))

	for _, f := range filters {
		attr := f.Attribute
		switch f.Operator {
		case "eq":
			clauses = append(clauses, fmt.Sprintf("JSON_UNQUOTE(JSON_EXTRACT(data, '$.%s')) = ?", attr))
			args = append(args, fmt.Sprintf("%v", f.Value))
		case "neq":
			clauses = append(clauses, fmt.Sprintf("JSON_UNQUOTE(JSON_EXTRACT(data, '$.%s')) != ?", attr))
			args = append(args, fmt.Sprintf("%v", f.Value))
		case "gt":
			clauses = append(clauses, fmt.Sprintf("CAST(JSON_EXTRACT(data, '$.%s') AS DECIMAL(20,6)) > ?", attr))
			args = append(args, toFloat(f.Value))
		case "gte":
			clauses = append(clauses, fmt.Sprintf("CAST(JSON_EXTRACT(data, '$.%s') AS DECIMAL(20,6)) >= ?", attr))
			args = append(args, toFloat(f.Value))
		case "lt":
			clauses = append(clauses, fmt.Sprintf("CAST(JSON_EXTRACT(data, '$.%s') AS DECIMAL(20,6)) < ?", attr))
			args = append(args, toFloat(f.Value))
		case "lte":
			clauses = append(clauses, fmt.Sprintf("CAST(JSON_EXTRACT(data, '$.%s') AS DECIMAL(20,6)) <= ?", attr))
			args = append(args, toFloat(f.Value))
		case "like":
			clauses = append(clauses, fmt.Sprintf("JSON_UNQUOTE(JSON_EXTRACT(data, '$.%s')) LIKE ?", attr))
			args = append(args, fmt.Sprintf("%%%v%%", f.Value))
		case "in":
			if arr, ok := f.Value.([]interface{}); ok {
				placeholders := make([]string, 0, len(arr))
				for _, v := range arr {
					placeholders = append(placeholders, "?")
					args = append(args, fmt.Sprintf("%v", v))
				}
				if len(placeholders) > 0 {
					clauses = append(clauses, fmt.Sprintf("JSON_UNQUOTE(JSON_EXTRACT(data, '$.%s')) IN (%s)", attr, strings.Join(placeholders, ", ")))
				}
			}
		}
	}

	return strings.Join(clauses, " AND "), args
}

// Update updates existing CIR data in MySQL
func (m *MySQLPlugin) Update(query *models.CIRQuery, updates *models.CIRUpdate) (*models.StorageResult, error) {
	if !m.initialized {
		return nil, fmt.Errorf("plugin not initialized")
	}

	items, err := m.Retrieve(query)
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

		whereClause, args := m.buildWhereClause(query.Filters)
		updateArg := append([]interface{}{string(updatedData)}, args...)
		var updateSQL string
		if whereClause != "" {
			updateSQL = fmt.Sprintf("UPDATE `%s` SET data = ? WHERE %s", entityType, whereClause)
		} else {
			updateSQL = fmt.Sprintf("UPDATE `%s` SET data = ?", entityType)
		}

		if _, err := m.db.Exec(updateSQL, updateArg...); err != nil {
			continue
		}
		affectedItems++
		break
	}

	return &models.StorageResult{
		Success:       true,
		AffectedItems: affectedItems,
	}, nil
}

// Delete deletes CIR data from MySQL
func (m *MySQLPlugin) Delete(query *models.CIRQuery) (*models.StorageResult, error) {
	if !m.initialized {
		return nil, fmt.Errorf("plugin not initialized")
	}

	entityType := query.EntityType
	if entityType == "" {
		entityType = "default"
	}

	whereClause, args := m.buildWhereClause(query.Filters)

	var deleteSQL string
	if whereClause != "" {
		deleteSQL = fmt.Sprintf("DELETE FROM `%s` WHERE %s", entityType, whereClause)
	} else {
		deleteSQL = fmt.Sprintf("DELETE FROM `%s`", entityType)
	}

	result, err := m.db.Exec(deleteSQL, args...)
	if err != nil {
		if strings.Contains(err.Error(), "doesn't exist") || strings.Contains(err.Error(), "Table") {
			return &models.StorageResult{Success: true, AffectedItems: 0}, nil
		}
		return nil, fmt.Errorf("failed to delete from mysql: %w", err)
	}

	affected, _ := result.RowsAffected()
	return &models.StorageResult{
		Success:       true,
		AffectedItems: int(affected),
	}, nil
}

// GetMetadata returns metadata about the MySQL storage
func (m *MySQLPlugin) GetMetadata() (*models.StorageMetadata, error) {
	return &models.StorageMetadata{
		StorageType: "mysql",
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

// HealthCheck checks if the MySQL connection is healthy
func (m *MySQLPlugin) HealthCheck() (bool, error) {
	if !m.initialized {
		return false, fmt.Errorf("plugin not initialized")
	}

	if err := m.db.Ping(); err != nil {
		return false, fmt.Errorf("mysql ping failed: %w", err)
	}

	return true, nil
}

// inferEntityType attempts to infer the entity type from CIR data
func (m *MySQLPlugin) inferEntityType(cir *models.CIR) string {
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

