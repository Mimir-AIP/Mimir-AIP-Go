#!/bin/bash

# Data Profiling Feature - Usage Examples
# This script demonstrates how to use the data profiling API

BASE_URL="http://localhost:8080/api/v1"

echo "=== Mimir-AIP Data Profiling Examples ==="
echo ""

# Example 1: Upload a CSV file
echo "1. Uploading CSV file..."
UPLOAD_RESPONSE=$(curl -s -X POST "$BASE_URL/data/upload" \
  -F "file=@test_data.csv" \
  -F "plugin_type=Input" \
  -F "plugin_name=csv")

UPLOAD_ID=$(echo $UPLOAD_RESPONSE | jq -r '.upload_id')
echo "   Upload ID: $UPLOAD_ID"
echo ""

# Example 2: Preview data WITHOUT profiling
echo "2. Preview data without profiling..."
curl -s -X POST "$BASE_URL/data/preview" \
  -H "Content-Type: application/json" \
  -d "{
    \"upload_id\": \"$UPLOAD_ID\",
    \"plugin_type\": \"Input\",
    \"plugin_name\": \"csv\",
    \"max_rows\": 10
  }" | jq '.message, .data.rows[0]'
echo ""

# Example 3: Preview data WITH profiling (via request body)
echo "3. Preview data WITH profiling (request body)..."
PROFILE_RESPONSE=$(curl -s -X POST "$BASE_URL/data/preview" \
  -H "Content-Type: application/json" \
  -d "{
    \"upload_id\": \"$UPLOAD_ID\",
    \"plugin_type\": \"Input\",
    \"plugin_name\": \"csv\",
    \"max_rows\": 100,
    \"profile\": true
  }")

echo "   Overall Quality Score: $(echo $PROFILE_RESPONSE | jq -r '.profile.overall_quality_score')"
echo "   Total Rows: $(echo $PROFILE_RESPONSE | jq -r '.profile.total_rows')"
echo "   Total Columns: $(echo $PROFILE_RESPONSE | jq -r '.profile.total_columns')"
echo "   Suggested Primary Keys: $(echo $PROFILE_RESPONSE | jq -r '.profile.suggested_primary_keys | join(", ")')"
echo ""

# Example 4: Preview with profiling via query parameter
echo "4. Preview data WITH profiling (query parameter)..."
curl -s -X POST "$BASE_URL/data/preview?profile=true" \
  -H "Content-Type: application/json" \
  -d "{
    \"upload_id\": \"$UPLOAD_ID\",
    \"plugin_type\": \"Input\",
    \"plugin_name\": \"csv\",
    \"max_rows\": 50
  }" | jq '.profile.overall_quality_score'
echo ""

# Example 5: Get detailed column profiles
echo "5. Analyzing individual column profiles..."
COLUMN_PROFILES=$(echo $PROFILE_RESPONSE | jq -r '.profile.column_profiles')

echo "$COLUMN_PROFILES" | jq -r '.[] | "
Column: \(.column_name)
  Type: \(.data_type)
  Completeness: \(100 - .null_percent)%
  Uniqueness: \(.distinct_percent)%
  Quality Score: \(.data_quality_score)
  Issues: \(.quality_issues | join(", ") // "None")
"'
echo ""

# Example 6: Check for quality issues
echo "6. Identifying columns with quality issues..."
echo "$COLUMN_PROFILES" | jq -r '.[] | select(.quality_issues | length > 0) | 
  "\(.column_name): \(.quality_issues | join("; "))"'
echo ""

# Example 7: Get top values for a specific column
echo "7. Top values for first column..."
FIRST_COLUMN=$(echo "$COLUMN_PROFILES" | jq -r '.[0].column_name')
echo "   Column: $FIRST_COLUMN"
echo "$COLUMN_PROFILES" | jq -r '.[0].top_values[] | 
  "   \(.value): \(.count) occurrences (\(.frequency | tonumber | round)%)"'
echo ""

# Example 8: Filter high-quality columns (for primary key candidates)
echo "8. High-quality columns (potential primary keys)..."
echo "$COLUMN_PROFILES" | jq -r '.[] | select(.data_quality_score > 0.9) | 
  "\(.column_name) - Quality: \(.data_quality_score), Distinct: \(.distinct_percent)%, Nulls: \(.null_percent)%"'
echo ""

# Example 9: Numeric columns with statistics
echo "9. Numeric columns with statistics..."
echo "$COLUMN_PROFILES" | jq -r '.[] | select(.data_type == "numeric") | 
  "
\(.column_name):
  Range: \(.min_value) - \(.max_value)
  Mean: \(.mean | tonumber | round)
  Median: \(.median | tonumber | round)
  Std Dev: \(.std_dev | tonumber * 100 | round / 100)
"'
echo ""

# Example 10: String columns with length analysis
echo "10. String columns with length analysis..."
echo "$COLUMN_PROFILES" | jq -r '.[] | select(.data_type == "string") | 
  "\(.column_name): Lengths \(.min_length)-\(.max_length) (avg: \(.avg_length | tonumber | round))"'
echo ""

echo "=== Examples Complete ==="
