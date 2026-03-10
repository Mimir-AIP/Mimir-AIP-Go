package analysis

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mimir-aip/mimir-aip-go/pkg/extraction"
	"github.com/mimir-aip/mimir-aip-go/pkg/metadatastore"
	"github.com/mimir-aip/mimir-aip-go/pkg/models"
	"github.com/mimir-aip/mimir-aip-go/pkg/storage"
)

const analysisAlgorithmVersion = "2026-03-09"

// Service runs persisted resolver analyses and autonomous insight generation.
type Service struct {
	store             metadatastore.MetadataStore
	extractionService *extraction.Service
	storageService    *storage.Service
}

func NewService(store metadatastore.MetadataStore, extractionService *extraction.Service, storageService *storage.Service) *Service {
	return &Service{store: store, extractionService: extractionService, storageService: storageService}
}

func (s *Service) RunResolver(projectID string, storageIDs []string) (*models.AnalysisRun, []*models.ReviewItem, error) {
	if projectID == "" {
		return nil, nil, fmt.Errorf("project_id is required")
	}
	if len(storageIDs) < 2 {
		return nil, nil, fmt.Errorf("at least two storage_ids are required for cross-source resolution")
	}
	policy, policyMetrics, err := s.adjustedLinkPolicy(projectID)
	if err != nil {
		return nil, nil, err
	}
	result, err := s.extractionService.ExtractFromStorage(projectID, storageIDs, true, false)
	if err != nil {
		return nil, nil, fmt.Errorf("resolver extraction failed: %w", err)
	}

	now := time.Now().UTC()
	run := &models.AnalysisRun{
		ID:               uuid.New().String(),
		ProjectID:        projectID,
		Kind:             models.AnalysisRunKindResolver,
		Status:           models.AnalysisRunStatusCompleted,
		SourceIDs:        append([]string{}, storageIDs...),
		AlgorithmVersion: analysisAlgorithmVersion,
		PolicyVersion:    fmt.Sprintf("review=%.2f:auto=%.2f", policy.ReviewThreshold, policy.AutoAcceptThreshold),
		CreatedAt:        now,
		CompletedAt:      ptrTime(now),
	}

	reviewItems := make([]*models.ReviewItem, 0)
	decisionCounts := map[string]int{"reject": 0, "needs_review": 0, "auto_accept": 0}
	for _, link := range result.CrossSourceLinks {
		decision := extraction.DecideCrossSourceLink(link, policy)
		decisionCounts[string(decision)]++
		if decision == extraction.LinkReject {
			continue
		}
		status := models.ReviewItemStatusPending
		suggested := string(decision)
		if decision == extraction.LinkAutoAccept {
			status = models.ReviewItemStatusAutoAccepted
			suggested = string(models.ReviewDecisionAccept)
		}
		item := &models.ReviewItem{
			ID:                uuid.New().String(),
			ProjectID:         projectID,
			RunID:             run.ID,
			FindingType:       "cross_source_link",
			Status:            status,
			SuggestedDecision: suggested,
			Confidence:        link.Confidence,
			Payload: map[string]any{
				"storage_a":     link.StorageA,
				"column_a":      link.ColumnA,
				"entity_type_a": link.EntityTypeA,
				"storage_b":     link.StorageB,
				"column_b":      link.ColumnB,
				"entity_type_b": link.EntityTypeB,
			},
			Evidence: map[string]any{
				"value_overlap":      link.ValueOverlap,
				"name_similarity":    link.NameSimilarity,
				"shared_value_count": link.SharedValueCount,
			},
			Rationale: fmt.Sprintf("Detected cross-source candidate between %s.%s and %s.%s", link.EntityTypeA, link.ColumnA, link.EntityTypeB, link.ColumnB),
			CreatedAt: now,
			UpdatedAt: now,
		}
		if err := s.store.SaveReviewItem(item); err != nil {
			return nil, nil, fmt.Errorf("failed to persist review item: %w", err)
		}
		reviewItems = append(reviewItems, item)
	}

	metrics, err := s.ResolverMetrics(projectID)
	if err != nil {
		return nil, nil, err
	}
	for k, v := range policyMetrics {
		metrics[k] = v
	}
	metrics["total_links_detected"] = len(result.CrossSourceLinks)
	metrics["pending_review_count"] = decisionCounts[string(extraction.LinkNeedsReview)]
	metrics["auto_accepted_count"] = decisionCounts[string(extraction.LinkAutoAccept)]
	run.Metrics = metrics
	if err := s.store.SaveAnalysisRun(run); err != nil {
		return nil, nil, fmt.Errorf("failed to persist analysis run: %w", err)
	}
	return run, reviewItems, nil
}

func (s *Service) ListReviewItems(projectID, status string) ([]*models.ReviewItem, error) {
	items, err := s.store.ListReviewItems(projectID)
	if err != nil {
		return nil, err
	}
	if status == "" {
		return items, nil
	}
	filtered := make([]*models.ReviewItem, 0, len(items))
	for _, item := range items {
		if string(item.Status) == status {
			filtered = append(filtered, item)
		}
	}
	return filtered, nil
}

func (s *Service) DecideReviewItem(id string, req *models.ReviewDecisionRequest) (*models.ReviewItem, error) {
	item, err := s.store.GetReviewItem(id)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	switch req.Decision {
	case models.ReviewDecisionAccept:
		item.Status = models.ReviewItemStatusAccepted
	case models.ReviewDecisionReject:
		item.Status = models.ReviewItemStatusRejected
	default:
		return nil, fmt.Errorf("decision must be accept or reject")
	}
	item.Rationale = req.Rationale
	item.Reviewer = req.Reviewer
	item.ReviewedAt = &now
	item.UpdatedAt = now
	if err := s.store.SaveReviewItem(item); err != nil {
		return nil, fmt.Errorf("failed to persist review decision: %w", err)
	}
	return item, nil
}

func (s *Service) ResolverMetrics(projectID string) (map[string]any, error) {
	items, err := s.store.ListReviewItems(projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to list review items: %w", err)
	}
	const highConfidenceThreshold = 0.75
	highTotal := 0
	highAccepted := 0
	decisionCounts := map[string]int{"pending": 0, "accepted": 0, "rejected": 0, "auto_accepted": 0}
	for _, item := range items {
		decisionCounts[string(item.Status)]++
		if item.FindingType != "cross_source_link" || item.Confidence < highConfidenceThreshold {
			continue
		}
		switch item.Status {
		case models.ReviewItemStatusAccepted, models.ReviewItemStatusRejected, models.ReviewItemStatusAutoAccepted:
			highTotal++
			if item.Status != models.ReviewItemStatusRejected {
				highAccepted++
			}
		}
	}
	precision := 0.0
	if highTotal > 0 {
		precision = float64(highAccepted) / float64(highTotal)
	}
	runs, err := s.store.ListAnalysisRunsByProject(projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to list analysis runs: %w", err)
	}
	series := make([]map[string]any, 0)
	for _, run := range runs {
		if run.Kind != models.AnalysisRunKindResolver {
			continue
		}
		series = append(series, map[string]any{
			"run_id":     run.ID,
			"created_at": run.CreatedAt,
			"metrics":    run.Metrics,
			"status":     run.Status,
		})
	}
	return map[string]any{
		"high_confidence_threshold": highConfidenceThreshold,
		"high_confidence_precision": precision,
		"high_confidence_total":     highTotal,
		"high_confidence_accepted":  highAccepted,
		"decision_counts":           decisionCounts,
		"runs":                      series,
	}, nil
}

func (s *Service) GenerateProjectInsights(projectID string) (*models.AnalysisRun, []*models.Insight, error) {
	configs, err := s.storageService.GetProjectStorageConfigs(projectID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load project storage configs: %w", err)
	}
	now := time.Now().UTC()
	run := &models.AnalysisRun{
		ID:               uuid.New().String(),
		ProjectID:        projectID,
		Kind:             models.AnalysisRunKindInsights,
		Status:           models.AnalysisRunStatusCompleted,
		AlgorithmVersion: analysisAlgorithmVersion,
		CreatedAt:        now,
		CompletedAt:      ptrTime(now),
	}
	insights := make([]*models.Insight, 0)
	sourceIDs := make([]string, 0, len(configs))
	for _, cfg := range configs {
		sourceIDs = append(sourceIDs, cfg.ID)
		sourceInsights, err := s.generateStorageInsights(projectID, run.ID, cfg.ID)
		if err != nil {
			continue
		}
		insights = append(insights, sourceInsights...)
	}
	run.SourceIDs = sourceIDs
	run.Metrics = map[string]any{"insight_count": len(insights)}
	if err := s.store.SaveAnalysisRun(run); err != nil {
		return nil, nil, fmt.Errorf("failed to persist insight run: %w", err)
	}
	for _, insight := range insights {
		if err := s.store.SaveInsight(insight); err != nil {
			return nil, nil, fmt.Errorf("failed to persist insight: %w", err)
		}
	}
	return run, insights, nil
}

func (s *Service) ListInsights(projectID, severity string, minConfidence float64) ([]*models.Insight, error) {
	insights, err := s.store.ListInsightsByProject(projectID)
	if err != nil {
		return nil, err
	}
	filtered := make([]*models.Insight, 0, len(insights))
	for _, insight := range insights {
		if severity != "" && string(insight.Severity) != severity {
			continue
		}
		if minConfidence > 0 && insight.Confidence < minConfidence {
			continue
		}
		filtered = append(filtered, insight)
	}
	return filtered, nil
}

func (s *Service) StartInsightLoop(ctx context.Context, interval time.Duration) {
	if interval <= 0 {
		interval = 24 * time.Hour
	}
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				projects, err := s.store.ListProjects()
				if err != nil {
					continue
				}
				for _, project := range projects {
					if project.Status != models.ProjectStatusActive {
						continue
					}
					_, _, _ = s.GenerateProjectInsights(project.ID)
				}
			}
		}
	}()
}

func (s *Service) adjustedLinkPolicy(projectID string) (extraction.LinkPolicy, map[string]any, error) {
	policy := extraction.DefaultLinkPolicy()
	items, err := s.store.ListReviewItems(projectID)
	if err != nil {
		return policy, nil, fmt.Errorf("failed to list review items: %w", err)
	}
	accepted := 0
	rejected := 0
	for _, item := range items {
		if item.FindingType != "cross_source_link" {
			continue
		}
		switch item.Status {
		case models.ReviewItemStatusAccepted:
			accepted++
		case models.ReviewItemStatusRejected:
			rejected++
		}
	}
	total := accepted + rejected
	if total > 0 {
		rejectRate := float64(rejected) / float64(total)
		delta := (rejectRate - 0.5) * 0.2
		policy.ReviewThreshold = clamp(policy.ReviewThreshold+delta, 0.2, 0.7)
		policy.AutoAcceptThreshold = clamp(policy.AutoAcceptThreshold+delta, policy.ReviewThreshold+0.1, 0.95)
	}
	return policy, map[string]any{
		"review_threshold":      policy.ReviewThreshold,
		"auto_accept_threshold": policy.AutoAcceptThreshold,
		"accepted_feedback":     accepted,
		"rejected_feedback":     rejected,
	}, nil
}

func (s *Service) generateStorageInsights(projectID, runID, storageID string) ([]*models.Insight, error) {
	cirs, err := s.storageService.Retrieve(storageID, &models.CIRQuery{Limit: 2000})
	if err != nil || len(cirs) == 0 {
		return nil, err
	}
	buckets := bucketByDay(cirs)
	if len(buckets) < 4 {
		return nil, nil
	}
	days := sortedBucketKeys(buckets)
	latestDay := days[len(days)-1]
	latestCount := buckets[latestDay]
	previous := make([]float64, 0, len(days)-1)
	for _, day := range days[:len(days)-1] {
		previous = append(previous, float64(buckets[day]))
	}
	avg, stddev := meanStd(previous)
	insights := make([]*models.Insight, 0)
	now := time.Now().UTC()
	if latestCount >= int(math.Max(avg*1.5, 3)) && float64(latestCount) > avg+2*stddev {
		insights = append(insights, newInsight(projectID, runID, "anomaly_spike", severityForRatio(float64(latestCount), avg), confidenceForDeviation(float64(latestCount), avg, stddev), now, fmt.Sprintf("Recent ingest volume spiked on %s for storage %s", latestDay, storageID), map[string]any{"storage_id": storageID, "latest_day": latestDay, "latest_count": latestCount, "baseline_avg": avg, "baseline_stddev": stddev}, "Inspect the newest records and upstream source health."))
	}
	if len(days) >= 6 {
		recentAvg := meanInts(selectCounts(buckets, days[len(days)-3:]))
		baselineAvg := meanInts(selectCounts(buckets, days[len(days)-6:len(days)-3]))
		if baselineAvg > 0 && math.Abs(recentAvg-baselineAvg)/baselineAvg >= 0.5 {
			insights = append(insights, newInsight(projectID, runID, "trend_break", severityForRatio(recentAvg, baselineAvg), clamp(math.Abs(recentAvg-baselineAvg)/math.Max(baselineAvg, 1), 0.3, 0.95), now, fmt.Sprintf("Recent ingest trend shifted for storage %s", storageID), map[string]any{"storage_id": storageID, "recent_average": recentAvg, "baseline_average": baselineAvg, "window_days": 3}, "Review whether the source cadence or data availability changed."))
		}
	}
	for pair, counts := range cooccurrenceByDay(cirs) {
		pairDays := sortedBucketKeys(counts)
		if len(pairDays) < 4 {
			continue
		}
		latestPairDay := pairDays[len(pairDays)-1]
		latestPairCount := counts[latestPairDay]
		previousPair := make([]float64, 0, len(pairDays)-1)
		for _, day := range pairDays[:len(pairDays)-1] {
			previousPair = append(previousPair, float64(counts[day]))
		}
		pairAvg, pairStd := meanStd(previousPair)
		if latestPairCount < 2 || float64(latestPairCount) <= pairAvg+2*pairStd || float64(latestPairCount) < pairAvg*2 {
			continue
		}
		insights = append(insights, newInsight(projectID, runID, "cooccurrence_surge", severityForRatio(float64(latestPairCount), pairAvg), confidenceForDeviation(float64(latestPairCount), pairAvg, pairStd), now, fmt.Sprintf("Field co-occurrence surged for %s in storage %s", pair, storageID), map[string]any{"storage_id": storageID, "field_pair": pair, "latest_day": latestPairDay, "latest_count": latestPairCount, "baseline_avg": pairAvg}, "Inspect recent records matching this field combination for emerging patterns."))
		break
	}
	return insights, nil
}

func bucketByDay(cirs []*models.CIR) map[string]int {
	result := make(map[string]int)
	for _, cir := range cirs {
		ts := cir.Source.Timestamp.UTC()
		if ts.IsZero() {
			continue
		}
		key := ts.Format("2006-01-02")
		result[key]++
	}
	return result
}

func cooccurrenceByDay(cirs []*models.CIR) map[string]map[string]int {
	result := make(map[string]map[string]int)
	for _, cir := range cirs {
		data, err := cir.GetDataAsMap()
		if err != nil {
			continue
		}
		keys := make([]string, 0)
		for key, value := range data {
			if strings.TrimSpace(fmt.Sprintf("%v", value)) == "" {
				continue
			}
			keys = append(keys, key)
		}
		sort.Strings(keys)
		day := cir.Source.Timestamp.UTC().Format("2006-01-02")
		for i := 0; i < len(keys); i++ {
			for j := i + 1; j < len(keys); j++ {
				pair := keys[i] + "+" + keys[j]
				if result[pair] == nil {
					result[pair] = make(map[string]int)
				}
				result[pair][day]++
			}
		}
	}
	return result
}

func sortedBucketKeys[T ~map[string]int](buckets T) []string {
	keys := make([]string, 0, len(buckets))
	for key := range buckets {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func selectCounts(buckets map[string]int, keys []string) []int {
	result := make([]int, 0, len(keys))
	for _, key := range keys {
		result = append(result, buckets[key])
	}
	return result
}

func meanStd(values []float64) (float64, float64) {
	if len(values) == 0 {
		return 0, 0
	}
	mean := 0.0
	for _, value := range values {
		mean += value
	}
	mean /= float64(len(values))
	variance := 0.0
	for _, value := range values {
		variance += math.Pow(value-mean, 2)
	}
	variance /= float64(len(values))
	return mean, math.Sqrt(variance)
}

func meanInts(values []int) float64 {
	if len(values) == 0 {
		return 0
	}
	total := 0
	for _, value := range values {
		total += value
	}
	return float64(total) / float64(len(values))
}

func confidenceForDeviation(latest, avg, stddev float64) float64 {
	if latest <= avg {
		return 0.3
	}
	if stddev <= 0 {
		return clamp((latest-avg)/math.Max(avg, 1), 0.35, 0.95)
	}
	return clamp((latest-avg)/(2*stddev+avg), 0.35, 0.95)
}

func severityForRatio(latest, baseline float64) models.InsightSeverity {
	ratio := latest / math.Max(baseline, 1)
	switch {
	case ratio >= 4:
		return models.InsightSeverityCritical
	case ratio >= 2.5:
		return models.InsightSeverityHigh
	case ratio >= 1.5:
		return models.InsightSeverityMedium
	default:
		return models.InsightSeverityLow
	}
}

func newInsight(projectID, runID, kind string, severity models.InsightSeverity, confidence float64, now time.Time, explanation string, evidence map[string]any, action string) *models.Insight {
	return &models.Insight{
		ID:              uuid.New().String(),
		ProjectID:       projectID,
		RunID:           runID,
		Type:            kind,
		Severity:        severity,
		Confidence:      confidence,
		Explanation:     explanation,
		SuggestedAction: action,
		Evidence:        evidence,
		Status:          "open",
		CreatedAt:       now,
		UpdatedAt:       now,
	}
}

func ptrTime(t time.Time) *time.Time { return &t }

func clamp(value, min, max float64) float64 {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}
