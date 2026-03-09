package storage

import (
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/mimir-aip/mimir-aip-go/pkg/metadatastore"
	"github.com/mimir-aip/mimir-aip-go/pkg/models"
)

// Service provides storage management operations
type Service struct {
	store        metadatastore.MetadataStore
	plugins      map[string]models.StoragePlugin
	mu           sync.RWMutex
	pluginLoader *PluginLoader // nil when dynamic loading is not configured
}

// NewService creates a new storage service
func NewService(store metadatastore.MetadataStore) *Service {
	return &Service{
		store:   store,
		plugins: make(map[string]models.StoragePlugin),
	}
}

// RegisterPlugin registers a storage plugin
func (s *Service) RegisterPlugin(pluginType string, plugin models.StoragePlugin) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.plugins[pluginType] = plugin
	log.Printf("Registered storage plugin: %s", pluginType)
}

// GetPlugin retrieves a registered plugin by type
func (s *Service) GetPlugin(pluginType string) (models.StoragePlugin, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	plugin, ok := s.plugins[pluginType]
	if !ok {
		return nil, fmt.Errorf("storage plugin not found: %s", pluginType)
	}

	return plugin, nil
}

// CreateStorageConfig creates a new storage configuration for a project
func (s *Service) CreateStorageConfig(projectID, pluginType string, config map[string]interface{}) (*models.StorageConfig, error) {
	// Validate plugin type
	if _, err := s.GetPlugin(pluginType); err != nil {
		return nil, fmt.Errorf("invalid plugin type: %w", err)
	}

	now := time.Now().Format(time.RFC3339)
	storageConfig := &models.StorageConfig{
		ID:         uuid.New().String(),
		ProjectID:  projectID,
		PluginType: pluginType,
		Config:     config,
		Active:     true,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	// Save storage config
	if err := s.store.SaveStorageConfig(storageConfig); err != nil {
		return nil, fmt.Errorf("failed to save storage config: %w", err)
	}

	log.Printf("Created storage config %s for project %s using plugin %s", storageConfig.ID, projectID, pluginType)

	return storageConfig, nil
}

// GetStorageConfig retrieves a storage configuration by ID
func (s *Service) GetStorageConfig(storageID string) (*models.StorageConfig, error) {
	return s.store.GetStorageConfig(storageID)
}

// GetProjectStorageConfigs retrieves all storage configurations for a project
func (s *Service) GetProjectStorageConfigs(projectID string) ([]*models.StorageConfig, error) {
	return s.store.ListStorageConfigsByProject(projectID)
}

// UpdateStorageConfig updates a storage configuration
func (s *Service) UpdateStorageConfig(storageID string, config map[string]interface{}, active *bool) error {
	storageConfig, err := s.store.GetStorageConfig(storageID)
	if err != nil {
		return fmt.Errorf("storage config not found: %w", err)
	}

	if config != nil {
		storageConfig.Config = config
	}

	if active != nil {
		storageConfig.Active = *active
	}

	storageConfig.UpdatedAt = time.Now().Format(time.RFC3339)

	if err := s.store.SaveStorageConfig(storageConfig); err != nil {
		return fmt.Errorf("failed to update storage config: %w", err)
	}

	log.Printf("Updated storage config %s", storageID)

	return nil
}

// DeleteStorageConfig deletes a storage configuration
func (s *Service) DeleteStorageConfig(storageID string) error {
	if err := s.store.DeleteStorageConfig(storageID); err != nil {
		return fmt.Errorf("failed to delete storage config: %w", err)
	}

	log.Printf("Deleted storage config %s", storageID)

	return nil
}

// InitializeStorage initializes storage for a project using the specified configuration
func (s *Service) InitializeStorage(storageID string, ontology *models.OntologyDefinition) error {
	storageConfig, err := s.store.GetStorageConfig(storageID)
	if err != nil {
		return fmt.Errorf("storage config not found: %w", err)
	}

	plugin, err := s.GetPlugin(storageConfig.PluginType)
	if err != nil {
		return err
	}

	// Initialize plugin
	pluginConfig := &models.PluginConfig{
		ConnectionString: getConnectionString(storageConfig.Config),
		Credentials:      getCredentials(storageConfig.Config),
		Options:          getOptions(storageConfig.Config),
	}

	if err := plugin.Initialize(pluginConfig); err != nil {
		return fmt.Errorf("failed to initialize storage plugin: %w", err)
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

// Store stores CIR data in the specified storage
func (s *Service) Store(storageID string, cir *models.CIR) (*models.StorageResult, error) {
	// Validate CIR
	if err := cir.Validate(); err != nil {
		return nil, fmt.Errorf("invalid CIR: %w", err)
	}

	storageConfig, err := s.store.GetStorageConfig(storageID)
	if err != nil {
		return nil, fmt.Errorf("storage config not found: %w", err)
	}

	if !storageConfig.Active {
		return nil, fmt.Errorf("storage config is not active")
	}

	plugin, err := s.GetPlugin(storageConfig.PluginType)
	if err != nil {
		return nil, err
	}

	// Initialize plugin if not already done
	pluginConfig := &models.PluginConfig{
		ConnectionString: getConnectionString(storageConfig.Config),
		Credentials:      getCredentials(storageConfig.Config),
		Options:          getOptions(storageConfig.Config),
	}

	if err := plugin.Initialize(pluginConfig); err != nil {
		return nil, fmt.Errorf("failed to initialize storage plugin: %w", err)
	}

	// Store data
	result, err := plugin.Store(cir)
	if err != nil {
		return nil, fmt.Errorf("failed to store data: %w", err)
	}

	log.Printf("Stored CIR data in storage %s, affected items: %d", storageID, result.AffectedItems)

	return result, nil
}

// Retrieve retrieves data from storage using a query
func (s *Service) Retrieve(storageID string, query *models.CIRQuery) ([]*models.CIR, error) {
	storageConfig, err := s.store.GetStorageConfig(storageID)
	if err != nil {
		return nil, fmt.Errorf("storage config not found: %w", err)
	}

	if !storageConfig.Active {
		return nil, fmt.Errorf("storage config is not active")
	}

	plugin, err := s.GetPlugin(storageConfig.PluginType)
	if err != nil {
		return nil, err
	}

	// Initialize plugin
	pluginConfig := &models.PluginConfig{
		ConnectionString: getConnectionString(storageConfig.Config),
		Credentials:      getCredentials(storageConfig.Config),
		Options:          getOptions(storageConfig.Config),
	}

	if err := plugin.Initialize(pluginConfig); err != nil {
		return nil, fmt.Errorf("failed to initialize storage plugin: %w", err)
	}

	// Retrieve data
	results, err := plugin.Retrieve(query)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve data: %w", err)
	}

	log.Printf("Retrieved %d CIR objects from storage %s", len(results), storageID)

	return results, nil
}

// Update updates data in storage
func (s *Service) Update(storageID string, query *models.CIRQuery, updates *models.CIRUpdate) (*models.StorageResult, error) {
	storageConfig, err := s.store.GetStorageConfig(storageID)
	if err != nil {
		return nil, fmt.Errorf("storage config not found: %w", err)
	}

	if !storageConfig.Active {
		return nil, fmt.Errorf("storage config is not active")
	}

	plugin, err := s.GetPlugin(storageConfig.PluginType)
	if err != nil {
		return nil, err
	}

	// Initialize plugin
	pluginConfig := &models.PluginConfig{
		ConnectionString: getConnectionString(storageConfig.Config),
		Credentials:      getCredentials(storageConfig.Config),
		Options:          getOptions(storageConfig.Config),
	}

	if err := plugin.Initialize(pluginConfig); err != nil {
		return nil, fmt.Errorf("failed to initialize storage plugin: %w", err)
	}

	// Update data
	result, err := plugin.Update(query, updates)
	if err != nil {
		return nil, fmt.Errorf("failed to update data: %w", err)
	}

	log.Printf("Updated data in storage %s, affected items: %d", storageID, result.AffectedItems)

	return result, nil
}

// Delete deletes data from storage
func (s *Service) Delete(storageID string, query *models.CIRQuery) (*models.StorageResult, error) {
	storageConfig, err := s.store.GetStorageConfig(storageID)
	if err != nil {
		return nil, fmt.Errorf("storage config not found: %w", err)
	}

	if !storageConfig.Active {
		return nil, fmt.Errorf("storage config is not active")
	}

	plugin, err := s.GetPlugin(storageConfig.PluginType)
	if err != nil {
		return nil, err
	}

	// Initialize plugin
	pluginConfig := &models.PluginConfig{
		ConnectionString: getConnectionString(storageConfig.Config),
		Credentials:      getCredentials(storageConfig.Config),
		Options:          getOptions(storageConfig.Config),
	}

	if err := plugin.Initialize(pluginConfig); err != nil {
		return nil, fmt.Errorf("failed to initialize storage plugin: %w", err)
	}

	// Delete data
	result, err := plugin.Delete(query)
	if err != nil {
		return nil, fmt.Errorf("failed to delete data: %w", err)
	}

	log.Printf("Deleted data from storage %s, affected items: %d", storageID, result.AffectedItems)

	return result, nil
}

// GetStorageMetadata retrieves metadata about the storage system
func (s *Service) GetStorageMetadata(storageID string) (*models.StorageMetadata, error) {
	storageConfig, err := s.store.GetStorageConfig(storageID)
	if err != nil {
		return nil, fmt.Errorf("storage config not found: %w", err)
	}

	plugin, err := s.GetPlugin(storageConfig.PluginType)
	if err != nil {
		return nil, err
	}

	// Initialize plugin
	pluginConfig := &models.PluginConfig{
		ConnectionString: getConnectionString(storageConfig.Config),
		Credentials:      getCredentials(storageConfig.Config),
		Options:          getOptions(storageConfig.Config),
	}

	if err := plugin.Initialize(pluginConfig); err != nil {
		return nil, fmt.Errorf("failed to initialize storage plugin: %w", err)
	}

	return plugin.GetMetadata()
}

// HealthCheck performs a health check on the storage
func (s *Service) HealthCheck(storageID string) (bool, error) {
	storageConfig, err := s.store.GetStorageConfig(storageID)
	if err != nil {
		return false, fmt.Errorf("storage config not found: %w", err)
	}

	plugin, err := s.GetPlugin(storageConfig.PluginType)
	if err != nil {
		return false, err
	}

	// Initialize plugin
	pluginConfig := &models.PluginConfig{
		ConnectionString: getConnectionString(storageConfig.Config),
		Credentials:      getCredentials(storageConfig.Config),
		Options:          getOptions(storageConfig.Config),
	}

	if err := plugin.Initialize(pluginConfig); err != nil {
		return false, fmt.Errorf("failed to initialize storage plugin: %w", err)
	}

	return plugin.HealthCheck()
}

// GetIngestionHealth computes ingestion freshness/completeness/drift metrics
// across all storage configs in a project.
func (s *Service) GetIngestionHealth(projectID string) (*models.IngestionHealthReport, error) {
	configs, err := s.store.ListStorageConfigsByProject(projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to list storage configs: %w", err)
	}
	report := &models.IngestionHealthReport{
		ProjectID:   projectID,
		GeneratedAt: time.Now().UTC(),
		Sources:     make([]models.IngestionHealthSource, 0, len(configs)),
	}
	if len(configs) == 0 {
		report.Status = models.IngestionHealthCritical
		report.Recommendations = []string{"No storage sources configured for this project"}
		return report, nil
	}

	total := 0.0
	for _, cfg := range configs {
		source := models.IngestionHealthSource{
			StorageID:  cfg.ID,
			PluginType: cfg.PluginType,
			Status:     models.IngestionHealthCritical,
		}
		if !cfg.Active {
			source.Findings = append(source.Findings, "Storage config is inactive")
			report.Sources = append(report.Sources, source)
			continue
		}
		cirs, err := s.Retrieve(cfg.ID, &models.CIRQuery{Limit: 1000})
		if err != nil {
			source.Findings = append(source.Findings, fmt.Sprintf("Failed to retrieve sample data: %v", err))
			report.Sources = append(report.Sources, source)
			continue
		}
		source.SampleSize = len(cirs)
		if len(cirs) == 0 {
			source.Findings = append(source.Findings, "No ingested records found")
			report.Sources = append(report.Sources, source)
			continue
		}
		source.LastIngestedAt, source.FreshnessScore = freshnessFromSample(cirs)
		source.CompletenessScore = completenessFromSample(cirs)
		source.SchemaDriftScore = schemaDriftFromSample(cirs)
		source.OverallScore = roundTo3(0.45*source.FreshnessScore + 0.35*source.CompletenessScore + 0.20*source.SchemaDriftScore)
		source.Status = scoreToIngestionStatus(source.OverallScore)
		source.Findings = append(source.Findings, describeSourceFinding(source)...)
		total += source.OverallScore
		report.Sources = append(report.Sources, source)
	}
	report.OverallScore = roundTo3(total / float64(len(report.Sources)))
	report.Status = scoreToIngestionStatus(report.OverallScore)
	report.Recommendations = healthRecommendations(report)
	return report, nil
}

func freshnessFromSample(cirs []*models.CIR) (*time.Time, float64) {
	var latest *time.Time
	for _, cir := range cirs {
		ts := cir.Source.Timestamp.UTC()
		if ts.IsZero() {
			continue
		}
		if latest == nil || ts.After(*latest) {
			cp := ts
			latest = &cp
		}
	}
	if latest == nil {
		return nil, 0.10
	}
	age := time.Since(*latest)
	switch {
	case age <= time.Hour:
		return latest, 1.0
	case age <= 24*time.Hour:
		return latest, 0.85
	case age <= 7*24*time.Hour:
		return latest, 0.55
	default:
		return latest, 0.20
	}
}

func completenessFromSample(cirs []*models.CIR) float64 {
	totalFields := 0
	nonEmptyFields := 0
	for _, cir := range cirs {
		rows := cirRows(cir)
		for _, row := range rows {
			for _, val := range row {
				totalFields++
				if valuePresent(val) {
					nonEmptyFields++
				}
			}
		}
	}
	if totalFields == 0 {
		return 0.0
	}
	return roundTo3(float64(nonEmptyFields) / float64(totalFields))
}

func schemaDriftFromSample(cirs []*models.CIR) float64 {
	signatures := make(map[string]int)
	totalRows := 0
	for _, cir := range cirs {
		rows := cirRows(cir)
		for _, row := range rows {
			totalRows++
			sig := schemaSignature(row)
			signatures[sig]++
		}
	}
	if totalRows == 0 || len(signatures) <= 1 {
		return 1.0
	}
	maxCount := 0
	for _, c := range signatures {
		if c > maxCount {
			maxCount = c
		}
	}
	driftRatio := 1 - (float64(maxCount) / float64(totalRows))
	return roundTo3(1 - driftRatio)
}

func scoreToIngestionStatus(score float64) models.IngestionHealthStatus {
	switch {
	case score >= 0.80:
		return models.IngestionHealthHealthy
	case score >= 0.50:
		return models.IngestionHealthWarning
	default:
		return models.IngestionHealthCritical
	}
}

func describeSourceFinding(source models.IngestionHealthSource) []string {
	findings := make([]string, 0, 3)
	if source.FreshnessScore < 0.55 {
		findings = append(findings, "Data freshness is degraded")
	}
	if source.CompletenessScore < 0.70 {
		findings = append(findings, "Record completeness is low")
	}
	if source.SchemaDriftScore < 0.70 {
		findings = append(findings, "Schema drift detected in sampled rows")
	}
	if len(findings) == 0 {
		findings = append(findings, "Ingestion quality is stable")
	}
	return findings
}

func healthRecommendations(report *models.IngestionHealthReport) []string {
	recs := make([]string, 0, 3)
	if report.Status == models.IngestionHealthCritical {
		recs = append(recs, "Review failing or inactive storage sources and restore ingestion")
	}
	for _, src := range report.Sources {
		if src.Status == models.IngestionHealthCritical {
			recs = append(recs, fmt.Sprintf("Investigate source %s (%s)", src.StorageID, src.PluginType))
		}
	}
	if len(recs) == 0 {
		recs = append(recs, "No immediate ingestion remediation required")
	}
	return recs
}

func cirRows(cir *models.CIR) []map[string]interface{} {
	if cir == nil || cir.Data == nil {
		return nil
	}
	rows := make([]map[string]interface{}, 0)
	switch d := cir.Data.(type) {
	case map[string]interface{}:
		rows = append(rows, d)
	case []interface{}:
		for _, item := range d {
			if m, ok := item.(map[string]interface{}); ok {
				rows = append(rows, m)
			}
		}
	}
	return rows
}

func valuePresent(v interface{}) bool {
	if v == nil {
		return false
	}
	s := strings.TrimSpace(fmt.Sprintf("%v", v))
	return s != "" && s != "<nil>"
}

func schemaSignature(row map[string]interface{}) string {
	if len(row) == 0 {
		return "_empty"
	}
	keys := make([]string, 0, len(row))
	for k := range row {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return strings.Join(keys, "|")
}

func roundTo3(v float64) float64 {
	return float64(int(v*1000+0.5)) / 1000
}

// SetPluginLoader attaches a PluginLoader to the service, enabling dynamic
// storage plugin installation. Must be called before InstallExternalPlugin.
func (s *Service) SetPluginLoader(loader *PluginLoader) {
	s.pluginLoader = loader
}

// InstallExternalPlugin clones, compiles, and registers a storage plugin from
// a Git repository. The plugin metadata is persisted so it survives restarts.
func (s *Service) InstallExternalPlugin(req *models.ExternalStoragePluginInstallRequest) (*models.ExternalStoragePlugin, error) {
	if s.pluginLoader == nil {
		return nil, fmt.Errorf("dynamic storage plugin loading is not configured")
	}
	if req.RepositoryURL == "" {
		return nil, fmt.Errorf("repository_url is required")
	}

	gitRef := req.GitRef
	if gitRef == "" {
		gitRef = "main"
	}

	// Derive a plugin name from the repository URL (last path segment without .git).
	name := repoName(req.RepositoryURL)

	// Compile (or use cached .so) and load the plugin.
	sp, commitHash, err := s.pluginLoader.CompileAndLoad(name, req.RepositoryURL, gitRef, "")

	now := time.Now().UTC()
	record := &models.ExternalStoragePlugin{
		Name:          name,
		RepositoryURL: req.RepositoryURL,
		GitCommitHash: commitHash,
		Status:        "active",
		InstalledAt:   now,
		UpdatedAt:     now,
	}

	if err != nil {
		record.Status = "error"
		record.ErrorMessage = err.Error()
		// Persist the error record so the user can see what went wrong.
		_ = s.store.SaveExternalStoragePlugin(record)
		return record, fmt.Errorf("failed to compile storage plugin %s: %w", name, err)
	}

	// Register in the live plugin map.
	s.RegisterPlugin(name, sp)

	if err := s.store.SaveExternalStoragePlugin(record); err != nil {
		return nil, fmt.Errorf("plugin compiled but failed to persist metadata: %w", err)
	}

	log.Printf("Installed external storage plugin: %s @ %s", name, commitHash)
	return record, nil
}

// ListExternalPlugins returns all persisted external storage plugin records.
func (s *Service) ListExternalPlugins() ([]*models.ExternalStoragePlugin, error) {
	return s.store.ListExternalStoragePlugins()
}

// GetExternalPlugin returns metadata for a single external storage plugin.
func (s *Service) GetExternalPlugin(name string) (*models.ExternalStoragePlugin, error) {
	return s.store.GetExternalStoragePlugin(name)
}

// UninstallExternalPlugin removes the plugin from the live registry and
// deletes its persisted metadata. The .so file is removed from the cache.
// Note: Go plugins cannot be unloaded from memory; the process must restart
// for the removal to take full effect on in-flight storage operations.
func (s *Service) UninstallExternalPlugin(name string) error {
	// Remove from live map.
	s.mu.Lock()
	delete(s.plugins, name)
	s.mu.Unlock()

	// Remove .so and meta from cache.
	if s.pluginLoader != nil {
		_ = os.Remove(s.pluginLoader.soPath(name))
		_ = os.Remove(s.pluginLoader.metaPath(name))
	}

	if err := s.store.DeleteExternalStoragePlugin(name); err != nil {
		return fmt.Errorf("failed to delete external storage plugin record: %w", err)
	}

	log.Printf("Uninstalled external storage plugin: %s", name)
	return nil
}

// LoadInstalledExternalPlugins re-registers all persisted external storage
// plugins on startup. If a cached .so exists it is used directly; otherwise
// the plugin is recompiled from its recorded repository URL and commit hash.
func (s *Service) LoadInstalledExternalPlugins() error {
	if s.pluginLoader == nil {
		return nil
	}

	records, err := s.store.ListExternalStoragePlugins()
	if err != nil {
		return fmt.Errorf("failed to list persisted storage plugins: %w", err)
	}

	var loadErrs []string
	for _, rec := range records {
		if rec.Status == "error" {
			log.Printf("Skipping external storage plugin %s (status=error)", rec.Name)
			continue
		}

		// Try cached .so first.
		sp, loadErr := s.pluginLoader.LoadCached(rec.Name)
		if loadErr != nil {
			// Cache miss — recompile from stored repo+commit.
			log.Printf("Recompiling external storage plugin %s from %s @ %s", rec.Name, rec.RepositoryURL, rec.GitCommitHash)
			var compileHash string
			sp, compileHash, loadErr = s.pluginLoader.CompileAndLoad(
				rec.Name, rec.RepositoryURL, rec.GitCommitHash, rec.GitCommitHash,
			)
			if loadErr != nil {
				msg := fmt.Sprintf("%s: %v", rec.Name, loadErr)
				loadErrs = append(loadErrs, msg)
				rec.Status = "error"
				rec.ErrorMessage = loadErr.Error()
				_ = s.store.SaveExternalStoragePlugin(rec)
				continue
			}
			_ = compileHash
		}

		s.RegisterPlugin(rec.Name, sp)
		log.Printf("Loaded external storage plugin: %s", rec.Name)
	}

	if len(loadErrs) > 0 {
		return fmt.Errorf("failed to load %d external storage plugin(s): %s", len(loadErrs), strings.Join(loadErrs, "; "))
	}
	return nil
}

// repoName derives a plugin name from a Git URL — last path segment without ".git".
func repoName(repoURL string) string {
	parts := strings.Split(strings.TrimRight(repoURL, "/"), "/")
	name := parts[len(parts)-1]
	name = strings.TrimSuffix(name, ".git")
	return strings.ToLower(name)
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
