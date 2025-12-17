package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProfileColumnData_Numeric(t *testing.T) {
	columnName := "age"
	values := []any{25, 30, 35, 40, 45, 50}

	profile := profileColumnData(columnName, values)

	assert.Equal(t, "age", profile.ColumnName)
	assert.Equal(t, "numeric", profile.DataType)
	assert.Equal(t, 6, profile.TotalCount)
	assert.Equal(t, 6, profile.DistinctCount)
	assert.Equal(t, 0, profile.NullCount)
	assert.InDelta(t, 37.5, profile.Mean, 0.01)
	assert.InDelta(t, 37.5, profile.Median, 0.01)
	assert.InDelta(t, 25.0, profile.MinValue, 0.01)
	assert.InDelta(t, 50.0, profile.MaxValue, 0.01)
	assert.Greater(t, profile.DataQualityScore, 0.9)
}

func TestProfileColumnData_String(t *testing.T) {
	columnName := "name"
	values := []any{"Alice", "Bob", "Charlie", "David", "Eve"}

	profile := profileColumnData(columnName, values)

	assert.Equal(t, "name", profile.ColumnName)
	assert.Equal(t, "string", profile.DataType)
	assert.Equal(t, 5, profile.TotalCount)
	assert.Equal(t, 5, profile.DistinctCount)
	assert.Equal(t, 0, profile.NullCount)
	assert.Equal(t, 3, profile.MinLength)
	assert.Equal(t, 7, profile.MaxLength)
	assert.InDelta(t, 4.6, profile.AvgLength, 0.1)
}

func TestProfileColumnData_WithNulls(t *testing.T) {
	columnName := "email"
	values := []any{"test@example.com", nil, "user@test.com", "", "admin@example.com"}

	profile := profileColumnData(columnName, values)

	assert.Equal(t, "email", profile.ColumnName)
	assert.Equal(t, 5, profile.TotalCount)
	assert.Equal(t, 3, profile.DistinctCount)
	assert.Equal(t, 2, profile.NullCount)
	assert.InDelta(t, 40.0, profile.NullPercent, 0.1)
	assert.Greater(t, len(profile.QualityIssues), 0)
}

func TestProfileColumnData_HighCardinality(t *testing.T) {
	columnName := "id"
	values := []any{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}

	profile := profileColumnData(columnName, values)

	assert.Equal(t, "id", profile.ColumnName)
	assert.Equal(t, "numeric", profile.DataType)
	assert.Equal(t, 10, profile.TotalCount)
	assert.Equal(t, 10, profile.DistinctCount)
	assert.InDelta(t, 100.0, profile.DistinctPercent, 0.1)
	assert.Greater(t, profile.DataQualityScore, 0.9)
}

func TestProfileColumnData_LowCardinality(t *testing.T) {
	columnName := "status"
	values := []any{"active", "active", "active", "inactive", "active", "active",
		"active", "active", "inactive", "active", "active", "active"}

	profile := profileColumnData(columnName, values)

	assert.Equal(t, "status", profile.ColumnName)
	assert.Equal(t, 12, profile.TotalCount)
	assert.Equal(t, 2, profile.DistinctCount)
	assert.Contains(t, profile.QualityIssues, "Very low cardinality (2 distinct values)")
}

func TestProfileColumnData_DominantValue(t *testing.T) {
	columnName := "category"
	values := make([]any, 100)
	for i := 0; i < 95; i++ {
		values[i] = "A"
	}
	for i := 95; i < 100; i++ {
		values[i] = "B"
	}

	profile := profileColumnData(columnName, values)

	assert.Equal(t, 100, profile.TotalCount)
	assert.Equal(t, 2, profile.DistinctCount)
	assert.Greater(t, len(profile.TopValues), 0)
	assert.InDelta(t, 95.0, profile.TopValues[0].Frequency, 0.1)

	// Check if dominant value issue is present
	found := false
	for _, issue := range profile.QualityIssues {
		if containsString(issue, "Single value dominates") {
			found = true
			break
		}
	}
	assert.True(t, found, "Expected dominant value quality issue")
}

func TestProfileColumnData_EmptyDataset(t *testing.T) {
	columnName := "empty"
	values := []any{}

	profile := profileColumnData(columnName, values)

	assert.Equal(t, "empty", profile.ColumnName)
	assert.Equal(t, "unknown", profile.DataType)
	assert.Equal(t, 0, profile.TotalCount)
	assert.Equal(t, 0.0, profile.DataQualityScore)
	assert.Contains(t, profile.QualityIssues, "No data available")
}

func TestGetTopValues(t *testing.T) {
	valueCounts := map[string]int{
		"apple":      10,
		"banana":     25,
		"cherry":     5,
		"date":       15,
		"elderberry": 3,
	}
	totalCount := 58

	topValues := getTopValues(valueCounts, totalCount, 3)

	assert.Equal(t, 3, len(topValues))
	assert.Equal(t, "banana", topValues[0].Value)
	assert.Equal(t, 25, topValues[0].Count)
	assert.InDelta(t, 43.1, topValues[0].Frequency, 0.1)
}

func TestCalculateMean(t *testing.T) {
	values := []float64{10.0, 20.0, 30.0, 40.0, 50.0}
	mean := calculateMean(values)
	assert.InDelta(t, 30.0, mean, 0.01)
}

func TestCalculateMedian_OddLength(t *testing.T) {
	values := []float64{1.0, 3.0, 5.0, 7.0, 9.0}
	median := calculateMedian(values)
	assert.InDelta(t, 5.0, median, 0.01)
}

func TestCalculateMedian_EvenLength(t *testing.T) {
	values := []float64{1.0, 2.0, 3.0, 4.0}
	median := calculateMedian(values)
	assert.InDelta(t, 2.5, median, 0.01)
}

func TestCalculateStdDev(t *testing.T) {
	values := []float64{2.0, 4.0, 4.0, 4.0, 5.0, 5.0, 7.0, 9.0}
	mean := calculateMean(values)
	stdDev := calculateStdDev(values, mean)
	assert.Greater(t, stdDev, 0.0)
	assert.InDelta(t, 2.0, stdDev, 0.1)
}

func TestParseNumeric(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected float64
		wantErr  bool
	}{
		{"int", 42, 42.0, false},
		{"int64", int64(42), 42.0, false},
		{"float64", 42.5, 42.5, false},
		{"string number", "42.5", 42.5, false},
		{"string non-number", "abc", 0.0, true},
		{"bool", true, 0.0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseNumeric(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.InDelta(t, tt.expected, result, 0.01)
			}
		})
	}
}

func TestDetectQualityIssues_HighNullRate(t *testing.T) {
	profile := ColumnProfile{
		ColumnName:      "test",
		NullPercent:     60.0,
		DistinctCount:   10,
		DistinctPercent: 50.0,
	}

	issues := detectQualityIssues(profile, 40)
	assert.Greater(t, len(issues), 0)
	found := false
	for _, issue := range issues {
		if containsString(issue, "High null rate") {
			found = true
			break
		}
	}
	assert.True(t, found, "Expected high null rate issue")
}

func TestCalculateDataQualityScore(t *testing.T) {
	tests := []struct {
		name          string
		profile       ColumnProfile
		nonNullCount  int
		expectedRange [2]float64 // min, max
	}{
		{
			name: "perfect data",
			profile: ColumnProfile{
				ColumnName:      "id",
				NullPercent:     0.0,
				DistinctCount:   100,
				DistinctPercent: 100.0,
				QualityIssues:   []string{},
			},
			nonNullCount:  100,
			expectedRange: [2]float64{0.95, 1.0},
		},
		{
			name: "high null rate",
			profile: ColumnProfile{
				ColumnName:      "optional_field",
				NullPercent:     80.0,
				DistinctCount:   5,
				DistinctPercent: 50.0,
				QualityIssues:   []string{"High null rate (80.0%)"},
			},
			nonNullCount:  20,
			expectedRange: [2]float64{0.3, 0.7},
		},
		{
			name: "moderate quality",
			profile: ColumnProfile{
				ColumnName:      "category",
				NullPercent:     10.0,
				DistinctCount:   8,
				DistinctPercent: 80.0,
				QualityIssues:   []string{},
			},
			nonNullCount:  90,
			expectedRange: [2]float64{0.8, 1.0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := calculateDataQualityScore(tt.profile, tt.nonNullCount)
			assert.GreaterOrEqual(t, score, tt.expectedRange[0],
				"Score should be >= %v, got %v", tt.expectedRange[0], score)
			assert.LessOrEqual(t, score, tt.expectedRange[1],
				"Score should be <= %v, got %v", tt.expectedRange[1], score)
		})
	}
}

func TestProfileDataset(t *testing.T) {
	// Create enough rows to trigger primary key suggestion (>10 rows)
	rows := make([]any, 15)
	for i := 0; i < 15; i++ {
		email := "user" + string(rune(i%5)) + "@example.com"
		if i == 2 {
			email = "" // Add one null email
		}
		rows[i] = map[string]any{
			"id":    i + 1,
			"name":  "User" + string(rune(65+i)),
			"age":   25 + i,
			"email": email,
		}
	}

	data := map[string]any{
		"rows": rows,
	}

	summary := profileDataset(data, 1000)

	assert.Equal(t, 15, summary.TotalRows)
	assert.Equal(t, 4, summary.TotalColumns)
	assert.Greater(t, len(summary.ColumnProfiles), 0)
	assert.Greater(t, summary.OverallQualityScore, 0.0)

	// Check if id is suggested as primary key (100% distinct, 0% null)
	assert.Contains(t, summary.SuggestedPrimaryKeys, "id")
}

func TestProfileDataset_EmptyData(t *testing.T) {
	data := map[string]any{
		"rows": []any{},
	}

	summary := profileDataset(data, 1000)

	assert.Equal(t, 0, summary.TotalRows)
	assert.Equal(t, 0, len(summary.ColumnProfiles))
}

func TestProfileDataset_Sampling(t *testing.T) {
	// Create a large dataset
	rows := make([]any, 15000)
	for i := 0; i < 15000; i++ {
		rows[i] = map[string]any{
			"id":   i + 1,
			"name": "User" + string(rune(i)),
		}
	}

	data := map[string]any{
		"rows": rows,
	}

	// Profile with sampling (should sample 10000 rows)
	summary := profileDataset(data, 10000)

	assert.Equal(t, 15000, summary.TotalRows)
	assert.Equal(t, 2, summary.TotalColumns)
	assert.Greater(t, len(summary.ColumnProfiles), 0)
}

func TestSampleRows(t *testing.T) {
	rows := make([]any, 1000)
	for i := 0; i < 1000; i++ {
		rows[i] = i
	}

	sampled := sampleRows(rows, 100)

	assert.Equal(t, 100, len(sampled))
	// First element should be 0
	assert.Equal(t, 0, sampled[0])
}

func TestSampleRows_SmallDataset(t *testing.T) {
	rows := []any{1, 2, 3}
	sampled := sampleRows(rows, 10)

	// Should return all rows when sample size > row count
	assert.Equal(t, 3, len(sampled))
}

// Helper function to check if a string contains a substring
func containsString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
