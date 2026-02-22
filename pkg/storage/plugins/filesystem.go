package plugins

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/mimir-aip/mimir-aip-go/pkg/models"
	"github.com/mimir-aip/mimir-aip-go/pkg/storage"
)

// FilesystemPlugin implements the StoragePlugin interface for local filesystem storage
type FilesystemPlugin struct {
	basePath    string
	initialized bool
}

// NewFilesystemPlugin creates a new filesystem storage plugin
func NewFilesystemPlugin() *FilesystemPlugin {
	return &FilesystemPlugin{}
}

// Initialize initializes the filesystem plugin with configuration
func (f *FilesystemPlugin) Initialize(config *models.PluginConfig) error {
	// Get base path from connection string or options
	basePath := config.ConnectionString
	if basePath == "" {
		if path, ok := config.Options["base_path"].(string); ok {
			basePath = path
		} else {
			return fmt.Errorf("base_path is required for filesystem storage")
		}
	}

	// Create base directory if it doesn't exist
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return fmt.Errorf("failed to create base directory: %w", err)
	}

	f.basePath = basePath
	f.initialized = true

	return nil
}

// CreateSchema creates directory structure based on ontology
func (f *FilesystemPlugin) CreateSchema(ontology *models.OntologyDefinition) error {
	if !f.initialized {
		return fmt.Errorf("plugin not initialized")
	}

	// Create a directory for each entity type
	for _, entity := range ontology.Entities {
		entityDir := filepath.Join(f.basePath, entity.Name)
		if err := os.MkdirAll(entityDir, 0755); err != nil {
			return fmt.Errorf("failed to create entity directory %s: %w", entity.Name, err)
		}
	}

	// Store ontology definition
	ontologyPath := filepath.Join(f.basePath, "ontology.json")
	ontologyData, err := json.MarshalIndent(ontology, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal ontology: %w", err)
	}

	if err := ioutil.WriteFile(ontologyPath, ontologyData, 0644); err != nil {
		return fmt.Errorf("failed to write ontology file: %w", err)
	}

	return nil
}

// Store stores CIR data to the filesystem
func (f *FilesystemPlugin) Store(cir *models.CIR) (*models.StorageResult, error) {
	if !f.initialized {
		return nil, fmt.Errorf("plugin not initialized")
	}

	// Validate CIR
	if err := cir.Validate(); err != nil {
		return nil, fmt.Errorf("invalid CIR: %w", err)
	}

	// Determine entity type from CIR data or use default
	entityType := f.inferEntityType(cir)

	// Get entity directory
	entityDir := filepath.Join(f.basePath, entityType)
	if err := os.MkdirAll(entityDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create entity directory: %w", err)
	}

	// Check if data is array or single object
	affectedItems := 0

	if arr, err := cir.GetDataAsArray(); err == nil {
		// Store each item in the array
		for _, item := range arr {
			itemCIR := &models.CIR{
				Version:  cir.Version,
				Source:   cir.Source,
				Data:     item,
				Metadata: cir.Metadata,
			}

			if err := f.storeItem(entityDir, itemCIR); err != nil {
				return nil, fmt.Errorf("failed to store item: %w", err)
			}
			affectedItems++
		}
	} else {
		// Store single item
		if err := f.storeItem(entityDir, cir); err != nil {
			return nil, fmt.Errorf("failed to store item: %w", err)
		}
		affectedItems = 1
	}

	return &models.StorageResult{
		Success:       true,
		AffectedItems: affectedItems,
	}, nil
}

// storeItem stores a single CIR item
func (f *FilesystemPlugin) storeItem(entityDir string, cir *models.CIR) error {
	// Generate unique filename
	itemID := uuid.New().String()
	filename := fmt.Sprintf("%s.json", itemID)
	filePath := filepath.Join(entityDir, filename)

	// Marshal CIR to JSON
	data, err := json.MarshalIndent(cir, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal CIR: %w", err)
	}

	// Write to file
	if err := ioutil.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// Retrieve retrieves data from the filesystem using a query
func (f *FilesystemPlugin) Retrieve(query *models.CIRQuery) ([]*models.CIR, error) {
	if !f.initialized {
		return nil, fmt.Errorf("plugin not initialized")
	}

	// Validate query
	if err := storage.ValidateCIRQuery(query); err != nil {
		return nil, fmt.Errorf("invalid query: %w", err)
	}

	// Determine which entity directory to search
	entityType := query.EntityType
	if entityType == "" {
		entityType = "default"
	}

	entityDir := filepath.Join(f.basePath, entityType)

	// Check if directory exists
	if _, err := os.Stat(entityDir); os.IsNotExist(err) {
		return []*models.CIR{}, nil // Return empty list if entity type doesn't exist
	}

	// Read all files in the entity directory
	files, err := ioutil.ReadDir(entityDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read entity directory: %w", err)
	}

	results := make([]*models.CIR, 0)

	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".json") {
			continue
		}

		filePath := filepath.Join(entityDir, file.Name())
		data, err := ioutil.ReadFile(filePath)
		if err != nil {
			continue // Skip files that can't be read
		}

		var cir models.CIR
		if err := json.Unmarshal(data, &cir); err != nil {
			continue // Skip invalid JSON
		}

		// Apply filters
		if f.matchesFilters(&cir, query.Filters) {
			results = append(results, &cir)
		}
	}

	// Apply limit and offset
	if query.Offset > 0 && query.Offset < len(results) {
		results = results[query.Offset:]
	}

	if query.Limit > 0 && query.Limit < len(results) {
		results = results[:query.Limit]
	}

	return results, nil
}

// matchesFilters checks if a CIR object matches the query filters
func (f *FilesystemPlugin) matchesFilters(cir *models.CIR, filters []models.CIRCondition) bool {
	if len(filters) == 0 {
		return true
	}

	// Get data as map
	dataMap, err := cir.GetDataAsMap()
	if err != nil {
		return false
	}

	// Check all filters
	for _, filter := range filters {
		value, exists := dataMap[filter.Attribute]
		if !exists {
			return false
		}

		if !f.evaluateCondition(value, filter.Operator, filter.Value) {
			return false
		}
	}

	return true
}

// evaluateCondition evaluates a filter condition
func (f *FilesystemPlugin) evaluateCondition(value interface{}, operator string, expected interface{}) bool {
	switch operator {
	case "eq":
		return fmt.Sprintf("%v", value) == fmt.Sprintf("%v", expected)
	case "neq":
		return fmt.Sprintf("%v", value) != fmt.Sprintf("%v", expected)
	case "like":
		valueStr := fmt.Sprintf("%v", value)
		expectedStr := fmt.Sprintf("%v", expected)
		return strings.Contains(strings.ToLower(valueStr), strings.ToLower(expectedStr))
	default:
		return false
	}
}

// Update updates existing CIR data (filesystem implementation is simple: delete and re-create)
func (f *FilesystemPlugin) Update(query *models.CIRQuery, updates *models.CIRUpdate) (*models.StorageResult, error) {
	if !f.initialized {
		return nil, fmt.Errorf("plugin not initialized")
	}

	// Retrieve matching items
	items, err := f.Retrieve(query)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve items: %w", err)
	}

	affectedItems := 0

	for _, item := range items {
		// Apply updates to data
		dataMap, err := item.GetDataAsMap()
		if err != nil {
			continue
		}

		// Apply updates
		for key, value := range updates.Updates {
			dataMap[key] = value
		}

		// Update the CIR data
		item.Data = dataMap
		item.UpdateSize()

		// Find and update the file
		entityType := f.inferEntityType(item)
		entityDir := filepath.Join(f.basePath, entityType)

		// For simplicity, we'll just re-store the item
		// In a real implementation, we'd find the exact file and update it
		if err := f.storeItem(entityDir, item); err != nil {
			continue
		}

		affectedItems++
	}

	return &models.StorageResult{
		Success:       true,
		AffectedItems: affectedItems,
	}, nil
}

// Delete deletes CIR data from the filesystem
func (f *FilesystemPlugin) Delete(query *models.CIRQuery) (*models.StorageResult, error) {
	if !f.initialized {
		return nil, fmt.Errorf("plugin not initialized")
	}

	// Determine which entity directory to search
	entityType := query.EntityType
	if entityType == "" {
		entityType = "default"
	}

	entityDir := filepath.Join(f.basePath, entityType)

	// Check if directory exists
	if _, err := os.Stat(entityDir); os.IsNotExist(err) {
		return &models.StorageResult{Success: true, AffectedItems: 0}, nil
	}

	// Read all files in the entity directory
	files, err := ioutil.ReadDir(entityDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read entity directory: %w", err)
	}

	affectedItems := 0

	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".json") {
			continue
		}

		filePath := filepath.Join(entityDir, file.Name())
		data, err := ioutil.ReadFile(filePath)
		if err != nil {
			continue
		}

		var cir models.CIR
		if err := json.Unmarshal(data, &cir); err != nil {
			continue
		}

		// Check if matches filters
		if f.matchesFilters(&cir, query.Filters) {
			if err := os.Remove(filePath); err != nil {
				continue
			}
			affectedItems++
		}
	}

	return &models.StorageResult{
		Success:       true,
		AffectedItems: affectedItems,
	}, nil
}

// GetMetadata returns metadata about the filesystem storage
func (f *FilesystemPlugin) GetMetadata() (*models.StorageMetadata, error) {
	return &models.StorageMetadata{
		StorageType: "filesystem",
		Version:     "1.0.0",
		Capabilities: []string{
			"store",
			"retrieve",
			"update",
			"delete",
			"schema_creation",
		},
	}, nil
}

// HealthCheck checks if the filesystem storage is accessible
func (f *FilesystemPlugin) HealthCheck() (bool, error) {
	if !f.initialized {
		return false, fmt.Errorf("plugin not initialized")
	}

	// Check if base path exists and is writable
	testFile := filepath.Join(f.basePath, ".healthcheck")
	if err := ioutil.WriteFile(testFile, []byte("ok"), 0644); err != nil {
		return false, fmt.Errorf("filesystem not writable: %w", err)
	}

	// Clean up test file
	os.Remove(testFile)

	return true, nil
}

// inferEntityType attempts to infer the entity type from CIR data
func (f *FilesystemPlugin) inferEntityType(cir *models.CIR) string {
	// Try to get entity type from source metadata
	if entityType, ok := cir.GetParameter("entity_type"); ok {
		if typeStr, ok := entityType.(string); ok {
			return typeStr
		}
	}

	// Try to infer from data structure
	if dataMap, err := cir.GetDataAsMap(); err == nil {
		// Simple heuristic: if it has certain fields, classify it
		if _, hasName := dataMap["name"]; hasName {
			if _, hasDept := dataMap["department"]; hasDept {
				return "Employee"
			}
			if _, hasLoc := dataMap["location"]; hasLoc {
				return "Company"
			}
		}
	}

	// Default entity type
	return "default"
}
