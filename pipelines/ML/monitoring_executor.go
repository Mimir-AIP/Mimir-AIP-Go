package ml

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	storage "github.com/Mimir-AIP/Mimir-AIP-Go/pipelines/Storage"
)

// MonitoringExecutor orchestrates the execution of monitoring jobs
type MonitoringExecutor struct {
	Storage    *storage.PersistenceBackend
	RuleEngine *RuleEngine
}

// NewMonitoringExecutor creates a new monitoring executor
func NewMonitoringExecutor(storageBE *storage.PersistenceBackend) *MonitoringExecutor {
	return &MonitoringExecutor{
		Storage:    storageBE,
		RuleEngine: NewRuleEngine(storageBE),
	}
}

// ExecuteMonitoringJob runs a monitoring job by ID
func (me *MonitoringExecutor) ExecuteMonitoringJob(ctx context.Context, jobID string) error {
	startTime := time.Now()

	// Get job details from database
	job, err := me.Storage.GetMonitoringJob(ctx, jobID)
	if err != nil {
		return fmt.Errorf("failed to get monitoring job: %w", err)
	}

	log.Printf("[Monitoring] Starting job: %s", job.Name)

	// Parse metrics JSON array
	var metrics []string
	if err := json.Unmarshal([]byte(job.Metrics), &metrics); err != nil {
		return fmt.Errorf("failed to parse metrics: %w", err)
	}

	// Track statistics
	metricsChecked := 0
	totalAnomalies := 0

	// Process each metric
	for _, metricName := range metrics {
		log.Printf("[Monitoring] Checking metric: %s", metricName)

		// Query time-series data for this metric (last 30 days)
		cutoffTime := time.Now().AddDate(0, 0, -30)
		timeSeries, err := me.getTimeSeriesData(ctx, job.OntologyID, "", metricName, cutoffTime, time.Now())
		if err != nil {
			log.Printf("[Monitoring] Warning: Failed to get time-series data for %s: %v", metricName, err)
			continue
		}

		if timeSeries == nil || len(timeSeries.Points) == 0 {
			log.Printf("[Monitoring] No data available for metric: %s", metricName)
			continue
		}

		// Get current value (most recent point)
		currentValue := timeSeries.Points[len(timeSeries.Points)-1].Value

		// Evaluate rules for this metric - returns count of anomalies detected
		anomalyCount, err := me.RuleEngine.EvaluateRules(
			ctx,
			job.OntologyID,
			"", // entityID - empty for ontology-level metrics
			metricName,
			currentValue,
			timeSeries,
		)
		if err != nil {
			log.Printf("[Monitoring] Warning: Failed to evaluate rules for %s: %v", metricName, err)
			continue
		}

		if anomalyCount > 0 {
			log.Printf("[Monitoring] Detected %d anomalies for metric: %s", anomalyCount, metricName)
			totalAnomalies += anomalyCount
		}

		metricsChecked++
	}

	// Record execution in monitoring_job_runs
	completedAt := time.Now()
	run := &storage.MonitoringJobRun{
		JobID:          jobID,
		StartedAt:      startTime,
		CompletedAt:    &completedAt,
		Status:         "success",
		MetricsChecked: metricsChecked,
		AlertsCreated:  totalAnomalies, // Keep field name for backwards compatibility
	}
	if err := me.Storage.RecordMonitoringRun(ctx, run); err != nil {
		log.Printf("[Monitoring] Warning: Failed to record monitoring run: %v", err)
	}

	// Update job status
	status := "success"
	if err := me.Storage.UpdateMonitoringJobStatus(ctx, jobID, status, totalAnomalies); err != nil {
		log.Printf("[Monitoring] Warning: Failed to update job status: %v", err)
	}

	log.Printf("[Monitoring] Job completed: checked %d metrics, detected %d anomalies", metricsChecked, totalAnomalies)
	return nil
}

// getTimeSeriesData retrieves time-series data from the database
func (me *MonitoringExecutor) getTimeSeriesData(
	ctx context.Context,
	ontologyID string,
	entityID string,
	metricName string,
	startTime time.Time,
	endTime time.Time,
) (*TimeSeries, error) {
	query := `
		SELECT timestamp, value
		FROM time_series_data
		WHERE ontology_id = ? AND metric_name = ? AND timestamp >= ? AND timestamp <= ?
	`
	args := []interface{}{ontologyID, metricName, startTime, endTime}

	if entityID != "" {
		query += " AND entity_id = ?"
		args = append(args, entityID)
	}

	query += " ORDER BY timestamp ASC"

	rows, err := me.Storage.GetDB().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query time-series data: %w", err)
	}
	defer rows.Close()

	var points []TimeSeriesPoint
	for rows.Next() {
		var timestamp time.Time
		var value float64
		if err := rows.Scan(&timestamp, &value); err != nil {
			return nil, fmt.Errorf("failed to scan time-series point: %w", err)
		}
		points = append(points, TimeSeriesPoint{
			Timestamp: timestamp,
			Value:     value,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating time-series rows: %w", err)
	}

	return &TimeSeries{
		EntityID:   entityID,
		MetricName: metricName,
		Points:     points,
	}, nil
}
