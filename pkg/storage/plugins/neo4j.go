package plugins

import (
	"context"
	"fmt"
	"time"

	"github.com/mimir-aip/mimir-aip-go/pkg/models"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// Neo4jPlugin implements the StoragePlugin interface for Neo4j graph database storage.
type Neo4jPlugin struct {
	driver   neo4j.DriverWithContext
	database string
	ctx      context.Context
}

// NewNeo4jPlugin creates a new Neo4j storage plugin.
func NewNeo4jPlugin() *Neo4jPlugin {
	return &Neo4jPlugin{
		ctx: context.Background(),
	}
}

// Initialize opens a Neo4j driver and verifies connectivity.
// Config keys: uri (bolt://...), username, password, database (default "neo4j")
func (p *Neo4jPlugin) Initialize(config *models.PluginConfig) error {
	uri := config.ConnectionString
	if uri == "" {
		if u, ok := config.Options["uri"].(string); ok {
			uri = u
		} else {
			return fmt.Errorf("uri is required for neo4j storage")
		}
	}

	username := ""
	password := ""
	if u, ok := config.Options["username"].(string); ok {
		username = u
	}
	if pw, ok := config.Options["password"].(string); ok {
		password = pw
	}

	p.database = "neo4j"
	if db, ok := config.Options["database"].(string); ok && db != "" {
		p.database = db
	}

	driver, err := neo4j.NewDriverWithContext(uri, neo4j.BasicAuth(username, password, ""))
	if err != nil {
		return fmt.Errorf("failed to create neo4j driver: %w", err)
	}

	if err := driver.VerifyConnectivity(p.ctx); err != nil {
		driver.Close(p.ctx)
		return fmt.Errorf("neo4j connectivity check failed: %w", err)
	}

	p.driver = driver
	return nil
}

// CreateSchema creates uniqueness constraints for each entity type in the ontology.
func (p *Neo4jPlugin) CreateSchema(ontology *models.OntologyDefinition) error {
	if p.driver == nil {
		return fmt.Errorf("plugin not initialized")
	}

	session := p.driver.NewSession(p.ctx, neo4j.SessionConfig{DatabaseName: p.database})
	defer session.Close(p.ctx)

	for _, entity := range ontology.Entities {
		query := fmt.Sprintf(
			"CREATE CONSTRAINT IF NOT EXISTS FOR (n:%s) REQUIRE n.id IS UNIQUE",
			entity.Name,
		)
		if _, err := session.Run(p.ctx, query, nil); err != nil {
			return fmt.Errorf("failed to create constraint for %s: %w", entity.Name, err)
		}
	}

	return nil
}

// Store persists a CIR record as a Neo4j node using MERGE on id.
func (p *Neo4jPlugin) Store(cir *models.CIR) (*models.StorageResult, error) {
	if p.driver == nil {
		return nil, fmt.Errorf("plugin not initialized")
	}

	if err := cir.Validate(); err != nil {
		return nil, fmt.Errorf("invalid CIR: %w", err)
	}

	entityType := p.inferEntityType(cir)

	dataMap, err := cir.GetDataAsMap()
	if err != nil {
		return nil, fmt.Errorf("failed to get CIR data as map: %w", err)
	}

	// Use "id" field from data if present, otherwise fall back to source URI
	nodeID := cir.Source.URI
	if idVal, ok := dataMap["id"]; ok {
		nodeID = fmt.Sprintf("%v", idVal)
	}

	session := p.driver.NewSession(p.ctx, neo4j.SessionConfig{DatabaseName: p.database})
	defer session.Close(p.ctx)

	query := fmt.Sprintf("MERGE (n:%s {id: $id}) SET n += $props", entityType)
	params := map[string]interface{}{
		"id":    nodeID,
		"props": dataMap,
	}

	if _, err := session.Run(p.ctx, query, params); err != nil {
		return nil, fmt.Errorf("failed to store node: %w", err)
	}

	return &models.StorageResult{Success: true, AffectedItems: 1}, nil
}

// Retrieve queries Neo4j nodes matching the CIRQuery filters.
func (p *Neo4jPlugin) Retrieve(query *models.CIRQuery) ([]*models.CIR, error) {
	if p.driver == nil {
		return nil, fmt.Errorf("plugin not initialized")
	}

	entityType := query.EntityType
	if entityType == "" {
		entityType = "Entity"
	}

	cypher := fmt.Sprintf("MATCH (n:%s)", entityType)
	params := map[string]interface{}{}

	if len(query.Filters) > 0 {
		cypher += " WHERE "
		for i, f := range query.Filters {
			if i > 0 {
				cypher += " AND "
			}
			paramKey := fmt.Sprintf("val%d", i)
			switch f.Operator {
			case "eq":
				cypher += fmt.Sprintf("n.%s = $%s", f.Attribute, paramKey)
			case "neq":
				cypher += fmt.Sprintf("n.%s <> $%s", f.Attribute, paramKey)
			case "like":
				cypher += fmt.Sprintf("toLower(toString(n.%s)) CONTAINS toLower($%s)", f.Attribute, paramKey)
			default:
				cypher += fmt.Sprintf("n.%s = $%s", f.Attribute, paramKey)
			}
			params[paramKey] = f.Value
		}
	}

	cypher += " RETURN n"

	if query.Offset > 0 {
		cypher += fmt.Sprintf(" SKIP %d", query.Offset)
	}
	if query.Limit > 0 {
		cypher += fmt.Sprintf(" LIMIT %d", query.Limit)
	}

	session := p.driver.NewSession(p.ctx, neo4j.SessionConfig{DatabaseName: p.database})
	defer session.Close(p.ctx)

	result, err := session.Run(p.ctx, cypher, params)
	if err != nil {
		return nil, fmt.Errorf("failed to run retrieve query: %w", err)
	}

	var cirs []*models.CIR
	for result.Next(p.ctx) {
		record := result.Record()
		nodeVal, ok := record.Get("n")
		if !ok {
			continue
		}
		node, ok := nodeVal.(neo4j.Node)
		if !ok {
			continue
		}
		props := make(map[string]interface{})
		for k, v := range node.Props {
			props[k] = v
		}
		cirs = append(cirs, &models.CIR{
			Version: models.CIRVersion,
			Source: models.CIRSource{
				Type:      models.SourceTypeDatabase,
				URI:       "neo4j:" + p.database,
				Timestamp: time.Now(),
				Format:    models.DataFormatJSON,
			},
			Data: props,
		})
	}

	if err := result.Err(); err != nil {
		return nil, fmt.Errorf("error iterating neo4j results: %w", err)
	}

	return cirs, nil
}

// Update applies field updates to all nodes matching the CIRQuery.
func (p *Neo4jPlugin) Update(query *models.CIRQuery, updates *models.CIRUpdate) (*models.StorageResult, error) {
	if p.driver == nil {
		return nil, fmt.Errorf("plugin not initialized")
	}

	entityType := query.EntityType
	if entityType == "" {
		entityType = "Entity"
	}

	cypher := fmt.Sprintf("MATCH (n:%s)", entityType)
	params := map[string]interface{}{"updates": updates.Updates}

	if len(query.Filters) > 0 {
		cypher += " WHERE "
		for i, f := range query.Filters {
			if i > 0 {
				cypher += " AND "
			}
			paramKey := fmt.Sprintf("val%d", i)
			cypher += fmt.Sprintf("n.%s = $%s", f.Attribute, paramKey)
			params[paramKey] = f.Value
		}
	}

	cypher += " SET n += $updates RETURN count(n) AS affected"

	session := p.driver.NewSession(p.ctx, neo4j.SessionConfig{DatabaseName: p.database})
	defer session.Close(p.ctx)

	result, err := session.Run(p.ctx, cypher, params)
	if err != nil {
		return nil, fmt.Errorf("failed to run update query: %w", err)
	}

	affected := 0
	if result.Next(p.ctx) {
		if v, ok := result.Record().Get("affected"); ok {
			if n, ok := v.(int64); ok {
				affected = int(n)
			}
		}
	}

	return &models.StorageResult{Success: true, AffectedItems: affected}, nil
}

// Delete removes all nodes matching the CIRQuery using DETACH DELETE.
func (p *Neo4jPlugin) Delete(query *models.CIRQuery) (*models.StorageResult, error) {
	if p.driver == nil {
		return nil, fmt.Errorf("plugin not initialized")
	}

	entityType := query.EntityType
	if entityType == "" {
		entityType = "Entity"
	}

	cypher := fmt.Sprintf("MATCH (n:%s)", entityType)
	params := map[string]interface{}{}

	if len(query.Filters) > 0 {
		cypher += " WHERE "
		for i, f := range query.Filters {
			if i > 0 {
				cypher += " AND "
			}
			paramKey := fmt.Sprintf("val%d", i)
			cypher += fmt.Sprintf("n.%s = $%s", f.Attribute, paramKey)
			params[paramKey] = f.Value
		}
	}

	cypher += " WITH n, count(n) AS cnt DETACH DELETE n RETURN cnt"

	session := p.driver.NewSession(p.ctx, neo4j.SessionConfig{DatabaseName: p.database})
	defer session.Close(p.ctx)

	result, err := session.Run(p.ctx, cypher, params)
	if err != nil {
		return nil, fmt.Errorf("failed to run delete query: %w", err)
	}

	affected := 0
	if result.Next(p.ctx) {
		if v, ok := result.Record().Get("cnt"); ok {
			if n, ok := v.(int64); ok {
				affected = int(n)
			}
		}
	}

	return &models.StorageResult{Success: true, AffectedItems: affected}, nil
}

// GetMetadata returns metadata about the Neo4j storage plugin.
func (p *Neo4jPlugin) GetMetadata() (*models.StorageMetadata, error) {
	return &models.StorageMetadata{
		StorageType: "neo4j",
		Version:     "1.0.0",
		Capabilities: []string{
			"store",
			"retrieve",
			"update",
			"delete",
			"schema_creation",
		},
	}, nil
}

// HealthCheck verifies Neo4j connectivity by running a trivial query.
func (p *Neo4jPlugin) HealthCheck() (bool, error) {
	if p.driver == nil {
		return false, fmt.Errorf("plugin not initialized")
	}

	session := p.driver.NewSession(p.ctx, neo4j.SessionConfig{DatabaseName: p.database})
	defer session.Close(p.ctx)

	result, err := session.Run(p.ctx, "RETURN 1 AS ok", nil)
	if err != nil {
		return false, fmt.Errorf("neo4j health check query failed: %w", err)
	}

	if !result.Next(p.ctx) {
		return false, fmt.Errorf("neo4j health check returned no results")
	}

	return true, nil
}

// inferEntityType attempts to determine the entity type from CIR source metadata.
func (p *Neo4jPlugin) inferEntityType(cir *models.CIR) string {
	if entityType, ok := cir.GetParameter("entity_type"); ok {
		if typeStr, ok := entityType.(string); ok {
			return typeStr
		}
	}
	return "Entity"
}
