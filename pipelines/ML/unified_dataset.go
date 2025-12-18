package ml

import (
	"fmt"
	"time"
)

// UnifiedDataset represents a generic tabular/structured dataset from any source
// This is the universal intermediate format that all data adapters must produce
type UnifiedDataset struct {
	// Source metadata
	Source     string                 `json:"source"`      // "csv", "excel", "json", "pipeline", "custom_plugin"
	SourceInfo map[string]interface{} `json:"source_info"` // Plugin-specific metadata

	// Schema information
	Columns     []ColumnMetadata `json:"columns"`
	ColumnCount int              `json:"column_count"`

	// Data rows (generic map representation)
	Rows     []map[string]interface{} `json:"rows"`
	RowCount int                      `json:"row_count"`

	// Optional time-series detection
	TimeSeriesConfig *TimeSeriesInfo `json:"timeseries_config,omitempty"`

	// Metadata
	CreatedAt time.Time              `json:"created_at"`
	Metadata  map[string]interface{} `json:"metadata"`
}

// ColumnMetadata describes a column's characteristics
type ColumnMetadata struct {
	Name         string `json:"name"`
	Index        int    `json:"index"`
	DataType     string `json:"data_type"` // "string", "numeric", "datetime", "boolean", "mixed"
	IsNumeric    bool   `json:"is_numeric"`
	IsDateTime   bool   `json:"is_datetime"`
	IsTimeSeries bool   `json:"is_timeseries"` // True if this is a time/date column
	HasNulls     bool   `json:"has_nulls"`
	NullCount    int    `json:"null_count"`

	// Statistical summaries (for numeric columns)
	Stats *ColumnStats `json:"stats,omitempty"`

	// Sample values for preview
	SampleValues []interface{} `json:"sample_values"`
	UniqueCount  int           `json:"unique_count,omitempty"` // For categorical analysis
}

// ColumnStats contains statistical information for numeric columns
type ColumnStats struct {
	Min    float64 `json:"min"`
	Max    float64 `json:"max"`
	Mean   float64 `json:"mean"`
	Median float64 `json:"median,omitempty"`
	StdDev float64 `json:"std_dev,omitempty"`
	Sum    float64 `json:"sum"`
	Count  int     `json:"count"`
}

// TimeSeriesInfo contains information about detected time-series structure
type TimeSeriesInfo struct {
	DateColumn    string    `json:"date_column"`    // Column containing timestamps
	MetricColumns []string  `json:"metric_columns"` // Numeric columns that vary over time
	Frequency     string    `json:"frequency"`      // "daily", "weekly", "monthly", "hourly", "irregular"
	StartDate     time.Time `json:"start_date"`
	EndDate       time.Time `json:"end_date"`
	HasGaps       bool      `json:"has_gaps"`  // True if there are missing time periods
	IsSorted      bool      `json:"is_sorted"` // True if rows are sorted by date
}

// DataSourceConfig represents the configuration for extracting data from a source
type DataSourceConfig struct {
	Type   string `json:"type"`             // "csv", "excel", "json", "pipeline", "storage", "plugin:<name>"
	Format string `json:"format,omitempty"` // For direct uploads: "csv", "excel", "json"

	// Direct upload data
	Data interface{} `json:"data,omitempty"`

	// Pipeline-based ingestion
	PipelineID string `json:"pipeline_id,omitempty"`
	OutputKey  string `json:"output_key,omitempty"` // Which pipeline output to use

	// Storage query-based
	StorageID string                 `json:"storage_id,omitempty"`
	Query     map[string]interface{} `json:"query,omitempty"`

	// Plugin-based (custom data sources)
	PluginName   string                 `json:"plugin_name,omitempty"` // Name of custom ingestion plugin
	PluginConfig map[string]interface{} `json:"plugin_config,omitempty"`

	// Options for data processing
	Options DataSourceOptions `json:"options,omitempty"`
}

// DataSourceOptions contains optional configuration for data extraction
type DataSourceOptions struct {
	// Column mapping
	ColumnMapping   map[string]string `json:"column_mapping,omitempty"`   // Rename columns
	SelectedColumns []string          `json:"selected_columns,omitempty"` // Only use these columns

	// Time-series hints
	ForceTimeSeriesDetection bool     `json:"force_timeseries,omitempty"`
	DateColumn               string   `json:"date_column,omitempty"`
	MetricColumns            []string `json:"metric_columns,omitempty"`

	// Filtering
	RowFilter map[string]interface{} `json:"row_filter,omitempty"` // Filter rows by conditions
	Limit     int                    `json:"limit,omitempty"`      // Max rows to extract

	// Type hints
	TypeHints map[string]string `json:"type_hints,omitempty"` // Force column types
}

// Validate checks if the UnifiedDataset is valid
func (ud *UnifiedDataset) Validate() error {
	if ud.RowCount == 0 {
		return fmt.Errorf("dataset is empty (0 rows)")
	}

	if ud.ColumnCount == 0 {
		return fmt.Errorf("dataset has no columns")
	}

	if len(ud.Columns) != ud.ColumnCount {
		return fmt.Errorf("column count mismatch: expected %d, got %d", ud.ColumnCount, len(ud.Columns))
	}

	if len(ud.Rows) != ud.RowCount {
		return fmt.Errorf("row count mismatch: expected %d, got %d", ud.RowCount, len(ud.Rows))
	}

	return nil
}

// GetColumn returns metadata for a named column
func (ud *UnifiedDataset) GetColumn(name string) (*ColumnMetadata, error) {
	for _, col := range ud.Columns {
		if col.Name == name {
			return &col, nil
		}
	}
	return nil, fmt.Errorf("column not found: %s", name)
}

// GetNumericColumns returns all numeric columns
func (ud *UnifiedDataset) GetNumericColumns() []ColumnMetadata {
	var numericCols []ColumnMetadata
	for _, col := range ud.Columns {
		if col.IsNumeric {
			numericCols = append(numericCols, col)
		}
	}
	return numericCols
}

// GetDateTimeColumns returns all datetime columns
func (ud *UnifiedDataset) GetDateTimeColumns() []ColumnMetadata {
	var dateCols []ColumnMetadata
	for _, col := range ud.Columns {
		if col.IsDateTime {
			dateCols = append(dateCols, col)
		}
	}
	return dateCols
}

// IsTimeSeries returns true if this dataset has time-series structure
func (ud *UnifiedDataset) IsTimeSeries() bool {
	return ud.TimeSeriesConfig != nil &&
		ud.TimeSeriesConfig.DateColumn != "" &&
		len(ud.TimeSeriesConfig.MetricColumns) > 0
}

// GetValue retrieves a value from a row by column name
func (ud *UnifiedDataset) GetValue(rowIndex int, columnName string) (interface{}, error) {
	if rowIndex < 0 || rowIndex >= ud.RowCount {
		return nil, fmt.Errorf("row index out of range: %d", rowIndex)
	}

	row := ud.Rows[rowIndex]
	value, exists := row[columnName]
	if !exists {
		return nil, fmt.Errorf("column not found in row: %s", columnName)
	}

	return value, nil
}

// Summary returns a human-readable summary of the dataset
func (ud *UnifiedDataset) Summary() string {
	summary := fmt.Sprintf("Dataset from source '%s'\n", ud.Source)
	summary += fmt.Sprintf("  Rows: %d\n", ud.RowCount)
	summary += fmt.Sprintf("  Columns: %d\n", ud.ColumnCount)

	if ud.IsTimeSeries() {
		summary += fmt.Sprintf("  Time-Series: Yes (date column: %s, %d metrics)\n",
			ud.TimeSeriesConfig.DateColumn,
			len(ud.TimeSeriesConfig.MetricColumns))
	} else {
		summary += "  Time-Series: No\n"
	}

	summary += "\nColumns:\n"
	for _, col := range ud.Columns {
		summary += fmt.Sprintf("  - %s (%s)%s\n",
			col.Name,
			col.DataType,
			func() string {
				if col.IsTimeSeries {
					return " [DATE/TIME]"
				}
				if col.IsNumeric {
					return " [NUMERIC]"
				}
				return ""
			}())
	}

	return summary
}

// NewUnifiedDataset creates a new UnifiedDataset with basic initialization
func NewUnifiedDataset(source string) *UnifiedDataset {
	return &UnifiedDataset{
		Source:     source,
		SourceInfo: make(map[string]interface{}),
		Columns:    []ColumnMetadata{},
		Rows:       []map[string]interface{}{},
		Metadata:   make(map[string]interface{}),
		CreatedAt:  time.Now(),
	}
}
