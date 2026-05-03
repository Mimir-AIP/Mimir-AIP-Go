package storage

import (
	"fmt"
	"github.com/mimir-aip/mimir-aip-go/pkg/metadatastore"
	"github.com/mimir-aip/mimir-aip-go/pkg/models"
	"github.com/mimir-aip/mimir-aip-go/pkg/pluginruntime"
	"log"
	"reflect"
)

// storagePluginFactory returns a fresh plugin instance for one configured storage operation.
type storagePluginFactory func() models.StoragePlugin

// Service provides storage management operations.
type Service struct {
	store        metadatastore.MetadataStore
	plugins      *pluginruntime.Registry[storagePluginFactory]
	pluginLoader *PluginLoader // nil when dynamic loading is not configured
}

// NewService creates a new storage service.
func NewService(store metadatastore.MetadataStore) *Service {
	return &Service{
		store:   store,
		plugins: pluginruntime.NewRegistry[storagePluginFactory](),
	}
}

// RegisterPlugin registers a storage plugin prototype used to create isolated instances per config.
func (s *Service) RegisterPlugin(pluginType string, plugin models.StoragePlugin) {
	s.plugins.Register(pluginType, newStoragePluginFactory(plugin))
	log.Printf("Registered storage plugin: %s", pluginType)
}

// GetPlugin retrieves a fresh plugin instance by type.
func (s *Service) GetPlugin(pluginType string) (models.StoragePlugin, error) {
	factory, ok := s.plugins.Get(pluginType)
	if !ok {
		return nil, fmt.Errorf("storage plugin not found: %s", pluginType)
	}

	plugin := factory()
	if plugin == nil {
		return nil, fmt.Errorf("storage plugin factory returned nil: %s", pluginType)
	}

	return plugin, nil
}

func newStoragePluginFactory(prototype models.StoragePlugin) storagePluginFactory {
	return func() models.StoragePlugin {
		if prototype == nil {
			return nil
		}

		value := reflect.ValueOf(prototype)
		if !value.IsValid() {
			return nil
		}
		if value.Kind() != reflect.Pointer || value.IsNil() {
			return prototype
		}

		clone := reflect.New(value.Elem().Type())
		clone.Elem().Set(value.Elem())
		plugin, _ := clone.Interface().(models.StoragePlugin)
		return plugin
	}
}

func (s *Service) configuredPlugin(storageConfig *models.StorageConfig) (models.StoragePlugin, error) {
	plugin, err := s.GetPlugin(storageConfig.PluginType)
	if err != nil {
		return nil, err
	}

	pluginConfig := &models.PluginConfig{
		ConnectionString: getConnectionString(storageConfig.Config),
		Credentials:      getCredentials(storageConfig.Config),
		Options:          getOptions(storageConfig.Config),
	}
	if err := plugin.Initialize(pluginConfig); err != nil {
		return nil, fmt.Errorf("failed to initialize storage plugin: %w", err)
	}

	return plugin, nil
}

// Helper functions to extract config values

func getConnectionString(config map[string]interface{}) string {
	if cs, ok := config["connection_string"].(string); ok {
		return cs
	}
	return ""
}

func getCredentials(config map[string]interface{}) map[string]interface{} {
	if creds, ok := config["credentials"].(map[string]interface{}); ok {
		return creds
	}
	return make(map[string]interface{})
}

func getOptions(config map[string]interface{}) map[string]interface{} {
	if opts, ok := config["options"].(map[string]interface{}); ok {
		return opts
	}
	return make(map[string]interface{})
}
