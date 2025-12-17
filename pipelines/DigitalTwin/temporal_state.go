package DigitalTwin

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// TemporalStateManager manages time-series state data for simulations
type TemporalStateManager struct {
	db *sql.DB
}

// NewTemporalStateManager creates a new temporal state manager
func NewTemporalStateManager(db *sql.DB) *TemporalStateManager {
	return &TemporalStateManager{db: db}
}

// StoreSnapshot persists a state snapshot to the database
func (tsm *TemporalStateManager) StoreSnapshot(snapshot StateSnapshot) error {
	stateJSON, err := json.Marshal(snapshot.State)
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	metricsJSON, err := json.Marshal(snapshot.Metrics)
	if err != nil {
		return fmt.Errorf("failed to marshal metrics: %w", err)
	}

	query := `
		INSERT INTO twin_state_snapshots (run_id, timestamp, step_number, state, description, metrics)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	_, err = tsm.db.Exec(query,
		snapshot.RunID,
		snapshot.Timestamp,
		snapshot.Step,
		string(stateJSON),
		snapshot.Description,
		string(metricsJSON),
	)

	if err != nil {
		return fmt.Errorf("failed to store snapshot: %w", err)
	}

	return nil
}

// GetSnapshotsByRunID retrieves all snapshots for a simulation run
func (tsm *TemporalStateManager) GetSnapshotsByRunID(runID string) ([]StateSnapshot, error) {
	query := `
		SELECT run_id, timestamp, step_number, state, description, metrics
		FROM twin_state_snapshots
		WHERE run_id = ?
		ORDER BY step_number ASC
	`

	rows, err := tsm.db.Query(query, runID)
	if err != nil {
		return nil, fmt.Errorf("failed to query snapshots: %w", err)
	}
	defer rows.Close()

	var snapshots []StateSnapshot

	for rows.Next() {
		var snapshot StateSnapshot
		var stateJSON, metricsJSON string

		err := rows.Scan(
			&snapshot.RunID,
			&snapshot.Timestamp,
			&snapshot.Step,
			&stateJSON,
			&snapshot.Description,
			&metricsJSON,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan snapshot: %w", err)
		}

		// Unmarshal state
		snapshot.State = make(map[string]interface{})
		if err := json.Unmarshal([]byte(stateJSON), &snapshot.State); err != nil {
			return nil, fmt.Errorf("failed to unmarshal state: %w", err)
		}

		// Unmarshal metrics
		snapshot.Metrics = make(map[string]float64)
		if err := json.Unmarshal([]byte(metricsJSON), &snapshot.Metrics); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metrics: %w", err)
		}

		snapshots = append(snapshots, snapshot)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating snapshots: %w", err)
	}

	return snapshots, nil
}

// GetSnapshotAtStep retrieves the snapshot at a specific step
func (tsm *TemporalStateManager) GetSnapshotAtStep(runID string, step int) (*StateSnapshot, error) {
	query := `
		SELECT run_id, timestamp, step_number, state, description, metrics
		FROM twin_state_snapshots
		WHERE run_id = ? AND step_number = ?
	`

	var snapshot StateSnapshot
	var stateJSON, metricsJSON string

	err := tsm.db.QueryRow(query, runID, step).Scan(
		&snapshot.RunID,
		&snapshot.Timestamp,
		&snapshot.Step,
		&stateJSON,
		&snapshot.Description,
		&metricsJSON,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("no snapshot found for run %s at step %d", runID, step)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query snapshot: %w", err)
	}

	// Unmarshal state
	snapshot.State = make(map[string]interface{})
	if err := json.Unmarshal([]byte(stateJSON), &snapshot.State); err != nil {
		return nil, fmt.Errorf("failed to unmarshal state: %w", err)
	}

	// Unmarshal metrics
	snapshot.Metrics = make(map[string]float64)
	if err := json.Unmarshal([]byte(metricsJSON), &snapshot.Metrics); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metrics: %w", err)
	}

	return &snapshot, nil
}

// GetSnapshotRange retrieves snapshots within a step range
func (tsm *TemporalStateManager) GetSnapshotRange(runID string, startStep, endStep int) ([]StateSnapshot, error) {
	query := `
		SELECT run_id, timestamp, step_number, state, description, metrics
		FROM twin_state_snapshots
		WHERE run_id = ? AND step_number BETWEEN ? AND ?
		ORDER BY step_number ASC
	`

	rows, err := tsm.db.Query(query, runID, startStep, endStep)
	if err != nil {
		return nil, fmt.Errorf("failed to query snapshot range: %w", err)
	}
	defer rows.Close()

	var snapshots []StateSnapshot

	for rows.Next() {
		var snapshot StateSnapshot
		var stateJSON, metricsJSON string

		err := rows.Scan(
			&snapshot.RunID,
			&snapshot.Timestamp,
			&snapshot.Step,
			&stateJSON,
			&snapshot.Description,
			&metricsJSON,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan snapshot: %w", err)
		}

		// Unmarshal state
		snapshot.State = make(map[string]interface{})
		if err := json.Unmarshal([]byte(stateJSON), &snapshot.State); err != nil {
			return nil, fmt.Errorf("failed to unmarshal state: %w", err)
		}

		// Unmarshal metrics
		snapshot.Metrics = make(map[string]float64)
		if err := json.Unmarshal([]byte(metricsJSON), &snapshot.Metrics); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metrics: %w", err)
		}

		snapshots = append(snapshots, snapshot)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating snapshot range: %w", err)
	}

	return snapshots, nil
}

// DeleteSnapshotsByRunID removes all snapshots for a run
func (tsm *TemporalStateManager) DeleteSnapshotsByRunID(runID string) error {
	query := `DELETE FROM twin_state_snapshots WHERE run_id = ?`
	_, err := tsm.db.Exec(query, runID)
	if err != nil {
		return fmt.Errorf("failed to delete snapshots: %w", err)
	}
	return nil
}

// TimeSeriesAnalysis provides analysis of temporal data
type TimeSeriesAnalysis struct {
	RunID            string                       `json:"run_id"`
	StartTime        time.Time                    `json:"start_time"`
	EndTime          time.Time                    `json:"end_time"`
	TotalSteps       int                          `json:"total_steps"`
	SnapshotCount    int                          `json:"snapshot_count"`
	MetricTimeSeries map[string][]TimePoint       `json:"metric_time_series"`
	EntityTimeSeries map[string][]EntityTimePoint `json:"entity_time_series"`
	Trends           []Trend                      `json:"trends"`
	Anomalies        []Anomaly                    `json:"anomalies"`
}

// TimePoint represents a metric value at a specific time
type TimePoint struct {
	Step      int       `json:"step"`
	Timestamp time.Time `json:"timestamp"`
	Value     float64   `json:"value"`
}

// EntityTimePoint represents entity state at a specific time
type EntityTimePoint struct {
	Step        int       `json:"step"`
	Timestamp   time.Time `json:"timestamp"`
	Status      string    `json:"status"`
	Utilization float64   `json:"utilization"`
	Available   bool      `json:"available"`
}

// Trend represents a detected trend in the data
type Trend struct {
	MetricName string  `json:"metric_name"`
	Direction  string  `json:"direction"` // "increasing", "decreasing", "stable"
	Slope      float64 `json:"slope"`
	Confidence float64 `json:"confidence"` // 0.0 to 1.0
	StartStep  int     `json:"start_step"`
	EndStep    int     `json:"end_step"`
}

// Anomaly represents a detected anomaly in the data
type Anomaly struct {
	Step          int       `json:"step"`
	Timestamp     time.Time `json:"timestamp"`
	MetricName    string    `json:"metric_name"`
	ExpectedValue float64   `json:"expected_value"`
	ActualValue   float64   `json:"actual_value"`
	Deviation     float64   `json:"deviation"`
	Severity      string    `json:"severity"`
	Description   string    `json:"description"`
}

// AnalyzeTimeSeries performs time-series analysis on simulation snapshots
func (tsm *TemporalStateManager) AnalyzeTimeSeries(runID string) (*TimeSeriesAnalysis, error) {
	snapshots, err := tsm.GetSnapshotsByRunID(runID)
	if err != nil {
		return nil, fmt.Errorf("failed to get snapshots: %w", err)
	}

	if len(snapshots) == 0 {
		return nil, fmt.Errorf("no snapshots found for run %s", runID)
	}

	analysis := &TimeSeriesAnalysis{
		RunID:            runID,
		StartTime:        snapshots[0].Timestamp,
		EndTime:          snapshots[len(snapshots)-1].Timestamp,
		TotalSteps:       snapshots[len(snapshots)-1].Step,
		SnapshotCount:    len(snapshots),
		MetricTimeSeries: make(map[string][]TimePoint),
		EntityTimeSeries: make(map[string][]EntityTimePoint),
		Trends:           []Trend{},
		Anomalies:        []Anomaly{},
	}

	// Extract metric time series
	metricNames := []string{
		"average_utilization",
		"peak_utilization",
		"active_entities",
		"failed_entities",
		"degraded_entities",
	}

	for _, metricName := range metricNames {
		timeSeries := []TimePoint{}
		for _, snapshot := range snapshots {
			if value, exists := snapshot.Metrics[metricName]; exists {
				timeSeries = append(timeSeries, TimePoint{
					Step:      snapshot.Step,
					Timestamp: snapshot.Timestamp,
					Value:     value,
				})
			}
		}
		if len(timeSeries) > 0 {
			analysis.MetricTimeSeries[metricName] = timeSeries
		}
	}

	// Detect trends
	for metricName, timeSeries := range analysis.MetricTimeSeries {
		if len(timeSeries) < 3 {
			continue
		}

		trend := detectTrend(metricName, timeSeries)
		if trend != nil {
			analysis.Trends = append(analysis.Trends, *trend)
		}
	}

	// Detect anomalies
	for metricName, timeSeries := range analysis.MetricTimeSeries {
		anomalies := detectAnomalies(metricName, timeSeries)
		analysis.Anomalies = append(analysis.Anomalies, anomalies...)
	}

	return analysis, nil
}

// detectTrend detects trends in time series data using linear regression
func detectTrend(metricName string, timeSeries []TimePoint) *Trend {
	if len(timeSeries) < 3 {
		return nil
	}

	// Simple linear regression
	n := float64(len(timeSeries))
	var sumX, sumY, sumXY, sumX2 float64

	for i, point := range timeSeries {
		x := float64(i)
		y := point.Value
		sumX += x
		sumY += y
		sumXY += x * y
		sumX2 += x * x
	}

	// Calculate slope
	slope := (n*sumXY - sumX*sumY) / (n*sumX2 - sumX*sumX)

	// Determine direction
	direction := "stable"
	if slope > 0.01 {
		direction = "increasing"
	} else if slope < -0.01 {
		direction = "decreasing"
	}

	// Calculate confidence (R-squared)
	meanY := sumY / n
	var ssRes, ssTot float64
	for i, point := range timeSeries {
		predicted := slope*float64(i) + (sumY-slope*sumX)/n
		ssRes += (point.Value - predicted) * (point.Value - predicted)
		ssTot += (point.Value - meanY) * (point.Value - meanY)
	}

	rSquared := 1.0
	if ssTot > 0 {
		rSquared = 1.0 - (ssRes / ssTot)
	}

	return &Trend{
		MetricName: metricName,
		Direction:  direction,
		Slope:      slope,
		Confidence: rSquared,
		StartStep:  timeSeries[0].Step,
		EndStep:    timeSeries[len(timeSeries)-1].Step,
	}
}

// detectAnomalies detects anomalies using moving average and standard deviation
func detectAnomalies(metricName string, timeSeries []TimePoint) []Anomaly {
	if len(timeSeries) < 5 {
		return []Anomaly{}
	}

	var anomalies []Anomaly
	windowSize := 5
	threshold := 2.0 // Standard deviations

	for i := windowSize; i < len(timeSeries); i++ {
		// Calculate moving average and std dev for window
		window := timeSeries[i-windowSize : i]

		var sum, sumSq float64
		for _, point := range window {
			sum += point.Value
			sumSq += point.Value * point.Value
		}

		mean := sum / float64(windowSize)
		variance := (sumSq / float64(windowSize)) - (mean * mean)
		stdDev := 0.0
		if variance > 0 {
			stdDev = variance
			// Approximate square root
			for j := 0; j < 10; j++ {
				stdDev = (stdDev + variance/stdDev) / 2
			}
		}

		// Check if current value is anomalous
		currentValue := timeSeries[i].Value
		deviation := (currentValue - mean)
		if stdDev > 0 {
			deviation = deviation / stdDev
		}

		if deviation > threshold || deviation < -threshold {
			severity := "medium"
			if deviation > 3.0 || deviation < -3.0 {
				severity = "high"
			}

			anomalies = append(anomalies, Anomaly{
				Step:          timeSeries[i].Step,
				Timestamp:     timeSeries[i].Timestamp,
				MetricName:    metricName,
				ExpectedValue: mean,
				ActualValue:   currentValue,
				Deviation:     deviation,
				Severity:      severity,
				Description:   fmt.Sprintf("%s deviated %.2f standard deviations from expected", metricName, deviation),
			})
		}
	}

	return anomalies
}

// CompareSnapshots compares two state snapshots and returns the differences
func CompareSnapshots(snapshot1, snapshot2 *StateSnapshot) map[string]interface{} {
	comparison := map[string]interface{}{
		"step_diff":    snapshot2.Step - snapshot1.Step,
		"time_diff":    snapshot2.Timestamp.Sub(snapshot1.Timestamp).Seconds(),
		"metric_diffs": make(map[string]float64),
		"changes":      []string{},
	}

	// Compare metrics
	for metric, value2 := range snapshot2.Metrics {
		if value1, exists := snapshot1.Metrics[metric]; exists {
			diff := value2 - value1
			if diff != 0 {
				comparison["metric_diffs"].(map[string]float64)[metric] = diff

				changeDesc := fmt.Sprintf("%s: %.2f -> %.2f (%.2f change)",
					metric, value1, value2, diff)
				comparison["changes"] = append(comparison["changes"].([]string), changeDesc)
			}
		}
	}

	return comparison
}

// GetMetricHistory extracts the history of a specific metric across all snapshots
func (tsm *TemporalStateManager) GetMetricHistory(runID string, metricName string) ([]TimePoint, error) {
	snapshots, err := tsm.GetSnapshotsByRunID(runID)
	if err != nil {
		return nil, fmt.Errorf("failed to get snapshots: %w", err)
	}

	var history []TimePoint
	for _, snapshot := range snapshots {
		if value, exists := snapshot.Metrics[metricName]; exists {
			history = append(history, TimePoint{
				Step:      snapshot.Step,
				Timestamp: snapshot.Timestamp,
				Value:     value,
			})
		}
	}

	return history, nil
}

// GetEntityHistory extracts the state history of a specific entity
func (tsm *TemporalStateManager) GetEntityHistory(runID string, entityURI string) ([]EntityTimePoint, error) {
	snapshots, err := tsm.GetSnapshotsByRunID(runID)
	if err != nil {
		return nil, fmt.Errorf("failed to get snapshots: %w", err)
	}

	var history []EntityTimePoint
	for _, snapshot := range snapshots {
		// Navigate nested state structure to find entity
		if stateData, ok := snapshot.State["Entities"].(map[string]interface{}); ok {
			if entityData, ok := stateData[entityURI].(map[string]interface{}); ok {
				point := EntityTimePoint{
					Step:      snapshot.Step,
					Timestamp: snapshot.Timestamp,
				}

				if status, ok := entityData["status"].(string); ok {
					point.Status = status
				}
				if util, ok := entityData["utilization"].(float64); ok {
					point.Utilization = util
				}
				if avail, ok := entityData["available"].(bool); ok {
					point.Available = avail
				}

				history = append(history, point)
			}
		}
	}

	return history, nil
}

// ExportTimeSeriesCSV exports time series data to CSV format
func ExportTimeSeriesCSV(timeSeries []TimePoint, metricName string) string {
	csv := "step,timestamp,value\n"
	for _, point := range timeSeries {
		csv += fmt.Sprintf("%d,%s,%.4f\n",
			point.Step,
			point.Timestamp.Format(time.RFC3339),
			point.Value)
	}
	return csv
}

// CalculateMetricStatistics calculates basic statistics for a metric
func CalculateMetricStatistics(timeSeries []TimePoint) map[string]float64 {
	if len(timeSeries) == 0 {
		return map[string]float64{}
	}

	stats := make(map[string]float64)

	var sum, min, max float64
	min = timeSeries[0].Value
	max = timeSeries[0].Value

	for _, point := range timeSeries {
		sum += point.Value
		if point.Value < min {
			min = point.Value
		}
		if point.Value > max {
			max = point.Value
		}
	}

	mean := sum / float64(len(timeSeries))

	// Calculate variance and standard deviation
	var sumSquaredDiff float64
	for _, point := range timeSeries {
		diff := point.Value - mean
		sumSquaredDiff += diff * diff
	}
	variance := sumSquaredDiff / float64(len(timeSeries))
	stdDev := variance
	// Approximate square root
	for i := 0; i < 10; i++ {
		stdDev = (stdDev + variance/stdDev) / 2
	}

	stats["mean"] = mean
	stats["min"] = min
	stats["max"] = max
	stats["variance"] = variance
	stats["std_dev"] = stdDev
	stats["range"] = max - min

	return stats
}
