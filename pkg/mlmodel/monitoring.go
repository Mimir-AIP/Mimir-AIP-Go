package mlmodel

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/mimir-aip/mimir-aip-go/pkg/metadatastore"
	"github.com/mimir-aip/mimir-aip-go/pkg/mlmodel/training"
	"github.com/mimir-aip/mimir-aip-go/pkg/models"
	"github.com/mimir-aip/mimir-aip-go/pkg/storage"
)

// MonitoringService periodically validates trained models and flags performance degradation.
type MonitoringService struct {
	store                metadatastore.MetadataStore
	mlService            *Service
	storageService       *storage.Service
	ticker               *time.Ticker
	done                 chan struct{}
	degradationThreshold float64 // accuracy drop that triggers degraded status (default 0.05)
}

// NewMonitoringService creates a new monitoring service.
func NewMonitoringService(store metadatastore.MetadataStore, mlService *Service, storageService *storage.Service) *MonitoringService {
	return &MonitoringService{
		store:                store,
		mlService:            mlService,
		storageService:       storageService,
		degradationThreshold: 0.05,
	}
}

// Start launches the background goroutine that checks all models every hour.
func (m *MonitoringService) Start() {
	m.done = make(chan struct{})
	m.ticker = time.NewTicker(1 * time.Hour)
	go func() {
		for {
			select {
			case <-m.ticker.C:
				m.checkAllModels()
			case <-m.done:
				return
			}
		}
	}()
	log.Println("Model monitoring service started (hourly checks)")
}

// Stop stops the background monitoring goroutine.
func (m *MonitoringService) Stop() {
	if m.ticker != nil {
		m.ticker.Stop()
	}
	if m.done != nil {
		close(m.done)
	}
}

// checkAllModels iterates all trained models and calls CheckModel on each.
func (m *MonitoringService) checkAllModels() {
	log.Println("Model monitoring: starting periodic check")
	allModels, err := m.store.ListMLModels()
	if err != nil {
		log.Printf("Model monitoring: failed to list models: %v", err)
		return
	}

	checked := 0
	for _, model := range allModels {
		if model.Status != models.ModelStatusTrained && model.Status != models.ModelStatusDegraded {
			continue
		}
		if err := m.CheckModel(model); err != nil {
			log.Printf("Model monitoring: check failed for %s (%s): %v", model.ID, model.Name, err)
		} else {
			checked++
		}
	}
	log.Printf("Model monitoring: checked %d models", checked)
}

// CheckModel validates a single model against recent storage data and updates its status
// if performance has degraded beyond the threshold.
func (m *MonitoringService) CheckModel(model *models.MLModel) error {
	if model.ModelArtifactPath == "" {
		return fmt.Errorf("model %s has no artifact", model.ID)
	}
	if model.PerformanceMetrics == nil {
		return fmt.Errorf("model %s has no baseline performance metrics", model.ID)
	}

	// Retrieve recent data from project storage
	storageConfigs, err := m.storageService.GetProjectStorageConfigs(model.ProjectID)
	if err != nil || len(storageConfigs) == 0 {
		return fmt.Errorf("no storage configs for project %s: %v", model.ProjectID, err)
	}

	var allCIRs []*models.CIR
	for _, cfg := range storageConfigs {
		cirs, err := m.storageService.Retrieve(cfg.ID, &models.CIRQuery{})
		if err != nil {
			continue
		}
		allCIRs = append(allCIRs, cirs...)
	}

	if len(allCIRs) == 0 {
		return fmt.Errorf("no data available to validate model %s", model.ID)
	}

	artifactBytes, err := os.ReadFile(model.ModelArtifactPath)
	if err != nil {
		return fmt.Errorf("failed to read model artifact for monitoring: %w", err)
	}
	var artifact struct {
		FeatureNames []string       `json:"feature_names"`
		Metadata     map[string]any `json:"metadata,omitempty"`
	}
	if err := json.Unmarshal(artifactBytes, &artifact); err != nil {
		return fmt.Errorf("failed to parse model artifact for monitoring: %w", err)
	}
	if len(artifact.FeatureNames) == 0 {
		return fmt.Errorf("model %s has no feature_names in artifact", model.ID)
	}

	labelColumn := "label"
	if v, ok := artifact.Metadata["label_column"].(string); ok && v != "" {
		labelColumn = v
	}

	features, labels := extractFeaturesAndLabels(allCIRs, artifact.FeatureNames, labelColumn)
	if len(features) == 0 {
		return fmt.Errorf("no monitoring rows matched artifact feature schema for model %s", model.ID)
	}

	data := &training.TrainingData{
		TestFeatures: features,
		TestLabels:   labels,
	}

	currentMetrics, err := m.mlService.ValidateModel(model.ID, data)
	if err != nil {
		return fmt.Errorf("validation failed for model %s: %w", model.ID, err)
	}

	now := time.Now().UTC()
	if model.Metadata == nil {
		model.Metadata = make(map[string]interface{})
	}
	model.Metadata["last_monitored_at"] = now.Format(time.RFC3339)
	model.PerformanceMetrics.LastMonitoredAt = now

	baselineAccuracy := model.PerformanceMetrics.Accuracy
	drop := baselineAccuracy - currentMetrics.Accuracy

	if drop > m.degradationThreshold {
		log.Printf("WARNING: Model %s (%s) degraded — accuracy dropped %.2f%% (baseline: %.1f%%, current: %.1f%%)",
			model.ID, model.Name, drop*100, baselineAccuracy*100, currentMetrics.Accuracy*100)
		model.Status = models.ModelStatusDegraded
		model.PerformanceMetrics.DegradationDetected = true
	}

	if err := m.store.SaveMLModel(model); err != nil {
		return fmt.Errorf("failed to save model after monitoring: %w", err)
	}

	return nil
}

// extractFeaturesAndLabels converts CIR records into a deterministic numeric feature matrix.
func extractFeaturesAndLabels(cirs []*models.CIR, featureNames []string, labelColumn string) ([][]float64, []float64) {
	features := make([][]float64, 0, len(cirs))
	labels := make([]float64, 0, len(cirs))

	for _, cir := range cirs {
		dataMap, ok := cir.Data.(map[string]interface{})
		if !ok {
			continue
		}
		labelVal, ok := numericValue(dataMap[labelColumn])
		if !ok {
			continue
		}
		row := make([]float64, len(featureNames))
		valid := true
		for i, featureName := range featureNames {
			value, ok := numericValue(dataMap[featureName])
			if !ok {
				valid = false
				break
			}
			row[i] = value
		}
		if !valid {
			continue
		}
		features = append(features, row)
		labels = append(labels, labelVal)
	}

	return features, labels
}

func numericValue(v interface{}) (float64, bool) {
	switch value := v.(type) {
	case float64:
		return value, true
	case float32:
		return float64(value), true
	case int:
		return float64(value), true
	case int64:
		return float64(value), true
	case int32:
		return float64(value), true
	case bool:
		if value {
			return 1, true
		}
		return 0, true
	default:
		return 0, false
	}
}
