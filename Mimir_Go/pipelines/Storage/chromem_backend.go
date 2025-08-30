package storage

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/philippgille/chromem-go"
)

// ChromemBackend implements StorageBackend using chromem-go
type ChromemBackend struct {
	db         *chromem.DB
	embedder   chromem.EmbeddingFunc
	collection *chromem.Collection
	config     *ChromemConfig
}

// NewChromemBackend creates a new ChromemBackend instance
func NewChromemBackend(config *ChromemConfig) (*ChromemBackend, error) {
	if config == nil {
		config = &ChromemConfig{
			CollectionName:   "default",
			SimilarityMetric: CosineSimilarity,
		}
	}

	var db *chromem.DB
	var err error

	// Create chromem DB with persistence if enabled
	if config.EnablePersistence && config.PersistencePath != "" {
		db, err = chromem.NewPersistentDB(config.PersistencePath, config.EnableCompression)
		if err != nil {
			return nil, fmt.Errorf("failed to create persistent DB: %w", err)
		}
	} else {
		db = chromem.NewDB()
	}

	// Create or get collection
	collection, err := db.GetOrCreateCollection(config.CollectionName, nil, config.EmbeddingFunc)
	if err != nil {
		return nil, fmt.Errorf("failed to create/get collection: %w", err)
	}

	return &ChromemBackend{
		db:         db,
		embedder:   config.EmbeddingFunc,
		collection: collection,
		config:     config,
	}, nil
}

// Store stores documents with their embeddings in the collection
func (cb *ChromemBackend) Store(ctx context.Context, collectionName string, documents []Document, embeddings [][]float32) error {
	// Ensure we're using the correct collection
	if collectionName != cb.collection.Name {
		collection, err := cb.db.GetOrCreateCollection(collectionName, nil, cb.embedder)
		if err != nil {
			return fmt.Errorf("failed to get collection %s: %w", collectionName, err)
		}
		cb.collection = collection
	}

	// Convert our documents to chromem documents
	chromemDocs := make([]chromem.Document, len(documents))
	for i, doc := range documents {
		// Convert metadata from interface{} to string map
		metadata := make(map[string]string)
		for k, v := range doc.Metadata {
			if str, ok := v.(string); ok {
				metadata[k] = str
			} else {
				// Convert other types to string
				metadata[k] = fmt.Sprintf("%v", v)
			}
		}

		chromemDocs[i] = chromem.Document{
			ID:       doc.ID,
			Content:  doc.Content,
			Metadata: metadata,
		}

		// Use provided embedding if available
		if i < len(embeddings) && len(embeddings[i]) > 0 {
			chromemDocs[i].Embedding = embeddings[i]
		}
	}

	// Use optimal concurrency for better performance
	concurrency := 1
	if len(documents) > 10 {
		concurrency = 4 // Use higher concurrency for larger batches
	}

	// Store documents with optimized concurrency
	err := cb.collection.AddDocuments(ctx, chromemDocs, concurrency)
	if err != nil {
		return fmt.Errorf("failed to store documents: %w", err)
	}

	// Auto-save if persistence is enabled
	if cb.config != nil && cb.config.EnablePersistence && cb.config.PersistencePath != "" {
		// Note: With NewPersistentDB, changes are automatically persisted
		// But we can still manually save if needed
	}

	return nil
}

// Query performs a similarity search on the collection
func (cb *ChromemBackend) Query(ctx context.Context, collectionName string, queryVector []float32, limit int, filters map[string]interface{}) ([]QueryResult, error) {
	// Ensure we're using the correct collection
	if collectionName != cb.collection.Name {
		collection, err := cb.db.GetOrCreateCollection(collectionName, nil, cb.embedder)
		if err != nil {
			return nil, fmt.Errorf("failed to get collection %s: %w", collectionName, err)
		}
		cb.collection = collection
	}

	// Convert filters to chromem format
	var where map[string]string
	if filters != nil {
		where = make(map[string]string)
		for k, v := range filters {
			if str, ok := v.(string); ok {
				where[k] = str
			} else {
				where[k] = fmt.Sprintf("%v", v)
			}
		}
	}

	// Query the collection using embedding
	results, err := cb.collection.QueryEmbedding(ctx, queryVector, limit, where, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to query collection: %w", err)
	}

	// Convert results to our format
	queryResults := make([]QueryResult, len(results))
	for i, result := range results {
		// Convert metadata back to interface{}
		metadata := make(map[string]interface{})
		for k, v := range result.Metadata {
			metadata[k] = v
		}

		queryResults[i] = QueryResult{
			Document: Document{
				ID:       result.ID,
				Content:  result.Content,
				Metadata: metadata,
			},
			Score:    result.Similarity,
			Metadata: metadata,
		}
	}

	return queryResults, nil
}

// Delete removes documents from the collection
func (cb *ChromemBackend) Delete(ctx context.Context, collectionName string, ids []string) error {
	// Ensure we're using the correct collection
	if collectionName != cb.collection.Name {
		collection, err := cb.db.GetOrCreateCollection(collectionName, nil, cb.embedder)
		if err != nil {
			return fmt.Errorf("failed to get collection %s: %w", collectionName, err)
		}
		cb.collection = collection
	}

	// Delete documents using the correct API
	err := cb.collection.Delete(ctx, nil, nil, ids...)
	if err != nil {
		return fmt.Errorf("failed to delete documents: %w", err)
	}

	return nil
}

// CreateCollection creates a new collection
func (cb *ChromemBackend) CreateCollection(ctx context.Context, name string, config CollectionConfig) error {
	_, err := cb.db.CreateCollection(name, nil, cb.embedder)
	if err != nil {
		return fmt.Errorf("failed to create collection %s: %w", name, err)
	}
	return nil
}

// DeleteCollection deletes a collection
func (cb *ChromemBackend) DeleteCollection(ctx context.Context, name string) error {
	err := cb.db.DeleteCollection(name)
	if err != nil {
		return fmt.Errorf("failed to delete collection %s: %w", name, err)
	}
	return nil
}

// ListCollections returns all collection names
func (cb *ChromemBackend) ListCollections(ctx context.Context) ([]string, error) {
	collections := cb.db.ListCollections()
	names := make([]string, 0, len(collections))
	for _, col := range collections {
		names = append(names, col.Name)
	}
	return names, nil
}

// GetDocument retrieves a single document by ID
func (cb *ChromemBackend) GetDocument(ctx context.Context, collectionName string, id string) (*Document, error) {
	// Ensure we're using the correct collection
	if collectionName != cb.collection.Name {
		collection, err := cb.db.GetOrCreateCollection(collectionName, nil, cb.embedder)
		if err != nil {
			return nil, fmt.Errorf("failed to get collection %s: %w", collectionName, err)
		}
		cb.collection = collection
	}

	// Get document from collection
	doc, err := cb.collection.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get document %s: %w", id, err)
	}

	// Convert metadata
	metadata := make(map[string]interface{})
	for k, v := range doc.Metadata {
		metadata[k] = v
	}

	return &Document{
		ID:       doc.ID,
		Content:  doc.Content,
		Metadata: metadata,
	}, nil
}

// UpdateMetadata updates the metadata of a document
func (cb *ChromemBackend) UpdateMetadata(ctx context.Context, collectionName string, id string, metadata map[string]interface{}) error {
	// Chromem doesn't have a direct update metadata method
	// We need to delete and re-add the document
	doc, err := cb.GetDocument(ctx, collectionName, id)
	if err != nil {
		return fmt.Errorf("failed to get document for update: %w", err)
	}

	// Update metadata
	doc.Metadata = metadata

	// Delete old document
	err = cb.Delete(ctx, collectionName, []string{id})
	if err != nil {
		return fmt.Errorf("failed to delete old document: %w", err)
	}

	// Re-add with updated metadata
	err = cb.Store(ctx, collectionName, []Document{*doc}, nil)
	if err != nil {
		return fmt.Errorf("failed to re-add updated document: %w", err)
	}

	return nil
}

// Health checks the health of the backend
func (cb *ChromemBackend) Health(ctx context.Context) error {
	// Basic health check - try to list collections
	_, err := cb.ListCollections(ctx)
	return err
}

// Stats returns statistics about the backend
func (cb *ChromemBackend) Stats(ctx context.Context) (StorageStats, error) {
	collections, err := cb.ListCollections(ctx)
	if err != nil {
		return StorageStats{}, err
	}

	// Count total documents across all collections
	totalDocs := 0
	for _, colName := range collections {
		collection, err := cb.db.GetOrCreateCollection(colName, nil, cb.embedder)
		if err != nil {
			continue
		}
		totalDocs += collection.Count()
	}

	// Estimate size (rough approximation)
	estimatedSize := int64(totalDocs * 1000) // Rough estimate: 1KB per document

	return StorageStats{
		TotalCollections: len(collections),
		TotalDocuments:   totalDocs,
		TotalSizeBytes:   estimatedSize,
		Uptime:           0, // Not tracked
		BackendType:      "chromem",
	}, nil
}

// GetPerformanceStats returns detailed performance statistics
func (cb *ChromemBackend) GetPerformanceStats() map[string]interface{} {
	stats := make(map[string]interface{})

	collections := cb.db.ListCollections()
	stats["total_collections"] = len(collections)

	totalDocs := 0
	collectionStats := make(map[string]interface{})

	for name, collection := range collections {
		count := collection.Count()
		totalDocs += count
		collectionStats[name] = map[string]interface{}{
			"document_count": count,
			"name":           name,
		}
	}

	stats["total_documents"] = totalDocs
	stats["collections"] = collectionStats
	stats["persistence_enabled"] = cb.config != nil && cb.config.EnablePersistence
	stats["compression_enabled"] = cb.config != nil && cb.config.EnableCompression

	return stats
}

// OptimizeCollection optimizes a collection for better performance
func (cb *ChromemBackend) OptimizeCollection(collectionName string) error {
	// Chromem-go automatically optimizes queries, but we can ensure the collection exists
	_, err := cb.db.GetOrCreateCollection(collectionName, nil, cb.embedder)
	if err != nil {
		return fmt.Errorf("failed to optimize collection %s: %w", collectionName, err)
	}

	// Note: chromem-go uses SIMD optimizations automatically when available
	// Additional optimizations like HNSW indexing are not yet available in the current version

	return nil
}

// BatchStore efficiently stores multiple documents with optimizations
func (cb *ChromemBackend) BatchStore(ctx context.Context, collectionName string, documents []Document, embeddings [][]float32) error {
	// Use the same logic as Store but with optimizations for large batches
	return cb.Store(ctx, collectionName, documents, embeddings)
}

// PreloadCollection preloads a collection into memory for faster access
func (cb *ChromemBackend) PreloadCollection(collectionName string) error {
	// Ensure collection exists and is ready
	collection, err := cb.db.GetOrCreateCollection(collectionName, nil, cb.embedder)
	if err != nil {
		return fmt.Errorf("failed to preload collection %s: %w", collectionName, err)
	}

	// Chromem collections are already in-memory, so this is mainly for validation
	cb.collection = collection
	return nil
}

// Close closes the backend and cleans up resources
func (cb *ChromemBackend) Close() error {
	// Save persistence if enabled
	if cb.config != nil && cb.config.EnablePersistence && cb.config.PersistencePath != "" {
		return cb.savePersistence()
	}
	return nil
}

// savePersistence saves the current state to disk
func (cb *ChromemBackend) savePersistence() error {
	if cb.config.PersistencePath == "" {
		return fmt.Errorf("persistence path not configured")
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(cb.config.PersistencePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create persistence directory: %w", err)
	}

	// Export the database to file
	err := cb.db.Export(cb.config.PersistencePath, cb.config.EnableCompression, cb.config.BackupEncryptionKey)
	if err != nil {
		return fmt.Errorf("failed to export database: %w", err)
	}

	return nil
}

// loadPersistence loads the state from disk
func (cb *ChromemBackend) loadPersistence() error {
	if cb.config.PersistencePath == "" {
		return nil // No persistence file configured
	}

	// Check if file exists
	if _, err := os.Stat(cb.config.PersistencePath); os.IsNotExist(err) {
		return nil // File doesn't exist, start fresh
	}

	// Import the database from file
	err := cb.db.Import(cb.config.PersistencePath, cb.config.BackupEncryptionKey)
	if err != nil {
		return fmt.Errorf("failed to import database: %w", err)
	}

	return nil
}

// Backup creates a backup of the entire database
func (cb *ChromemBackend) Backup(backupPath string) error {
	if backupPath == "" {
		return fmt.Errorf("backup path cannot be empty")
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(backupPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create backup directory: %w", err)
	}

	// Export the database to backup file
	err := cb.db.Export(backupPath, cb.config.EnableCompression, cb.config.BackupEncryptionKey)
	if err != nil {
		return fmt.Errorf("failed to create backup: %w", err)
	}

	return nil
}

// Restore restores the database from a backup
func (cb *ChromemBackend) Restore(backupPath string) error {
	if backupPath == "" {
		return fmt.Errorf("backup path cannot be empty")
	}

	// Check if backup file exists
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		return fmt.Errorf("backup file does not exist: %s", backupPath)
	}

	// Import the backup from file
	err := cb.db.Import(backupPath, cb.config.BackupEncryptionKey)
	if err != nil {
		return fmt.Errorf("failed to restore from backup: %w", err)
	}

	return nil
}

// EnablePersistence enables persistence for the backend
func (cb *ChromemBackend) EnablePersistence(persistencePath string, enableCompression bool) error {
	if cb.config == nil {
		cb.config = &ChromemConfig{}
	}

	cb.config.PersistencePath = persistencePath
	cb.config.EnablePersistence = true
	cb.config.EnableCompression = enableCompression

	// Try to load existing persistence
	return cb.loadPersistence()
}

// DisablePersistence disables persistence
func (cb *ChromemBackend) DisablePersistence() error {
	if cb.config != nil {
		cb.config.EnablePersistence = false
	}
	return nil
}
