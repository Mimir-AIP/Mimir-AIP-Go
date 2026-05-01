package mlmodel

import (
	"fmt"
	"sort"
	"time"

	"github.com/mimir-aip/mimir-aip-go/pkg/models"
)

const modelMetadataFeatureProvenance = "feature_provenance"

func (s *Service) buildFeatureProvenance(model *models.MLModel, storageIDs []string) (*models.MLFeatureProvenance, error) {
	if model == nil {
		return nil, fmt.Errorf("model cannot be nil")
	}
	compiled, err := s.store.GetCompiledOntology(model.OntologyID)
	if err != nil {
		return &models.MLFeatureProvenance{OntologyID: model.OntologyID, StorageIDs: append([]string(nil), storageIDs...), GeneratedAt: time.Now().UTC()}, nil
	}
	provenance := &models.MLFeatureProvenance{OntologyID: model.OntologyID, ContentHash: compiled.ContentHash, StorageIDs: append([]string(nil), storageIDs...), GeneratedAt: time.Now().UTC()}
	features := map[string]*models.MLSemanticFeature{}
	for _, storageID := range storageIDs {
		if s.storageService == nil {
			continue
		}
		cirs, err := s.storageService.RetrieveForProject(model.ProjectID, storageID, &models.CIRQuery{Limit: 1000})
		if err != nil {
			return nil, fmt.Errorf("retrieve semantic training data from storage %s: %w", storageID, err)
		}
		for _, cir := range cirs {
			if cir == nil || cir.Metadata.Ontology == nil || cir.Metadata.Ontology.OntologyID != model.OntologyID || cir.Metadata.Ontology.ContentHash != compiled.ContentHash {
				continue
			}
			for propertyID, property := range cir.Metadata.Ontology.Properties {
				key := cir.Metadata.Ontology.ClassID + ":" + propertyID
				feature := features[key]
				if feature == nil {
					feature = &models.MLSemanticFeature{ClassID: cir.Metadata.Ontology.ClassID, PropertyID: propertyID, Range: property.Range}
					features[key] = feature
				}
				feature.SourceFields = appendUnique(feature.SourceFields, property.SourceField)
				feature.StorageIDs = appendUnique(feature.StorageIDs, storageID)
			}
		}
	}
	keys := make([]string, 0, len(features))
	for key := range features {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		feature := features[key]
		sort.Strings(feature.SourceFields)
		sort.Strings(feature.StorageIDs)
		provenance.Features = append(provenance.Features, *feature)
	}
	return provenance, nil
}

func (s *Service) ensureModelOntologyCompatible(model *models.MLModel) error {
	if model == nil || model.Metadata == nil {
		return nil
	}
	raw, ok := model.Metadata[modelMetadataFeatureProvenance]
	if !ok {
		return nil
	}
	provenanceMap, ok := raw.(map[string]interface{})
	if !ok {
		return nil
	}
	contentHash, _ := provenanceMap["content_hash"].(string)
	if contentHash == "" {
		return nil
	}
	compiled, err := s.store.GetCompiledOntology(model.OntologyID)
	if err != nil {
		return fmt.Errorf("compiled ontology not found for model compatibility check: %w", err)
	}
	if compiled.ContentHash != contentHash {
		return fmt.Errorf("model ontology hash mismatch: trained with %s, current ontology is %s", contentHash, compiled.ContentHash)
	}
	return nil
}

func appendUnique(values []string, value string) []string {
	if value == "" {
		return values
	}
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}
