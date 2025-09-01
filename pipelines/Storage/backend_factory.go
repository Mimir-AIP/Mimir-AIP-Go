package storage

import (
	"fmt"

	"github.com/philippgille/chromem-go"
)

// BackendType represents the type of storage backend
type BackendType string

const (
	BackendTypeChromem  BackendType = "chromem"
	BackendTypePinecone BackendType = "pinecone"
	BackendTypeWeaviate BackendType = "weaviate"
	BackendTypeQdrant   BackendType = "qdrant"
)

// StorageConfig represents the configuration for storage backends
type StorageConfig struct {
	Type           BackendType `json:"type"`
	DefaultBackend BackendType `json:"default_backend,omitempty"`

	// Chromem-specific config
	ChromemConfig *ChromemConfig `json:"chromem,omitempty"`

	// Pinecone-specific config
	PineconeConfig *PineconeConfig `json:"pinecone,omitempty"`

	// Weaviate-specific config
	WeaviateConfig *WeaviateConfig `json:"weaviate,omitempty"`

	// Qdrant-specific config
	QdrantConfig *QdrantConfig `json:"qdrant,omitempty"`
}

// ChromemConfig holds configuration for ChromemBackend
type ChromemConfig struct {
	PersistencePath     string                `json:"persistence_path,omitempty"`
	BackupPath          string                `json:"backup_path,omitempty"`
	EnablePersistence   bool                  `json:"enable_persistence"`
	EnableCompression   bool                  `json:"enable_compression"`
	BackupEncryptionKey string                `json:"backup_encryption_key,omitempty"`
	EmbeddingFunc       chromem.EmbeddingFunc `json:"-"` // Function for embedding, not serializable
	CollectionName      string                `json:"collection_name"`
	SimilarityMetric    SimilarityMetric      `json:"similarity_metric"`
}

// PineconeConfig holds configuration for Pinecone backend
type PineconeConfig struct {
	APIKey      string `json:"api_key"`
	Environment string `json:"environment"`
	ProjectID   string `json:"project_id"`
	IndexName   string `json:"index_name"`
}

// WeaviateConfig holds configuration for Weaviate backend
type WeaviateConfig struct {
	URL       string `json:"url"`
	APIKey    string `json:"api_key,omitempty"`
	ClassName string `json:"class_name"`
}

// QdrantConfig holds configuration for Qdrant backend
type QdrantConfig struct {
	URL        string `json:"url"`
	APIKey     string `json:"api_key,omitempty"`
	Collection string `json:"collection"`
}

// BackendFactory creates storage backends based on configuration
type BackendFactory struct{}

// NewBackendFactory creates a new BackendFactory
func NewBackendFactory() *BackendFactory {
	return &BackendFactory{}
}

// CreateBackend creates a storage backend based on the configuration
func (f *BackendFactory) CreateBackend(config *StorageConfig) (StorageBackend, error) {
	if config == nil {
		return nil, fmt.Errorf("storage config cannot be nil")
	}

	backendType := config.Type
	if backendType == "" {
		backendType = config.DefaultBackend
	}
	if backendType == "" {
		backendType = BackendTypeChromem // Default to Chromem
	}

	switch backendType {
	case BackendTypeChromem:
		return f.createChromemBackend(config.ChromemConfig)
	case BackendTypePinecone:
		return f.createPineconeBackend(config.PineconeConfig)
	case BackendTypeWeaviate:
		return f.createWeaviateBackend(config.WeaviateConfig)
	case BackendTypeQdrant:
		return f.createQdrantBackend(config.QdrantConfig)
	default:
		return nil, fmt.Errorf("unknown storage backend type: %s", backendType)
	}
}

// createChromemBackend creates a ChromemBackend instance
func (f *BackendFactory) createChromemBackend(config *ChromemConfig) (*ChromemBackend, error) {
	if config == nil {
		config = &ChromemConfig{
			CollectionName:   "default",
			SimilarityMetric: CosineSimilarity,
		}
	}

	// If no embedding function is provided, use default (OpenAI)
	if config.EmbeddingFunc == nil {
		config.EmbeddingFunc = chromem.NewEmbeddingFuncDefault()
	}

	// Set default values for persistence
	if config.EnablePersistence && config.PersistencePath == "" {
		config.PersistencePath = "./data/chromem.db"
	}

	return NewChromemBackend(config)
}

// createPineconeBackend creates a PineconeBackend instance
func (f *BackendFactory) createPineconeBackend(config *PineconeConfig) (StorageBackend, error) {
	if config == nil {
		return nil, fmt.Errorf("Pinecone config cannot be nil")
	}

	// TODO: Implement PineconeBackend
	return nil, fmt.Errorf("Pinecone backend not yet implemented")
}

// createWeaviateBackend creates a WeaviateBackend instance
func (f *BackendFactory) createWeaviateBackend(config *WeaviateConfig) (StorageBackend, error) {
	if config == nil {
		return nil, fmt.Errorf("Weaviate config cannot be nil")
	}

	// TODO: Implement WeaviateBackend
	return nil, fmt.Errorf("Weaviate backend not yet implemented")
}

// createQdrantBackend creates a QdrantBackend instance
func (f *BackendFactory) createQdrantBackend(config *QdrantConfig) (StorageBackend, error) {
	if config == nil {
		return nil, fmt.Errorf("Qdrant config cannot be nil")
	}

	// TODO: Implement QdrantBackend
	return nil, fmt.Errorf("Qdrant backend not yet implemented")
}

// ValidateConfig validates the storage configuration
func (f *BackendFactory) ValidateConfig(config *StorageConfig) error {
	if config == nil {
		return fmt.Errorf("storage config cannot be nil")
	}

	backendType := config.Type
	if backendType == "" {
		backendType = config.DefaultBackend
	}

	switch backendType {
	case BackendTypeChromem:
		return f.validateChromemConfig(config.ChromemConfig)
	case BackendTypePinecone:
		return f.validatePineconeConfig(config.PineconeConfig)
	case BackendTypeWeaviate:
		return f.validateWeaviateConfig(config.WeaviateConfig)
	case BackendTypeQdrant:
		return f.validateQdrantConfig(config.QdrantConfig)
	case "":
		return fmt.Errorf("no backend type specified")
	default:
		return fmt.Errorf("unknown storage backend type: %s", backendType)
	}
}

// validateChromemConfig validates Chromem-specific configuration
func (f *BackendFactory) validateChromemConfig(config *ChromemConfig) error {
	if config != nil && config.CollectionName == "" {
		return fmt.Errorf("collection name cannot be empty for Chromem backend")
	}
	return nil
}

// validatePineconeConfig validates Pinecone-specific configuration
func (f *BackendFactory) validatePineconeConfig(config *PineconeConfig) error {
	if config == nil {
		return fmt.Errorf("Pinecone config cannot be nil")
	}
	if config.APIKey == "" {
		return fmt.Errorf("API key is required for Pinecone backend")
	}
	if config.IndexName == "" {
		return fmt.Errorf("index name is required for Pinecone backend")
	}
	return nil
}

// validateWeaviateConfig validates Weaviate-specific configuration
func (f *BackendFactory) validateWeaviateConfig(config *WeaviateConfig) error {
	if config == nil {
		return fmt.Errorf("Weaviate config cannot be nil")
	}
	if config.URL == "" {
		return fmt.Errorf("URL is required for Weaviate backend")
	}
	if config.ClassName == "" {
		return fmt.Errorf("class name is required for Weaviate backend")
	}
	return nil
}

// validateQdrantConfig validates Qdrant-specific configuration
func (f *BackendFactory) validateQdrantConfig(config *QdrantConfig) error {
	if config == nil {
		return fmt.Errorf("Qdrant config cannot be nil")
	}
	if config.URL == "" {
		return fmt.Errorf("URL is required for Qdrant backend")
	}
	if config.Collection == "" {
		return fmt.Errorf("collection is required for Qdrant backend")
	}
	return nil
}

// GetSupportedBackends returns a list of supported backend types
func (f *BackendFactory) GetSupportedBackends() []BackendType {
	return []BackendType{
		BackendTypeChromem,
		BackendTypePinecone,
		BackendTypeWeaviate,
		BackendTypeQdrant,
	}
}

// GetDefaultConfig returns a default configuration for the specified backend type
func (f *BackendFactory) GetDefaultConfig(backendType BackendType) *StorageConfig {
	config := &StorageConfig{
		Type: backendType,
	}

	switch backendType {
	case BackendTypeChromem:
		config.ChromemConfig = &ChromemConfig{
			CollectionName:   "default",
			SimilarityMetric: CosineSimilarity,
		}
	case BackendTypePinecone:
		config.PineconeConfig = &PineconeConfig{
			Environment: "us-east-1",
		}
	case BackendTypeWeaviate:
		config.WeaviateConfig = &WeaviateConfig{
			URL: "http://localhost:8080",
		}
	case BackendTypeQdrant:
		config.QdrantConfig = &QdrantConfig{
			URL: "http://localhost:6333",
		}
	}

	return config
}
