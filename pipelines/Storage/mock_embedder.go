package storage

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/philippgille/chromem-go"
)

// MockEmbedder provides a mock embedding function for testing
type MockEmbedder struct {
	dimensions int
	random     *rand.Rand
}

// NewMockEmbedder creates a new mock embedder with specified dimensions
func NewMockEmbedder(dimensions int) *MockEmbedder {
	return &MockEmbedder{
		dimensions: dimensions,
		random:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// EmbedText creates a mock embedding for a single text
func (me *MockEmbedder) EmbedText(ctx context.Context, text string) ([]float32, error) {
	return me.generateEmbedding(text), nil
}

// EmbedTexts creates mock embeddings for multiple texts
func (me *MockEmbedder) EmbedTexts(ctx context.Context, texts []string) ([][]float32, error) {
	embeddings := make([][]float32, len(texts))
	for i, text := range texts {
		embeddings[i] = me.generateEmbedding(text)
	}
	return embeddings, nil
}

// GetDimensions returns the dimensionality of embeddings
func (me *MockEmbedder) GetDimensions() int {
	return me.dimensions
}

// GetProvider returns the provider name
func (me *MockEmbedder) GetProvider() EmbeddingProvider {
	return "mock"
}

// ToChromemFunc converts to chromem-go EmbeddingFunc
func (me *MockEmbedder) ToChromemFunc() chromem.EmbeddingFunc {
	return func(ctx context.Context, text string) ([]float32, error) {
		return me.generateEmbedding(text), nil
	}
}

// generateEmbedding creates a deterministic but random-looking embedding based on text
func (me *MockEmbedder) generateEmbedding(text string) []float32 {
	embedding := make([]float32, me.dimensions)

	// Create a simple hash from the text to make embeddings deterministic
	hash := 0
	for _, char := range text {
		hash = hash*31 + int(char)
	}

	// Use the hash to seed the random number generator for deterministic results
	seededRand := rand.New(rand.NewSource(int64(hash)))

	// Generate embedding values
	for i := range embedding {
		// Create values between -1 and 1
		embedding[i] = float32(seededRand.Float64()*2 - 1)
	}

	// Normalize the embedding to unit length (required by chromem-go)
	return normalizeEmbedding(embedding)
}

// normalizeEmbedding normalizes a vector to unit length
func normalizeEmbedding(embedding []float32) []float32 {
	var sum float32
	for _, val := range embedding {
		sum += val * val
	}

	if sum == 0 {
		return embedding // Avoid division by zero
	}

	norm := sqrt32(sum)
	normalized := make([]float32, len(embedding))
	for i, val := range embedding {
		normalized[i] = val / norm
	}

	return normalized
}

// sqrt32 calculates square root for float32
func sqrt32(x float32) float32 {
	if x < 0 {
		return 0
	}

	// Simple approximation using Newton's method
	z := x / 2
	for i := 0; i < 10; i++ {
		z = (z + x/z) / 2
	}

	return z
}

// MockEmbeddingService implements EmbeddingService for testing
type MockEmbeddingService struct {
	embedder *MockEmbedder
}

// NewMockEmbeddingService creates a new mock embedding service
func NewMockEmbeddingService(dimensions int) *MockEmbeddingService {
	return &MockEmbeddingService{
		embedder: NewMockEmbedder(dimensions),
	}
}

func (mes *MockEmbeddingService) EmbedText(ctx context.Context, text string) ([]float32, error) {
	return mes.embedder.EmbedText(ctx, text)
}

func (mes *MockEmbeddingService) EmbedTexts(ctx context.Context, texts []string) ([][]float32, error) {
	return mes.embedder.EmbedTexts(ctx, texts)
}

func (mes *MockEmbeddingService) GetDimensions() int {
	return mes.embedder.GetDimensions()
}

func (mes *MockEmbeddingService) GetProvider() EmbeddingProvider {
	return "mock"
}

func (mes *MockEmbeddingService) ToChromemFunc() chromem.EmbeddingFunc {
	return mes.embedder.ToChromemFunc()
}

// CreateMockEmbeddingFunc creates a chromem-go compatible mock embedding function
func CreateMockEmbeddingFunc(dimensions int) chromem.EmbeddingFunc {
	mockEmbedder := NewMockEmbedder(dimensions)
	return mockEmbedder.ToChromemFunc()
}

// Example usage and testing functions

// TestMockEmbedder demonstrates how to use the mock embedder
func TestMockEmbedder() {
	fmt.Println("🧪 Testing Mock Embedder...")

	mockEmbedder := NewMockEmbedder(384) // Common embedding dimension

	ctx := context.Background()

	// Test single text embedding
	text := "This is a test document"
	embedding, err := mockEmbedder.EmbedText(ctx, text)
	if err != nil {
		fmt.Printf("❌ Failed to create embedding: %v\n", err)
		return
	}

	fmt.Printf("✅ Created embedding with %d dimensions\n", len(embedding))

	// Test multiple texts
	texts := []string{
		"First test document",
		"Second test document",
		"Third test document",
	}

	embeddings, err := mockEmbedder.EmbedTexts(ctx, texts)
	if err != nil {
		fmt.Printf("❌ Failed to create multiple embeddings: %v\n", err)
		return
	}

	fmt.Printf("✅ Created %d embeddings\n", len(embeddings))

	// Verify embeddings are normalized (unit length)
	for i, emb := range embeddings {
		var sum float32
		for _, val := range emb {
			sum += val * val
		}
		fmt.Printf("   Embedding %d magnitude: %.6f (should be ~1.0)\n", i, sum)
	}

	// Test deterministic behavior
	embedding1, _ := mockEmbedder.EmbedText(ctx, "test")
	embedding2, _ := mockEmbedder.EmbedText(ctx, "test")

	identical := true
	for i := range embedding1 {
		if embedding1[i] != embedding2[i] {
			identical = false
			break
		}
	}

	if identical {
		fmt.Println("✅ Embeddings are deterministic for same input")
	} else {
		fmt.Println("⚠️  Embeddings are not deterministic")
	}

	fmt.Println("🎉 Mock Embedder Test Complete!")
}

// CreateTestStorageConfig creates a test configuration with mock embedder
func CreateTestStorageConfig() *StorageConfig {
	return &StorageConfig{
		Type: BackendTypeChromem,
		ChromemConfig: &ChromemConfig{
			CollectionName:    "test_collection",
			SimilarityMetric:  CosineSimilarity,
			EmbeddingFunc:     CreateMockEmbeddingFunc(384),
			EnablePersistence: false, // Disable for tests to avoid file I/O
			EnableCompression: false,
		},
	}
}

// CreatePersistentTestStorageConfig creates a test configuration with persistence enabled
func CreatePersistentTestStorageConfig() *StorageConfig {
	return &StorageConfig{
		Type: BackendTypeChromem,
		ChromemConfig: &ChromemConfig{
			CollectionName:    "persistent_test_collection",
			SimilarityMetric:  CosineSimilarity,
			EmbeddingFunc:     CreateMockEmbeddingFunc(384),
			EnablePersistence: true,
			PersistencePath:   "./test_data/persistent_test.db",
			EnableCompression: true,
		},
	}
}

// CreateTestDocuments creates sample documents for testing
func CreateTestDocuments() []Document {
	return []Document{
		{
			ID:      "doc1",
			Content: "The sky is blue because of Rayleigh scattering",
			Metadata: map[string]interface{}{
				"category": "science",
				"topic":    "physics",
				"type":     "explanation",
			},
		},
		{
			ID:      "doc2",
			Content: "Photosynthesis is the process by which plants make food",
			Metadata: map[string]interface{}{
				"category": "biology",
				"topic":    "plants",
				"type":     "process",
			},
		},
		{
			ID:      "doc3",
			Content: "Machine learning algorithms learn from data",
			Metadata: map[string]interface{}{
				"category": "technology",
				"topic":    "AI",
				"type":     "definition",
			},
		},
	}
}
