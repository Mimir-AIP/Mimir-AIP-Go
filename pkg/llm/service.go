package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/mimir-aip/mimir-aip-go/pkg/metadatastore"
	"github.com/mimir-aip/mimir-aip-go/pkg/models"
)

const (
	modelCacheTTL  = 1 * time.Hour
	labelBatchSize = 50
)

// Service wraps a Provider with a TTL model cache and graceful degradation.
// A nil *Service is safe to use — all methods check the receiver.
type Service struct {
	provider     Provider
	enabled      bool
	defaultModel string

	mu           sync.RWMutex
	cachedModels []Model
	cacheExpiry  time.Time

	// registry maps provider names to Provider implementations (built-in + external).
	registry map[string]Provider
	// store persists external provider metadata (nil = no persistence).
	store metadatastore.MetadataStore
	// loader compiles and loads external provider .so files (nil = no dynamic loading).
	loader *Loader
}

// NewService creates a Service backed by provider.  When enabled is false the
// service is present but all operations are no-ops.
func NewService(provider Provider, defaultModel string, enabled bool) *Service {
	return &Service{
		provider:     provider,
		enabled:      enabled,
		defaultModel: defaultModel,
		registry:     make(map[string]Provider),
	}
}

// WithStore attaches a MetadataStore for external provider persistence.
func (s *Service) WithStore(store metadatastore.MetadataStore) *Service {
	s.store = store
	return s
}

// WithLoader attaches a Loader for dynamic provider compilation.
func (s *Service) WithLoader(loader *Loader) *Service {
	s.loader = loader
	return s
}

// SetActiveProvider switches the service to the given provider without rebuilding
// the registry, store, or loader attachments.
func (s *Service) SetActiveProvider(p Provider, defaultModel string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.provider = p
	s.defaultModel = defaultModel
	s.enabled = true
	// Invalidate model cache so the new provider's models are fetched fresh.
	s.cachedModels = nil
	s.cacheExpiry = time.Time{}
}

// RegisterProvider adds a named provider to the in-memory registry.
func (s *Service) RegisterProvider(name string, p Provider) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.registry[name] = p
}

// GetProvider retrieves a provider from the registry by name.
func (s *Service) GetProvider(name string) (Provider, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	p, ok := s.registry[name]
	if !ok {
		return nil, fmt.Errorf("llm: provider %q not found in registry", name)
	}
	return p, nil
}

// ListRegisteredProviders returns the names of all registered providers.
func (s *Service) ListRegisteredProviders() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	names := make([]string, 0, len(s.registry))
	for name := range s.registry {
		names = append(names, name)
	}
	return names
}

// InstallExternalProvider clones, compiles, and registers an LLM provider from
// a Git repository.  The metadata is persisted so it survives restarts.
func (s *Service) InstallExternalProvider(req *models.ExternalLLMProviderInstallRequest) (*models.ExternalLLMProvider, error) {
	if s.loader == nil {
		return nil, fmt.Errorf("dynamic LLM provider loading is not configured")
	}
	if req.RepositoryURL == "" {
		return nil, fmt.Errorf("repository_url is required")
	}

	gitRef := req.GitRef
	if gitRef == "" {
		gitRef = "main"
	}

	name := repoName(req.RepositoryURL)

	p, commitHash, err := s.loader.CompileAndLoad(name, req.RepositoryURL, gitRef, "")

	now := time.Now().UTC()
	record := &models.ExternalLLMProvider{
		Name:          name,
		RepositoryURL: req.RepositoryURL,
		GitCommitHash: commitHash,
		Status:        "active",
		InstalledAt:   now,
		UpdatedAt:     now,
	}

	if err != nil {
		record.Status = "error"
		record.ErrorMessage = err.Error()
		if s.store != nil {
			_ = s.store.SaveExternalLLMProvider(record)
		}
		return record, fmt.Errorf("failed to compile LLM provider %s: %w", name, err)
	}

	s.RegisterProvider(name, p)

	if s.store != nil {
		if saveErr := s.store.SaveExternalLLMProvider(record); saveErr != nil {
			return nil, fmt.Errorf("provider compiled but failed to persist metadata: %w", saveErr)
		}
	}

	log.Printf("Installed external LLM provider: %s @ %s", name, commitHash)
	return record, nil
}

// ListExternalProviders returns all persisted external LLM provider records.
func (s *Service) ListExternalProviders() ([]*models.ExternalLLMProvider, error) {
	if s.store == nil {
		return []*models.ExternalLLMProvider{}, nil
	}
	return s.store.ListExternalLLMProviders()
}

// GetExternalProvider returns metadata for a single external LLM provider.
func (s *Service) GetExternalProvider(name string) (*models.ExternalLLMProvider, error) {
	if s.store == nil {
		return nil, fmt.Errorf("external LLM provider not found: %s", name)
	}
	return s.store.GetExternalLLMProvider(name)
}

// UninstallExternalProvider removes the provider from the live registry and
// deletes its persisted metadata.  The .so file is removed from the cache.
// Note: Go plugins cannot be unloaded from memory; a process restart is needed
// for removal to take full effect on in-flight calls.
func (s *Service) UninstallExternalProvider(name string) error {
	s.mu.Lock()
	delete(s.registry, name)
	s.mu.Unlock()

	if s.loader != nil {
		_ = os.Remove(s.loader.soPath(name))
		_ = os.Remove(s.loader.metaPath(name))
	}

	if s.store != nil {
		if err := s.store.DeleteExternalLLMProvider(name); err != nil {
			return fmt.Errorf("failed to delete external LLM provider record: %w", err)
		}
	}

	log.Printf("Uninstalled external LLM provider: %s", name)
	return nil
}

// LoadInstalledExternalProviders re-registers all persisted external LLM
// providers on startup.  If a cached .so exists it is used directly; otherwise
// the provider is recompiled from its recorded repository URL and commit hash.
func (s *Service) LoadInstalledExternalProviders() error {
	if s.loader == nil || s.store == nil {
		return nil
	}

	records, err := s.store.ListExternalLLMProviders()
	if err != nil {
		return fmt.Errorf("failed to list persisted LLM providers: %w", err)
	}

	var loadErrs []string
	for _, rec := range records {
		if rec.Status == "error" {
			log.Printf("Skipping external LLM provider %s (status=error)", rec.Name)
			continue
		}

		p, loadErr := s.loader.LoadCached(rec.Name)
		if loadErr != nil {
			log.Printf("Recompiling external LLM provider %s from %s @ %s", rec.Name, rec.RepositoryURL, rec.GitCommitHash)
			var compileHash string
			p, compileHash, loadErr = s.loader.CompileAndLoad(
				rec.Name, rec.RepositoryURL, rec.GitCommitHash, rec.GitCommitHash,
			)
			if loadErr != nil {
				loadErrs = append(loadErrs, fmt.Sprintf("%s: %v", rec.Name, loadErr))
				rec.Status = "error"
				rec.ErrorMessage = loadErr.Error()
				_ = s.store.SaveExternalLLMProvider(rec)
				continue
			}
			_ = compileHash
		}

		s.RegisterProvider(rec.Name, p)
		log.Printf("Loaded external LLM provider: %s", rec.Name)
	}

	if len(loadErrs) > 0 {
		return fmt.Errorf("failed to load %d external LLM provider(s): %s", len(loadErrs), strings.Join(loadErrs, "; "))
	}
	return nil
}

// IsEnabled returns true when the service is configured and enabled.
// Safe to call on a nil receiver.
func (s *Service) IsEnabled() bool {
	if s == nil {
		return false
	}
	return s.enabled && s.provider != nil
}

// ProviderName returns the provider name, or "none" when not enabled.
func (s *Service) ProviderName() string {
	if !s.IsEnabled() {
		return "none"
	}
	return s.provider.Name()
}

// ListModels returns the cached model list, refreshing it when the TTL has
// expired.  Uses double-checked locking: fast read path avoids write lock.
func (s *Service) ListModels(ctx context.Context) ([]Model, error) {
	if !s.IsEnabled() {
		return nil, nil
	}

	// Fast path — check under read lock.
	s.mu.RLock()
	if time.Now().Before(s.cacheExpiry) && s.cachedModels != nil {
		models := s.cachedModels
		s.mu.RUnlock()
		return models, nil
	}
	s.mu.RUnlock()

	// Slow path — fetch and cache under write lock.
	s.mu.Lock()
	defer s.mu.Unlock()

	// Double-check: another goroutine may have refreshed while we waited.
	if time.Now().Before(s.cacheExpiry) && s.cachedModels != nil {
		return s.cachedModels, nil
	}

	models, err := s.provider.ListModels(ctx)
	if err != nil {
		return nil, fmt.Errorf("llm: list models: %w", err)
	}

	s.cachedModels = models
	s.cacheExpiry = time.Now().Add(modelCacheTTL)
	return models, nil
}

// LabelEntityTypes assigns a PascalCase entity type to each name in names by
// batching them into groups of labelBatchSize and calling the LLM.
// It NEVER returns an error — failures are logged and a partial/empty map is
// returned so the caller can always proceed with heuristic behaviour.
func (s *Service) LabelEntityTypes(ctx context.Context, names []string, sourceName string, contextCols []string) map[string]string {
	result := make(map[string]string, len(names))
	if !s.IsEnabled() || len(names) == 0 {
		return result
	}

	for i := 0; i < len(names); i += labelBatchSize {
		end := i + labelBatchSize
		if end > len(names) {
			end = len(names)
		}
		batch := names[i:end]

		labels, err := s.labelBatch(ctx, batch, sourceName, contextCols)
		if err != nil {
			log.Printf("llm: LabelEntityTypes batch %d-%d failed: %v", i, end, err)
			continue
		}
		for k, v := range labels {
			result[k] = v
		}
	}
	return result
}

// labelBatch calls the LLM for a single batch of names and returns the parsed
// entity-type map.
func (s *Service) labelBatch(ctx context.Context, names []string, sourceName string, contextCols []string) (map[string]string, error) {
	// Build the JSON array of names for the prompt.
	nameJSON, err := json.Marshal(names)
	if err != nil {
		return nil, fmt.Errorf("marshal names: %w", err)
	}

	system := `You are an ontology engineer building a knowledge graph.
Assign a concise PascalCase entity type to each entity name.
Respond ONLY with valid JSON: {"entity_name": "EntityType", ...}.
No explanations, no markdown, just the raw JSON object.`

	colStr := strings.Join(contextCols, ", ")
	user := fmt.Sprintf("Source: %s\nContext columns: %s\nEntities: %s",
		sourceName, colStr, string(nameJSON))

	resp, err := s.provider.Complete(ctx, CompletionRequest{
		Model: s.defaultModel,
		Messages: []Message{
			{Role: "system", Content: system},
			{Role: "user", Content: user},
		},
		MaxTokens:   512,
		Temperature: 0.0,
	})
	if err != nil {
		return nil, fmt.Errorf("complete: %w", err)
	}

	raw := strings.TrimSpace(resp.Content)

	// Strip markdown code fences if present (```json ... ``` or ``` ... ```).
	if strings.HasPrefix(raw, "```") {
		// Remove the opening fence line.
		if idx := strings.Index(raw, "\n"); idx != -1 {
			raw = raw[idx+1:]
		}
		// Remove the closing fence.
		if idx := strings.LastIndex(raw, "```"); idx != -1 {
			raw = raw[:idx]
		}
		raw = strings.TrimSpace(raw)
	}

	var labels map[string]string
	if err := json.Unmarshal([]byte(raw), &labels); err != nil {
		return nil, fmt.Errorf("parse response JSON %q: %w", raw, err)
	}
	return labels, nil
}
