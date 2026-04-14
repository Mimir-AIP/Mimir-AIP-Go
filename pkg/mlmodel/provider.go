package mlmodel

import (
	"fmt"
	"sort"

	"github.com/mimir-aip/mimir-aip-go/pkg/models"
	"github.com/mimir-aip/mimir-aip-go/pkg/pluginruntime"
)

type ProviderTrainRequest struct {
	Model        *models.MLModel
	TrainingData interface{}
}

type ProviderTrainResult struct {
	ArtifactData       []byte
	PerformanceMetrics *models.PerformanceMetrics
	TrainingMetrics    *models.TrainingMetrics
	AdditionalMetadata map[string]any
}

type ProviderInferRequest struct {
	Model *models.MLModel
	Input map[string]any
}

type ProviderInferResult struct {
	Output     any
	Confidence float64
	Metadata   map[string]any
}

type Provider interface {
	Metadata() models.MLProviderMetadata
	ValidateModel(model *models.MLModel) error
	Train(req *ProviderTrainRequest) (*ProviderTrainResult, error)
	Infer(req *ProviderInferRequest) (*ProviderInferResult, error)
}

type providerFactory func() Provider

type ProviderRegistry struct {
	providers *pluginruntime.Registry[providerFactory]
}

func NewProviderRegistry() *ProviderRegistry {
	return &ProviderRegistry{providers: pluginruntime.NewRegistry[providerFactory]()}
}

func (r *ProviderRegistry) Register(name string, provider Provider) {
	if provider == nil {
		return
	}
	r.providers.Register(name, func() Provider {
		return provider
	})
}

func (r *ProviderRegistry) Get(name string) (Provider, bool) {
	factory, ok := r.providers.Get(name)
	if !ok || factory == nil {
		return nil, false
	}
	provider := factory()
	if provider == nil {
		return nil, false
	}
	return provider, true
}

func (r *ProviderRegistry) Names() []string {
	names := r.providers.Names()
	sort.Strings(names)
	return names
}

func normalizeProviderIdentity(model *models.MLModel) (string, string, error) {
	if model == nil {
		return "", "", fmt.Errorf("model is required")
	}
	provider := model.Provider
	providerModel := model.ProviderModel
	if provider == "" {
		provider = "builtin"
	}
	if provider == "builtin" && providerModel == "" && model.Type != "" {
		providerModel = string(model.Type)
	}
	if providerModel == "" {
		return "", "", fmt.Errorf("provider_model is required")
	}
	return provider, providerModel, nil
}
