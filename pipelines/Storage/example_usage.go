package storage

import (
	"context"
	"fmt"
	"log"

	"github.com/Mimir-AIP/Mimir-AIP-Go/pipelines"
	"github.com/philippgille/chromem-go"
)

// ExampleUsage demonstrates how to use the storage system
func ExampleUsage() {
	ctx := context.Background()

	// 1. Create a storage backend using the factory
	factory := NewBackendFactory()

	// Configure for Chromem backend with persistence
	config := &StorageConfig{
		Type: BackendTypeChromem,
		ChromemConfig: &ChromemConfig{
			CollectionName:      "knowledge_base",
			SimilarityMetric:    CosineSimilarity,
			EmbeddingFunc:       chromem.NewEmbeddingFuncDefault(), // Requires OPENAI_API_KEY env var
			EnablePersistence:   true,
			PersistencePath:     "./data/knowledge_base.db",
			EnableCompression:   true,
			BackupEncryptionKey: "", // Set this for encrypted backups
		},
	}

	// Validate configuration
	if err := factory.ValidateConfig(config); err != nil {
		log.Fatalf("Invalid configuration: %v", err)
	}

	// Create the backend
	backend, err := factory.CreateBackend(config)
	if err != nil {
		log.Fatalf("Failed to create backend: %v", err)
	}
	defer backend.Close()

	// 2. Create a storage plugin
	storagePlugin := NewStoragePlugin(backend)

	// 3. Example: Store documents
	fmt.Println("=== Storing Documents ===")

	// Create some sample documents
	documents := []Document{
		{
			ID:      "doc1",
			Content: "The sky is blue because of Rayleigh scattering",
			Metadata: map[string]interface{}{
				"source": "science",
				"topic":  "physics",
			},
		},
		{
			ID:      "doc2",
			Content: "Photosynthesis is the process by which plants make food",
			Metadata: map[string]interface{}{
				"source": "biology",
				"topic":  "plants",
			},
		},
		{
			ID:      "doc3",
			Content: "Machine learning is a subset of artificial intelligence",
			Metadata: map[string]interface{}{
				"source": "technology",
				"topic":  "AI",
			},
		},
	}

	// Store documents using the plugin
	stepConfig := pipelines.StepConfig{
		Name:   "Store Knowledge",
		Plugin: "Storage.vector",
		Config: map[string]interface{}{
			"operation":  "store",
			"collection": "knowledge_base",
			"documents":  documents,
		},
		Output: "store_result",
	}

	_, err = storagePlugin.ExecuteStep(ctx, stepConfig, pipelines.NewPluginContext())
	if err != nil {
		log.Fatalf("Failed to store documents: %v", err)
	}

	fmt.Printf("Stored %d documents successfully\n", len(documents))

	// 4. Example: Query documents
	fmt.Println("\n=== Querying Documents ===")

	queryStepConfig := pipelines.StepConfig{
		Name:   "Query Knowledge",
		Plugin: "Storage.vector",
		Config: map[string]interface{}{
			"operation":  "query",
			"collection": "knowledge_base",
			"query":      "How do plants make food?",
			"limit":      2,
		},
		Output: "query_result",
	}

	queryResult, err := storagePlugin.ExecuteStep(ctx, queryStepConfig, pipelines.NewPluginContext())
	if err != nil {
		log.Fatalf("Failed to query documents: %v", err)
	}

	// Extract query results
	if queryData, exists := queryResult.Get("query_result"); exists {
		if queryMap, ok := queryData.(map[string]interface{}); ok {
			if results, ok := queryMap["results"].([]interface{}); ok {
				fmt.Printf("Found %d results:\n", len(results))
				for i, result := range results {
					if resultMap, ok := result.(map[string]interface{}); ok {
						fmt.Printf("%d. %s (score: %.3f)\n",
							i+1,
							resultMap["content"],
							resultMap["score"])
					}
				}
			}
		}
	}

	// 5. Example: Get specific document
	fmt.Println("\n=== Getting Specific Document ===")

	getStepConfig := pipelines.StepConfig{
		Name:   "Get Document",
		Plugin: "Storage.vector",
		Config: map[string]interface{}{
			"operation":  "get",
			"collection": "knowledge_base",
			"id":         "doc2",
		},
		Output: "document",
	}

	getResult, err := storagePlugin.ExecuteStep(ctx, getStepConfig, pipelines.NewPluginContext())
	if err != nil {
		log.Fatalf("Failed to get document: %v", err)
	}

	if docData, exists := getResult.Get("document"); exists {
		if docMap, ok := docData.(map[string]interface{}); ok {
			fmt.Printf("Retrieved document: %s\n", docMap["content"])
		}
	}

	// 6. Example: List collections
	fmt.Println("\n=== Listing Collections ===")

	listStepConfig := pipelines.StepConfig{
		Name:   "List Collections",
		Plugin: "Storage.vector",
		Config: map[string]interface{}{
			"operation": "list_collections",
		},
		Output: "collections",
	}

	listResult, err := storagePlugin.ExecuteStep(ctx, listStepConfig, pipelines.NewPluginContext())
	if err != nil {
		log.Fatalf("Failed to list collections: %v", err)
	}

	if collections, exists := listResult.Get("collections"); exists {
		if colSlice, ok := collections.([]string); ok {
			fmt.Printf("Available collections: %v\n", colSlice)
		}
	}

	fmt.Println("\n=== Storage Example Complete ===")
}

// ExamplePipelineUsage shows how storage can be used in a complete pipeline
func ExamplePipelineUsage() {
	fmt.Println("=== Example Pipeline with Storage ===")

	// This would be defined in a YAML pipeline file
	pipelineConfig := `
name: "Knowledge Pipeline"
steps:
  - name: "Collect Data"
    plugin: "Input.api"
    config:
      url: "https://api.example.com/data"
      method: "GET"
    output: "raw_data"

  - name: "Process Data"
    plugin: "Data_Processing.transform"
    config:
      operation: "extract_text"
    output: "processed_data"

  - name: "Store Knowledge"
    plugin: "Storage.vector"
    config:
      operation: "store"
      collection: "knowledge_base"
    output: "store_result"

  - name: "Query Knowledge"
    plugin: "Storage.vector"
    config:
      operation: "query"
      collection: "knowledge_base"
      query: "What is the meaning of life?"
      limit: 3
    output: "insights"

  - name: "Generate Report"
    plugin: "Output.json"
    config:
      filename: "knowledge_report.json"
      pretty: true
    output: "report"
`

	fmt.Println("Pipeline configuration:")
	fmt.Println(pipelineConfig)

	fmt.Println("\nThis pipeline demonstrates:")
	fmt.Println("1. Collecting data from external sources")
	fmt.Println("2. Processing and transforming the data")
	fmt.Println("3. Storing the processed data in vector storage")
	fmt.Println("4. Querying the stored knowledge for insights")
	fmt.Println("5. Generating a report with the results")
}

// ExamplePersistenceAndPerformance demonstrates persistence and performance features
func ExamplePersistenceAndPerformance() {
	fmt.Println("=== Persistence and Performance Example ===")

	ctx := context.Background()

	// Create backend with persistence enabled
	factory := NewBackendFactory()
	config := &StorageConfig{
		Type: BackendTypeChromem,
		ChromemConfig: &ChromemConfig{
			CollectionName:    "performance_test",
			SimilarityMetric:  CosineSimilarity,
			EmbeddingFunc:     CreateMockEmbeddingFunc(384), // Use mock for testing
			EnablePersistence: true,
			PersistencePath:   "./data/performance_test.db",
			EnableCompression: true,
		},
	}

	backend, err := factory.CreateBackend(config)
	if err != nil {
		log.Fatalf("Failed to create backend: %v", err)
	}
	defer backend.Close()

	// Create storage plugin
	plugin := NewStoragePlugin(backend)

	// Store some test documents
	documents := []Document{
		{
			ID:      "perf_doc_1",
			Content: "This is a performance test document",
			Metadata: map[string]interface{}{
				"type": "performance_test",
			},
		},
		{
			ID:      "perf_doc_2",
			Content: "Another document for performance testing",
			Metadata: map[string]interface{}{
				"type": "performance_test",
			},
		},
	}

	// Convert documents for plugin
	docMaps := make([]map[string]interface{}, len(documents))
	for i, doc := range documents {
		docMaps[i] = map[string]interface{}{
			"id":       doc.ID,
			"content":  doc.Content,
			"metadata": doc.Metadata,
		}
	}

	stepConfig := pipelines.StepConfig{
		Name:   "Store Performance Test Documents",
		Plugin: "Storage.vector",
		Config: map[string]interface{}{
			"operation":  "store",
			"collection": "performance_test",
			"documents":  docMaps,
		},
		Output: "store_result",
	}

	_, err = plugin.ExecuteStep(ctx, stepConfig, pipelines.NewPluginContext())
	if err != nil {
		log.Fatalf("Failed to store documents: %v", err)
	}

	fmt.Println("✅ Documents stored successfully with persistence enabled")

	// Demonstrate backup functionality
	if chromemBackend, ok := backend.(*ChromemBackend); ok {
		// Create backup
		backupPath := "./data/backup/performance_test_backup.db"
		err = chromemBackend.Backup(backupPath)
		if err != nil {
			log.Printf("Failed to create backup: %v", err)
		} else {
			fmt.Printf("✅ Backup created at: %s\n", backupPath)
		}

		// Get performance stats
		stats := chromemBackend.GetPerformanceStats()
		fmt.Printf("📊 Performance Stats: %+v\n", stats)

		// Optimize collection
		err = chromemBackend.OptimizeCollection("performance_test")
		if err != nil {
			log.Printf("Failed to optimize collection: %v", err)
		} else {
			fmt.Println("✅ Collection optimized for performance")
		}
	}

	fmt.Println("\n=== Persistence Features ===")
	fmt.Println("• Automatic persistence to disk")
	fmt.Println("• Gzip compression enabled")
	fmt.Println("• Backup and restore capabilities")
	fmt.Println("• Optimized for concurrent operations")
	fmt.Println("• SIMD acceleration (when available)")

	fmt.Println("\n=== Performance Optimizations ===")
	fmt.Println("• Concurrent document processing")
	fmt.Println("• Efficient vector similarity search")
	fmt.Println("• Memory-optimized storage")
	fmt.Println("• Fast collection switching")
}

// ExampleBackendSwitching shows how to switch between different backends
func ExampleBackendSwitching() {
	fmt.Println("=== Backend Switching Example ===")

	// Example configurations for different backends
	configs := []struct {
		name   string
		config *StorageConfig
	}{
		{
			name: "Chromem (Local)",
			config: &StorageConfig{
				Type: BackendTypeChromem,
				ChromemConfig: &ChromemConfig{
					CollectionName: "local_kb",
					EmbeddingFunc:  chromem.NewEmbeddingFuncDefault(),
				},
			},
		},
		{
			name: "Pinecone (Cloud)",
			config: &StorageConfig{
				Type: BackendTypePinecone,
				PineconeConfig: &PineconeConfig{
					APIKey:      "${PINECONE_API_KEY}",
					Environment: "us-east-1",
					IndexName:   "cloud_kb",
				},
			},
		},
		{
			name: "Weaviate (Cloud/Local)",
			config: &StorageConfig{
				Type: BackendTypeWeaviate,
				WeaviateConfig: &WeaviateConfig{
					URL:       "http://localhost:8080",
					ClassName: "KnowledgeBase",
				},
			},
		},
	}

	factory := NewBackendFactory()

	for _, example := range configs {
		fmt.Printf("\n%s Configuration:\n", example.name)
		fmt.Printf("  Type: %s\n", example.config.Type)

		// Note: Actual backend creation would require proper setup
		// (API keys, running services, etc.)
		if err := factory.ValidateConfig(example.config); err != nil {
			fmt.Printf("  Status: Configuration invalid - %v\n", err)
		} else {
			fmt.Printf("  Status: Configuration valid\n")
		}
	}

	fmt.Println("\nTo switch backends:")
	fmt.Println("1. Update the StorageConfig.Type field")
	fmt.Println("2. Provide the appropriate backend-specific configuration")
	fmt.Println("3. Restart the application or reload configuration")
	fmt.Println("4. All existing pipeline code continues to work unchanged!")
}
