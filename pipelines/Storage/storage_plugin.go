package storage

import (
	"context"
	"fmt"

	"github.com/Mimir-AIP/Mimir-AIP-Go/pipelines"
)

// StoragePlugin provides pipeline integration for storage operations
// It acts as a bridge between the pipeline system and various storage backends
type StoragePlugin struct {
	backend          StorageBackend
	embeddingService EmbeddingService
}

// NewStoragePlugin creates a new StoragePlugin with the given backend
func NewStoragePlugin(backend StorageBackend) *StoragePlugin {
	return &StoragePlugin{
		backend: backend,
	}
}

// NewStoragePluginWithEmbedding creates a new StoragePlugin with backend and embedding service
func NewStoragePluginWithEmbedding(backend StorageBackend, embeddingService EmbeddingService) *StoragePlugin {
	return &StoragePlugin{
		backend:          backend,
		embeddingService: embeddingService,
	}
}

// ExecuteStep executes a storage operation based on the step configuration
func (sp *StoragePlugin) ExecuteStep(ctx context.Context, stepConfig pipelines.StepConfig, globalContext *pipelines.PluginContext) (*pipelines.PluginContext, error) {
	operation, ok := stepConfig.Config["operation"].(string)
	if !ok {
		return pipelines.NewPluginContext(), fmt.Errorf("operation is required and must be a string")
	}

	switch operation {
	case "store":
		return sp.handleStore(ctx, stepConfig, globalContext)
	case "query":
		return sp.handleQuery(ctx, stepConfig, globalContext)
	case "batch_store":
		return sp.handleBatchStore(ctx, stepConfig, globalContext)
	case "delete":
		return sp.handleDelete(ctx, stepConfig, globalContext)
	case "get":
		return sp.handleGet(ctx, stepConfig, globalContext)
	case "create_collection":
		return sp.handleCreateCollection(ctx, stepConfig, globalContext)
	case "list_collections":
		return sp.handleListCollections(ctx, stepConfig, globalContext)
	default:
		return pipelines.NewPluginContext(), fmt.Errorf("unknown storage operation: %s", operation)
	}
}

// handleStore stores documents in the specified collection
func (sp *StoragePlugin) handleStore(ctx context.Context, stepConfig pipelines.StepConfig, globalContext *pipelines.PluginContext) (*pipelines.PluginContext, error) {
	collection, ok := stepConfig.Config["collection"].(string)
	if !ok || collection == "" {
		return pipelines.NewPluginContext(), fmt.Errorf("collection is required and must be a string")
	}

	// Get documents from config or context
	var documents []Document
	if docsConfig, exists := stepConfig.Config["documents"]; exists {
		// Try different ways to extract documents
		if docsSlice, ok := docsConfig.([]interface{}); ok {
			documents = sp.convertToDocuments(docsSlice)
		} else if docsSlice, ok := docsConfig.([]map[string]interface{}); ok {
			// Handle direct []map[string]interface{} input
			documents = sp.convertToDocumentsFromMaps(docsSlice)
		} else if docSlice, ok := docsConfig.([]Document); ok {
			// Handle direct []Document input
			documents = docSlice
		}
	}

	// If no documents in config, try to get from context
	if len(documents) == 0 {
		if docsFromContext, exists := globalContext.Get(stepConfig.Output); exists {
			if docsSlice, ok := docsFromContext.([]interface{}); ok {
				documents = sp.convertToDocuments(docsSlice)
			}
		}
	}

	if len(documents) == 0 {
		return pipelines.NewPluginContext(), fmt.Errorf("no documents provided for storage")
	}

	// Store documents
	err := sp.backend.Store(ctx, collection, documents, nil) // Let backend handle embeddings
	if err != nil {
		return pipelines.NewPluginContext(), fmt.Errorf("failed to store documents: %w", err)
	}

	result := pipelines.NewPluginContext()
	result.Set("store_result", map[string]interface{}{
		"operation":  "store",
		"collection": collection,
		"count":      float64(len(documents)),
		"successful": true,
	})

	return result, nil
}

// handleQuery performs a similarity search
func (sp *StoragePlugin) handleQuery(ctx context.Context, stepConfig pipelines.StepConfig, globalContext *pipelines.PluginContext) (*pipelines.PluginContext, error) {
	collection, ok := stepConfig.Config["collection"].(string)
	if !ok || collection == "" {
		return pipelines.NewPluginContext(), fmt.Errorf("collection is required and must be a string")
	}

	// Get query parameters
	queryText, _ := stepConfig.Config["query"].(string)
	if queryText == "" {
		return pipelines.NewPluginContext(), fmt.Errorf("query is required and must be a string")
	}

	limit := 10 // default
	if limitConfig, exists := stepConfig.Config["limit"]; exists {
		if limitVal, ok := limitConfig.(int); ok {
			limit = limitVal
		}
	}

	// Convert text query to vector using embedding service
	var queryVector []float32
	var err error

	if sp.embeddingService != nil {
		queryVector, err = sp.embeddingService.EmbedText(ctx, queryText)
		if err != nil {
			return pipelines.NewPluginContext(), fmt.Errorf("failed to embed query text: %w", err)
		}
	} else {
		// Fallback to simple method if no embedding service
		queryVector = sp.simpleTextToVector(queryText)
	}

	// Perform query
	results, err := sp.backend.Query(ctx, collection, queryVector, limit, nil)
	if err != nil {
		return pipelines.NewPluginContext(), fmt.Errorf("failed to query collection: %w", err)
	}

	result := pipelines.NewPluginContext()
	result.Set("query_result", map[string]interface{}{
		"operation":  "query",
		"collection": collection,
		"query":      queryText,
		"limit":      float64(limit),
		"count":      float64(len(results)),
		"results":    sp.convertQueryResults(results),
	})

	return result, nil
}

// handleBatchStore handles batch storage operations
func (sp *StoragePlugin) handleBatchStore(ctx context.Context, stepConfig pipelines.StepConfig, globalContext *pipelines.PluginContext) (*pipelines.PluginContext, error) {
	collection, ok := stepConfig.Config["collection"].(string)
	if !ok || collection == "" {
		return pipelines.NewPluginContext(), fmt.Errorf("collection is required and must be a string")
	}

	var documents []Document
	if docsConfig, exists := stepConfig.Config["documents"]; exists {
		if docsSlice, ok := docsConfig.([]interface{}); ok {
			documents = sp.convertToDocuments(docsSlice)
		} else if docsSlice, ok := docsConfig.([]map[string]interface{}); ok {
			documents = sp.convertToDocumentsFromMaps(docsSlice)
		} else if docSlice, ok := docsConfig.([]Document); ok {
			documents = docSlice
		}
	}

	if len(documents) == 0 {
		return pipelines.NewPluginContext(), fmt.Errorf("no documents provided for batch storage")
	}

	err := sp.backend.Store(ctx, collection, documents, nil)
	if err != nil {
		return pipelines.NewPluginContext(), fmt.Errorf("failed to batch store documents: %w", err)
	}

	result := pipelines.NewPluginContext()
	result.Set("batch_result", map[string]interface{}{
		"operation":  "store",
		"collection": collection,
		"count":      float64(len(documents)),
		"successful": true,
	})

	return result, nil
}

// handleDelete removes documents from storage
func (sp *StoragePlugin) handleDelete(ctx context.Context, stepConfig pipelines.StepConfig, globalContext *pipelines.PluginContext) (*pipelines.PluginContext, error) {
	collection, ok := stepConfig.Config["collection"].(string)
	if !ok || collection == "" {
		return pipelines.NewPluginContext(), fmt.Errorf("collection is required and must be a string")
	}

	idsConfig, exists := stepConfig.Config["ids"]
	if !exists {
		return pipelines.NewPluginContext(), fmt.Errorf("ids are required for delete operation")
	}

	var ids []string
	if idsSlice, ok := idsConfig.([]interface{}); ok {
		for _, id := range idsSlice {
			if idStr, ok := id.(string); ok {
				ids = append(ids, idStr)
			}
		}
	} else if idsSlice, ok := idsConfig.([]string); ok {
		ids = idsSlice
	}

	if len(ids) == 0 {
		return pipelines.NewPluginContext(), fmt.Errorf("no valid ids provided for deletion")
	}

	err := sp.backend.Delete(ctx, collection, ids)
	if err != nil {
		return pipelines.NewPluginContext(), fmt.Errorf("failed to delete documents: %w", err)
	}

	result := pipelines.NewPluginContext()
	result.Set("delete_result", map[string]interface{}{
		"operation":  "delete",
		"collection": collection,
		"count":      float64(len(ids)),
		"successful": true,
	})

	return result, nil
}

// handleGet retrieves a single document by ID
func (sp *StoragePlugin) handleGet(ctx context.Context, stepConfig pipelines.StepConfig, globalContext *pipelines.PluginContext) (*pipelines.PluginContext, error) {
	collection, ok := stepConfig.Config["collection"].(string)
	if !ok || collection == "" {
		return pipelines.NewPluginContext(), fmt.Errorf("collection is required and must be a string")
	}

	id, ok := stepConfig.Config["id"].(string)
	if !ok || id == "" {
		return pipelines.NewPluginContext(), fmt.Errorf("id is required and must be a string")
	}

	doc, err := sp.backend.GetDocument(ctx, collection, id)
	if err != nil {
		return pipelines.NewPluginContext(), fmt.Errorf("failed to get document: %w", err)
	}

	result := pipelines.NewPluginContext()
	result.Set("document", map[string]interface{}{
		"id":       doc.ID,
		"content":  doc.Content,
		"metadata": doc.Metadata,
	})

	return result, nil
}

// handleCreateCollection creates a new collection
func (sp *StoragePlugin) handleCreateCollection(ctx context.Context, stepConfig pipelines.StepConfig, globalContext *pipelines.PluginContext) (*pipelines.PluginContext, error) {
	name, ok := stepConfig.Config["name"].(string)
	if !ok || name == "" {
		return pipelines.NewPluginContext(), fmt.Errorf("name is required and must be a string")
	}

	config := CollectionConfig{
		Name:        name,
		Dimension:   1536, // Default for OpenAI embeddings
		Metric:      CosineSimilarity,
		Persistence: true,
	}

	err := sp.backend.CreateCollection(ctx, name, config)
	if err != nil {
		return pipelines.NewPluginContext(), fmt.Errorf("failed to create collection: %w", err)
	}

	result := pipelines.NewPluginContext()
	result.Set("create_result", map[string]interface{}{
		"operation":  "create_collection",
		"name":       name,
		"successful": true,
	})

	return result, nil
}

// handleListCollections lists all collections
func (sp *StoragePlugin) handleListCollections(ctx context.Context, stepConfig pipelines.StepConfig, globalContext *pipelines.PluginContext) (*pipelines.PluginContext, error) {
	collections, err := sp.backend.ListCollections(ctx)
	if err != nil {
		return pipelines.NewPluginContext(), fmt.Errorf("failed to list collections: %w", err)
	}

	result := pipelines.NewPluginContext()
	result.Set("collections", collections)
	return result, nil
}

// Helper methods

// convertToDocuments converts interface{} slice to Document slice
func (sp *StoragePlugin) convertToDocuments(docs []interface{}) []Document {
	documents := make([]Document, 0, len(docs))
	for _, doc := range docs {
		if docMap, ok := doc.(map[string]interface{}); ok {
			document := Document{
				Metadata: make(map[string]interface{}),
			}

			if id, exists := docMap["id"]; exists {
				if idStr, ok := id.(string); ok {
					document.ID = idStr
				}
			}

			if content, exists := docMap["content"]; exists {
				if contentStr, ok := content.(string); ok {
					document.Content = contentStr
				}
			}

			if metadata, exists := docMap["metadata"]; exists {
				if metaMap, ok := metadata.(map[string]interface{}); ok {
					document.Metadata = metaMap
				}
			}

			documents = append(documents, document)
		}
	}
	return documents
}

// convertToDocumentsFromMaps converts []map[string]interface{} to Document slice
func (sp *StoragePlugin) convertToDocumentsFromMaps(docs []map[string]interface{}) []Document {
	documents := make([]Document, 0, len(docs))
	for _, docMap := range docs {
		document := Document{
			Metadata: make(map[string]interface{}),
		}

		if id, exists := docMap["id"]; exists {
			if idStr, ok := id.(string); ok {
				document.ID = idStr
			}
		}

		if content, exists := docMap["content"]; exists {
			if contentStr, ok := content.(string); ok {
				document.Content = contentStr
			}
		}

		if metadata, exists := docMap["metadata"]; exists {
			if metaMap, ok := metadata.(map[string]interface{}); ok {
				document.Metadata = metaMap
			}
		}

		documents = append(documents, document)
	}
	return documents
}

// convertQueryResults converts QueryResult slice to interface{} for pipeline context
func (sp *StoragePlugin) convertQueryResults(results []QueryResult) []interface{} {
	converted := make([]interface{}, len(results))
	for i, result := range results {
		converted[i] = map[string]interface{}{
			"id":       result.Document.ID,
			"content":  result.Document.Content,
			"metadata": result.Document.Metadata,
			"score":    result.Score,
		}
	}
	return converted
}

// simpleTextToVector creates a simple vector from text (placeholder implementation)
// In a real implementation, this would use proper embeddings
func (sp *StoragePlugin) simpleTextToVector(text string) []float32 {
	// This is a placeholder - real implementation would use embeddings
	// For now, we'll create a simple hash-based vector
	vector := make([]float32, 1536) // Standard dimension for many embedding models
	for i := range vector {
		vector[i] = float32(len(text)%100) / 100.0 // Simple deterministic value
	}
	return vector
}

// Plugin interface methods

func (sp *StoragePlugin) GetPluginType() string {
	return "Storage"
}

func (sp *StoragePlugin) GetPluginName() string {
	return "vector"
}

func (sp *StoragePlugin) ValidateConfig(config map[string]interface{}) error {
	if _, exists := config["operation"]; !exists {
		return fmt.Errorf("operation is required")
	}
	return nil
}
