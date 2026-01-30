package ontology

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDeterministicExtractor_ExtractFromCSV tests CSV extraction with actual data
func TestDeterministicExtractor_ExtractFromCSV(t *testing.T) {
	// Create a deterministic extractor
	config := ExtractionConfig{
		OntologyID:     "test-ontology-1",
		SourceType:     SourceTypeCSV,
		ExtractionType: ExtractionDeterministic,
	}
	extractor := NewDeterministicExtractor(config)

	// Create ontology context with sample classes and properties
	ontology := &OntologyContext{
		BaseURI: "http://example.org/ontology",
		Metadata: &OntologyMetadata{
			ID:        "test-ontology-1",
			TDB2Graph: "http://example.org/graph",
		},
		Classes: []OntologyClass{
			{URI: "http://example.org/ontology/Product", Label: "Product"},
			{URI: "http://example.org/ontology/Person", Label: "Person"},
		},
		Properties: []OntologyProperty{
			{URI: "http://example.org/ontology/hasName", Label: "name"},
			{URI: "http://example.org/ontology/hasPrice", Label: "price"},
			{URI: "http://example.org/ontology/hasCategory", Label: "category"},
		},
	}

	// Test 1: Extract from product CSV data
	t.Run("Extract product data from CSV", func(t *testing.T) {
		csvData := `name,price,category
Laptop,999.99,Electronics
Mouse,29.99,Electronics
Desk,299.99,Furniture`

		result, err := extractor.Extract(csvData, ontology)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, 3, result.EntitiesExtracted)
		assert.Greater(t, result.TriplesGenerated, 0)
		assert.Equal(t, ExtractionDeterministic, result.ExtractionType)
		assert.Equal(t, 1.0, result.Confidence)

		// Verify entities were created
		require.Len(t, result.Entities, 3)

		// Check first entity
		firstEntity := result.Entities[0]
		assert.Contains(t, firstEntity.URI, "http://example.org/ontology")
		assert.NotEmpty(t, firstEntity.Type)
		assert.NotNil(t, firstEntity.Properties)

		// Verify triples were generated
		require.NotEmpty(t, result.Triples)

		// Check for type triple
		foundTypeTriple := false
		for _, triple := range result.Triples {
			if triple.Predicate == "http://www.w3.org/1999/02/22-rdf-syntax-ns#type" {
				foundTypeTriple = true
				break
			}
		}
		assert.True(t, foundTypeTriple, "Should have type triples")
	})

	// Test 2: Extract from person CSV data
	t.Run("Extract person data from CSV", func(t *testing.T) {
		csvData := `name,age,department
Alice,30,Engineering
Bob,25,Sales
Charlie,35,Marketing`

		result, err := extractor.Extract(csvData, ontology)
		require.NoError(t, err)
		assert.Equal(t, 3, result.EntitiesExtracted)
		assert.Greater(t, result.TriplesGenerated, 3) // At least type + properties

		// Verify each entity has properties
		for _, entity := range result.Entities {
			assert.NotEmpty(t, entity.Properties)
		}
	})

	// Test 3: Extract with missing values
	t.Run("Handle CSV with missing values", func(t *testing.T) {
		csvData := `name,price,category
Laptop,999.99,Electronics
Broken,,Electronics
Desk,299.99,`

		result, err := extractor.Extract(csvData, ontology)
		require.NoError(t, err)
		assert.Equal(t, 3, result.EntitiesExtracted)

		// Should have warnings about empty values
		// but still produce valid entities
		assert.NotEmpty(t, result.Entities)
	})

	// Test 4: Extract empty CSV
	t.Run("Handle empty CSV", func(t *testing.T) {
		csvData := ""

		result, err := extractor.Extract(csvData, ontology)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, 0, result.EntitiesExtracted)
		assert.Equal(t, 0, result.TriplesGenerated)
	})

	// Test 5: CSV with only headers
	t.Run("Handle CSV with only headers", func(t *testing.T) {
		csvData := "name,price,category\n"

		result, err := extractor.Extract(csvData, ontology)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, 0, result.EntitiesExtracted)
	})
}

// TestDeterministicExtractor_ExtractFromJSON tests JSON extraction
func TestDeterministicExtractor_ExtractFromJSON(t *testing.T) {
	config := ExtractionConfig{
		OntologyID:     "test-ontology-2",
		SourceType:     SourceTypeJSON,
		ExtractionType: ExtractionDeterministic,
	}
	extractor := NewDeterministicExtractor(config)

	ontology := &OntologyContext{
		BaseURI: "http://example.org/ontology",
		Metadata: &OntologyMetadata{
			ID:        "test-ontology-2",
			TDB2Graph: "http://example.org/graph",
		},
		Classes: []OntologyClass{
			{URI: "http://example.org/ontology/Employee", Label: "Employee"},
		},
		Properties: []OntologyProperty{
			{URI: "http://example.org/ontology/name", Label: "name"},
			{URI: "http://example.org/ontology/age", Label: "age"},
		},
	}

	// Test 1: Extract from JSON string
	t.Run("Extract from JSON string", func(t *testing.T) {
		jsonData := `[
			{"name": "Alice", "age": 30, "department": "Engineering"},
			{"name": "Bob", "age": 25, "department": "Sales"},
			{"name": "Charlie", "age": 35, "department": "Marketing"}
		]`

		result, err := extractor.Extract(jsonData, ontology)
		require.NoError(t, err)
		assert.Equal(t, 3, result.EntitiesExtracted)
		assert.Greater(t, result.TriplesGenerated, 0)

		// Verify entities
		require.Len(t, result.Entities, 3)

		// Check properties
		firstEntity := result.Entities[0]
		assert.Contains(t, firstEntity.Properties, "http://example.org/ontology/name")
		assert.Equal(t, "Alice", firstEntity.Properties["http://example.org/ontology/name"])
	})

	// Test 2: Extract from JSON array
	t.Run("Extract from JSON array of objects", func(t *testing.T) {
		jsonArray := []map[string]any{
			{"name": "Product A", "price": 19.99, "quantity": 100},
			{"name": "Product B", "price": 29.99, "quantity": 50},
			{"name": "Product C", "price": 39.99, "quantity": 75},
		}

		result, err := extractor.Extract(jsonArray, ontology)
		require.NoError(t, err)
		assert.Equal(t, 3, result.EntitiesExtracted)
		assert.Equal(t, 1.0, result.Confidence)

		// Verify all properties were captured
		for _, entity := range result.Entities {
			assert.NotEmpty(t, entity.Properties)
			assert.NotEmpty(t, entity.URI)
		}
	})

	// Test 3: Extract with nested objects
	t.Run("Extract from JSON with nested objects", func(t *testing.T) {
		jsonData := `[
			{"name": "Company A", "address": {"city": "NYC", "country": "USA"}},
			{"name": "Company B", "address": {"city": "London", "country": "UK"}}
		]`

		result, err := extractor.Extract(jsonData, ontology)
		require.NoError(t, err)
		assert.Equal(t, 2, result.EntitiesExtracted)

		// Nested values should be converted to strings
		for _, entity := range result.Entities {
			assert.NotEmpty(t, entity.Properties)
		}
	})

	// Test 4: Empty JSON array
	t.Run("Handle empty JSON array", func(t *testing.T) {
		jsonData := "[]"

		result, err := extractor.Extract(jsonData, ontology)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, 0, result.EntitiesExtracted)
	})

	// Test 5: Invalid JSON string
	t.Run("Handle invalid JSON", func(t *testing.T) {
		jsonData := `{"invalid json`

		_, err := extractor.Extract(jsonData, ontology)
		assert.Error(t, err)
	})
}

// TestDeterministicExtractor_EntityTypeInference tests entity type inference
func TestDeterministicExtractor_EntityTypeInference(t *testing.T) {
	config := ExtractionConfig{
		OntologyID:     "test-ontology-3",
		SourceType:     SourceTypeCSV,
		ExtractionType: ExtractionDeterministic,
	}
	extractor := NewDeterministicExtractor(config)

	// Test 1: Infer type from explicit "type" column
	t.Run("Infer type from type column", func(t *testing.T) {
		ontology := &OntologyContext{
			BaseURI: "http://example.org/ontology",
			Classes: []OntologyClass{
				{URI: "http://example.org/ontology/Product", Label: "Product"},
				{URI: "http://example.org/ontology/Service", Label: "Service"},
			},
		}

		csvData := `type,name,price
Product,Laptop,999.99
Service,Support,50.00
Product,Mouse,29.99`

		result, err := extractor.Extract(csvData, ontology)
		require.NoError(t, err)
		require.Len(t, result.Entities, 3)

		// First entity should be Product type
		assert.Equal(t, "http://example.org/ontology/Product", result.Entities[0].Type)
		// Second entity should be Service type
		assert.Equal(t, "http://example.org/ontology/Service", result.Entities[1].Type)
	})

	// Test 2: Infer type from "class" column
	t.Run("Infer type from class column", func(t *testing.T) {
		ontology := &OntologyContext{
			BaseURI: "http://example.org/ontology",
			Classes: []OntologyClass{
				{URI: "http://example.org/ontology/Person", Label: "Person"},
				{URI: "http://example.org/ontology/Organization", Label: "Organization"},
			},
		}

		csvData := `class,name
Person,Alice
Organization,Acme Corp
Person,Bob`

		result, err := extractor.Extract(csvData, ontology)
		require.NoError(t, err)
		require.Len(t, result.Entities, 3)

		assert.Equal(t, "http://example.org/ontology/Person", result.Entities[0].Type)
		assert.Equal(t, "http://example.org/ontology/Organization", result.Entities[1].Type)
	})

	// Test 3: Default to first class if no type column
	t.Run("Default to first class when no type info", func(t *testing.T) {
		ontology := &OntologyContext{
			BaseURI: "http://example.org/ontology",
			Classes: []OntologyClass{
				{URI: "http://example.org/ontology/DefaultClass", Label: "DefaultClass"},
			},
		}

		csvData := `name,value
Item1,100
Item2,200`

		result, err := extractor.Extract(csvData, ontology)
		require.NoError(t, err)

		for _, entity := range result.Entities {
			assert.Equal(t, "http://example.org/ontology/DefaultClass", entity.Type)
		}
	})

	// Test 4: Fallback to owl:Thing when no classes defined
	t.Run("Fallback to owl:Thing when no classes", func(t *testing.T) {
		ontology := &OntologyContext{
			BaseURI: "http://example.org/ontology",
			Classes: []OntologyClass{}, // Empty
		}

		csvData := `name,value
Item1,100`

		result, err := extractor.Extract(csvData, ontology)
		require.NoError(t, err)

		assert.Equal(t, "http://www.w3.org/2002/07/owl#Thing", result.Entities[0].Type)
	})

	// Test 5: Infer type from JSON object
	t.Run("Infer type from JSON type field", func(t *testing.T) {
		config.SourceType = SourceTypeJSON
		extractor := NewDeterministicExtractor(config)

		ontology := &OntologyContext{
			BaseURI: "http://example.org/ontology",
			Classes: []OntologyClass{
				{URI: "http://example.org/ontology/Employee", Label: "Employee"},
			},
		}

		jsonData := `[
			{"type": "Employee", "name": "Alice"},
			{"type": "Employee", "name": "Bob"}
		]`

		result, err := extractor.Extract(jsonData, ontology)
		require.NoError(t, err)

		for _, entity := range result.Entities {
			assert.Equal(t, "http://example.org/ontology/Employee", entity.Type)
		}
	})
}

// TestDeterministicExtractor_PropertyMapping tests property mapping
func TestDeterministicExtractor_PropertyMapping(t *testing.T) {
	config := ExtractionConfig{
		OntologyID:     "test-ontology-4",
		SourceType:     SourceTypeCSV,
		ExtractionType: ExtractionDeterministic,
	}

	ontology := &OntologyContext{
		BaseURI: "http://example.org/ontology",
		Properties: []OntologyProperty{
			{URI: "http://example.org/ontology/hasName", Label: "name"},
			{URI: "http://example.org/ontology/hasPrice", Label: "price"},
			{URI: "http://example.org/ontology/description", Label: "description"},
			{URI: "http://example.org/ontology/sku", Label: "sku"},
		},
	}

	extractor := NewDeterministicExtractor(config)

	// Test 1: Automatic property mapping via fuzzy matching
	t.Run("Auto-map properties via fuzzy matching", func(t *testing.T) {
		csvData := `name,price,description
Laptop,999.99,A high-end laptop
Mouse,29.99,Wireless mouse`

		result, err := extractor.Extract(csvData, ontology)
		require.NoError(t, err)

		// Check that properties were mapped
		for _, entity := range result.Entities {
			// Should have properties with ontology URIs
			assert.NotEmpty(t, entity.Properties)
		}

		// Verify triples use mapped properties
		foundMappedProperty := false
		for _, triple := range result.Triples {
			if triple.Predicate == "http://example.org/ontology/hasName" ||
				triple.Predicate == "http://example.org/ontology/hasPrice" {
				foundMappedProperty = true
				break
			}
		}
		assert.True(t, foundMappedProperty, "Should have mapped properties")
	})

	// Test 2: Property mapping with confidence
	t.Run("Property mapping confidence scores", func(t *testing.T) {
		// Get inferred mappings
		mappings := extractor.inferMappings([]string{"name", "price", "unknown_field"}, ontology)

		assert.NotEmpty(t, mappings)

		// Check that high-confidence mappings exist
		for _, mapping := range mappings {
			if mapping.SourceField == "name" || mapping.SourceField == "price" {
				assert.Greater(t, mapping.Confidence, 0.6, "Should have high confidence for obvious matches")
			}
		}
	})

	// Test 3: Handle unmapped fields
	t.Run("Handle unmapped fields gracefully", func(t *testing.T) {
		csvData := `name,price,custom_field
Laptop,999.99,Custom value`

		result, err := extractor.Extract(csvData, ontology)
		require.NoError(t, err)

		// Should still extract all data
		assert.Equal(t, 1, result.EntitiesExtracted)
		assert.Greater(t, len(result.Triples), 0)
	})

	// Test 4: Property mapping with pre-defined mappings
	t.Run("Use pre-defined property mappings", func(t *testing.T) {
		customMappings := []PropertyMapping{
			{SourceField: "product_name", PropertyURI: "http://example.org/ontology/hasName", Confidence: 1.0},
			{SourceField: "cost", PropertyURI: "http://example.org/ontology/hasPrice", Confidence: 1.0},
		}

		config.Mappings = customMappings
		extractor := NewDeterministicExtractor(config)

		csvData := `product_name,cost
Widget,10.99
Gadget,24.99`

		result, err := extractor.Extract(csvData, ontology)
		require.NoError(t, err)

		// Verify custom mappings were used - check properties map
		firstEntity := result.Entities[0]
		assert.Equal(t, "Widget", firstEntity.Properties["http://example.org/ontology/hasName"])
		assert.Equal(t, "10.99", firstEntity.Properties["http://example.org/ontology/hasPrice"])
	})
}

// TestDeterministicExtractor_TripleGeneration tests triple generation
func TestDeterministicExtractor_TripleGeneration(t *testing.T) {
	config := ExtractionConfig{
		OntologyID:     "test-ontology-5",
		SourceType:     SourceTypeCSV,
		ExtractionType: ExtractionDeterministic,
	}
	extractor := NewDeterministicExtractor(config)

	ontology := &OntologyContext{
		BaseURI: "http://example.org/ontology",
		Classes: []OntologyClass{
			{URI: "http://example.org/ontology/Product", Label: "Product"},
		},
		Properties: []OntologyProperty{
			{URI: "http://example.org/ontology/hasName", Label: "name"},
			{URI: "http://example.org/ontology/hasPrice", Label: "price"},
		},
	}

	// Test 1: Generate correct number of triples
	t.Run("Generate correct triple count", func(t *testing.T) {
		csvData := `name,price
Laptop,999.99
Mouse,29.99`

		result, err := extractor.Extract(csvData, ontology)
		require.NoError(t, err)

		// 2 entities * (1 type triple + 2 property triples) = 6 triples minimum
		// Plus label triples
		assert.GreaterOrEqual(t, result.TriplesGenerated, 6)

		// Count actual triples
		typeCount := 0
		propertyCount := 0
		labelCount := 0

		for _, triple := range result.Triples {
			switch triple.Predicate {
			case "http://www.w3.org/1999/02/22-rdf-syntax-ns#type":
				typeCount++
			case "http://www.w3.org/2000/01/rdf-schema#label":
				labelCount++
			default:
				propertyCount++
			}
		}

		assert.Equal(t, 2, typeCount, "Should have type triple for each entity")
		assert.Equal(t, 2, labelCount, "Should have label triple for each entity")
		assert.GreaterOrEqual(t, propertyCount, 4, "Should have property triples")
	})

	// Test 2: Triples have correct structure
	t.Run("Triple structure validation", func(t *testing.T) {
		csvData := `name,price
Laptop,999.99`

		result, err := extractor.Extract(csvData, ontology)
		require.NoError(t, err)

		for _, triple := range result.Triples {
			// All triples must have subject, predicate, object
			assert.NotEmpty(t, triple.Subject, "Subject should not be empty")
			assert.NotEmpty(t, triple.Predicate, "Predicate should not be empty")
			assert.NotEmpty(t, triple.Object, "Object should not be empty")

			// Subject should be a URI
			assert.Contains(t, triple.Subject, "http://")
		}
	})

	// Test 3: Datatype inference in triples
	t.Run("Datatype inference", func(t *testing.T) {
		csvData := `name,count,price,active
Laptop,10,999.99,true
Mouse,5,29.99,false`

		result, err := extractor.Extract(csvData, ontology)
		require.NoError(t, err)

		// Check for datatype annotations
		foundDatatype := false
		for _, triple := range result.Triples {
			if triple.Datatype != "" {
				foundDatatype = true
				// Verify expected datatypes
				assert.Contains(t, []string{
					"http://www.w3.org/2001/XMLSchema#boolean",
					"http://www.w3.org/2001/XMLSchema#integer",
					"http://www.w3.org/2001/XMLSchema#decimal",
					"http://www.w3.org/2001/XMLSchema#string",
				}, triple.Datatype)
			}
		}
		assert.True(t, foundDatatype, "Should have datatype annotations")
	})

	// Test 4: Skip empty values in triples
	t.Run("Skip empty values", func(t *testing.T) {
		csvData := `name,price,optional
Laptop,999.99,
Mouse,,29.99`

		result, err := extractor.Extract(csvData, ontology)
		require.NoError(t, err)

		// Should still have triples for non-empty values
		assert.Greater(t, result.TriplesGenerated, 0)
	})
}

// TestDeterministicExtractor_WithSampleData tests with various sample data
func TestDeterministicExtractor_WithSampleData(t *testing.T) {
	config := ExtractionConfig{
		OntologyID:     "test-ontology-6",
		SourceType:     SourceTypeCSV,
		ExtractionType: ExtractionDeterministic,
	}
	extractor := NewDeterministicExtractor(config)

	// Test 1: Product catalog data
	t.Run("Extract product catalog", func(t *testing.T) {
		ontology := &OntologyContext{
			BaseURI: "http://example.org/products",
			Classes: []OntologyClass{
				{URI: "http://example.org/products/Product", Label: "Product"},
			},
		}

		csvData := `sku,name,category,price,stock
SKU001,Ultra Laptop,Electronics,1299.99,50
SKU002,Wireless Mouse,Electronics,49.99,200
SKU003,USB-C Hub,Electronics,79.99,150
SKU004,Standing Desk,Furniture,399.99,30
SKU005,Ergonomic Chair,Furniture,599.99,25`

		result, err := extractor.Extract(csvData, ontology)
		require.NoError(t, err)
		assert.Equal(t, 5, result.EntitiesExtracted)
		assert.Greater(t, result.TriplesGenerated, 15)

		// Verify all products extracted
		require.Len(t, result.Entities, 5)

		// Check that entities have all properties
		for _, entity := range result.Entities {
			assert.GreaterOrEqual(t, len(entity.Properties), 4)
		}
	})

	// Test 2: Employee data
	t.Run("Extract employee data", func(t *testing.T) {
		ontology := &OntologyContext{
			BaseURI: "http://example.org/hr",
			Classes: []OntologyClass{
				{URI: "http://example.org/hr/Employee", Label: "Employee"},
			},
		}

		csvData := `id,name,department,salary,hire_date,start_year
E001,Alice Johnson,Engineering,85000,2019-03-15,2019
E002,Bob Smith,Sales,65000,2020-07-01,2020
E003,Carol White,Marketing,70000,2018-11-20,2018
E004,David Brown,Engineering,90000,2017-05-10,2017
E005,Eve Davis,HR,60000,2021-01-15,2021`

		result, err := extractor.Extract(csvData, ontology)
		require.NoError(t, err)
		assert.Equal(t, 5, result.EntitiesExtracted)

		// Check entity properties
		firstEntity := result.Entities[0]
		assert.NotEmpty(t, firstEntity.Label)
		// Property URIs are generated with "prop_" prefix
		assert.Contains(t, firstEntity.Properties, "http://example.org/hr/prop_id")
		assert.Equal(t, "E001", firstEntity.Properties["http://example.org/hr/prop_id"])
		assert.Equal(t, "Alice Johnson", firstEntity.Label)
	})

	// Test 3: Financial transactions
	t.Run("Extract financial transactions", func(t *testing.T) {
		ontology := &OntologyContext{
			BaseURI: "http://example.org/finance",
			Classes: []OntologyClass{
				{URI: "http://example.org/finance/Transaction", Label: "Transaction"},
			},
		}

		csvData := `transaction_id,date,amount,currency,type,account
TXN001,2024-01-15,1500.00,USD,CREDIT,ACC001
TXN002,2024-01-16,-200.00,USD,DEBIT,ACC001
TXN003,2024-01-17,5000.00,EUR,CREDIT,ACC002
TXN004,2024-01-18,-150.00,GBP,DEBIT,ACC003
TXN005,2024-01-19,2500.00,USD,CREDIT,ACC001`

		result, err := extractor.Extract(csvData, ontology)
		require.NoError(t, err)
		assert.Equal(t, 5, result.EntitiesExtracted)

		// Verify transaction types in properties
		for _, entity := range result.Entities {
			assert.NotEmpty(t, entity.Properties)
		}
	})

	// Test 4: Large dataset
	t.Run("Extract large dataset", func(t *testing.T) {
		ontology := &OntologyContext{
			BaseURI: "http://example.org/large",
			Classes: []OntologyClass{
				{URI: "http://example.org/large/Item", Label: "Item"},
			},
		}

		// Create large CSV (100 rows)
		csvData := "id,name,value\n"
		for i := 1; i <= 100; i++ {
			csvData += fmt.Sprintf("ID%d,Item%d,%d\n", i, i, i*10)
		}

		result, err := extractor.Extract(csvData, ontology)
		require.NoError(t, err)
		assert.Equal(t, 100, result.EntitiesExtracted)
		assert.Greater(t, result.TriplesGenerated, 300)
	})
}

// TestDeterministicExtractor_GetSupportedSourceTypes tests supported types
func TestDeterministicExtractor_GetSupportedSourceTypes(t *testing.T) {
	config := ExtractionConfig{
		OntologyID:     "test",
		SourceType:     SourceTypeCSV,
		ExtractionType: ExtractionDeterministic,
	}
	extractor := NewDeterministicExtractor(config)

	supported := extractor.GetSupportedSourceTypes()
	assert.Contains(t, supported, SourceTypeCSV)
	assert.Contains(t, supported, SourceTypeJSON)
	assert.NotContains(t, supported, SourceTypeText)
	assert.NotContains(t, supported, SourceTypeHTML)
}

// TestDeterministicExtractor_GetType tests extraction type
func TestDeterministicExtractor_GetType(t *testing.T) {
	config := ExtractionConfig{
		OntologyID:     "test",
		SourceType:     SourceTypeCSV,
		ExtractionType: ExtractionDeterministic,
	}
	extractor := NewDeterministicExtractor(config)

	assert.Equal(t, ExtractionDeterministic, extractor.GetType())
}
