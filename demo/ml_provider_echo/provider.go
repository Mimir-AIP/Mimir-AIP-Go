package main

import (
	"fmt"
	"sort"
	"strings"

	"github.com/mimir-aip/mimir-aip-go/pkg/mlmodel"
	"github.com/mimir-aip/mimir-aip-go/pkg/models"
)

type demoEchoProvider struct{}

func (p *demoEchoProvider) Metadata() models.MLProviderMetadata {
	return models.MLProviderMetadata{
		Name:               "demo-echo-provider",
		DisplayName:        "Demo Echo Provider",
		Description:        "Inference-only demo provider useful for validating provider loading and normalized outputs.",
		SupportsTraining:   false,
		SupportsInference:  true,
		SupportsMonitoring: false,
		Capabilities:       []models.MLProviderCapability{models.MLProviderCapabilityInfer, models.MLProviderCapabilityGenerate, models.MLProviderCapabilityClassify},
		Models: []models.MLProviderModel{
			{Name: "echo_classifier", DisplayName: "Echo Classifier", Capabilities: []models.MLProviderCapability{models.MLProviderCapabilityInfer, models.MLProviderCapabilityClassify}},
			{Name: "echo_generator", DisplayName: "Echo Generator", Capabilities: []models.MLProviderCapability{models.MLProviderCapabilityInfer, models.MLProviderCapabilityGenerate}},
		},
	}
}

func (p *demoEchoProvider) ValidateModel(model *models.MLModel) error {
	if model == nil {
		return fmt.Errorf("model is required")
	}
	if model.ProviderModel != "echo_classifier" && model.ProviderModel != "echo_generator" {
		return fmt.Errorf("unsupported demo echo provider model: %s", model.ProviderModel)
	}
	return nil
}

func (p *demoEchoProvider) Train(req *mlmodel.ProviderTrainRequest) (*mlmodel.ProviderTrainResult, error) {
	return nil, fmt.Errorf("demo-echo-provider does not support training")
}

func (p *demoEchoProvider) Infer(req *mlmodel.ProviderInferRequest) (*mlmodel.ProviderInferResult, error) {
	if req == nil || req.Model == nil {
		return nil, fmt.Errorf("model is required")
	}
	if err := p.ValidateModel(req.Model); err != nil {
		return nil, err
	}
	keys := make([]string, 0, len(req.Input))
	total := 0.0
	for key, value := range req.Input {
		keys = append(keys, key)
		switch v := value.(type) {
		case float64:
			total += v
		case int:
			total += float64(v)
		}
	}
	sort.Strings(keys)
	if req.Model.ProviderModel == "echo_classifier" {
		label := "low"
		if total >= 10 {
			label = "high"
		}
		return &mlmodel.ProviderInferResult{Output: label, Confidence: 0.65, Metadata: map[string]any{"provider": "demo-echo-provider", "observed_keys": keys, "total": total}}, nil
	}
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s=%v", key, req.Input[key]))
	}
	return &mlmodel.ProviderInferResult{Output: fmt.Sprintf("Generated summary: %s", strings.Join(parts, ", ")), Confidence: 0.55, Metadata: map[string]any{"provider": "demo-echo-provider", "observed_keys": keys}}, nil
}

var MLProvider mlmodel.Provider = &demoEchoProvider{}
