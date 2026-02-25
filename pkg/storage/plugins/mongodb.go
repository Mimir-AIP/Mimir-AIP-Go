package plugins

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/mimir-aip/mimir-aip-go/pkg/models"
	"github.com/mimir-aip/mimir-aip-go/pkg/storage"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// MongoDBPlugin implements the StoragePlugin interface for MongoDB storage
type MongoDBPlugin struct {
	client      *mongo.Client
	db          *mongo.Database
	dbName      string
	initialized bool
}

// NewMongoDBPlugin creates a new MongoDB storage plugin
func NewMongoDBPlugin() *MongoDBPlugin {
	return &MongoDBPlugin{}
}

// Initialize initializes the MongoDB plugin with configuration
func (m *MongoDBPlugin) Initialize(config *models.PluginConfig) error {
	uri := config.ConnectionString
	if uri == "" {
		return fmt.Errorf("connection string is required for mongodb storage")
	}

	dbName := "mimir"
	if config.Options != nil {
		if name, ok := config.Options["database"].(string); ok && name != "" {
			dbName = name
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	clientOpts := options.Client().ApplyURI(uri)
	client, err := mongo.Connect(ctx, clientOpts)
	if err != nil {
		return fmt.Errorf("failed to connect to mongodb: %w", err)
	}

	if err := client.Ping(ctx, nil); err != nil {
		client.Disconnect(ctx)
		return fmt.Errorf("failed to ping mongodb: %w", err)
	}

	m.client = client
	m.db = client.Database(dbName)
	m.dbName = dbName
	m.initialized = true
	return nil
}

// CreateSchema creates indexes for each entity type
func (m *MongoDBPlugin) CreateSchema(ontology *models.OntologyDefinition) error {
	if !m.initialized {
		return fmt.Errorf("plugin not initialized")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	for _, entity := range ontology.Entities {
		collection := m.db.Collection(entity.Name)
		indexModel := mongo.IndexModel{
			Keys: bson.D{{Key: "entity_type", Value: 1}},
		}
		if _, err := collection.Indexes().CreateOne(ctx, indexModel); err != nil {
			return fmt.Errorf("failed to create index for %s: %w", entity.Name, err)
		}
	}

	return nil
}

// Store stores CIR data into MongoDB
func (m *MongoDBPlugin) Store(cir *models.CIR) (*models.StorageResult, error) {
	if !m.initialized {
		return nil, fmt.Errorf("plugin not initialized")
	}

	if err := cir.Validate(); err != nil {
		return nil, fmt.Errorf("invalid CIR: %w", err)
	}

	entityType := m.inferEntityType(cir)
	affectedItems := 0

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	collection := m.db.Collection(entityType)

	if arr, err := cir.GetDataAsArray(); err == nil {
		for _, item := range arr {
			itemCIR := &models.CIR{
				Version:  cir.Version,
				Source:   cir.Source,
				Data:     item,
				Metadata: cir.Metadata,
			}
			if err := m.insertDocument(ctx, collection, entityType, itemCIR); err != nil {
				return nil, fmt.Errorf("failed to store item: %w", err)
			}
			affectedItems++
		}
	} else {
		if err := m.insertDocument(ctx, collection, entityType, cir); err != nil {
			return nil, fmt.Errorf("failed to store item: %w", err)
		}
		affectedItems = 1
	}

	return &models.StorageResult{
		Success:       true,
		AffectedItems: affectedItems,
	}, nil
}

func (m *MongoDBPlugin) insertDocument(ctx context.Context, collection *mongo.Collection, entityType string, cir *models.CIR) error {
	dataMap, err := cir.GetDataAsMap()
	if err != nil {
		// Fall back to JSON marshaling
		jsonBytes, jsonErr := json.Marshal(cir.Data)
		if jsonErr != nil {
			return fmt.Errorf("failed to get data as map: %w", err)
		}
		dataMap = map[string]interface{}{"_raw": string(jsonBytes)}
	}

	doc := bson.M{
		"_id":         uuid.New().String(),
		"cir_version": cir.Version,
		"source_uri":  cir.Source.URI,
		"source_type": string(cir.Source.Type),
		"entity_type": entityType,
		"data":        dataMap,
		"created_at":  time.Now(),
	}

	_, err = collection.InsertOne(ctx, doc)
	return err
}

// Retrieve retrieves CIR data from MongoDB using a query
func (m *MongoDBPlugin) Retrieve(query *models.CIRQuery) ([]*models.CIR, error) {
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

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	collection := m.db.Collection(entityType)

	filter := m.buildFilter(query.Filters)

	findOpts := options.Find()
	if query.Limit > 0 {
		limit := int64(query.Limit)
		findOpts.SetLimit(limit)
	}
	if query.Offset > 0 {
		findOpts.SetSkip(int64(query.Offset))
	}

	cursor, err := collection.Find(ctx, filter, findOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to query mongodb: %w", err)
	}
	defer cursor.Close(ctx)

	results := make([]*models.CIR, 0)
	for cursor.Next(ctx) {
		var doc bson.M
		if err := cursor.Decode(&doc); err != nil {
			continue
		}

		cir := m.documentToCIR(doc)
		if cir != nil {
			results = append(results, cir)
		}
	}

	return results, nil
}

// buildFilter builds a MongoDB filter from CIR conditions
func (m *MongoDBPlugin) buildFilter(filters []models.CIRCondition) bson.M {
	if len(filters) == 0 {
		return bson.M{}
	}

	andConditions := make([]bson.M, 0, len(filters))

	for _, f := range filters {
		dataKey := fmt.Sprintf("data.%s", f.Attribute)
		switch f.Operator {
		case "eq":
			andConditions = append(andConditions, bson.M{dataKey: bson.M{"$eq": f.Value}})
		case "neq":
			andConditions = append(andConditions, bson.M{dataKey: bson.M{"$ne": f.Value}})
		case "gt":
			andConditions = append(andConditions, bson.M{dataKey: bson.M{"$gt": toFloat(f.Value)}})
		case "gte":
			andConditions = append(andConditions, bson.M{dataKey: bson.M{"$gte": toFloat(f.Value)}})
		case "lt":
			andConditions = append(andConditions, bson.M{dataKey: bson.M{"$lt": toFloat(f.Value)}})
		case "lte":
			andConditions = append(andConditions, bson.M{dataKey: bson.M{"$lte": toFloat(f.Value)}})
		case "like":
			pattern := fmt.Sprintf(".*%v.*", f.Value)
			andConditions = append(andConditions, bson.M{dataKey: bson.M{"$regex": pattern, "$options": "i"}})
		case "in":
			andConditions = append(andConditions, bson.M{dataKey: bson.M{"$in": f.Value}})
		}
	}

	if len(andConditions) == 1 {
		return andConditions[0]
	}
	return bson.M{"$and": andConditions}
}

// documentToCIR converts a MongoDB document back to a CIR object
func (m *MongoDBPlugin) documentToCIR(doc bson.M) *models.CIR {
	cir := &models.CIR{}

	if v, ok := doc["cir_version"].(string); ok {
		cir.Version = v
	} else {
		cir.Version = "1.0"
	}

	sourceType := ""
	if v, ok := doc["source_type"].(string); ok {
		sourceType = v
	}
	sourceURI := ""
	if v, ok := doc["source_uri"].(string); ok {
		sourceURI = v
	}

	cir.Source = models.CIRSource{
		Type:       models.SourceType(sourceType),
		URI:        sourceURI,
		Timestamp:  time.Now(),
		Format:     models.DataFormatJSON,
		Parameters: make(map[string]interface{}),
	}

	if data, ok := doc["data"]; ok {
		cir.Data = data
	} else {
		cir.Data = doc
	}

	cir.Metadata = models.CIRMetadata{}

	return cir
}

// Update updates existing CIR data in MongoDB
func (m *MongoDBPlugin) Update(query *models.CIRQuery, updates *models.CIRUpdate) (*models.StorageResult, error) {
	if !m.initialized {
		return nil, fmt.Errorf("plugin not initialized")
	}

	entityType := query.EntityType
	if entityType == "" {
		entityType = "default"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	collection := m.db.Collection(entityType)
	filter := m.buildFilter(query.Filters)

	// Build $set update document
	setDoc := bson.M{}
	for key, value := range updates.Updates {
		setDoc[fmt.Sprintf("data.%s", key)] = value
	}

	updateDoc := bson.M{"$set": setDoc}

	result, err := collection.UpdateMany(ctx, filter, updateDoc)
	if err != nil {
		return nil, fmt.Errorf("failed to update mongodb: %w", err)
	}

	return &models.StorageResult{
		Success:       true,
		AffectedItems: int(result.ModifiedCount),
	}, nil
}

// Delete deletes CIR data from MongoDB
func (m *MongoDBPlugin) Delete(query *models.CIRQuery) (*models.StorageResult, error) {
	if !m.initialized {
		return nil, fmt.Errorf("plugin not initialized")
	}

	entityType := query.EntityType
	if entityType == "" {
		entityType = "default"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	collection := m.db.Collection(entityType)
	filter := m.buildFilter(query.Filters)

	result, err := collection.DeleteMany(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to delete from mongodb: %w", err)
	}

	return &models.StorageResult{
		Success:       true,
		AffectedItems: int(result.DeletedCount),
	}, nil
}

// GetMetadata returns metadata about the MongoDB storage
func (m *MongoDBPlugin) GetMetadata() (*models.StorageMetadata, error) {
	return &models.StorageMetadata{
		StorageType: "mongodb",
		Version:     "1.0.0",
		Capabilities: []string{
			"store",
			"retrieve",
			"update",
			"delete",
			"schema_creation",
			"document_query",
			"flexible_schema",
		},
	}, nil
}

// HealthCheck checks if the MongoDB connection is healthy
func (m *MongoDBPlugin) HealthCheck() (bool, error) {
	if !m.initialized {
		return false, fmt.Errorf("plugin not initialized")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := m.client.Ping(ctx, nil); err != nil {
		return false, fmt.Errorf("mongodb ping failed: %w", err)
	}

	return true, nil
}

// inferEntityType attempts to infer the entity type from CIR data
func (m *MongoDBPlugin) inferEntityType(cir *models.CIR) string {
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

