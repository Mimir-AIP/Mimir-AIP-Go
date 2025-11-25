# Mimir AIP Storage System

The Storage system provides a modular, backend-agnostic vector storage solution for Mimir AIP. It allows you to store and query vector embeddings with support for multiple storage backends.

## Architecture

The storage system is designed with modularity in mind:

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Storage       │    │  StoragePlugin  │    │  Storage       │
│   Plugins       │────│  Interface      │────│  Backend       │
│                 │    │                 │    │  Abstraction   │
│ • store         │    │ • Store()       │    │                │
│ • query         │    │ • Query()       │    │ • Store()      │
│ • batch_store   │    │ • BatchStore()  │    │ • Query()      │
└─────────────────┘    └─────────────────┘    └─────────────────┘
                                                       │
                    ┌──────────────────────────────────┼──────────────────────────────────┐
                    │                                  │                                  │
          ┌─────────▼─────────┐              ┌─────────▼─────────┐              ┌─────────▼─────────┐
          │  ChromemBackend  │              │ PineconeBackend   │              │ WeaviateBackend  │
          │                  │              │                    │              │                  │
          │ • Chromem-Go     │              │ • Pinecone API    │              │ • Weaviate API   │
          │ • Local storage  │              │ • Cloud storage   │              │ • Cloud/Local    │
          └──────────────────┘              └────────────────────┘              └──────────────────┘
```

## Supported Backends

### Chromem (Default)
- **Type**: Local vector database
- **Library**: [chromem-go](https://github.com/philippgille/chromem-go)
- **Features**: In-memory with optional persistence, cosine similarity
- **Use Case**: Development, small to medium datasets

### Pinecone (Cloud)
- **Type**: Managed cloud vector database
- **Features**: Scalable, high-performance, managed service
- **Use Case**: Production, large datasets

### Weaviate (Cloud/Local)
- **Type**: Open-source vector database
- **Features**: GraphQL API, hybrid search, cloud or self-hosted
- **Use Case**: Flexible deployment options

### Qdrant (Cloud/Local)
- **Type**: Open-source vector database
- **Features**: High-performance, filtering, cloud or self-hosted
- **Use Case**: High-performance search requirements

## Quick Start

### 1. Basic Setup

```go
package main

import (
    "context"
    "log"

    "github.com/Mimir-AIP/Mimir-AIP-Go/pipelines/Storage"
    "github.com/philippgille/chromem-go"
)

func main() {
    // Create storage configuration
    config := &storage.StorageConfig{
        Type: storage.BackendTypeChromem,
        ChromemConfig: &storage.ChromemConfig{
            CollectionName:   "my_collection",
            SimilarityMetric: storage.CosineSimilarity,
            EmbeddingFunc:    chromem.NewEmbeddingFuncDefault(),
        },
    }

    // Create backend factory
    factory := storage.NewBackendFactory()

    // Validate configuration
    if err := factory.ValidateConfig(config); err != nil {
        log.Fatalf("Invalid config: %v", err)
    }

    // Create backend
    backend, err := factory.CreateBackend(config)
    if err != nil {
        log.Fatalf("Failed to create backend: %v", err)
    }
    defer backend.Close()

    // Create storage plugin
    plugin := storage.NewStoragePlugin(backend)

    // Use in your application...
}
```

### 2. Using in Pipelines

```yaml
# pipeline.yaml
name: "Knowledge Pipeline"
steps:
  - name: "Collect Data"
    plugin: "Input.api"
    config:
      url: "https://api.example.com/data"
    output: "raw_data"

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
      query: "What is machine learning?"
      limit: 5
    output: "insights"
```

### 3. Configuration

```yaml
# config.yaml
storage:
  type: "chromem"  # or "pinecone", "weaviate", "qdrant"
  chromem:
    collection_name: "knowledge_base"
    similarity_metric: "cosine"
    persistence_path: "./data"
  pinecone:
    api_key: "${PINECONE_API_KEY}"
    environment: "us-east-1"
    index_name: "my-index"
  weaviate:
    url: "http://localhost:8080"
    class_name: "Document"
```

## Storage Operations

### Store Documents

```go
documents := []storage.Document{
    {
        ID:      "doc1",
        Content: "The sky is blue because of Rayleigh scattering",
        Metadata: map[string]interface{}{
            "source": "science",
            "topic":  "physics",
        },
    },
}

stepConfig := pipelines.StepConfig{
    Name:   "Store Documents",
    Plugin: "Storage.vector",
    Config: map[string]interface{}{
        "operation":  "store",
        "collection": "knowledge_base",
        "documents":  documents,
    },
    Output: "store_result",
}

result, err := storagePlugin.ExecuteStep(ctx, stepConfig, *pipelines.NewPluginContext())
```

### Query Documents

```go
stepConfig := pipelines.StepConfig{
    Name:   "Query Documents",
    Plugin: "Storage.vector",
    Config: map[string]interface{}{
        "operation":  "query",
        "collection": "knowledge_base",
        "query":      "How does photosynthesis work?",
        "limit":      3,
    },
    Output: "query_result",
}

result, err := storagePlugin.ExecuteStep(ctx, stepConfig, *pipelines.NewPluginContext())
```

### Delete Documents

```go
stepConfig := pipelines.StepConfig{
    Name:   "Delete Documents",
    Plugin: "Storage.vector",
    Config: map[string]interface{}{
        "operation":  "delete",
        "collection": "knowledge_base",
        "ids":        []string{"doc1", "doc2"},
    },
    Output: "delete_result",
}
```

### Get Single Document

```go
stepConfig := pipelines.StepConfig{
    Name:   "Get Document",
    Plugin: "Storage.vector",
    Config: map[string]interface{}{
        "operation":  "get",
        "collection": "knowledge_base",
        "id":         "doc1",
    },
    Output: "document",
}
```

## Backend Switching

### Switching from Chromem to Pinecone

1. **Update Configuration**:
```yaml
storage:
  type: "pinecone"
  pinecone:
    api_key: "${PINECONE_API_KEY}"
    environment: "us-east-1"
    index_name: "my-knowledge-base"
```

2. **No Code Changes Required**:
```go
// This code works with any backend
config := &storage.StorageConfig{
    Type: storage.BackendTypePinecone, // Changed from BackendTypeChromem
    PineconeConfig: &storage.PineconeConfig{
        APIKey:      os.Getenv("PINECONE_API_KEY"),
        Environment: "us-east-1",
        IndexName:   "my-knowledge-base",
    },
}
```

3. **Pipeline Compatibility**:
All existing pipelines continue to work without modification because they use the `Storage.vector` plugin interface.

## Advanced Features

### Custom Embedding Functions

```go
// Use Ollama for local embeddings
config.ChromemConfig.EmbeddingFunc = chromem.NewEmbeddingFuncOllama("nomic-embed-text", "")

// Use OpenAI with custom model
config.ChromemConfig.EmbeddingFunc = chromem.NewEmbeddingFuncOpenAI("your-api-key", chromem.EmbeddingModelOpenAI3Large)
```

### Batch Operations

```go
stepConfig := pipelines.StepConfig{
    Name:   "Batch Store",
    Plugin: "Storage.vector",
    Config: map[string]interface{}{
        "operation": "batch_store",
        "collection": "knowledge_base",
        "documents": largeDocumentArray,
    },
}
```

### Collection Management

```go
// Create collection
stepConfig := pipelines.StepConfig{
    Name:   "Create Collection",
    Plugin: "Storage.vector",
    Config: map[string]interface{}{
        "operation": "create_collection",
        "name":      "new_collection",
    },
}

// List collections
stepConfig := pipelines.StepConfig{
    Name:   "List Collections",
    Plugin: "Storage.vector",
    Config: map[string]interface{}{
        "operation": "list_collections",
    },
}
```

## Performance Considerations

### Chromem Backend
- **Best for**: Development, datasets < 100K documents
- **Memory**: In-memory storage (optional persistence)
- **Performance**: Fast queries, low latency
- **Scaling**: Single instance, no clustering

### Pinecone Backend
- **Best for**: Production, large datasets
- **Memory**: Cloud-managed
- **Performance**: High throughput, low latency
- **Scaling**: Automatic scaling, multi-region

### Choosing a Backend

| Criteria | Chromem | Pinecone | Weaviate | Qdrant |
|----------|---------|----------|----------|--------|
| Setup Complexity | Low | Medium | Medium | Medium |
| Scaling | Limited | Excellent | Good | Good |
| Cost | Free | Paid | Free/Cloud | Free/Cloud |
| Local Development | Excellent | Poor | Good | Good |
| Production Ready | Small projects | Large projects | Medium projects | Medium projects |

## Error Handling

The storage system provides comprehensive error handling:

```go
result, err := storagePlugin.ExecuteStep(ctx, stepConfig, *pipelines.NewPluginContext())
if err != nil {
    // Handle storage errors
    log.Printf("Storage operation failed: %v", err)
    return
}

// Check operation-specific results
if storeResult, exists := result.Get("store_result"); exists {
    if resultMap, ok := storeResult.(map[string]interface{}); ok {
        if success, ok := resultMap["successful"].(bool); ok && !success {
            log.Printf("Storage operation was not successful")
        }
    }
}
```

## Migration Guide

### From No Storage to Storage

1. Add storage dependency: `go get github.com/philippgille/chromem-go`
2. Create storage configuration in your config file
3. Add storage steps to your pipelines
4. Update your application to initialize storage backend

### Between Backends

1. Export data from current backend (if supported)
2. Update configuration to new backend type
3. Import data to new backend (if supported)
4. Update any backend-specific configurations
5. Test with a subset of data first

## Troubleshooting

### Common Issues

1. **"Embedding function required"**
   - Ensure you have set an embedding function in your configuration
   - For Chromem: `EmbeddingFunc: chromem.NewEmbeddingFuncDefault()`

2. **"Collection not found"**
   - Collections are created automatically on first use
   - Check collection name spelling in your configuration

3. **"API key required"**
   - For cloud backends, ensure API keys are set in environment variables
   - Check your configuration matches the backend requirements

4. **Memory issues with large datasets**
   - Consider switching to a cloud backend for large datasets
   - Use batch operations for large data imports

### Debug Mode

Enable debug logging to troubleshoot issues:

```go
// Set log level to debug
os.Setenv("MIMIR_LOG_LEVEL", "debug")

// Check backend health
if err := backend.Health(ctx); err != nil {
    log.Printf("Backend health check failed: %v", err)
}

// Get backend statistics
if stats, err := backend.Stats(ctx); err == nil {
    log.Printf("Backend stats: %+v", stats)
}
```

## Contributing

### Adding New Backends

1. Implement the `StorageBackend` interface
2. Add backend type constant
3. Update `BackendFactory.CreateBackend()`
4. Add configuration struct
5. Update validation logic
6. Add documentation and examples

### Testing

```bash
# Run storage tests
go test ./pipelines/Storage/...

# Run with race detection
go test -race ./pipelines/Storage/...

# Run benchmarks
go test -bench=. ./pipelines/Storage/...
```

## License

This storage system is part of Mimir AIP and follows the same license terms.