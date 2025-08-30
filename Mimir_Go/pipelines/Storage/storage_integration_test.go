package storage

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/Mimir-AIP/Mimir-AIP-Go/pipelines"
	"github.com/philippgille/chromem-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStorageSystemIntegration(t *testing.T) {
	ctx := context.Background()

	// Setup storage backend
	factory := NewBackendFactory()
	config := &StorageConfig{
		Type: BackendTypeChromem,
		ChromemConfig: &ChromemConfig{
			CollectionName:   "test_collection",
			SimilarityMetric: CosineSimilarity,
			EmbeddingFunc:    CreateMockEmbeddingFunc(384),
		},
	}

	backend, err := factory.CreateBackend(config)
	require.NoError(t, err)
	defer backend.Close()

	// Setup embedding service
	embeddingFactory := NewEmbeddingServiceFactory()
	embeddingService := embeddingFactory.CreateDefaultService()

	// Setup storage plugin
	plugin := NewStoragePluginWithEmbedding(backend, embeddingService)

	t.Run("Basic CRUD Operations", func(t *testing.T) {
		testBasicCRUDOperations(t, ctx, plugin)
	})

	t.Run("Query Operations", func(t *testing.T) {
		testQueryOperations(t, ctx, plugin)
	})

	t.Run("Batch Operations", func(t *testing.T) {
		testBatchOperations(t, ctx, plugin)
	})

	t.Run("Collection Management", func(t *testing.T) {
		testCollectionManagement(t, ctx, plugin)
	})

	t.Run("Error Handling", func(t *testing.T) {
		testErrorHandling(t, ctx, plugin)
	})
}

func testBasicCRUDOperations(t *testing.T, ctx context.Context, plugin *StoragePlugin) {
	// Test document storage
	documents := []Document{
		{
			ID:      "test_doc_1",
			Content: "This is a test document about artificial intelligence",
			Metadata: map[string]interface{}{
				"category": "technology",
				"tags":     []string{"AI", "ML"},
			},
		},
		{
			ID:      "test_doc_2",
			Content: "Machine learning is a subset of AI that focuses on algorithms",
			Metadata: map[string]interface{}{
				"category": "technology",
				"tags":     []string{"ML", "algorithms"},
			},
		},
	}

	// Convert documents to the format expected by the plugin
	docMaps := make([]map[string]interface{}, len(documents))
	for i, doc := range documents {
		docMaps[i] = map[string]interface{}{
			"id":       doc.ID,
			"content":  doc.Content,
			"metadata": doc.Metadata,
		}
	}

	stepConfig := pipelines.StepConfig{
		Name:   "Store Test Documents",
		Plugin: "Storage.vector",
		Config: map[string]interface{}{
			"operation":  "store",
			"collection": "test_collection",
			"documents":  docMaps,
		},
		Output: "store_result",
	}

	result, err := plugin.ExecuteStep(ctx, stepConfig, *pipelines.NewPluginContext())
	require.NoError(t, err)

	// Verify storage result
	storeResult, exists := result.Get("store_result")
	require.True(t, exists)

	resultMap, ok := storeResult.(map[string]interface{})
	require.True(t, ok)

	assert.Equal(t, "store", resultMap["operation"])
	assert.Equal(t, "test_collection", resultMap["collection"])
	assert.Equal(t, float64(2), resultMap["count"])
	assert.True(t, resultMap["successful"].(bool))

	// Test document retrieval
	getStepConfig := pipelines.StepConfig{
		Name:   "Get Test Document",
		Plugin: "Storage.vector",
		Config: map[string]interface{}{
			"operation":  "get",
			"collection": "test_collection",
			"id":         "test_doc_1",
		},
		Output: "document",
	}

	getResult, err := plugin.ExecuteStep(ctx, getStepConfig, *pipelines.NewPluginContext())
	require.NoError(t, err)

	docResult, exists := getResult.Get("document")
	require.True(t, exists)

	docMap, ok := docResult.(map[string]interface{})
	require.True(t, ok)

	assert.Equal(t, "test_doc_1", docMap["id"])
	assert.Equal(t, "This is a test document about artificial intelligence", docMap["content"])
	assert.Equal(t, "technology", docMap["metadata"].(map[string]interface{})["category"])

	// Test document deletion
	deleteStepConfig := pipelines.StepConfig{
		Name:   "Delete Test Document",
		Plugin: "Storage.vector",
		Config: map[string]interface{}{
			"operation":  "delete",
			"collection": "test_collection",
			"ids":        []string{"test_doc_1"},
		},
		Output: "delete_result",
	}

	deleteResult, err := plugin.ExecuteStep(ctx, deleteStepConfig, *pipelines.NewPluginContext())
	require.NoError(t, err)

	deleteResultMap, exists := deleteResult.Get("delete_result")
	require.True(t, exists)

	deleteMap, ok := deleteResultMap.(map[string]interface{})
	require.True(t, ok)

	assert.Equal(t, "delete", deleteMap["operation"])
	assert.Equal(t, float64(1), deleteMap["count"])
	assert.True(t, deleteMap["successful"].(bool))
}

func testQueryOperations(t *testing.T, ctx context.Context, plugin *StoragePlugin) {
	// First, store some documents for querying
	documents := []Document{
		{
			ID:      "query_doc_1",
			Content: "The weather is beautiful today with clear blue skies",
			Metadata: map[string]interface{}{
				"topic": "weather",
			},
		},
		{
			ID:      "query_doc_2",
			Content: "Machine learning algorithms can predict weather patterns",
			Metadata: map[string]interface{}{
				"topic": "technology",
			},
		},
		{
			ID:      "query_doc_3",
			Content: "Blue is a primary color in the visible light spectrum",
			Metadata: map[string]interface{}{
				"topic": "science",
			},
		},
	}

	// Convert documents to the format expected by the plugin
	docMaps := make([]map[string]interface{}, len(documents))
	for i, doc := range documents {
		docMaps[i] = map[string]interface{}{
			"id":       doc.ID,
			"content":  doc.Content,
			"metadata": doc.Metadata,
		}
	}

	storeStepConfig := pipelines.StepConfig{
		Name:   "Store Query Test Documents",
		Plugin: "Storage.vector",
		Config: map[string]interface{}{
			"operation":  "store",
			"collection": "query_test_collection",
			"documents":  docMaps,
		},
		Output: "store_result",
	}

	_, err := plugin.ExecuteStep(ctx, storeStepConfig, *pipelines.NewPluginContext())
	require.NoError(t, err)

	// Test semantic search
	queryStepConfig := pipelines.StepConfig{
		Name:   "Query Weather Documents",
		Plugin: "Storage.vector",
		Config: map[string]interface{}{
			"operation":  "query",
			"collection": "query_test_collection",
			"query":      "What is the weather like?",
			"limit":      2,
		},
		Output: "query_result",
	}

	queryResult, err := plugin.ExecuteStep(ctx, queryStepConfig, *pipelines.NewPluginContext())
	require.NoError(t, err)

	queryResultMap, exists := queryResult.Get("query_result")
	require.True(t, exists)

	resultMap, ok := queryResultMap.(map[string]interface{})
	require.True(t, ok)

	assert.Equal(t, "query", resultMap["operation"])
	assert.Equal(t, "query_test_collection", resultMap["collection"])
	assert.Equal(t, "What is the weather like?", resultMap["query"])
	assert.Equal(t, float64(2), resultMap["limit"])

	results, ok := resultMap["results"].([]interface{})
	require.True(t, ok)
	assert.True(t, len(results) > 0)

	// The first result should be about weather
	firstResult, ok := results[0].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "query_doc_1", firstResult["id"])
}

func testBatchOperations(t *testing.T, ctx context.Context, plugin *StoragePlugin) {
	// Create a larger set of documents for batch testing
	documents := make([]Document, 10)
	for i := 0; i < 10; i++ {
		documents[i] = Document{
			ID:      fmt.Sprintf("batch_doc_%d", i),
			Content: fmt.Sprintf("This is batch document number %d with some content", i),
			Metadata: map[string]interface{}{
				"batch": i,
				"type":  "test",
			},
		}
	}

	// Convert documents to the format expected by the plugin
	docMaps := make([]map[string]interface{}, len(documents))
	for i, doc := range documents {
		docMaps[i] = map[string]interface{}{
			"id":       doc.ID,
			"content":  doc.Content,
			"metadata": doc.Metadata,
		}
	}

	batchStepConfig := pipelines.StepConfig{
		Name:   "Batch Store Documents",
		Plugin: "Storage.vector",
		Config: map[string]interface{}{
			"operation":  "batch_store",
			"collection": "batch_test_collection",
			"documents":  docMaps,
		},
		Output: "batch_result",
	}

	batchResult, err := plugin.ExecuteStep(ctx, batchStepConfig, *pipelines.NewPluginContext())
	require.NoError(t, err)

	batchResultMap, exists := batchResult.Get("batch_result")
	require.True(t, exists)

	resultMap, ok := batchResultMap.(map[string]interface{})
	require.True(t, ok)

	assert.Equal(t, "store", resultMap["operation"])
	assert.Equal(t, float64(10), resultMap["count"])
	assert.True(t, resultMap["successful"].(bool))
}

func testCollectionManagement(t *testing.T, ctx context.Context, plugin *StoragePlugin) {
	collectionName := "management_test_collection"

	// Test collection creation
	createStepConfig := pipelines.StepConfig{
		Name:   "Create Test Collection",
		Plugin: "Storage.vector",
		Config: map[string]interface{}{
			"operation": "create_collection",
			"name":      collectionName,
		},
		Output: "create_result",
	}

	createResult, err := plugin.ExecuteStep(ctx, createStepConfig, *pipelines.NewPluginContext())
	require.NoError(t, err)

	createResultMap, exists := createResult.Get("create_result")
	require.True(t, exists)

	resultMap, ok := createResultMap.(map[string]interface{})
	require.True(t, ok)

	assert.Equal(t, "create_collection", resultMap["operation"])
	assert.Equal(t, collectionName, resultMap["name"])
	assert.True(t, resultMap["successful"].(bool))

	// Test collection listing
	listStepConfig := pipelines.StepConfig{
		Name:   "List Collections",
		Plugin: "Storage.vector",
		Config: map[string]interface{}{
			"operation": "list_collections",
		},
		Output: "collections",
	}

	listResult, err := plugin.ExecuteStep(ctx, listStepConfig, *pipelines.NewPluginContext())
	require.NoError(t, err)

	collections, exists := listResult.Get("collections")
	require.True(t, exists)

	colSlice, ok := collections.([]string)
	require.True(t, ok)

	assert.Contains(t, colSlice, collectionName)
}

func testErrorHandling(t *testing.T, ctx context.Context, plugin *StoragePlugin) {
	// Test querying non-existent collection
	queryStepConfig := pipelines.StepConfig{
		Name:   "Query Non-existent Collection",
		Plugin: "Storage.vector",
		Config: map[string]interface{}{
			"operation":  "query",
			"collection": "non_existent_collection",
			"query":      "test query",
		},
		Output: "query_result",
	}

	_, err := plugin.ExecuteStep(ctx, queryStepConfig, *pipelines.NewPluginContext())
	// This should return an error for non-existent collection
	assert.Error(t, err)

	// Test getting non-existent document
	getStepConfig := pipelines.StepConfig{
		Name:   "Get Non-existent Document",
		Plugin: "Storage.vector",
		Config: map[string]interface{}{
			"operation":  "get",
			"collection": "test_collection",
			"id":         "non_existent_doc",
		},
		Output: "document",
	}

	_, err = plugin.ExecuteStep(ctx, getStepConfig, *pipelines.NewPluginContext())
	assert.Error(t, err)

	// Test invalid operation
	invalidStepConfig := pipelines.StepConfig{
		Name:   "Invalid Operation",
		Plugin: "Storage.vector",
		Config: map[string]interface{}{
			"operation": "invalid_operation",
		},
		Output: "result",
	}

	_, err = plugin.ExecuteStep(ctx, invalidStepConfig, *pipelines.NewPluginContext())
	assert.Error(t, err)
}

func TestEmbeddingServiceIntegration(t *testing.T) {
	ctx := context.Background()

	// Test OpenAI embedding service (requires API key)
	if os.Getenv("OPENAI_API_KEY") != "" {
		t.Run("OpenAI Embedding Service", func(t *testing.T) {
			testOpenAIEmbeddingService(t, ctx)
		})
	}

	// Test Ollama embedding service (if available)
	t.Run("Ollama Embedding Service", func(t *testing.T) {
		testOllamaEmbeddingService(t, ctx)
	})

	// Test text processing utilities
	t.Run("Text Processing", func(t *testing.T) {
		testTextProcessing(t)
	})
}

func testOpenAIEmbeddingService(t *testing.T, ctx context.Context) {
	config := EmbeddingConfig{
		Provider:   ProviderOpenAI,
		APIKey:     os.Getenv("OPENAI_API_KEY"),
		Model:      "text-embedding-3-small",
		Dimensions: 1536,
	}

	factory := NewEmbeddingServiceFactory()
	service, err := factory.CreateService(config)
	require.NoError(t, err)

	// Test single text embedding
	text := "This is a test document for embedding"
	embedding, err := service.EmbedText(ctx, text)
	require.NoError(t, err)
	assert.Len(t, embedding, 1536)

	// Test multiple text embedding
	texts := []string{
		"First test document",
		"Second test document",
		"Third test document",
	}
	embeddings, err := service.EmbedTexts(ctx, texts)
	require.NoError(t, err)
	assert.Len(t, embeddings, 3)
	assert.Len(t, embeddings[0], 1536)

	// Test embedding validation
	err = ValidateEmbedding(embedding)
	assert.NoError(t, err)

	// Test embedding normalization
	normalized := NormalizeEmbedding(embedding)
	assert.Len(t, normalized, 1536)
}

func testOllamaEmbeddingService(t *testing.T, ctx context.Context) {
	config := EmbeddingConfig{
		Provider:   ProviderOllama,
		BaseURL:    "http://localhost:11434",
		Model:      "nomic-embed-text",
		Dimensions: 768,
	}

	factory := NewEmbeddingServiceFactory()
	service, err := factory.CreateService(config)
	require.NoError(t, err)

	// Test single text embedding
	text := "This is a test document for Ollama embedding"
	embedding, err := service.EmbedText(ctx, text)
	if err != nil {
		t.Skip("Ollama not available, skipping test")
		return
	}

	assert.Len(t, embedding, 768)
}

func testTextProcessing(t *testing.T) {
	processor := NewTextProcessor()

	// Test text preprocessing
	input := "  This is   a test   document!  "
	expected := "This is a test document!"
	result := processor.PreprocessText(input)
	assert.Equal(t, expected, result)

	// Test text chunking
	longText := "This is a very long document that should be split into smaller chunks for better embedding performance and to handle token limits in various embedding models."
	chunks := processor.ChunkText(longText, 50)
	assert.True(t, len(chunks) > 1)

	for _, chunk := range chunks {
		assert.True(t, len(chunk) <= 50)
	}
}

func TestBackendFactoryIntegration(t *testing.T) {
	factory := NewBackendFactory()

	// Test configuration validation
	validConfig := &StorageConfig{
		Type: BackendTypeChromem,
		ChromemConfig: &ChromemConfig{
			CollectionName: "test",
		},
	}

	err := factory.ValidateConfig(validConfig)
	assert.NoError(t, err)

	// Test invalid configuration
	invalidConfig := &StorageConfig{
		Type:           BackendTypePinecone,
		PineconeConfig: &PineconeConfig{
			// Missing required fields
		},
	}

	err = factory.ValidateConfig(invalidConfig)
	assert.Error(t, err)

	// Test default configuration creation
	defaultConfig := factory.GetDefaultConfig(BackendTypeChromem)
	assert.Equal(t, BackendTypeChromem, defaultConfig.Type)
	assert.NotNil(t, defaultConfig.ChromemConfig)
}

func BenchmarkStorageOperations(b *testing.B) {
	ctx := context.Background()

	// Setup
	factory := NewBackendFactory()
	config := &StorageConfig{
		Type: BackendTypeChromem,
		ChromemConfig: &ChromemConfig{
			CollectionName:   "benchmark_collection",
			SimilarityMetric: CosineSimilarity,
			EmbeddingFunc:    chromem.NewEmbeddingFuncDefault(),
		},
	}

	backend, _ := factory.CreateBackend(config)
	defer backend.Close()

	plugin := NewStoragePlugin(backend)

	// Benchmark document storage
	b.Run("StoreDocuments", func(b *testing.B) {
		documents := []Document{
			{
				ID:      "bench_doc",
				Content: "This is a benchmark document for performance testing",
				Metadata: map[string]interface{}{
					"type": "benchmark",
				},
			},
		}

		stepConfig := pipelines.StepConfig{
			Name:   "Benchmark Store",
			Plugin: "Storage.vector",
			Config: map[string]interface{}{
				"operation":  "store",
				"collection": "benchmark_collection",
				"documents":  documents,
			},
			Output: "store_result",
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = plugin.ExecuteStep(ctx, stepConfig, *pipelines.NewPluginContext())
		}
	})

	// Benchmark document querying
	b.Run("QueryDocuments", func(b *testing.B) {
		stepConfig := pipelines.StepConfig{
			Name:   "Benchmark Query",
			Plugin: "Storage.vector",
			Config: map[string]interface{}{
				"operation":  "query",
				"collection": "benchmark_collection",
				"query":      "benchmark document",
				"limit":      1,
			},
			Output: "query_result",
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = plugin.ExecuteStep(ctx, stepConfig, *pipelines.NewPluginContext())
		}
	})
}
