package plugins

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mimir-aip/mimir-aip-go/pkg/models"
	"github.com/mimir-aip/mimir-aip-go/pkg/storage"
	"github.com/redis/go-redis/v9"
)

// RedisPlugin implements the StoragePlugin interface for Redis storage
type RedisPlugin struct {
	client      *redis.Client
	initialized bool
}

// NewRedisPlugin creates a new Redis storage plugin
func NewRedisPlugin() *RedisPlugin {
	return &RedisPlugin{}
}

// Initialize initializes the Redis plugin with configuration
func (r *RedisPlugin) Initialize(config *models.PluginConfig) error {
	redisURL := config.ConnectionString
	if redisURL == "" {
		return fmt.Errorf("connection string is required for redis storage")
	}

	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return fmt.Errorf("failed to parse redis URL: %w", err)
	}

	client := redis.NewClient(opts)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		client.Close()
		return fmt.Errorf("failed to ping redis: %w", err)
	}

	r.client = client
	r.initialized = true
	return nil
}

// CreateSchema is a no-op for Redis (schemaless)
func (r *RedisPlugin) CreateSchema(ontology *models.OntologyDefinition) error {
	if !r.initialized {
		return fmt.Errorf("plugin not initialized")
	}
	// Redis is schemaless; no schema creation needed
	return nil
}

// Store stores CIR data into Redis
func (r *RedisPlugin) Store(cir *models.CIR) (*models.StorageResult, error) {
	if !r.initialized {
		return nil, fmt.Errorf("plugin not initialized")
	}

	if err := cir.Validate(); err != nil {
		return nil, fmt.Errorf("invalid CIR: %w", err)
	}

	entityType := r.inferEntityType(cir)
	affectedItems := 0

	if arr, err := cir.GetDataAsArray(); err == nil {
		for _, item := range arr {
			itemCIR := &models.CIR{
				Version:  cir.Version,
				Source:   cir.Source,
				Data:     item,
				Metadata: cir.Metadata,
			}
			if err := r.storeItem(entityType, itemCIR); err != nil {
				return nil, fmt.Errorf("failed to store item: %w", err)
			}
			affectedItems++
		}
	} else {
		if err := r.storeItem(entityType, cir); err != nil {
			return nil, fmt.Errorf("failed to store item: %w", err)
		}
		affectedItems = 1
	}

	return &models.StorageResult{
		Success:       true,
		AffectedItems: affectedItems,
	}, nil
}

func (r *RedisPlugin) storeItem(entityType string, cir *models.CIR) error {
	data, err := json.Marshal(cir)
	if err != nil {
		return fmt.Errorf("failed to marshal CIR: %w", err)
	}

	id := uuid.New().String()
	key := fmt.Sprintf("mimir:%s:%s", entityType, id)
	indexKey := fmt.Sprintf("mimir:%s:index", entityType)
	score := float64(time.Now().UnixMicro())

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pipe := r.client.Pipeline()
	pipe.Set(ctx, key, string(data), 0)
	pipe.ZAdd(ctx, indexKey, redis.Z{
		Score:  score,
		Member: id,
	})
	_, err = pipe.Exec(ctx)
	return err
}

// Retrieve retrieves CIR data from Redis using a query
func (r *RedisPlugin) Retrieve(query *models.CIRQuery) ([]*models.CIR, error) {
	if !r.initialized {
		return nil, fmt.Errorf("plugin not initialized")
	}

	if err := storage.ValidateCIRQuery(query); err != nil {
		return nil, fmt.Errorf("invalid query: %w", err)
	}

	entityType := query.EntityType
	if entityType == "" {
		entityType = "default"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	indexKey := fmt.Sprintf("mimir:%s:index", entityType)

	ids, err := r.client.ZRange(ctx, indexKey, 0, -1).Result()
	if err != nil {
		if err == redis.Nil {
			return []*models.CIR{}, nil
		}
		return nil, fmt.Errorf("failed to read redis index: %w", err)
	}

	results := make([]*models.CIR, 0)

	for _, id := range ids {
		key := fmt.Sprintf("mimir:%s:%s", entityType, id)

		getCtx, getCancel := context.WithTimeout(context.Background(), 5*time.Second)
		val, err := r.client.Get(getCtx, key).Result()
		getCancel()

		if err != nil {
			continue
		}

		var cir models.CIR
		if err := json.Unmarshal([]byte(val), &cir); err != nil {
			continue
		}

		if r.matchesFilters(&cir, query.Filters) {
			results = append(results, &cir)
		}
	}

	// Apply offset and limit
	if query.Offset > 0 && query.Offset < len(results) {
		results = results[query.Offset:]
	}
	if query.Limit > 0 && query.Limit < len(results) {
		results = results[:query.Limit]
	}

	return results, nil
}

// Update updates existing CIR data in Redis
func (r *RedisPlugin) Update(query *models.CIRQuery, updates *models.CIRUpdate) (*models.StorageResult, error) {
	if !r.initialized {
		return nil, fmt.Errorf("plugin not initialized")
	}

	entityType := query.EntityType
	if entityType == "" {
		entityType = "default"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	indexKey := fmt.Sprintf("mimir:%s:index", entityType)

	ids, err := r.client.ZRange(ctx, indexKey, 0, -1).Result()
	if err != nil {
		if err == redis.Nil {
			return &models.StorageResult{Success: true, AffectedItems: 0}, nil
		}
		return nil, fmt.Errorf("failed to read redis index: %w", err)
	}

	affectedItems := 0

	for _, id := range ids {
		key := fmt.Sprintf("mimir:%s:%s", entityType, id)

		getCtx, getCancel := context.WithTimeout(context.Background(), 5*time.Second)
		val, err := r.client.Get(getCtx, key).Result()
		getCancel()

		if err != nil {
			continue
		}

		var cir models.CIR
		if err := json.Unmarshal([]byte(val), &cir); err != nil {
			continue
		}

		if !r.matchesFilters(&cir, query.Filters) {
			continue
		}

		dataMap, err := cir.GetDataAsMap()
		if err != nil {
			continue
		}

		for k, v := range updates.Updates {
			dataMap[k] = v
		}
		cir.Data = dataMap
		cir.UpdateSize()

		updatedData, err := json.Marshal(&cir)
		if err != nil {
			continue
		}

		setCtx, setCancel := context.WithTimeout(context.Background(), 5*time.Second)
		setErr := r.client.Set(setCtx, key, string(updatedData), 0).Err()
		setCancel()

		if setErr != nil {
			continue
		}
		affectedItems++
	}

	return &models.StorageResult{
		Success:       true,
		AffectedItems: affectedItems,
	}, nil
}

// Delete deletes CIR data from Redis
func (r *RedisPlugin) Delete(query *models.CIRQuery) (*models.StorageResult, error) {
	if !r.initialized {
		return nil, fmt.Errorf("plugin not initialized")
	}

	entityType := query.EntityType
	if entityType == "" {
		entityType = "default"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	indexKey := fmt.Sprintf("mimir:%s:index", entityType)

	ids, err := r.client.ZRange(ctx, indexKey, 0, -1).Result()
	if err != nil {
		if err == redis.Nil {
			return &models.StorageResult{Success: true, AffectedItems: 0}, nil
		}
		return nil, fmt.Errorf("failed to read redis index: %w", err)
	}

	affectedItems := 0

	for _, id := range ids {
		key := fmt.Sprintf("mimir:%s:%s", entityType, id)

		getCtx, getCancel := context.WithTimeout(context.Background(), 5*time.Second)
		val, err := r.client.Get(getCtx, key).Result()
		getCancel()

		if err != nil {
			continue
		}

		var cir models.CIR
		if err := json.Unmarshal([]byte(val), &cir); err != nil {
			continue
		}

		if !r.matchesFilters(&cir, query.Filters) {
			continue
		}

		pipe := r.client.Pipeline()
		delCtx, delCancel := context.WithTimeout(context.Background(), 5*time.Second)
		pipe.Del(delCtx, key)
		pipe.ZRem(delCtx, indexKey, id)
		_, pipeErr := pipe.Exec(delCtx)
		delCancel()

		if pipeErr != nil {
			continue
		}
		affectedItems++
	}

	return &models.StorageResult{
		Success:       true,
		AffectedItems: affectedItems,
	}, nil
}

// GetMetadata returns metadata about the Redis storage
func (r *RedisPlugin) GetMetadata() (*models.StorageMetadata, error) {
	return &models.StorageMetadata{
		StorageType: "redis",
		Version:     "1.0.0",
		Capabilities: []string{
			"store",
			"retrieve",
			"update",
			"delete",
			"high_performance",
			"in_memory",
			"sorted_sets",
		},
	}, nil
}

// HealthCheck checks if the Redis connection is healthy
func (r *RedisPlugin) HealthCheck() (bool, error) {
	if !r.initialized {
		return false, fmt.Errorf("plugin not initialized")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := r.client.Ping(ctx).Err(); err != nil {
		return false, fmt.Errorf("redis ping failed: %w", err)
	}

	return true, nil
}

// inferEntityType attempts to infer the entity type from CIR data
func (r *RedisPlugin) inferEntityType(cir *models.CIR) string {
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

// matchesFilters checks if a CIR object matches the query filters
func (r *RedisPlugin) matchesFilters(cir *models.CIR, filters []models.CIRCondition) bool {
	if len(filters) == 0 {
		return true
	}

	dataMap, err := cir.GetDataAsMap()
	if err != nil {
		return false
	}

	for _, filter := range filters {
		value, exists := dataMap[filter.Attribute]
		if !exists {
			return false
		}

		if !r.evaluateCondition(value, filter.Operator, filter.Value) {
			return false
		}
	}

	return true
}

// evaluateCondition evaluates a filter condition
func (r *RedisPlugin) evaluateCondition(value interface{}, operator string, expected interface{}) bool {
	switch operator {
	case "eq":
		return fmt.Sprintf("%v", value) == fmt.Sprintf("%v", expected)
	case "neq":
		return fmt.Sprintf("%v", value) != fmt.Sprintf("%v", expected)
	case "like":
		valueStr := fmt.Sprintf("%v", value)
		expectedStr := fmt.Sprintf("%v", expected)
		return strings.Contains(strings.ToLower(valueStr), strings.ToLower(expectedStr))
	case "gt":
		return toFloat(value) > toFloat(expected)
	case "gte":
		return toFloat(value) >= toFloat(expected)
	case "lt":
		return toFloat(value) < toFloat(expected)
	case "lte":
		return toFloat(value) <= toFloat(expected)
	case "in":
		if arr, ok := expected.([]interface{}); ok {
			valueStr := fmt.Sprintf("%v", value)
			for _, v := range arr {
				if fmt.Sprintf("%v", v) == valueStr {
					return true
				}
			}
		}
		return false
	default:
		return false
	}
}

// ensure uuid import is used
var _ = uuid.New
