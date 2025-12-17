package storage

import (
	"context"
	"time"
)

// EmbeddingFunc is a function that creates embeddings for a given text
type EmbeddingFunc func(ctx context.Context, text string) ([]float32, error)

// StorageBackend defines the interface for any vector storage system
// This abstraction allows different storage backends (local, cloud, etc.) to be used interchangeably
type StorageBackend interface {
	// Core vector operations
	Store(ctx context.Context, collection string, documents []Document, embeddings [][]float32) error
	Query(ctx context.Context, collection string, queryVector []float32, limit int, filters map[string]any) ([]QueryResult, error)
	Delete(ctx context.Context, collection string, ids []string) error

	// Collection management
	CreateCollection(ctx context.Context, name string, config CollectionConfig) error
	DeleteCollection(ctx context.Context, name string) error
	ListCollections(ctx context.Context) ([]string, error)

	// Document operations
	GetDocument(ctx context.Context, collection string, id string) (*Document, error)
	UpdateMetadata(ctx context.Context, collection string, id string, metadata map[string]any) error

	// Health and status
	Health(ctx context.Context) error
	Stats(ctx context.Context) (StorageStats, error)

	// Cleanup
	Close() error
}

// Document represents a document to be stored with its metadata
type Document struct {
	ID        string         `json:"id"`
	Content   string         `json:"content"`
	Metadata  map[string]any `json:"metadata,omitempty"`
	Embedding []float32      `json:"embedding,omitempty"` // Optional: pre-computed embedding
}

// QueryResult represents the result of a vector similarity query
type QueryResult struct {
	Document Document       `json:"document"`
	Score    float32        `json:"score"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

// CollectionConfig defines configuration for a vector collection
type CollectionConfig struct {
	Name        string           `json:"name"`
	Dimension   int              `json:"dimension"`
	Metric      SimilarityMetric `json:"metric"`
	IndexType   string           `json:"index_type,omitempty"`
	Persistence bool             `json:"persistence"`
}

// SimilarityMetric defines the distance metric for vector similarity
type SimilarityMetric string

const (
	CosineSimilarity  SimilarityMetric = "cosine"
	EuclideanDistance SimilarityMetric = "euclidean"
	DotProduct        SimilarityMetric = "dot_product"
	ManhattanDistance SimilarityMetric = "manhattan"
)

// StorageStats provides statistics about the storage backend
type StorageStats struct {
	TotalCollections int           `json:"total_collections"`
	TotalDocuments   int           `json:"total_documents"`
	TotalSizeBytes   int64         `json:"total_size_bytes"`
	Uptime           time.Duration `json:"uptime"`
	BackendType      string        `json:"backend_type"`
}

// BatchOperation represents a batch operation for multiple documents
type BatchOperation struct {
	Collection string
	Documents  []Document
	Embeddings [][]float32
	Operation  BatchOpType
}

// BatchOpType defines the type of batch operation
type BatchOpType string

const (
	BatchStore  BatchOpType = "store"
	BatchDelete BatchOpType = "delete"
	BatchUpdate BatchOpType = "update"
)

// BatchResult represents the result of a batch operation
type BatchResult struct {
	Successful int          `json:"successful"`
	Failed     int          `json:"failed"`
	Errors     []BatchError `json:"errors,omitempty"`
}

// BatchError represents an error in a batch operation
type BatchError struct {
	Index   int    `json:"index"`
	ID      string `json:"id"`
	Message string `json:"message"`
}
