package schema_inference

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestDetectForeignKeysByName tests FK detection based on naming patterns
func TestDetectForeignKeysByName(t *testing.T) {
	config := InferenceConfig{
		SampleSize:        100,
		EnableConstraints: true,
		EnableFKDetection: true,
		FKMinConfidence:   0.8,
	}
	engine := NewSchemaInferenceEngine(config)

	// Test data: orders referencing users
	data := []map[string]interface{}{
		{"id": 1, "user_id": 100, "amount": 50.0},
		{"id": 2, "user_id": 101, "amount": 75.0},
		{"id": 3, "user_id": 100, "amount": 30.0},
		{"id": 4, "user_id": 102, "amount": 120.0},
	}

	schema, err := engine.InferSchema(data, "orders")
	assert.NoError(t, err)
	assert.NotNil(t, schema)

	// Check that user_id is detected as FK
	var userIdCol *ColumnSchema
	for i := range schema.Columns {
		if schema.Columns[i].Name == "user_id" {
			userIdCol = &schema.Columns[i]
			break
		}
	}

	assert.NotNil(t, userIdCol, "user_id column should exist")
	assert.True(t, userIdCol.IsForeignKey, "user_id should be detected as FK")
}

// TestDetectForeignKeysByNamePatterns tests various FK naming patterns
func TestDetectForeignKeysByNamePatterns(t *testing.T) {
	config := InferenceConfig{
		EnableConstraints: true,
		EnableFKDetection: false, // We'll test the method directly
	}
	engine := NewSchemaInferenceEngine(config)

	tests := []struct {
		name               string
		columnName         string
		expectedConfidence float64
		shouldDetect       bool
	}{
		{
			name:               "Standard _id pattern",
			columnName:         "user_id",
			expectedConfidence: 0.7,
			shouldDetect:       true,
		},
		{
			name:               "Ref pattern",
			columnName:         "order_ref",
			expectedConfidence: 0.7,
			shouldDetect:       true,
		},
		{
			name:               "FK prefix pattern",
			columnName:         "fk_customer",
			expectedConfidence: 0.8,
			shouldDetect:       true,
		},
		{
			name:               "Regular column",
			columnName:         "email",
			expectedConfidence: 0.0,
			shouldDetect:       false,
		},
		{
			name:               "ID without prefix",
			columnName:         "id",
			expectedConfidence: 0.0,
			shouldDetect:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			columns := []ColumnSchema{
				{Name: "id", IsPrimaryKey: true},
			}
			confidence, _ := engine.detectFKByName(tt.columnName, columns)

			if tt.shouldDetect {
				assert.Greater(t, confidence, 0.0, "Should detect FK pattern")
				assert.GreaterOrEqual(t, confidence, tt.expectedConfidence-0.3, "Confidence should be in expected range")
			} else {
				assert.Equal(t, 0.0, confidence, "Should not detect FK pattern")
			}
		})
	}
}

// TestDetectForeignKeysByCardinality tests FK detection based on cardinality
func TestDetectForeignKeysByCardinality(t *testing.T) {
	config := InferenceConfig{}
	engine := NewSchemaInferenceEngine(config)

	tests := []struct {
		name               string
		cardinality        int
		rowCount           int
		expectedConfidence float64
		shouldDetect       bool
	}{
		{
			name:               "Good FK cardinality - 30%",
			cardinality:        30,
			rowCount:           100,
			expectedConfidence: 0.7,
			shouldDetect:       true,
		},
		{
			name:               "Good FK cardinality - 50%",
			cardinality:        50,
			rowCount:           100,
			expectedConfidence: 0.7,
			shouldDetect:       true,
		},
		{
			name:               "Too low cardinality - 2%",
			cardinality:        2,
			rowCount:           100,
			expectedConfidence: 0.0,
			shouldDetect:       false,
		},
		{
			name:               "Too high cardinality - 95%",
			cardinality:        95,
			rowCount:           100,
			expectedConfidence: 0.0,
			shouldDetect:       false,
		},
		{
			name:               "Edge case - 10%",
			cardinality:        10,
			rowCount:           100,
			expectedConfidence: 0.5,
			shouldDetect:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			col := ColumnSchema{
				Name:               "test_col",
				Cardinality:        tt.cardinality,
				CardinalityPercent: float64(tt.cardinality) / float64(tt.rowCount),
			}

			confidence := engine.detectFKByCardinality(col, tt.rowCount)

			if tt.shouldDetect {
				assert.Greater(t, confidence, 0.0, "Should detect FK by cardinality")
			} else {
				assert.Equal(t, 0.0, confidence, "Should not detect FK by cardinality")
			}
		})
	}
}

// TestDetectForeignKeysByValueOverlap tests FK detection based on value matching
func TestDetectForeignKeysByValueOverlap(t *testing.T) {
	config := InferenceConfig{}
	engine := NewSchemaInferenceEngine(config)

	tests := []struct {
		name               string
		sourceValues       map[string]bool
		targetValues       map[string]bool
		expectedConfidence float64
		shouldDetect       bool
	}{
		{
			name: "Perfect overlap - 100%",
			sourceValues: map[string]bool{
				"1": true,
				"2": true,
				"3": true,
			},
			targetValues: map[string]bool{
				"1": true,
				"2": true,
				"3": true,
				"4": true,
			},
			expectedConfidence: 1.0,
			shouldDetect:       true,
		},
		{
			name: "Good overlap - 80%",
			sourceValues: map[string]bool{
				"1": true,
				"2": true,
				"3": true,
				"4": true,
				"5": true,
			},
			targetValues: map[string]bool{
				"1": true,
				"2": true,
				"3": true,
				"4": true,
			},
			expectedConfidence: 0.8,
			shouldDetect:       true,
		},
		{
			name: "Poor overlap - 50%",
			sourceValues: map[string]bool{
				"1": true,
				"2": true,
				"3": true,
				"4": true,
			},
			targetValues: map[string]bool{
				"1": true,
				"2": true,
			},
			expectedConfidence: 0.0,
			shouldDetect:       false,
		},
		{
			name: "No overlap",
			sourceValues: map[string]bool{
				"1": true,
				"2": true,
			},
			targetValues: map[string]bool{
				"3": true,
				"4": true,
			},
			expectedConfidence: 0.0,
			shouldDetect:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			confidence, matched, total := engine.detectFKByValueOverlap(tt.sourceValues, tt.targetValues)

			if tt.shouldDetect {
				assert.Greater(t, confidence, 0.0, "Should detect FK by value overlap")
				assert.GreaterOrEqual(t, confidence, tt.expectedConfidence-0.1, "Confidence should be in expected range")
				assert.Greater(t, matched, 0, "Should have matched values")
			} else {
				assert.Equal(t, 0.0, confidence, "Should not detect FK by value overlap")
			}

			assert.Equal(t, len(tt.sourceValues), total, "Total should match source value count")
		})
	}
}

// TestForeignKeyDetectionEndToEnd tests complete FK detection workflow
func TestForeignKeyDetectionEndToEnd(t *testing.T) {
	config := InferenceConfig{
		SampleSize:        100,
		EnableConstraints: true,
		EnableFKDetection: true,
		FKMinConfidence:   0.7,
	}
	engine := NewSchemaInferenceEngine(config)

	// Test data with clear FK relationships
	// Users table
	users := []map[string]interface{}{
		{"id": 1, "name": "Alice", "email": "alice@example.com"},
		{"id": 2, "name": "Bob", "email": "bob@example.com"},
		{"id": 3, "name": "Charlie", "email": "charlie@example.com"},
	}

	// Orders table referencing users
	orders := []map[string]interface{}{
		{"order_id": 101, "user_id": 1, "amount": 50.0, "status": "completed"},
		{"order_id": 102, "user_id": 2, "amount": 75.0, "status": "pending"},
		{"order_id": 103, "user_id": 1, "amount": 30.0, "status": "completed"},
		{"order_id": 104, "user_id": 3, "amount": 120.0, "status": "shipped"},
		{"order_id": 105, "user_id": 2, "amount": 45.0, "status": "pending"},
	}

	// Test users schema
	usersSchema, err := engine.InferSchema(users, "users")
	assert.NoError(t, err)
	assert.NotNil(t, usersSchema)

	// Verify id column is detected as PK
	var idCol *ColumnSchema
	for i := range usersSchema.Columns {
		if usersSchema.Columns[i].Name == "id" {
			idCol = &usersSchema.Columns[i]
			break
		}
	}
	assert.NotNil(t, idCol)
	assert.True(t, idCol.IsPrimaryKey, "id should be detected as primary key")
	assert.True(t, idCol.IsUnique, "id should be unique")

	// Test orders schema
	ordersSchema, err := engine.InferSchema(orders, "orders")
	assert.NoError(t, err)
	assert.NotNil(t, ordersSchema)

	// Verify user_id is detected as FK
	var userIdCol *ColumnSchema
	for i := range ordersSchema.Columns {
		if ordersSchema.Columns[i].Name == "user_id" {
			userIdCol = &ordersSchema.Columns[i]
			break
		}
	}

	assert.NotNil(t, userIdCol, "user_id column should exist")
	assert.True(t, userIdCol.IsForeignKey, "user_id should be detected as FK")

	// Check cardinality
	assert.Equal(t, 3, userIdCol.Cardinality, "user_id should have 3 unique values")
	assert.InDelta(t, 0.6, userIdCol.CardinalityPercent, 0.1, "Cardinality percent should be around 60%")
}

// TestForeignKeyRelationshipDetection tests FK relationship structure
func TestForeignKeyRelationshipDetection(t *testing.T) {
	config := InferenceConfig{
		SampleSize:        100,
		EnableConstraints: true,
		EnableFKDetection: true,
		FKMinConfidence:   0.7,
	}
	engine := NewSchemaInferenceEngine(config)

	// Combined dataset simulating a join scenario
	data := []map[string]interface{}{
		{"id": 1, "user_id": 10, "product_id": 100},
		{"id": 2, "user_id": 10, "product_id": 101},
		{"id": 3, "user_id": 11, "product_id": 100},
		{"id": 4, "user_id": 11, "product_id": 102},
		{"id": 5, "user_id": 12, "product_id": 101},
	}

	schema, err := engine.InferSchema(data, "order_items")
	assert.NoError(t, err)
	assert.NotNil(t, schema)

	// Check that FK relationships are detected
	// Note: In a single table, value overlap detection won't find matches
	// but name pattern detection should still work
	assert.GreaterOrEqual(t, len(schema.Columns), 3, "Should have at least 3 columns")

	// Verify columns have FK markers based on naming
	var userIdCol, productIdCol *ColumnSchema
	for i := range schema.Columns {
		if schema.Columns[i].Name == "user_id" {
			userIdCol = &schema.Columns[i]
		}
		if schema.Columns[i].Name == "product_id" {
			productIdCol = &schema.Columns[i]
		}
	}

	assert.NotNil(t, userIdCol, "user_id should exist")
	assert.NotNil(t, productIdCol, "product_id should exist")

	// These should be flagged as FK based on naming pattern
	assert.True(t, userIdCol.IsForeignKey, "user_id should be flagged as FK")
	assert.True(t, productIdCol.IsForeignKey, "product_id should be flagged as FK")
}

// TestUpdateColumnsWithFKInfo tests FK metadata update
func TestUpdateColumnsWithFKInfo(t *testing.T) {
	config := InferenceConfig{}
	engine := NewSchemaInferenceEngine(config)

	columns := []ColumnSchema{
		{Name: "id", IsPrimaryKey: true},
		{Name: "user_id", IsForeignKey: false},
		{Name: "name"},
	}

	fkRelationships := []ForeignKeyRelationship{
		{
			SourceColumn:         "user_id",
			TargetColumn:         "id",
			Confidence:           0.9,
			ReferentialIntegrity: 0.95,
			MatchedValues:        95,
			TotalValues:          100,
			DetectionMethods:     []string{"name_pattern", "value_overlap"},
		},
	}

	engine.updateColumnsWithFKInfo(columns, fkRelationships)

	// Check that user_id was updated
	var userIdCol *ColumnSchema
	for i := range columns {
		if columns[i].Name == "user_id" {
			userIdCol = &columns[i]
			break
		}
	}

	assert.NotNil(t, userIdCol)
	assert.True(t, userIdCol.IsForeignKey, "Should be marked as FK")
	assert.NotNil(t, userIdCol.FKMetadata, "Should have FK metadata")
	assert.Equal(t, "id", userIdCol.FKMetadata.ReferencedColumn)
	assert.Equal(t, 0.9, userIdCol.FKMetadata.Confidence)
	assert.Contains(t, userIdCol.FKMetadata.DetectionMethod, "name_pattern")
}

// TestForeignKeyMinConfidence tests confidence threshold filtering
func TestForeignKeyMinConfidence(t *testing.T) {
	// High confidence threshold
	highConfig := InferenceConfig{
		SampleSize:        100,
		EnableConstraints: true,
		EnableFKDetection: true,
		FKMinConfidence:   0.95, // Very high threshold
	}
	highEngine := NewSchemaInferenceEngine(highConfig)

	// Low confidence threshold
	lowConfig := InferenceConfig{
		SampleSize:        100,
		EnableConstraints: true,
		EnableFKDetection: true,
		FKMinConfidence:   0.5, // Low threshold
	}
	lowEngine := NewSchemaInferenceEngine(lowConfig)

	data := []map[string]interface{}{
		{"id": 1, "user_id": 10},
		{"id": 2, "user_id": 10},
		{"id": 3, "user_id": 11},
	}

	// High threshold - might detect fewer FKs
	highSchema, err := highEngine.InferSchema(data, "test")
	assert.NoError(t, err)
	highFKCount := len(highSchema.ForeignKeys)

	// Low threshold - should detect more FKs
	lowSchema, err := lowEngine.InferSchema(data, "test")
	assert.NoError(t, err)
	lowFKCount := len(lowSchema.ForeignKeys)

	// With lower threshold, we should detect at least as many FKs
	assert.GreaterOrEqual(t, lowFKCount, highFKCount, "Lower threshold should detect >= FKs")
}

// TestCalculateAverageConfidence tests confidence calculation
func TestCalculateAverageConfidence(t *testing.T) {
	config := InferenceConfig{}
	engine := NewSchemaInferenceEngine(config)

	tests := []struct {
		name     string
		scores   []float64
		expected float64
	}{
		{
			name:     "Single score",
			scores:   []float64{0.8},
			expected: 0.8,
		},
		{
			name:     "Multiple scores",
			scores:   []float64{0.8, 0.9, 0.7},
			expected: 0.8,
		},
		{
			name:     "Perfect scores",
			scores:   []float64{1.0, 1.0, 1.0},
			expected: 1.0,
		},
		{
			name:     "Empty scores",
			scores:   []float64{},
			expected: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := engine.calculateAverageConfidence(tt.scores)
			assert.InDelta(t, tt.expected, result, 0.01, "Average confidence should match")
		})
	}
}

// TestCompositeKeyDetection tests detection of composite foreign keys
func TestCompositeKeyDetection(t *testing.T) {
	config := InferenceConfig{
		SampleSize:        100,
		EnableConstraints: true,
		EnableFKDetection: true,
		FKMinConfidence:   0.7,
	}
	engine := NewSchemaInferenceEngine(config)

	// Data with multiple FK columns
	data := []map[string]interface{}{
		{"id": 1, "user_id": 10, "org_id": 1, "created_by_id": 10},
		{"id": 2, "user_id": 11, "org_id": 1, "created_by_id": 11},
		{"id": 3, "user_id": 10, "org_id": 2, "created_by_id": 10},
	}

	schema, err := engine.InferSchema(data, "documents")
	assert.NoError(t, err)
	assert.NotNil(t, schema)

	// Count FK columns
	fkCount := 0
	for _, col := range schema.Columns {
		if col.IsForeignKey {
			fkCount++
		}
	}

	// Should detect multiple FK columns based on naming pattern
	assert.GreaterOrEqual(t, fkCount, 2, "Should detect multiple FK columns")
}

// TestReferentialIntegrityCalculation tests referential integrity calculation
func TestReferentialIntegrityCalculation(t *testing.T) {
	config := InferenceConfig{}
	engine := NewSchemaInferenceEngine(config)

	sourceValues := map[string]bool{
		"1": true,
		"2": true,
		"3": true,
		"4": true,
		"5": true,
	}

	targetValues := map[string]bool{
		"1": true,
		"2": true,
		"3": true,
		"4": true,
	}

	confidence, matched, total := engine.detectFKByValueOverlap(sourceValues, targetValues)

	assert.Equal(t, 5, total, "Total should be 5")
	assert.Equal(t, 4, matched, "Matched should be 4")

	expectedIntegrity := 4.0 / 5.0 // 80%
	if confidence > 0 {
		actualIntegrity := float64(matched) / float64(total)
		assert.InDelta(t, expectedIntegrity, actualIntegrity, 0.01, "Referential integrity should be 80%")
	}
}
