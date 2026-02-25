package plugins

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"
	"github.com/mimir-aip/mimir-aip-go/pkg/models"
	"github.com/mimir-aip/mimir-aip-go/pkg/storage"
)

// ElasticsearchPlugin implements the StoragePlugin interface for Elasticsearch storage
type ElasticsearchPlugin struct {
	client      *elasticsearch.Client
	initialized bool
}

// NewElasticsearchPlugin creates a new Elasticsearch storage plugin
func NewElasticsearchPlugin() *ElasticsearchPlugin {
	return &ElasticsearchPlugin{}
}

// esDocument represents a document stored in Elasticsearch
type esDocument struct {
	CIRVersion string                 `json:"cir_version"`
	SourceURI  string                 `json:"source_uri"`
	SourceType string                 `json:"source_type"`
	EntityType string                 `json:"entity_type"`
	Data       map[string]interface{} `json:"data"`
	CreatedAt  string                 `json:"created_at"`
}

// Initialize initializes the Elasticsearch plugin with configuration
func (e *ElasticsearchPlugin) Initialize(config *models.PluginConfig) error {
	if config.ConnectionString == "" {
		return fmt.Errorf("connection string is required for elasticsearch storage")
	}

	addresses := strings.Split(config.ConnectionString, ",")
	for i, addr := range addresses {
		addresses[i] = strings.TrimSpace(addr)
	}

	esCfg := elasticsearch.Config{
		Addresses: addresses,
	}

	if config.Credentials != nil {
		if username, ok := config.Credentials["username"].(string); ok {
			esCfg.Username = username
		}
		if password, ok := config.Credentials["password"].(string); ok {
			esCfg.Password = password
		}
		if apiKey, ok := config.Credentials["api_key"].(string); ok {
			esCfg.APIKey = apiKey
		}
	}

	client, err := elasticsearch.NewClient(esCfg)
	if err != nil {
		return fmt.Errorf("failed to create elasticsearch client: %w", err)
	}

	// Verify connectivity
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req := esapi.InfoRequest{}
	resp, err := req.Do(ctx, client)
	if err != nil {
		return fmt.Errorf("failed to connect to elasticsearch: %w", err)
	}
	defer resp.Body.Close()

	if resp.IsError() {
		return fmt.Errorf("elasticsearch info returned error: %s", resp.Status())
	}

	e.client = client
	e.initialized = true
	return nil
}

// CreateSchema creates Elasticsearch indexes for each entity type
func (e *ElasticsearchPlugin) CreateSchema(ontology *models.OntologyDefinition) error {
	if !e.initialized {
		return fmt.Errorf("plugin not initialized")
	}

	for _, entity := range ontology.Entities {
		indexName := fmt.Sprintf("mimir-%s", strings.ToLower(entity.Name))

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		existsReq := esapi.IndicesExistsRequest{Index: []string{indexName}}
		existsResp, err := existsReq.Do(ctx, e.client)
		cancel()

		if err != nil {
			return fmt.Errorf("failed to check index existence for %s: %w", indexName, err)
		}
		existsResp.Body.Close()

		if existsResp.StatusCode == 200 {
			// Index already exists
			continue
		}

		mappings := map[string]interface{}{
			"mappings": map[string]interface{}{
				"properties": map[string]interface{}{
					"cir_version": map[string]interface{}{"type": "keyword"},
					"source_uri":  map[string]interface{}{"type": "keyword"},
					"source_type": map[string]interface{}{"type": "keyword"},
					"entity_type": map[string]interface{}{"type": "keyword"},
					"created_at":  map[string]interface{}{"type": "date"},
					"data":        map[string]interface{}{"type": "object", "dynamic": true},
				},
			},
		}

		mappingData, err := json.Marshal(mappings)
		if err != nil {
			return fmt.Errorf("failed to marshal mappings: %w", err)
		}

		createCtx, createCancel := context.WithTimeout(context.Background(), 10*time.Second)
		createReq := esapi.IndicesCreateRequest{
			Index: indexName,
			Body:  bytes.NewReader(mappingData),
		}
		createResp, createErr := createReq.Do(createCtx, e.client)
		createCancel()

		if createErr != nil {
			return fmt.Errorf("failed to create index %s: %w", indexName, createErr)
		}
		createResp.Body.Close()

		if createResp.IsError() {
			return fmt.Errorf("failed to create index %s: %s", indexName, createResp.Status())
		}
	}

	return nil
}

// Store stores CIR data into Elasticsearch
func (e *ElasticsearchPlugin) Store(cir *models.CIR) (*models.StorageResult, error) {
	if !e.initialized {
		return nil, fmt.Errorf("plugin not initialized")
	}

	if err := cir.Validate(); err != nil {
		return nil, fmt.Errorf("invalid CIR: %w", err)
	}

	entityType := e.inferEntityType(cir)
	affectedItems := 0

	if arr, err := cir.GetDataAsArray(); err == nil {
		for _, item := range arr {
			itemCIR := &models.CIR{
				Version:  cir.Version,
				Source:   cir.Source,
				Data:     item,
				Metadata: cir.Metadata,
			}
			if err := e.indexDocument(entityType, itemCIR); err != nil {
				return nil, fmt.Errorf("failed to store item: %w", err)
			}
			affectedItems++
		}
	} else {
		if err := e.indexDocument(entityType, cir); err != nil {
			return nil, fmt.Errorf("failed to store item: %w", err)
		}
		affectedItems = 1
	}

	return &models.StorageResult{
		Success:       true,
		AffectedItems: affectedItems,
	}, nil
}

func (e *ElasticsearchPlugin) indexDocument(entityType string, cir *models.CIR) error {
	dataMap, err := cir.GetDataAsMap()
	if err != nil {
		dataMap = map[string]interface{}{}
	}

	indexName := fmt.Sprintf("mimir-%s", strings.ToLower(entityType))

	doc := esDocument{
		CIRVersion: cir.Version,
		SourceURI:  cir.Source.URI,
		SourceType: string(cir.Source.Type),
		EntityType: entityType,
		Data:       dataMap,
		CreatedAt:  time.Now().UTC().Format(time.RFC3339),
	}

	docData, err := json.Marshal(doc)
	if err != nil {
		return fmt.Errorf("failed to marshal document: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req := esapi.IndexRequest{
		Index: indexName,
		Body:  bytes.NewReader(docData),
	}

	resp, err := req.Do(ctx, e.client)
	if err != nil {
		return fmt.Errorf("failed to index document: %w", err)
	}
	defer resp.Body.Close()

	if resp.IsError() {
		return fmt.Errorf("elasticsearch index error: %s", resp.Status())
	}

	return nil
}

// Retrieve retrieves CIR data from Elasticsearch using a query
func (e *ElasticsearchPlugin) Retrieve(query *models.CIRQuery) ([]*models.CIR, error) {
	if !e.initialized {
		return nil, fmt.Errorf("plugin not initialized")
	}

	if err := storage.ValidateCIRQuery(query); err != nil {
		return nil, fmt.Errorf("invalid query: %w", err)
	}

	entityType := query.EntityType
	if entityType == "" {
		entityType = "default"
	}

	indexName := fmt.Sprintf("mimir-%s", strings.ToLower(entityType))

	esQuery := e.buildESQuery(query.Filters)

	queryData, err := json.Marshal(esQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal es query: %w", err)
	}

	req := esapi.SearchRequest{
		Index: []string{indexName},
		Body:  bytes.NewReader(queryData),
	}

	if query.Limit > 0 {
		size := query.Limit
		req.Size = &size
	}
	if query.Offset > 0 {
		from := query.Offset
		req.From = &from
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	resp, err := req.Do(ctx, e.client)
	if err != nil {
		return nil, fmt.Errorf("failed to search elasticsearch: %w", err)
	}
	defer resp.Body.Close()

	if resp.IsError() {
		if resp.StatusCode == 404 {
			return []*models.CIR{}, nil
		}
		return nil, fmt.Errorf("elasticsearch search error: %s", resp.Status())
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read elasticsearch response: %w", err)
	}

	var searchResult map[string]interface{}
	if err := json.Unmarshal(body, &searchResult); err != nil {
		return nil, fmt.Errorf("failed to unmarshal elasticsearch response: %w", err)
	}

	results := make([]*models.CIR, 0)

	hitsOuter, ok := searchResult["hits"].(map[string]interface{})
	if !ok {
		return results, nil
	}

	hitsArr, ok := hitsOuter["hits"].([]interface{})
	if !ok {
		return results, nil
	}

	for _, hitRaw := range hitsArr {
		hit, ok := hitRaw.(map[string]interface{})
		if !ok {
			continue
		}

		source, ok := hit["_source"].(map[string]interface{})
		if !ok {
			continue
		}

		cir := e.sourceToCIR(source)
		if cir != nil {
			results = append(results, cir)
		}
	}

	return results, nil
}

// buildESQuery constructs an Elasticsearch query DSL from CIR conditions
func (e *ElasticsearchPlugin) buildESQuery(filters []models.CIRCondition) map[string]interface{} {
	if len(filters) == 0 {
		return map[string]interface{}{
			"query": map[string]interface{}{
				"match_all": map[string]interface{}{},
			},
		}
	}

	mustClauses := make([]interface{}, 0, len(filters))

	for _, f := range filters {
		dataField := fmt.Sprintf("data.%s", f.Attribute)
		switch f.Operator {
		case "eq":
			mustClauses = append(mustClauses, map[string]interface{}{
				"term": map[string]interface{}{dataField: f.Value},
			})
		case "neq":
			mustClauses = append(mustClauses, map[string]interface{}{
				"bool": map[string]interface{}{
					"must_not": []interface{}{
						map[string]interface{}{
							"term": map[string]interface{}{dataField: f.Value},
						},
					},
				},
			})
		case "gt":
			mustClauses = append(mustClauses, map[string]interface{}{
				"range": map[string]interface{}{
					dataField: map[string]interface{}{"gt": toFloat(f.Value)},
				},
			})
		case "gte":
			mustClauses = append(mustClauses, map[string]interface{}{
				"range": map[string]interface{}{
					dataField: map[string]interface{}{"gte": toFloat(f.Value)},
				},
			})
		case "lt":
			mustClauses = append(mustClauses, map[string]interface{}{
				"range": map[string]interface{}{
					dataField: map[string]interface{}{"lt": toFloat(f.Value)},
				},
			})
		case "lte":
			mustClauses = append(mustClauses, map[string]interface{}{
				"range": map[string]interface{}{
					dataField: map[string]interface{}{"lte": toFloat(f.Value)},
				},
			})
		case "like":
			mustClauses = append(mustClauses, map[string]interface{}{
				"wildcard": map[string]interface{}{
					dataField: map[string]interface{}{
						"value":            fmt.Sprintf("*%v*", f.Value),
						"case_insensitive": true,
					},
				},
			})
		case "in":
			mustClauses = append(mustClauses, map[string]interface{}{
				"terms": map[string]interface{}{dataField: f.Value},
			})
		}
	}

	return map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"must": mustClauses,
			},
		},
	}
}

// sourceToCIR converts an Elasticsearch hit source back to a CIR object
func (e *ElasticsearchPlugin) sourceToCIR(source map[string]interface{}) *models.CIR {
	cir := &models.CIR{}

	if v, ok := source["cir_version"].(string); ok {
		cir.Version = v
	} else {
		cir.Version = "1.0"
	}

	sourceType := ""
	if v, ok := source["source_type"].(string); ok {
		sourceType = v
	}
	sourceURI := ""
	if v, ok := source["source_uri"].(string); ok {
		sourceURI = v
	}

	cir.Source = models.CIRSource{
		Type:       models.SourceType(sourceType),
		URI:        sourceURI,
		Timestamp:  time.Now(),
		Format:     models.DataFormatJSON,
		Parameters: make(map[string]interface{}),
	}

	if data, ok := source["data"]; ok {
		cir.Data = data
	} else {
		cir.Data = source
	}

	cir.Metadata = models.CIRMetadata{}

	return cir
}

// Update updates existing CIR data in Elasticsearch using update_by_query
func (e *ElasticsearchPlugin) Update(query *models.CIRQuery, updates *models.CIRUpdate) (*models.StorageResult, error) {
	if !e.initialized {
		return nil, fmt.Errorf("plugin not initialized")
	}

	entityType := query.EntityType
	if entityType == "" {
		entityType = "default"
	}

	indexName := fmt.Sprintf("mimir-%s", strings.ToLower(entityType))

	// Build the script for updating fields
	scriptParts := make([]string, 0, len(updates.Updates))
	params := make(map[string]interface{})

	for key, value := range updates.Updates {
		paramKey := fmt.Sprintf("param_%s", key)
		scriptParts = append(scriptParts, fmt.Sprintf("ctx._source.data.%s = params.%s", key, paramKey))
		params[paramKey] = value
	}

	script := strings.Join(scriptParts, "; ")

	esQuery := e.buildESQuery(query.Filters)
	filterQuery := esQuery["query"]

	updateByQuery := map[string]interface{}{
		"query": filterQuery,
		"script": map[string]interface{}{
			"source": script,
			"lang":   "painless",
			"params": params,
		},
	}

	queryData, err := json.Marshal(updateByQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal update query: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req := esapi.UpdateByQueryRequest{
		Index: []string{indexName},
		Body:  bytes.NewReader(queryData),
	}

	resp, err := req.Do(ctx, e.client)
	if err != nil {
		return nil, fmt.Errorf("failed to update elasticsearch: %w", err)
	}
	defer resp.Body.Close()

	if resp.IsError() {
		if resp.StatusCode == 404 {
			return &models.StorageResult{Success: true, AffectedItems: 0}, nil
		}
		return nil, fmt.Errorf("elasticsearch update_by_query error: %s", resp.Status())
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read update response: %w", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal update response: %w", err)
	}

	updated := 0
	if v, ok := result["updated"].(float64); ok {
		updated = int(v)
	}

	return &models.StorageResult{
		Success:       true,
		AffectedItems: updated,
	}, nil
}

// Delete deletes CIR data from Elasticsearch using delete_by_query
func (e *ElasticsearchPlugin) Delete(query *models.CIRQuery) (*models.StorageResult, error) {
	if !e.initialized {
		return nil, fmt.Errorf("plugin not initialized")
	}

	entityType := query.EntityType
	if entityType == "" {
		entityType = "default"
	}

	indexName := fmt.Sprintf("mimir-%s", strings.ToLower(entityType))

	esQuery := e.buildESQuery(query.Filters)

	queryData, err := json.Marshal(esQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal delete query: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req := esapi.DeleteByQueryRequest{
		Index: []string{indexName},
		Body:  bytes.NewReader(queryData),
	}

	resp, err := req.Do(ctx, e.client)
	if err != nil {
		return nil, fmt.Errorf("failed to delete from elasticsearch: %w", err)
	}
	defer resp.Body.Close()

	if resp.IsError() {
		if resp.StatusCode == 404 {
			return &models.StorageResult{Success: true, AffectedItems: 0}, nil
		}
		return nil, fmt.Errorf("elasticsearch delete_by_query error: %s", resp.Status())
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read delete response: %w", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal delete response: %w", err)
	}

	deleted := 0
	if v, ok := result["deleted"].(float64); ok {
		deleted = int(v)
	}

	return &models.StorageResult{
		Success:       true,
		AffectedItems: deleted,
	}, nil
}

// GetMetadata returns metadata about the Elasticsearch storage
func (e *ElasticsearchPlugin) GetMetadata() (*models.StorageMetadata, error) {
	return &models.StorageMetadata{
		StorageType: "elasticsearch",
		Version:     "1.0.0",
		Capabilities: []string{
			"store",
			"retrieve",
			"update",
			"delete",
			"full_text_search",
			"schema_creation",
			"aggregations",
			"scalable",
		},
	}, nil
}

// HealthCheck checks if the Elasticsearch connection is healthy
func (e *ElasticsearchPlugin) HealthCheck() (bool, error) {
	if !e.initialized {
		return false, fmt.Errorf("plugin not initialized")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req := esapi.PingRequest{}
	resp, err := req.Do(ctx, e.client)
	if err != nil {
		return false, fmt.Errorf("elasticsearch ping failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.IsError() {
		return false, fmt.Errorf("elasticsearch ping returned error: %s", resp.Status())
	}

	return true, nil
}

// inferEntityType attempts to infer the entity type from CIR data
func (e *ElasticsearchPlugin) inferEntityType(cir *models.CIR) string {
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
