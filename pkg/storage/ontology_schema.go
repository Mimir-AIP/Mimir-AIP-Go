package storage

import (
	"fmt"
	"github.com/mimir-aip/mimir-aip-go/pkg/models"
	"log"
	"sort"
	"strings"
	"time"
)

// InitializeStorage initializes storage for a project using the specified configuration
func (s *Service) InitializeStorage(storageID string, ontology *models.OntologyDefinition) error {
	storageConfig, err := s.store.GetStorageConfig(storageID)
	if err != nil {
		return fmt.Errorf("storage config not found: %w", err)
	}

	plugin, err := s.configuredPlugin(storageConfig)
	if err != nil {
		return err
	}
	// Create schema if ontology is provided
	if ontology != nil {
		if err := plugin.CreateSchema(ontology); err != nil {
			return fmt.Errorf("failed to create storage schema: %w", err)
		}
		storageConfig.UpdatedAt = time.Now().Format(time.RFC3339)
		if err := s.store.SaveStorageConfig(storageConfig); err != nil {
			return fmt.Errorf("failed to update storage config: %w", err)
		}
	}

	log.Printf("Initialized storage %s with plugin %s", storageID, storageConfig.PluginType)

	return nil
}

func ontologyDefinitionFromCompiled(compiled *models.CompiledOntology) *models.OntologyDefinition {
	if compiled == nil {
		return nil
	}
	propertiesByDomain := map[string][]models.CompiledOntologyProperty{}
	for _, property := range compiled.Properties {
		if property.Kind != "datatype" {
			continue
		}
		if len(property.Domain) == 0 {
			continue
		}
		for _, domain := range property.Domain {
			propertiesByDomain[domain] = append(propertiesByDomain[domain], property)
		}
	}

	definition := &models.OntologyDefinition{
		Entities:      make([]models.EntityDefinition, 0, len(compiled.Classes)),
		Relationships: make([]models.RelationshipDefinition, 0, len(compiled.Relations)),
	}
	for _, class := range compiled.Classes {
		attributes := make([]models.AttributeDefinition, 0)
		primaryKey := make([]string, 0)
		for _, property := range propertiesByDomain[class.ID] {
			attr := models.AttributeDefinition{Name: property.Name, Type: storageTypeFromOntologyRange(property.Range), Nullable: true}
			attributes = append(attributes, attr)
			if isIdentityProperty(class.ID, property.Name) {
				primaryKey = append(primaryKey, property.Name)
			}
		}
		sort.Slice(attributes, func(i, j int) bool { return attributes[i].Name < attributes[j].Name })
		definition.Entities = append(definition.Entities, models.EntityDefinition{Name: class.Name, Attributes: attributes, PrimaryKey: primaryKey})
	}
	for _, relation := range compiled.Relations {
		definition.Relationships = append(definition.Relationships, models.RelationshipDefinition{Name: relation.Name, FromEntity: relation.FromClass, ToEntity: relation.ToClass, Type: "many-to-many"})
	}
	sort.Slice(definition.Entities, func(i, j int) bool { return definition.Entities[i].Name < definition.Entities[j].Name })
	sort.Slice(definition.Relationships, func(i, j int) bool {
		if definition.Relationships[i].FromEntity != definition.Relationships[j].FromEntity {
			return definition.Relationships[i].FromEntity < definition.Relationships[j].FromEntity
		}
		return definition.Relationships[i].Name < definition.Relationships[j].Name
	})
	return definition
}

func storageTypeFromOntologyRange(ranges []string) string {
	if len(ranges) == 0 {
		return "string"
	}
	switch strings.ToLower(ranges[0]) {
	case "integer", "int", "long", "short", "decimal", "float", "double":
		return "number"
	case "boolean", "bool":
		return "boolean"
	case "date", "datetime", "time":
		return "date"
	case "json", "object":
		return "json"
	default:
		return "string"
	}
}

func isIdentityProperty(classID, propertyName string) bool {
	property := strings.ToLower(propertyName)
	class := strings.ToLower(classID)
	return property == "id" || property == class+"id" || property == class+"_id"
}
