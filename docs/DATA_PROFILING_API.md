# Data Profiling API Documentation

## Overview

The Data Profiling feature provides comprehensive statistical analysis and quality assessment for uploaded datasets. It analyzes each column to provide insights into data distribution, quality issues, and potential primary keys.

## API Endpoint

### Preview Data with Profiling

**Endpoint:** `POST /api/v1/data/preview`

**Request Body:**
```json
{
  "upload_id": "upload_1234567890_data.csv",
  "plugin_type": "Input",
  "plugin_name": "csv",
  "config": {
    "has_headers": true,
    "delimiter": ","
  },
  "max_rows": 100,
  "profile": true
}
```

**Query Parameters:**
- `profile=true` - Alternative way to enable profiling via query parameter

**Response:**
```json
{
  "upload_id": "upload_1234567890_data.csv",
  "plugin_type": "Input",
  "plugin_name": "csv",
  "data": {
    "rows": [...],
    "preview_limited": true,
    "total_rows": 1000
  },
  "preview_rows": 100,
  "profile": {
    "total_rows": 1000,
    "total_columns": 5,
    "total_distinct_values": 3450,
    "overall_quality_score": 0.87,
    "suggested_primary_keys": ["id", "email"],
    "column_profiles": [...]
  },
  "message": "Data preview with profiling generated successfully"
}
```

## Column Profile Structure

Each column in the dataset receives a comprehensive profile:

```json
{
  "column_name": "age",
  "data_type": "numeric",
  "total_count": 1000,
  "distinct_count": 45,
  "distinct_percent": 45.0,
  "null_count": 10,
  "null_percent": 1.0,
  "min_value": 18,
  "max_value": 85,
  "mean": 42.5,
  "median": 41.0,
  "std_dev": 15.3,
  "top_values": [
    {
      "value": "35",
      "count": 45,
      "frequency": 4.5
    }
  ],
  "data_quality_score": 0.95,
  "quality_issues": []
}
```

### Column Profile Fields

#### Basic Metrics
- **column_name** (string): Name of the column
- **data_type** (string): Detected type - "numeric", "string", or "unknown"
- **total_count** (int): Total number of values including nulls
- **distinct_count** (int): Number of unique values
- **distinct_percent** (float): Percentage of distinct values (cardinality)
- **null_count** (int): Number of null or empty values
- **null_percent** (float): Percentage of null values

#### Numeric Statistics (numeric columns only)
- **min_value** (float): Minimum value
- **max_value** (float): Maximum value
- **mean** (float): Arithmetic mean
- **median** (float): Median value
- **std_dev** (float): Standard deviation

#### String Statistics (string columns only)
- **min_length** (int): Minimum string length
- **max_length** (int): Maximum string length
- **avg_length** (float): Average string length

#### Value Distribution
- **top_values** (array): Top 5 most frequent values with:
  - **value** (any): The actual value
  - **count** (int): Number of occurrences
  - **frequency** (float): Percentage of total

#### Data Quality
- **data_quality_score** (float): Overall quality score (0.0-1.0)
- **quality_issues** (array of strings): List of detected issues

## Data Quality Scoring

The quality score (0.0-1.0) is calculated based on:

### Negative Factors (reduce score)
- **High null rate**: Up to -0.5 penalty for 100% nulls
- **Critical quality issues**: -0.15 per critical issue:
  - High null rates (>50%)
  - Very low cardinality (<5 distinct values)
  - Duplicate values in ID columns

### Positive Factors (increase score)
- **High completeness**: +0.1 bonus for <5% nulls
- **Good cardinality**: +0.05 bonus for appropriate distinct values
- **Base score**: Starts at 1.0

## Quality Issues Detection

The system automatically detects and reports:

### 1. Null Rate Issues
- **High null rate (>50%)**: Indicates significant missing data
- **Moderate null rate (>25%)**: Warning about data completeness

### 2. Cardinality Issues
- **Very low cardinality**: < 5 distinct values in >10 rows
- **Low uniqueness**: < 80% distinct in >100 rows
- **Single value dominates**: One value represents >80% of data

### 3. String Issues
- **Extreme length variance**: Max/min length ratio > 100
- **Potential ID column issues**: ID-named columns with <95% uniqueness

## Primary Key Suggestions

The system suggests columns as potential primary keys if they meet:
- **Uniqueness**: >95% distinct values
- **Completeness**: <5% null values
- **Sample size**: >10 rows analyzed

## Usage Examples

### Example 1: Basic Profiling

```bash
curl -X POST http://localhost:8080/api/v1/data/preview \
  -H "Content-Type: application/json" \
  -d '{
    "upload_id": "upload_1234567890_customers.csv",
    "plugin_type": "Input",
    "plugin_name": "csv",
    "profile": true
  }'
```

### Example 2: Query Parameter Profiling

```bash
curl -X POST "http://localhost:8080/api/v1/data/preview?profile=true" \
  -H "Content-Type: application/json" \
  -d '{
    "upload_id": "upload_1234567890_sales.csv",
    "plugin_type": "Input",
    "plugin_name": "csv",
    "max_rows": 50
  }'
```

### Example 3: Large Dataset with Sampling

For large datasets, profiling automatically samples up to 10,000 rows:

```bash
curl -X POST http://localhost:8080/api/v1/data/preview \
  -H "Content-Type: application/json" \
  -d '{
    "upload_id": "upload_1234567890_bigdata.csv",
    "plugin_type": "Input",
    "plugin_name": "csv",
    "max_rows": 100,
    "profile": true
  }'
```

## Performance Considerations

### Sampling Strategy
- Datasets with ≤10,000 rows: Full profiling
- Datasets with >10,000 rows: Systematic sampling of 10,000 rows
- Sampling uses systematic selection for consistency

### Statistical Calculations
- **Numeric columns**: Mean, median, std dev, min/max calculated
- **String columns**: Length statistics (min/max/avg)
- **Top values**: Limited to top 5 most frequent

### Memory Efficiency
- Streaming value counting
- In-place sorting for median calculation
- Efficient distinct value tracking with maps

## Integration with Other Features

### Ontology Generation
Profiling results can inform:
- Property type selection (numeric vs string)
- Required vs optional properties (based on null rate)
- Cardinality constraints
- Primary key identification

### Data Quality Monitoring
- Track quality scores over time
- Alert on quality degradation
- Identify columns needing cleanup

### Data Validation
- Detect anomalies before import
- Validate data types
- Check completeness requirements

## Error Handling

### Missing Upload
```json
{
  "error": "Upload file not found: upload_invalid",
  "status": "error"
}
```

### Invalid Plugin
```json
{
  "error": "Plugin not found: Input.invalid",
  "status": "error"
}
```

### Parsing Failure
```json
{
  "error": "Data parsing failed: invalid CSV format",
  "status": "error"
}
```

## Best Practices

### 1. Enable Profiling Selectively
- Use profiling for initial data exploration
- Skip for repeated previews of same data
- Consider cost for very large datasets

### 2. Interpret Quality Scores
- **0.9-1.0**: Excellent quality, ready for import
- **0.7-0.9**: Good quality, minor issues
- **0.5-0.7**: Fair quality, review issues
- **<0.5**: Poor quality, requires cleanup

### 3. Review Quality Issues
- Address high null rates before import
- Verify low cardinality is expected (e.g., categories)
- Ensure ID columns have proper uniqueness

### 4. Use Primary Key Suggestions
- Validate suggested keys match business logic
- Consider composite keys if no single column suggested
- Verify uniqueness in full dataset

## Code Reference

The profiling implementation is located in:
- **handlers.go**: Main profiling logic (lines 2374+)
- **handlers_profiling_test.go**: Comprehensive test suite

Key functions:
- `profileColumnData(columnName string, values []any) ColumnProfile`
- `profileDataset(data map[string]any, sampleSize int) DataProfileSummary`
- `calculateDataQualityScore(profile ColumnProfile, nonNullCount int) float64`
- `detectQualityIssues(profile ColumnProfile, nonNullCount int) []string`
