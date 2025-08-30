package storage

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/philippgille/chromem-go"
)

// EmbeddingProvider represents different embedding providers
type EmbeddingProvider string

const (
	ProviderOpenAI  EmbeddingProvider = "openai"
	ProviderOllama  EmbeddingProvider = "ollama"
	ProviderLocalAI EmbeddingProvider = "localai"
	ProviderCohere  EmbeddingProvider = "cohere"
	ProviderJina    EmbeddingProvider = "jina"
	ProviderMistral EmbeddingProvider = "mistral"
	ProviderVertex  EmbeddingProvider = "vertex"
)

// EmbeddingConfig holds configuration for embedding services
type EmbeddingConfig struct {
	Provider   EmbeddingProvider `json:"provider"`
	APIKey     string            `json:"api_key,omitempty"`
	BaseURL    string            `json:"base_url,omitempty"`
	Model      string            `json:"model,omitempty"`
	Dimensions int               `json:"dimensions,omitempty"`
}

// EmbeddingService provides a unified interface for different embedding providers
type EmbeddingService interface {
	// EmbedText creates an embedding for a single text
	EmbedText(ctx context.Context, text string) ([]float32, error)

	// EmbedTexts creates embeddings for multiple texts
	EmbedTexts(ctx context.Context, texts []string) ([][]float32, error)

	// GetDimensions returns the dimensionality of embeddings
	GetDimensions() int

	// GetProvider returns the provider name
	GetProvider() EmbeddingProvider

	// ToChromemFunc converts to chromem-go EmbeddingFunc
	ToChromemFunc() chromem.EmbeddingFunc
}

// OpenAIEmbeddingService implements EmbeddingService for OpenAI
type OpenAIEmbeddingService struct {
	config EmbeddingConfig
}

// NewOpenAIEmbeddingService creates a new OpenAI embedding service
func NewOpenAIEmbeddingService(config EmbeddingConfig) *OpenAIEmbeddingService {
	if config.APIKey == "" {
		config.APIKey = os.Getenv("OPENAI_API_KEY")
	}
	if config.Model == "" {
		config.Model = "text-embedding-3-small"
	}
	if config.Dimensions == 0 {
		config.Dimensions = 1536 // Default for text-embedding-3-small
	}

	return &OpenAIEmbeddingService{config: config}
}

func (s *OpenAIEmbeddingService) EmbedText(ctx context.Context, text string) ([]float32, error) {
	embeddings, err := s.EmbedTexts(ctx, []string{text})
	if err != nil {
		return nil, err
	}
	return embeddings[0], nil
}

func (s *OpenAIEmbeddingService) EmbedTexts(ctx context.Context, texts []string) ([][]float32, error) {
	// Use chromem-go's OpenAI embedding function
	var model chromem.EmbeddingModelOpenAI
	switch s.config.Model {
	case "text-embedding-ada-002":
		model = chromem.EmbeddingModelOpenAI2Ada
	case "text-embedding-3-large":
		model = chromem.EmbeddingModelOpenAI3Large
	default:
		model = chromem.EmbeddingModelOpenAI3Small
	}

	embeddingFunc := chromem.NewEmbeddingFuncOpenAI(s.config.APIKey, model)

	// Process each text individually
	embeddings := make([][]float32, len(texts))
	for i, text := range texts {
		embedding, err := embeddingFunc(ctx, text)
		if err != nil {
			return nil, fmt.Errorf("failed to embed text %d: %w", i, err)
		}
		embeddings[i] = embedding
	}

	return embeddings, nil
}

func (s *OpenAIEmbeddingService) GetDimensions() int {
	return s.config.Dimensions
}

func (s *OpenAIEmbeddingService) GetProvider() EmbeddingProvider {
	return ProviderOpenAI
}

func (s *OpenAIEmbeddingService) ToChromemFunc() chromem.EmbeddingFunc {
	var model chromem.EmbeddingModelOpenAI
	switch s.config.Model {
	case "text-embedding-ada-002":
		model = chromem.EmbeddingModelOpenAI2Ada
	case "text-embedding-3-large":
		model = chromem.EmbeddingModelOpenAI3Large
	default:
		model = chromem.EmbeddingModelOpenAI3Small
	}
	return chromem.NewEmbeddingFuncOpenAI(s.config.APIKey, model)
}

// OllamaEmbeddingService implements EmbeddingService for Ollama
type OllamaEmbeddingService struct {
	config EmbeddingConfig
}

// NewOllamaEmbeddingService creates a new Ollama embedding service
func NewOllamaEmbeddingService(config EmbeddingConfig) *OllamaEmbeddingService {
	if config.BaseURL == "" {
		config.BaseURL = "http://localhost:11434"
	}
	if config.Model == "" {
		config.Model = "nomic-embed-text"
	}
	if config.Dimensions == 0 {
		config.Dimensions = 768 // Default for nomic-embed-text
	}

	return &OllamaEmbeddingService{config: config}
}

func (s *OllamaEmbeddingService) EmbedText(ctx context.Context, text string) ([]float32, error) {
	embeddings, err := s.EmbedTexts(ctx, []string{text})
	if err != nil {
		return nil, err
	}
	return embeddings[0], nil
}

func (s *OllamaEmbeddingService) EmbedTexts(ctx context.Context, texts []string) ([][]float32, error) {
	embeddingFunc := chromem.NewEmbeddingFuncOllama(s.config.Model, s.config.BaseURL)

	// Process each text individually
	embeddings := make([][]float32, len(texts))
	for i, text := range texts {
		embedding, err := embeddingFunc(ctx, text)
		if err != nil {
			return nil, fmt.Errorf("failed to embed text %d: %w", i, err)
		}
		embeddings[i] = embedding
	}

	return embeddings, nil
}

func (s *OllamaEmbeddingService) GetDimensions() int {
	return s.config.Dimensions
}

func (s *OllamaEmbeddingService) GetProvider() EmbeddingProvider {
	return ProviderOllama
}

func (s *OllamaEmbeddingService) ToChromemFunc() chromem.EmbeddingFunc {
	return chromem.NewEmbeddingFuncOllama(s.config.Model, s.config.BaseURL)
}

// LocalAIEmbeddingService implements EmbeddingService for LocalAI
type LocalAIEmbeddingService struct {
	config EmbeddingConfig
}

// NewLocalAIEmbeddingService creates a new LocalAI embedding service
func NewLocalAIEmbeddingService(config EmbeddingConfig) *LocalAIEmbeddingService {
	if config.Model == "" {
		config.Model = "bert-cpp"
	}
	if config.Dimensions == 0 {
		config.Dimensions = 384 // Default for bert-cpp
	}

	return &LocalAIEmbeddingService{config: config}
}

func (s *LocalAIEmbeddingService) EmbedText(ctx context.Context, text string) ([]float32, error) {
	embeddings, err := s.EmbedTexts(ctx, []string{text})
	if err != nil {
		return nil, err
	}
	return embeddings[0], nil
}

func (s *LocalAIEmbeddingService) EmbedTexts(ctx context.Context, texts []string) ([][]float32, error) {
	embeddingFunc := chromem.NewEmbeddingFuncLocalAI(s.config.Model)

	// Process each text individually
	embeddings := make([][]float32, len(texts))
	for i, text := range texts {
		embedding, err := embeddingFunc(ctx, text)
		if err != nil {
			return nil, fmt.Errorf("failed to embed text %d: %w", i, err)
		}
		embeddings[i] = embedding
	}

	return embeddings, nil
}

func (s *LocalAIEmbeddingService) GetDimensions() int {
	return s.config.Dimensions
}

func (s *LocalAIEmbeddingService) GetProvider() EmbeddingProvider {
	return ProviderLocalAI
}

func (s *LocalAIEmbeddingService) ToChromemFunc() chromem.EmbeddingFunc {
	return chromem.NewEmbeddingFuncLocalAI(s.config.Model)
}

// EmbeddingServiceFactory creates embedding services based on configuration
type EmbeddingServiceFactory struct{}

// NewEmbeddingServiceFactory creates a new factory
func NewEmbeddingServiceFactory() *EmbeddingServiceFactory {
	return &EmbeddingServiceFactory{}
}

// CreateService creates an embedding service based on configuration
func (f *EmbeddingServiceFactory) CreateService(config EmbeddingConfig) (EmbeddingService, error) {
	switch config.Provider {
	case ProviderOpenAI:
		return NewOpenAIEmbeddingService(config), nil
	case ProviderOllama:
		return NewOllamaEmbeddingService(config), nil
	case ProviderLocalAI:
		return NewLocalAIEmbeddingService(config), nil
	case "mock":
		return NewMockEmbeddingService(config.Dimensions), nil
	case ProviderCohere:
		return nil, fmt.Errorf("Cohere provider not yet implemented")
	case ProviderJina:
		return nil, fmt.Errorf("Jina provider not yet implemented")
	case ProviderMistral:
		return nil, fmt.Errorf("Mistral provider not yet implemented")
	case ProviderVertex:
		return nil, fmt.Errorf("Vertex provider not yet implemented")
	default:
		return nil, fmt.Errorf("unknown embedding provider: %s", config.Provider)
	}
}

// CreateDefaultService creates a default embedding service
func (f *EmbeddingServiceFactory) CreateDefaultService() EmbeddingService {
	// For testing/development, use mock embedder by default
	return NewMockEmbeddingService(384)
}

// CreateProductionService creates a production-ready embedding service
func (f *EmbeddingServiceFactory) CreateProductionService() EmbeddingService {
	// Try OpenAI first (most common for production)
	if apiKey := os.Getenv("OPENAI_API_KEY"); apiKey != "" {
		return NewOpenAIEmbeddingService(EmbeddingConfig{
			Provider:   ProviderOpenAI,
			APIKey:     apiKey,
			Model:      "text-embedding-3-small",
			Dimensions: 1536,
		})
	}

	// Fall back to Ollama for local production
	return NewOllamaEmbeddingService(EmbeddingConfig{
		Provider:   ProviderOllama,
		BaseURL:    "http://localhost:11434",
		Model:      "nomic-embed-text",
		Dimensions: 768,
	})
}

// TextProcessor provides utilities for text preprocessing before embedding
type TextProcessor struct{}

// NewTextProcessor creates a new text processor
func NewTextProcessor() *TextProcessor {
	return &TextProcessor{}
}

// PreprocessText cleans and prepares text for embedding
func (tp *TextProcessor) PreprocessText(text string) string {
	// Remove extra whitespace
	text = strings.TrimSpace(text)

	// Replace multiple spaces with single space
	for strings.Contains(text, "  ") {
		text = strings.ReplaceAll(text, "  ", " ")
	}

	// Remove control characters
	text = strings.Map(func(r rune) rune {
		if r < 32 && r != 9 && r != 10 && r != 13 { // Keep tab, newline, carriage return
			return -1
		}
		return r
	}, text)

	return text
}

// ChunkText splits long text into smaller chunks for embedding
func (tp *TextProcessor) ChunkText(text string, maxChunkSize int) []string {
	if len(text) <= maxChunkSize {
		return []string{text}
	}

	var chunks []string
	words := strings.Fields(text)
	currentChunk := ""

	for _, word := range words {
		if len(currentChunk)+len(word)+1 > maxChunkSize && currentChunk != "" {
			chunks = append(chunks, currentChunk)
			currentChunk = word
		} else {
			if currentChunk != "" {
				currentChunk += " "
			}
			currentChunk += word
		}
	}

	if currentChunk != "" {
		chunks = append(chunks, currentChunk)
	}

	return chunks
}

// ValidateEmbedding checks if an embedding vector is valid
func ValidateEmbedding(embedding []float32) error {
	if len(embedding) == 0 {
		return fmt.Errorf("embedding is empty")
	}

	// Check for NaN or infinite values
	for i, val := range embedding {
		if fmt.Sprintf("%f", val) == "NaN" {
			return fmt.Errorf("embedding contains NaN at index %d", i)
		}
	}

	return nil
}

// NormalizeEmbedding normalizes an embedding vector to unit length
func NormalizeEmbedding(embedding []float32) []float32 {
	var sum float32
	for _, val := range embedding {
		sum += val * val
	}

	if sum == 0 {
		return embedding // Avoid division by zero
	}

	norm := sqrt(sum)
	normalized := make([]float32, len(embedding))
	for i, val := range embedding {
		normalized[i] = val / norm
	}

	return normalized
}

// sqrt calculates square root for float32
func sqrt(x float32) float32 {
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
