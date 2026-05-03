package storage

import (
	"fmt"
	"github.com/mimir-aip/mimir-aip-go/pkg/models"
	"sort"
	"strings"
	"time"
)

func (s *Service) GetIngestionHealth(projectID string) (*models.IngestionHealthReport, error) {
	configs, err := s.store.ListStorageConfigsByProject(projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to list storage configs: %w", err)
	}
	report := &models.IngestionHealthReport{
		ProjectID:   projectID,
		GeneratedAt: time.Now().UTC(),
		Sources:     make([]models.IngestionHealthSource, 0, len(configs)),
	}
	if len(configs) == 0 {
		report.Status = models.IngestionHealthCritical
		report.Recommendations = []string{"No storage sources configured for this project"}
		return report, nil
	}

	total := 0.0
	for _, cfg := range configs {
		source := models.IngestionHealthSource{
			StorageID:  cfg.ID,
			PluginType: cfg.PluginType,
			Status:     models.IngestionHealthCritical,
		}
		if !cfg.Active {
			source.Findings = append(source.Findings, "Storage config is inactive")
			report.Sources = append(report.Sources, source)
			continue
		}
		cirs, err := s.Retrieve(cfg.ID, &models.CIRQuery{Limit: 1000})
		if err != nil {
			source.Findings = append(source.Findings, fmt.Sprintf("Failed to retrieve sample data: %v", err))
			report.Sources = append(report.Sources, source)
			continue
		}
		source.SampleSize = len(cirs)
		if len(cirs) == 0 {
			source.Findings = append(source.Findings, "No ingested records found")
			report.Sources = append(report.Sources, source)
			continue
		}
		source.LastIngestedAt, source.FreshnessScore = freshnessFromSample(cirs)
		source.CompletenessScore = completenessFromSample(cirs)
		source.SchemaDriftScore = schemaDriftFromSample(cirs)
		source.OverallScore = roundTo3(0.45*source.FreshnessScore + 0.35*source.CompletenessScore + 0.20*source.SchemaDriftScore)
		source.Status = scoreToIngestionStatus(source.OverallScore)
		source.Findings = append(source.Findings, describeSourceFinding(source)...)
		total += source.OverallScore
		report.Sources = append(report.Sources, source)
	}
	report.OverallScore = roundTo3(total / float64(len(report.Sources)))
	report.Status = scoreToIngestionStatus(report.OverallScore)
	report.Recommendations = healthRecommendations(report)
	return report, nil
}

func freshnessFromSample(cirs []*models.CIR) (*time.Time, float64) {
	var latest *time.Time
	for _, cir := range cirs {
		ts := cir.Source.Timestamp.UTC()
		if ts.IsZero() {
			continue
		}
		if latest == nil || ts.After(*latest) {
			cp := ts
			latest = &cp
		}
	}
	if latest == nil {
		return nil, 0.10
	}
	age := time.Since(*latest)
	switch {
	case age <= time.Hour:
		return latest, 1.0
	case age <= 24*time.Hour:
		return latest, 0.85
	case age <= 7*24*time.Hour:
		return latest, 0.55
	default:
		return latest, 0.20
	}
}

func completenessFromSample(cirs []*models.CIR) float64 {
	totalFields := 0
	nonEmptyFields := 0
	for _, cir := range cirs {
		rows := cirRows(cir)
		for _, row := range rows {
			for _, val := range row {
				totalFields++
				if valuePresent(val) {
					nonEmptyFields++
				}
			}
		}
	}
	if totalFields == 0 {
		return 0.0
	}
	return roundTo3(float64(nonEmptyFields) / float64(totalFields))
}

func schemaDriftFromSample(cirs []*models.CIR) float64 {
	signatures := make(map[string]int)
	totalRows := 0
	for _, cir := range cirs {
		rows := cirRows(cir)
		for _, row := range rows {
			totalRows++
			sig := schemaSignature(row)
			signatures[sig]++
		}
	}
	if totalRows == 0 || len(signatures) <= 1 {
		return 1.0
	}
	maxCount := 0
	for _, c := range signatures {
		if c > maxCount {
			maxCount = c
		}
	}
	driftRatio := 1 - (float64(maxCount) / float64(totalRows))
	return roundTo3(1 - driftRatio)
}

func scoreToIngestionStatus(score float64) models.IngestionHealthStatus {
	switch {
	case score >= 0.80:
		return models.IngestionHealthHealthy
	case score >= 0.50:
		return models.IngestionHealthWarning
	default:
		return models.IngestionHealthCritical
	}
}

func describeSourceFinding(source models.IngestionHealthSource) []string {
	findings := make([]string, 0, 3)
	if source.FreshnessScore < 0.55 {
		findings = append(findings, "Data freshness is degraded")
	}
	if source.CompletenessScore < 0.70 {
		findings = append(findings, "Record completeness is low")
	}
	if source.SchemaDriftScore < 0.70 {
		findings = append(findings, "Schema drift detected in sampled rows")
	}
	if len(findings) == 0 {
		findings = append(findings, "Ingestion quality is stable")
	}
	return findings
}

func healthRecommendations(report *models.IngestionHealthReport) []string {
	recs := make([]string, 0, 3)
	if report.Status == models.IngestionHealthCritical {
		recs = append(recs, "Review failing or inactive storage sources and restore ingestion")
	}
	for _, src := range report.Sources {
		if src.Status == models.IngestionHealthCritical {
			recs = append(recs, fmt.Sprintf("Investigate source %s (%s)", src.StorageID, src.PluginType))
		}
	}
	if len(recs) == 0 {
		recs = append(recs, "No immediate ingestion remediation required")
	}
	return recs
}

func cirRows(cir *models.CIR) []map[string]interface{} {
	if cir == nil || cir.Data == nil {
		return nil
	}
	rows := make([]map[string]interface{}, 0)
	switch d := cir.Data.(type) {
	case map[string]interface{}:
		rows = append(rows, d)
	case []interface{}:
		for _, item := range d {
			if m, ok := item.(map[string]interface{}); ok {
				rows = append(rows, m)
			}
		}
	}
	return rows
}

func valuePresent(v interface{}) bool {
	if v == nil {
		return false
	}
	s := strings.TrimSpace(fmt.Sprintf("%v", v))
	return s != "" && s != "<nil>"
}

func schemaSignature(row map[string]interface{}) string {
	if len(row) == 0 {
		return "_empty"
	}
	keys := make([]string, 0, len(row))
	for k := range row {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return strings.Join(keys, "|")
}

func roundTo3(v float64) float64 {
	return float64(int(v*1000+0.5)) / 1000
}

// SetPluginLoader attaches a PluginLoader to the service, enabling dynamic
