package ml

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"time"

	storage "github.com/Mimir-AIP/Mimir-AIP-Go/pipelines/Storage"
)

// RuleEngine evaluates monitoring rules against time-series data
type RuleEngine struct {
	Storage    *storage.PersistenceBackend
	TSAnalyzer *TimeSeriesAnalyzer
}

// NewRuleEngine creates a new rule engine
func NewRuleEngine(storageBE *storage.PersistenceBackend) *RuleEngine {
	return &RuleEngine{
		Storage:    storageBE,
		TSAnalyzer: NewTimeSeriesAnalyzer(),
	}
}

// Alert represents a monitoring alert
type Alert struct {
	ID         string
	OntologyID string
	EntityID   string
	MetricName string
	AlertType  string
	Severity   string
	Message    string
	Value      float64
	Threshold  string
	Status     string
	CreatedAt  time.Time
	ResolvedAt *time.Time
}

// RuleCondition represents parsed condition from monitoring rule
type RuleCondition struct {
	Operator      string      // "<", ">", "between", "change_percent", "z_score"
	Value         interface{} // float64 or []float64 for between
	Direction     string      // "increasing", "decreasing" (for trend rules)
	WindowMinutes int         // lookback window
}

// EvaluateRules checks data against all enabled rules for a metric
func (re *RuleEngine) EvaluateRules(
	ctx context.Context,
	ontologyID string,
	entityID string,
	metricName string,
	currentValue float64,
	timeSeries *TimeSeries,
) ([]Alert, error) {
	// Get all enabled rules for this metric
	rules, err := re.Storage.GetMonitoringRules(ctx, entityID, metricName)
	if err != nil {
		return nil, fmt.Errorf("failed to get monitoring rules: %w", err)
	}

	var alerts []Alert

	for _, rule := range rules {
		// Skip disabled rules
		if !rule.IsEnabled {
			continue
		}

		// Check if we should skip due to recent duplicate alert
		isDuplicate, err := re.CheckDuplicateAlert(ctx, entityID, metricName, rule.RuleType, 24)
		if err != nil {
			return nil, fmt.Errorf("failed to check duplicate alert: %w", err)
		}
		if isDuplicate {
			continue // Skip creating duplicate alert
		}

		// Parse rule condition
		condition, err := re.parseCondition(rule.Condition)
		if err != nil {
			return nil, fmt.Errorf("failed to parse condition for rule %s: %w", rule.ID, err)
		}

		// Evaluate rule based on type
		var violated bool
		var message string

		switch rule.RuleType {
		case "threshold":
			violated, message = re.evaluateThreshold(currentValue, condition)
		case "trend":
			violated, message = re.evaluateTrend(timeSeries, condition)
		case "anomaly":
			violated, message = re.evaluateAnomaly(timeSeries, currentValue, condition)
		default:
			continue // Skip unknown rule types
		}

		// Create alert if rule violated
		if violated {
			alert := Alert{
				OntologyID: ontologyID,
				EntityID:   entityID,
				MetricName: metricName,
				AlertType:  rule.RuleType,
				Severity:   rule.Severity,
				Message:    message,
				Value:      currentValue,
				Threshold:  rule.Condition,
				Status:     "active",
				CreatedAt:  time.Now(),
			}
			alerts = append(alerts, alert)
		}
	}

	return alerts, nil
}

// evaluateThreshold checks if value violates threshold condition
func (re *RuleEngine) evaluateThreshold(value float64, condition *RuleCondition) (bool, string) {
	switch condition.Operator {
	case "<":
		threshold := condition.Value.(float64)
		if value < threshold {
			return true, fmt.Sprintf("Value %.2f is below threshold %.2f", value, threshold)
		}
	case ">":
		threshold := condition.Value.(float64)
		if value > threshold {
			return true, fmt.Sprintf("Value %.2f is above threshold %.2f", value, threshold)
		}
	case "<=":
		threshold := condition.Value.(float64)
		if value <= threshold {
			return true, fmt.Sprintf("Value %.2f is at or below threshold %.2f", value, threshold)
		}
	case ">=":
		threshold := condition.Value.(float64)
		if value >= threshold {
			return true, fmt.Sprintf("Value %.2f is at or above threshold %.2f", value, threshold)
		}
	case "between":
		bounds := condition.Value.([]float64)
		if len(bounds) == 2 {
			if value < bounds[0] || value > bounds[1] {
				return true, fmt.Sprintf("Value %.2f is outside range [%.2f, %.2f]", value, bounds[0], bounds[1])
			}
		}
	}
	return false, ""
}

// evaluateTrend checks if data shows specified trend pattern
func (re *RuleEngine) evaluateTrend(ts *TimeSeries, condition *RuleCondition) (bool, string) {
	if len(ts.Points) < 3 {
		return false, "" // Need at least 3 points for trend
	}

	// Analyze trend using TimeSeriesAnalyzer
	windowDays := 30 // default
	if condition.WindowMinutes > 0 {
		windowDays = condition.WindowMinutes / (60 * 24)
	}

	trendResult, err := re.TSAnalyzer.DetectTrend(ts, windowDays)
	if err != nil {
		return false, ""
	}

	// Check if trend matches condition
	changeThreshold := 0.0
	if val, ok := condition.Value.(float64); ok {
		changeThreshold = val
	}

	// Check direction
	if condition.Direction != "" {
		if condition.Direction == "increasing" && trendResult.Trend != TrendIncreasing {
			return false, ""
		}
		if condition.Direction == "decreasing" && trendResult.Trend != TrendDecreasing {
			return false, ""
		}
	}

	// Check if change exceeds threshold
	changePercent := math.Abs(trendResult.PercentChange)
	if changePercent >= changeThreshold {
		return true, fmt.Sprintf("Detected %s trend with %.1f%% change (threshold: %.1f%%)",
			trendResult.Trend, trendResult.PercentChange, changeThreshold)
	}

	return false, ""
}

// evaluateAnomaly checks if current value is anomalous
func (re *RuleEngine) evaluateAnomaly(ts *TimeSeries, currentValue float64, condition *RuleCondition) (bool, string) {
	if len(ts.Points) < 5 {
		return false, "" // Need at least 5 points for anomaly detection
	}

	// Use TimeSeriesAnalyzer to detect anomalies
	anomalies, err := re.TSAnalyzer.DetectAnomalies(ts, "zscore")
	if err != nil {
		return false, ""
	}

	// Check if any recent anomalies
	if len(anomalies) > 0 {
		// Check the most recent anomaly
		lastAnomaly := anomalies[len(anomalies)-1]

		// Check if it matches current value (within 1% tolerance)
		if math.Abs(lastAnomaly.Value-currentValue)/currentValue < 0.01 {
			zScoreThreshold := 3.0
			if val, ok := condition.Value.(float64); ok {
				zScoreThreshold = val
			}

			if math.Abs(lastAnomaly.Deviation) >= zScoreThreshold {
				return true, fmt.Sprintf("Anomaly detected: value %.2f has z-score %.2f (threshold: %.1f)",
					lastAnomaly.Value, lastAnomaly.Deviation, zScoreThreshold)
			}
		}
	}

	return false, ""
}

// parseCondition parses JSON condition string into RuleCondition struct
func (re *RuleEngine) parseCondition(conditionJSON string) (*RuleCondition, error) {
	var raw map[string]interface{}
	if err := json.Unmarshal([]byte(conditionJSON), &raw); err != nil {
		return nil, fmt.Errorf("failed to parse condition JSON: %w", err)
	}

	condition := &RuleCondition{}

	// Parse threshold operators
	if val, ok := raw["<"]; ok {
		condition.Operator = "<"
		condition.Value = val.(float64)
		return condition, nil
	}
	if val, ok := raw[">"]; ok {
		condition.Operator = ">"
		condition.Value = val.(float64)
		return condition, nil
	}
	if val, ok := raw["<="]; ok {
		condition.Operator = "<="
		condition.Value = val.(float64)
		return condition, nil
	}
	if val, ok := raw[">="]; ok {
		condition.Operator = ">="
		condition.Value = val.(float64)
		return condition, nil
	}
	if val, ok := raw["between"]; ok {
		condition.Operator = "between"
		arr := val.([]interface{})
		condition.Value = []float64{arr[0].(float64), arr[1].(float64)}
		return condition, nil
	}

	// Parse trend conditions
	if val, ok := raw["change_percent"]; ok {
		condition.Operator = "change_percent"
		condition.Value = val.(float64)
		if dir, ok := raw["direction"].(string); ok {
			condition.Direction = dir
		}
		if window, ok := raw["window_minutes"].(float64); ok {
			condition.WindowMinutes = int(window)
		}
		return condition, nil
	}

	// Parse anomaly conditions
	if val, ok := raw["z_score"]; ok {
		condition.Operator = "z_score"
		condition.Value = val.(float64)
		return condition, nil
	}

	return nil, fmt.Errorf("unknown condition format: %s", conditionJSON)
}

// CheckDuplicateAlert checks if similar alert exists within specified hours
func (re *RuleEngine) CheckDuplicateAlert(
	ctx context.Context,
	entityID string,
	metricName string,
	alertType string,
	withinHours int,
) (bool, error) {
	// Query alerts table for similar alert in last N hours
	query := `
		SELECT COUNT(*) 
		FROM alerts 
		WHERE entity_id = ? 
		  AND metric_name = ? 
		  AND alert_type = ?
		  AND status = 'active'
		  AND created_at > datetime('now', '-' || ? || ' hours')
	`

	var count int
	// We need to access the database through the storage backend's DB connection
	// Since PersistenceBackend doesn't expose QueryRowContext directly, we'll need to add a method or use GetDB()
	// For now, let's assume we can access it (will be fixed in integration)
	row := re.Storage.GetDB().QueryRowContext(ctx, query, entityID, metricName, alertType, withinHours)
	err := row.Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check duplicate alert: %w", err)
	}

	return count > 0, nil
}

// SaveAlert saves an alert to the database
func (re *RuleEngine) SaveAlert(ctx context.Context, alert *Alert) error {
	query := `
		INSERT INTO alerts (
			ontology_id, entity_id, metric_name, alert_type, severity,
			message, value, threshold, status, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := re.Storage.GetDB().ExecContext(ctx, query,
		alert.OntologyID, alert.EntityID, alert.MetricName, alert.AlertType,
		alert.Severity, alert.Message, alert.Value, alert.Threshold,
		alert.Status, alert.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to save alert: %w", err)
	}

	return nil
}

// GetActiveAlerts retrieves all active alerts
func (re *RuleEngine) GetActiveAlerts(ctx context.Context, ontologyID string) ([]Alert, error) {
	query := `
		SELECT ontology_id, entity_id, metric_name, alert_type, severity,
			message, value, threshold, status, created_at
		FROM alerts
		WHERE status = 'active'
	`
	args := []interface{}{}

	if ontologyID != "" {
		query += " AND ontology_id = ?"
		args = append(args, ontologyID)
	}

	query += " ORDER BY created_at DESC"

	rows, err := re.Storage.GetDB().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get active alerts: %w", err)
	}
	defer rows.Close()

	var alerts []Alert
	for rows.Next() {
		var alert Alert
		err := rows.Scan(
			&alert.OntologyID, &alert.EntityID, &alert.MetricName, &alert.AlertType,
			&alert.Severity, &alert.Message, &alert.Value, &alert.Threshold,
			&alert.Status, &alert.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan alert: %w", err)
		}
		alerts = append(alerts, alert)
	}

	return alerts, rows.Err()
}

// ResolveAlert marks an alert as resolved
func (re *RuleEngine) ResolveAlert(ctx context.Context, alertID string) error {
	now := time.Now()
	query := `
		UPDATE alerts
		SET status = 'resolved', resolved_at = ?
		WHERE id = ?
	`
	_, err := re.Storage.GetDB().ExecContext(ctx, query, now, alertID)
	if err != nil {
		return fmt.Errorf("failed to resolve alert: %w", err)
	}
	return nil
}
