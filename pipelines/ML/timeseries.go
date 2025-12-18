package ml

import (
	"fmt"
	"math"
	"sort"
	"time"
)

// TimeSeriesPoint represents a single data point in a time series
type TimeSeriesPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Value     float64   `json:"value"`
}

// TimeSeries represents a time series for a specific entity/metric
type TimeSeries struct {
	EntityID   string            `json:"entity_id"`   // e.g., "product_123" or "supply_medical"
	MetricName string            `json:"metric_name"` // e.g., "price", "stock_level", "deliveries"
	Points     []TimeSeriesPoint `json:"points"`
	Metadata   map[string]string `json:"metadata,omitempty"` // Additional context
}

// TrendType represents the direction of a trend
type TrendType string

const (
	TrendIncreasing TrendType = "increasing"
	TrendDecreasing TrendType = "decreasing"
	TrendStable     TrendType = "stable"
	TrendVolatile   TrendType = "volatile"
)

// TrendResult contains the results of trend analysis
type TrendResult struct {
	Trend         TrendType `json:"trend"`
	Slope         float64   `json:"slope"`          // Rate of change per day
	PercentChange float64   `json:"percent_change"` // % change over window
	Confidence    float64   `json:"confidence"`     // 0-1, based on R²
	StartValue    float64   `json:"start_value"`
	EndValue      float64   `json:"end_value"`
	WindowDays    int       `json:"window_days"`
	IsSignificant bool      `json:"is_significant"` // Is change meaningful?
}

// AnomalyPoint represents an anomaly detected in time series
type AnomalyPoint struct {
	Timestamp     time.Time `json:"timestamp"`
	Value         float64   `json:"value"`
	ExpectedValue float64   `json:"expected_value"`
	Deviation     float64   `json:"deviation"`    // Standard deviations from normal
	Severity      string    `json:"severity"`     // low, medium, high, critical
	AnomalyType   string    `json:"anomaly_type"` // spike, drop, outlier, sudden_change
}

// ForecastPoint represents a forecasted value
type ForecastPoint struct {
	Timestamp       time.Time `json:"timestamp"`
	ForecastedValue float64   `json:"forecasted_value"`
	LowerBound      float64   `json:"lower_bound"` // Confidence interval
	UpperBound      float64   `json:"upper_bound"` // Confidence interval
	Confidence      float64   `json:"confidence"`  // 0-1
}

// TimeSeriesAnalyzer provides time series analysis capabilities
type TimeSeriesAnalyzer struct {
	MinDataPoints              int     // Minimum points needed for analysis
	AnomalySigmaThreshold      float64 // Standard deviations for anomaly detection
	TrendSignificanceThreshold float64 // Minimum R² for significant trend
}

// NewTimeSeriesAnalyzer creates a new analyzer with default settings
func NewTimeSeriesAnalyzer() *TimeSeriesAnalyzer {
	return &TimeSeriesAnalyzer{
		MinDataPoints:              7,   // At least a week of data
		AnomalySigmaThreshold:      2.5, // 2.5 std devs
		TrendSignificanceThreshold: 0.6, // R² > 0.6 considered significant
	}
}

// DetectTrend analyzes the trend in a time series
func (tsa *TimeSeriesAnalyzer) DetectTrend(ts *TimeSeries, windowDays int) (*TrendResult, error) {
	if len(ts.Points) < tsa.MinDataPoints {
		return nil, fmt.Errorf("insufficient data points: need %d, got %d", tsa.MinDataPoints, len(ts.Points))
	}

	// Sort by timestamp
	sortedPoints := make([]TimeSeriesPoint, len(ts.Points))
	copy(sortedPoints, ts.Points)
	sort.Slice(sortedPoints, func(i, j int) bool {
		return sortedPoints[i].Timestamp.Before(sortedPoints[j].Timestamp)
	})

	// Filter to window if specified
	var windowPoints []TimeSeriesPoint
	if windowDays > 0 {
		cutoffTime := time.Now().AddDate(0, 0, -windowDays)
		for _, p := range sortedPoints {
			if p.Timestamp.After(cutoffTime) {
				windowPoints = append(windowPoints, p)
			}
		}
		if len(windowPoints) < tsa.MinDataPoints {
			return nil, fmt.Errorf("insufficient points in window: need %d, got %d", tsa.MinDataPoints, len(windowPoints))
		}
	} else {
		windowPoints = sortedPoints
	}

	// Perform linear regression
	slope, _, rSquared := tsa.linearRegression(windowPoints)

	// Calculate percent change
	startValue := windowPoints[0].Value
	endValue := windowPoints[len(windowPoints)-1].Value
	percentChange := 0.0
	if startValue != 0 {
		percentChange = ((endValue - startValue) / math.Abs(startValue)) * 100
	}

	// Determine trend type
	trend := TrendStable
	isSignificant := rSquared >= tsa.TrendSignificanceThreshold

	if isSignificant {
		if math.Abs(slope) < 0.01 {
			trend = TrendStable
		} else if slope > 0 {
			trend = TrendIncreasing
		} else {
			trend = TrendDecreasing
		}
	} else {
		// Low R² indicates high volatility
		if rSquared < 0.3 {
			trend = TrendVolatile
		}
	}

	return &TrendResult{
		Trend:         trend,
		Slope:         slope,
		PercentChange: percentChange,
		Confidence:    rSquared,
		StartValue:    startValue,
		EndValue:      endValue,
		WindowDays:    windowDays,
		IsSignificant: isSignificant,
	}, nil
}

// linearRegression performs simple linear regression on time series data
// Returns: slope, intercept, R²
func (tsa *TimeSeriesAnalyzer) linearRegression(points []TimeSeriesPoint) (float64, float64, float64) {
	n := float64(len(points))
	if n == 0 {
		return 0, 0, 0
	}

	// Convert timestamps to days since first point
	baseTime := points[0].Timestamp
	var x, y []float64
	for _, p := range points {
		daysSince := p.Timestamp.Sub(baseTime).Hours() / 24.0
		x = append(x, daysSince)
		y = append(y, p.Value)
	}

	// Calculate means
	var sumX, sumY float64
	for i := range x {
		sumX += x[i]
		sumY += y[i]
	}
	meanX := sumX / n
	meanY := sumY / n

	// Calculate slope and intercept
	var numerator, denominator float64
	for i := range x {
		numerator += (x[i] - meanX) * (y[i] - meanY)
		denominator += (x[i] - meanX) * (x[i] - meanX)
	}

	slope := 0.0
	if denominator != 0 {
		slope = numerator / denominator
	}
	intercept := meanY - slope*meanX

	// Calculate R²
	var ssRes, ssTot float64
	for i := range x {
		predicted := slope*x[i] + intercept
		ssRes += (y[i] - predicted) * (y[i] - predicted)
		ssTot += (y[i] - meanY) * (y[i] - meanY)
	}

	rSquared := 0.0
	if ssTot != 0 {
		rSquared = 1 - (ssRes / ssTot)
	}

	return slope, intercept, rSquared
}

// DetectAnomalies detects anomalies in a time series using statistical methods
func (tsa *TimeSeriesAnalyzer) DetectAnomalies(ts *TimeSeries, method string) ([]AnomalyPoint, error) {
	if len(ts.Points) < tsa.MinDataPoints {
		return nil, fmt.Errorf("insufficient data points: need %d, got %d", tsa.MinDataPoints, len(ts.Points))
	}

	switch method {
	case "zscore":
		return tsa.detectAnomaliesZScore(ts)
	case "moving_average":
		return tsa.detectAnomaliesMovingAverage(ts, 7) // 7-day window
	case "iqr":
		return tsa.detectAnomaliesIQR(ts)
	default:
		return tsa.detectAnomaliesZScore(ts) // Default to z-score
	}
}

// detectAnomaliesZScore detects anomalies using z-score (standard deviations from mean)
func (tsa *TimeSeriesAnalyzer) detectAnomaliesZScore(ts *TimeSeries) ([]AnomalyPoint, error) {
	// Calculate mean and standard deviation
	var sum, sumSq float64
	n := float64(len(ts.Points))

	for _, p := range ts.Points {
		sum += p.Value
		sumSq += p.Value * p.Value
	}

	mean := sum / n
	variance := (sumSq / n) - (mean * mean)
	stdDev := math.Sqrt(variance)

	if stdDev == 0 {
		return nil, nil // No variation, no anomalies
	}

	// Find anomalies
	var anomalies []AnomalyPoint
	for _, p := range ts.Points {
		zScore := (p.Value - mean) / stdDev
		absZ := math.Abs(zScore)

		if absZ > tsa.AnomalySigmaThreshold {
			severity := "low"
			if absZ > 4 {
				severity = "critical"
			} else if absZ > 3.5 {
				severity = "high"
			} else if absZ > 3 {
				severity = "medium"
			}

			anomalyType := "outlier"
			if zScore > 0 {
				anomalyType = "spike"
			} else {
				anomalyType = "drop"
			}

			anomalies = append(anomalies, AnomalyPoint{
				Timestamp:     p.Timestamp,
				Value:         p.Value,
				ExpectedValue: mean,
				Deviation:     absZ,
				Severity:      severity,
				AnomalyType:   anomalyType,
			})
		}
	}

	return anomalies, nil
}

// detectAnomaliesMovingAverage detects anomalies using moving average deviation
func (tsa *TimeSeriesAnalyzer) detectAnomaliesMovingAverage(ts *TimeSeries, windowSize int) ([]AnomalyPoint, error) {
	if len(ts.Points) < windowSize {
		return nil, fmt.Errorf("insufficient data for window size %d", windowSize)
	}

	// Sort by timestamp
	sortedPoints := make([]TimeSeriesPoint, len(ts.Points))
	copy(sortedPoints, ts.Points)
	sort.Slice(sortedPoints, func(i, j int) bool {
		return sortedPoints[i].Timestamp.Before(sortedPoints[j].Timestamp)
	})

	var anomalies []AnomalyPoint

	// Calculate moving average and detect deviations
	for i := windowSize; i < len(sortedPoints); i++ {
		// Calculate moving average of previous window
		var windowSum float64
		for j := i - windowSize; j < i; j++ {
			windowSum += sortedPoints[j].Value
		}
		movingAvg := windowSum / float64(windowSize)

		// Calculate standard deviation of window
		var windowSumSq float64
		for j := i - windowSize; j < i; j++ {
			diff := sortedPoints[j].Value - movingAvg
			windowSumSq += diff * diff
		}
		stdDev := math.Sqrt(windowSumSq / float64(windowSize))

		if stdDev == 0 {
			continue
		}

		// Check if current point is an anomaly
		currentPoint := sortedPoints[i]
		deviation := math.Abs(currentPoint.Value-movingAvg) / stdDev

		if deviation > tsa.AnomalySigmaThreshold {
			severity := "low"
			if deviation > 4 {
				severity = "critical"
			} else if deviation > 3.5 {
				severity = "high"
			} else if deviation > 3 {
				severity = "medium"
			}

			anomalyType := "sudden_change"
			if currentPoint.Value > movingAvg {
				anomalyType = "spike"
			} else {
				anomalyType = "drop"
			}

			anomalies = append(anomalies, AnomalyPoint{
				Timestamp:     currentPoint.Timestamp,
				Value:         currentPoint.Value,
				ExpectedValue: movingAvg,
				Deviation:     deviation,
				Severity:      severity,
				AnomalyType:   anomalyType,
			})
		}
	}

	return anomalies, nil
}

// detectAnomaliesIQR detects anomalies using Interquartile Range (IQR) method
func (tsa *TimeSeriesAnalyzer) detectAnomaliesIQR(ts *TimeSeries) ([]AnomalyPoint, error) {
	// Extract values and sort
	values := make([]float64, len(ts.Points))
	for i, p := range ts.Points {
		values[i] = p.Value
	}
	sort.Float64s(values)

	n := len(values)
	q1 := values[n/4]
	q3 := values[3*n/4]
	iqr := q3 - q1

	lowerBound := q1 - 1.5*iqr
	upperBound := q3 + 1.5*iqr

	var anomalies []AnomalyPoint
	for _, p := range ts.Points {
		if p.Value < lowerBound || p.Value > upperBound {
			deviation := 0.0
			expectedValue := (q1 + q3) / 2 // Median
			if p.Value < lowerBound {
				deviation = (lowerBound - p.Value) / iqr
			} else {
				deviation = (p.Value - upperBound) / iqr
			}

			severity := "medium"
			if deviation > 3 {
				severity = "critical"
			} else if deviation > 2 {
				severity = "high"
			} else if deviation < 1 {
				severity = "low"
			}

			anomalyType := "outlier"
			if p.Value > upperBound {
				anomalyType = "spike"
			} else {
				anomalyType = "drop"
			}

			anomalies = append(anomalies, AnomalyPoint{
				Timestamp:     p.Timestamp,
				Value:         p.Value,
				ExpectedValue: expectedValue,
				Deviation:     deviation,
				Severity:      severity,
				AnomalyType:   anomalyType,
			})
		}
	}

	return anomalies, nil
}

// Forecast predicts future values using simple linear extrapolation
func (tsa *TimeSeriesAnalyzer) Forecast(ts *TimeSeries, daysAhead int) ([]ForecastPoint, error) {
	if len(ts.Points) < tsa.MinDataPoints {
		return nil, fmt.Errorf("insufficient data points: need %d, got %d", tsa.MinDataPoints, len(ts.Points))
	}

	// Sort by timestamp
	sortedPoints := make([]TimeSeriesPoint, len(ts.Points))
	copy(sortedPoints, ts.Points)
	sort.Slice(sortedPoints, func(i, j int) bool {
		return sortedPoints[i].Timestamp.Before(sortedPoints[j].Timestamp)
	})

	// Perform linear regression on recent data
	slope, intercept, rSquared := tsa.linearRegression(sortedPoints)

	// Calculate standard error for confidence intervals
	baseTime := sortedPoints[0].Timestamp
	var residuals []float64
	for _, p := range sortedPoints {
		daysSince := p.Timestamp.Sub(baseTime).Hours() / 24.0
		predicted := slope*daysSince + intercept
		residuals = append(residuals, p.Value-predicted)
	}

	// Calculate standard deviation of residuals
	var sumSq float64
	for _, r := range residuals {
		sumSq += r * r
	}
	stdError := math.Sqrt(sumSq / float64(len(residuals)))

	// Generate forecasts
	lastPoint := sortedPoints[len(sortedPoints)-1]
	lastDays := lastPoint.Timestamp.Sub(baseTime).Hours() / 24.0

	var forecasts []ForecastPoint
	for i := 1; i <= daysAhead; i++ {
		forecastTime := lastPoint.Timestamp.AddDate(0, 0, i)
		daysSince := lastDays + float64(i)
		forecastedValue := slope*daysSince + intercept

		// Confidence decreases with distance into future
		confidence := rSquared * math.Exp(-float64(i)*0.1) // Exponential decay

		// Confidence interval widens with distance (1.96 for 95% CI)
		margin := 1.96 * stdError * math.Sqrt(1+float64(i)*0.1)

		forecasts = append(forecasts, ForecastPoint{
			Timestamp:       forecastTime,
			ForecastedValue: forecastedValue,
			LowerBound:      forecastedValue - margin,
			UpperBound:      forecastedValue + margin,
			Confidence:      confidence,
		})
	}

	return forecasts, nil
}

// CompareTimeSeries compares two time series and returns insights
func (tsa *TimeSeriesAnalyzer) CompareTimeSeries(ts1, ts2 *TimeSeries) (*ComparisonResult, error) {
	if len(ts1.Points) < tsa.MinDataPoints || len(ts2.Points) < tsa.MinDataPoints {
		return nil, fmt.Errorf("insufficient data in one or both series")
	}

	// Analyze trends for both
	trend1, err := tsa.DetectTrend(ts1, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze first series: %w", err)
	}

	trend2, err := tsa.DetectTrend(ts2, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze second series: %w", err)
	}

	// Calculate correlation if timestamps align
	correlation := tsa.calculateCorrelation(ts1, ts2)

	return &ComparisonResult{
		Series1Trend: trend1,
		Series2Trend: trend2,
		Correlation:  correlation,
		IsConverging: trend1.Slope > 0 && trend2.Slope < 0 || trend1.Slope < 0 && trend2.Slope > 0,
		IsDiverging:  (trend1.Slope > 0 && trend2.Slope > 0 && trend1.Slope > trend2.Slope) || (trend1.Slope < 0 && trend2.Slope < 0 && trend1.Slope < trend2.Slope),
	}, nil
}

// ComparisonResult contains the results of comparing two time series
type ComparisonResult struct {
	Series1Trend *TrendResult `json:"series1_trend"`
	Series2Trend *TrendResult `json:"series2_trend"`
	Correlation  float64      `json:"correlation"`   // -1 to 1
	IsConverging bool         `json:"is_converging"` // Trends moving toward each other
	IsDiverging  bool         `json:"is_diverging"`  // Trends moving apart
}

// calculateCorrelation calculates Pearson correlation between two time series
func (tsa *TimeSeriesAnalyzer) calculateCorrelation(ts1, ts2 *TimeSeries) float64 {
	// Find overlapping timestamps (simplified - assumes exact matches)
	values1 := make(map[time.Time]float64)
	for _, p := range ts1.Points {
		// Round to day for matching
		day := time.Date(p.Timestamp.Year(), p.Timestamp.Month(), p.Timestamp.Day(), 0, 0, 0, 0, time.UTC)
		values1[day] = p.Value
	}

	var x, y []float64
	for _, p := range ts2.Points {
		day := time.Date(p.Timestamp.Year(), p.Timestamp.Month(), p.Timestamp.Day(), 0, 0, 0, 0, time.UTC)
		if val1, exists := values1[day]; exists {
			x = append(x, val1)
			y = append(y, p.Value)
		}
	}

	if len(x) < 2 {
		return 0 // Not enough overlapping points
	}

	// Calculate Pearson correlation
	n := float64(len(x))
	var sumX, sumY, sumXY, sumX2, sumY2 float64
	for i := range x {
		sumX += x[i]
		sumY += y[i]
		sumXY += x[i] * y[i]
		sumX2 += x[i] * x[i]
		sumY2 += y[i] * y[i]
	}

	numerator := n*sumXY - sumX*sumY
	denominator := math.Sqrt((n*sumX2 - sumX*sumX) * (n*sumY2 - sumY*sumY))

	if denominator == 0 {
		return 0
	}

	return numerator / denominator
}

// AggregateByCategory aggregates multiple time series (e.g., all products in a category)
func AggregateTimeSeries(series []*TimeSeries, aggregationType string) (*TimeSeries, error) {
	if len(series) == 0 {
		return nil, fmt.Errorf("no series to aggregate")
	}

	// Collect all unique timestamps
	timestampMap := make(map[time.Time][]float64)
	for _, ts := range series {
		for _, p := range ts.Points {
			// Round to day for aggregation
			day := time.Date(p.Timestamp.Year(), p.Timestamp.Month(), p.Timestamp.Day(), 0, 0, 0, 0, time.UTC)
			timestampMap[day] = append(timestampMap[day], p.Value)
		}
	}

	// Aggregate values for each timestamp
	var aggregatedPoints []TimeSeriesPoint
	for timestamp, values := range timestampMap {
		var aggregatedValue float64
		switch aggregationType {
		case "sum":
			for _, v := range values {
				aggregatedValue += v
			}
		case "mean", "average":
			for _, v := range values {
				aggregatedValue += v
			}
			aggregatedValue /= float64(len(values))
		case "min":
			aggregatedValue = values[0]
			for _, v := range values {
				if v < aggregatedValue {
					aggregatedValue = v
				}
			}
		case "max":
			aggregatedValue = values[0]
			for _, v := range values {
				if v > aggregatedValue {
					aggregatedValue = v
				}
			}
		case "median":
			sortedValues := make([]float64, len(values))
			copy(sortedValues, values)
			sort.Float64s(sortedValues)
			mid := len(sortedValues) / 2
			if len(sortedValues)%2 == 0 {
				aggregatedValue = (sortedValues[mid-1] + sortedValues[mid]) / 2
			} else {
				aggregatedValue = sortedValues[mid]
			}
		default:
			return nil, fmt.Errorf("unknown aggregation type: %s", aggregationType)
		}

		aggregatedPoints = append(aggregatedPoints, TimeSeriesPoint{
			Timestamp: timestamp,
			Value:     aggregatedValue,
		})
	}

	// Sort by timestamp
	sort.Slice(aggregatedPoints, func(i, j int) bool {
		return aggregatedPoints[i].Timestamp.Before(aggregatedPoints[j].Timestamp)
	})

	return &TimeSeries{
		EntityID:   "aggregated",
		MetricName: fmt.Sprintf("%s_aggregated", series[0].MetricName),
		Points:     aggregatedPoints,
		Metadata: map[string]string{
			"aggregation_type": aggregationType,
			"series_count":     fmt.Sprintf("%d", len(series)),
		},
	}, nil
}
