package connectors

import (
	"fmt"
	"sort"
	"strings"

	"github.com/mimir-aip/mimir-aip-go/pkg/models"
	"github.com/mimir-aip/mimir-aip-go/pkg/pipeline"
	"github.com/mimir-aip/mimir-aip-go/pkg/scheduler"
	"github.com/mimir-aip/mimir-aip-go/pkg/storage"
)

type connectorDescriptor struct {
	template   models.ConnectorTemplate
	buildSteps func(req *models.ConnectorSetupRequest) ([]models.PipelineStep, error)
}

// Service exposes bundled, use-case-agnostic ingestion connectors that compile down to
// ordinary pipelines and optional schedules.
type Service struct {
	pipelineService  *pipeline.Service
	schedulerService *scheduler.Service
	storageService   *storage.Service
	registry         map[string]connectorDescriptor
}

// NewService creates a connector materialization service.
func NewService(pipelineService *pipeline.Service, schedulerService *scheduler.Service, storageService *storage.Service) *Service {
	return &Service{
		pipelineService:  pipelineService,
		schedulerService: schedulerService,
		storageService:   storageService,
		registry:         builtInRegistry(),
	}
}

// ListTemplates returns the bundled connector catalog.
func (s *Service) ListTemplates() []models.ConnectorTemplate {
	templates := make([]models.ConnectorTemplate, 0, len(s.registry))
	for _, descriptor := range s.registry {
		templates = append(templates, descriptor.template)
	}
	sort.Slice(templates, func(i, j int) bool { return templates[i].Kind < templates[j].Kind })
	return templates
}

// GetTemplate retrieves one bundled connector template by kind.
func (s *Service) GetTemplate(kind string) (*models.ConnectorTemplate, error) {
	descriptor, ok := s.registry[strings.TrimSpace(kind)]
	if !ok {
		return nil, fmt.Errorf("connector template not found: %s", kind)
	}
	template := descriptor.template
	return &template, nil
}

// Materialize creates a standard pipeline and optional schedule from a connector template.
func (s *Service) Materialize(req *models.ConnectorSetupRequest) (*models.ConnectorSetupResponse, error) {
	if err := s.validateMaterializeRequest(req); err != nil {
		return nil, err
	}
	descriptor, ok := s.registry[strings.TrimSpace(req.Kind)]
	if !ok {
		return nil, fmt.Errorf("connector template not found: %s", req.Kind)
	}
	if err := s.validateStorageOwnership(req.ProjectID, req.StorageID); err != nil {
		return nil, err
	}
	if req.Schedule != nil && strings.TrimSpace(req.Schedule.CronSchedule) != "" && !descriptor.template.SupportsSchedule {
		return nil, fmt.Errorf("connector template %s does not support schedules", descriptor.template.Kind)
	}

	steps, err := descriptor.buildSteps(req)
	if err != nil {
		return nil, err
	}
	pipelineReq := &models.PipelineCreateRequest{
		ProjectID:   req.ProjectID,
		Name:        req.Name,
		Type:        descriptor.template.PipelineType,
		Description: req.Description,
		Steps:       steps,
	}
	createdPipeline, err := s.pipelineService.Create(pipelineReq)
	if err != nil {
		return nil, fmt.Errorf("failed to create connector pipeline: %w", err)
	}

	var createdSchedule *models.Schedule
	if req.Schedule != nil && strings.TrimSpace(req.Schedule.CronSchedule) != "" {
		scheduleName := strings.TrimSpace(req.Schedule.Name)
		if scheduleName == "" {
			scheduleName = req.Name + " schedule"
		}
		createdSchedule, err = s.schedulerService.Create(&models.ScheduleCreateRequest{
			ProjectID:    req.ProjectID,
			Name:         scheduleName,
			Pipelines:    []string{createdPipeline.ID},
			CronSchedule: req.Schedule.CronSchedule,
			Enabled:      req.Schedule.Enabled,
		})
		if err != nil {
			if rollbackErr := s.pipelineService.Delete(createdPipeline.ID); rollbackErr != nil {
				return nil, fmt.Errorf("failed to create connector schedule: %v (rollback failed: %w)", err, rollbackErr)
			}
			return nil, fmt.Errorf("failed to create connector schedule: %w", err)
		}
	}

	return &models.ConnectorSetupResponse{
		Template: descriptor.template,
		Pipeline: createdPipeline,
		Schedule: createdSchedule,
	}, nil
}

func (s *Service) validateMaterializeRequest(req *models.ConnectorSetupRequest) error {
	if req == nil {
		return fmt.Errorf("connector setup request is required")
	}
	if strings.TrimSpace(req.ProjectID) == "" {
		return fmt.Errorf("project_id is required")
	}
	if strings.TrimSpace(req.Kind) == "" {
		return fmt.Errorf("kind is required")
	}
	if strings.TrimSpace(req.Name) == "" {
		return fmt.Errorf("name is required")
	}
	if strings.TrimSpace(req.StorageID) == "" {
		return fmt.Errorf("storage_id is required")
	}
	return nil
}

func (s *Service) validateStorageOwnership(projectID, storageID string) error {
	config, err := s.storageService.GetStorageConfig(storageID)
	if err != nil {
		return fmt.Errorf("storage_id is invalid: %w", err)
	}
	if config.ProjectID != projectID {
		return fmt.Errorf("storage_id %s belongs to project %s, not %s", storageID, config.ProjectID, projectID)
	}
	return nil
}

func builtInRegistry() map[string]connectorDescriptor {
	registry := map[string]connectorDescriptor{}
	for _, descriptor := range []connectorDescriptor{
		newSQLIncrementalDescriptor(),
		newHTTPJSONPollDescriptor(),
		newRSSPollDescriptor(),
		newCSVDropDescriptor(),
	} {
		registry[descriptor.template.Kind] = descriptor
	}
	return registry
}

func newSQLIncrementalDescriptor() connectorDescriptor {
	template := models.ConnectorTemplate{
		Kind:             "sql_incremental",
		Label:            "Incremental SQL Table",
		Description:      "Incrementally polls a MySQL or PostgreSQL table using a cursor column and persists new rows into Mimir storage.",
		Category:         "database",
		PipelineType:     models.PipelineTypeIngestion,
		SupportsSchedule: true,
		Fields: []models.ConnectorField{
			{Name: "driver", Label: "Database Driver", Type: "select", Required: true, Default: "mysql", Options: []models.ConnectorFieldOption{{Value: "mysql", Label: "MySQL"}, {Value: "postgresql", Label: "PostgreSQL"}}},
			{Name: "dsn", Label: "Connection String", Type: "text", Required: true, Description: "Driver DSN used to connect to the source database."},
			{Name: "table", Label: "Source Table", Type: "text", Required: true, Description: "Table or view name to poll incrementally."},
			{Name: "cursor_column", Label: "Cursor Column", Type: "text", Required: true, Description: "Monotonic numeric or timestamp column used for incremental polling."},
			{Name: "limit", Label: "Batch Size", Type: "number", Required: false, Default: 500},
		},
	}
	return connectorDescriptor{
		template: template,
		buildSteps: func(req *models.ConnectorSetupRequest) ([]models.PipelineStep, error) {
			source := normalizedSourceConfig(req.SourceConfig)
			driver := stringValue(source, "driver", "mysql")
			dsn := requiredString(source, "dsn")
			table := requiredString(source, "table")
			cursorColumn := requiredString(source, "cursor_column")
			if dsn == "" || table == "" || cursorColumn == "" {
				return nil, fmt.Errorf("sql_incremental requires driver, dsn, table, and cursor_column")
			}
			limit := intValue(source, "limit", 500)
			loadCheckpoint, storeBatch, saveCheckpoint := connectorBaseSteps(template.Kind, req.StorageID)
			fetch := models.PipelineStep{
				Name:   "poll_sql_rows",
				Plugin: "default",
				Action: "poll_sql_incremental",
				Parameters: map[string]interface{}{
					"driver":        driver,
					"dsn":           dsn,
					"table":         table,
					"cursor_column": cursorColumn,
					"limit":         limit,
					"checkpoint":    "{{context.load_source_checkpoint.checkpoint}}",
				},
			}
			storeBatch.Parameters["items"] = "{{context.poll_sql_rows.items}}"
			storeBatch.Parameters["source_uri"] = fmt.Sprintf("sql://%s/%s", driver, table)
			saveCheckpoint.Parameters["checkpoint"] = "{{context.poll_sql_rows.checkpoint}}"
			return []models.PipelineStep{loadCheckpoint, fetch, storeBatch, saveCheckpoint}, nil
		},
	}
}

func newHTTPJSONPollDescriptor() connectorDescriptor {
	template := models.ConnectorTemplate{
		Kind:             "http_json_poll",
		Label:            "HTTP JSON Feed",
		Description:      "Polls a JSON HTTP endpoint with checkpointed ETag and Last-Modified support, then stores new items.",
		Category:         "http",
		PipelineType:     models.PipelineTypeIngestion,
		SupportsSchedule: true,
		Fields: []models.ConnectorField{
			{Name: "url", Label: "Endpoint URL", Type: "text", Required: true},
			{Name: "method", Label: "HTTP Method", Type: "select", Required: false, Default: "GET", Options: []models.ConnectorFieldOption{{Value: "GET", Label: "GET"}, {Value: "POST", Label: "POST"}}},
			{Name: "item_path", Label: "Item Path", Type: "text", Required: false, Description: "Optional dot path inside the JSON payload where item arrays live."},
		},
	}
	return connectorDescriptor{
		template: template,
		buildSteps: func(req *models.ConnectorSetupRequest) ([]models.PipelineStep, error) {
			source := normalizedSourceConfig(req.SourceConfig)
			url := requiredString(source, "url")
			if url == "" {
				return nil, fmt.Errorf("http_json_poll requires url")
			}
			loadCheckpoint, storeBatch, saveCheckpoint := connectorBaseSteps(template.Kind, req.StorageID)
			fetchParams := map[string]interface{}{
				"url":        url,
				"method":     stringValue(source, "method", "GET"),
				"checkpoint": "{{context.load_source_checkpoint.checkpoint}}",
			}
			if itemPath := strings.TrimSpace(stringValue(source, "item_path", "")); itemPath != "" {
				fetchParams["items_path"] = itemPath
			}
			fetch := models.PipelineStep{Name: "poll_http_feed", Plugin: "default", Action: "poll_http_json", Parameters: fetchParams}
			storeBatch.Parameters["items"] = "{{context.poll_http_feed.items}}"
			storeBatch.Parameters["source_uri"] = url
			storeBatch.Parameters["source_type"] = "api"
			saveCheckpoint.Parameters["checkpoint"] = "{{context.poll_http_feed.checkpoint}}"
			return []models.PipelineStep{loadCheckpoint, fetch, storeBatch, saveCheckpoint}, nil
		},
	}
}

func newRSSPollDescriptor() connectorDescriptor {
	template := models.ConnectorTemplate{
		Kind:             "rss_poll",
		Label:            "RSS or Atom Feed",
		Description:      "Polls an RSS or Atom feed and checkpoints previously seen items to avoid duplicates.",
		Category:         "feed",
		PipelineType:     models.PipelineTypeIngestion,
		SupportsSchedule: true,
		Fields:           []models.ConnectorField{{Name: "url", Label: "Feed URL", Type: "text", Required: true}},
	}
	return connectorDescriptor{
		template: template,
		buildSteps: func(req *models.ConnectorSetupRequest) ([]models.PipelineStep, error) {
			source := normalizedSourceConfig(req.SourceConfig)
			url := requiredString(source, "url")
			if url == "" {
				return nil, fmt.Errorf("rss_poll requires url")
			}
			loadCheckpoint, storeBatch, saveCheckpoint := connectorBaseSteps(template.Kind, req.StorageID)
			fetch := models.PipelineStep{
				Name:   "poll_rss_feed",
				Plugin: "default",
				Action: "poll_rss",
				Parameters: map[string]interface{}{
					"url":        url,
					"checkpoint": "{{context.load_source_checkpoint.checkpoint}}",
				},
			}
			storeBatch.Parameters["items"] = "{{context.poll_rss_feed.items}}"
			storeBatch.Parameters["source_uri"] = url
			storeBatch.Parameters["source_type"] = "feed"
			saveCheckpoint.Parameters["checkpoint"] = "{{context.poll_rss_feed.checkpoint}}"
			return []models.PipelineStep{loadCheckpoint, fetch, storeBatch, saveCheckpoint}, nil
		},
	}
}

func newCSVDropDescriptor() connectorDescriptor {
	template := models.ConnectorTemplate{
		Kind:             "csv_drop",
		Label:            "CSV File Drop",
		Description:      "Polls a directory or glob for new CSV files, ingests unseen file drops, and checkpoints processed files.",
		Category:         "file",
		PipelineType:     models.PipelineTypeIngestion,
		SupportsSchedule: true,
		Fields: []models.ConnectorField{
			{Name: "path_glob", Label: "Path Glob", Type: "text", Required: true, Description: "File glob such as /data/drop/*.csv or ./demo/*.csv."},
			{Name: "has_header", Label: "Has Header Row", Type: "boolean", Required: false, Default: true},
			{Name: "delimiter", Label: "Delimiter", Type: "text", Required: false, Default: ","},
		},
	}
	return connectorDescriptor{
		template: template,
		buildSteps: func(req *models.ConnectorSetupRequest) ([]models.PipelineStep, error) {
			source := normalizedSourceConfig(req.SourceConfig)
			pathGlob := requiredString(source, "path_glob")
			if pathGlob == "" {
				return nil, fmt.Errorf("csv_drop requires path_glob")
			}
			loadCheckpoint, storeBatch, saveCheckpoint := connectorBaseSteps(template.Kind, req.StorageID)
			fetch := models.PipelineStep{
				Name:   "poll_csv_drop",
				Plugin: "default",
				Action: "poll_csv_drop",
				Parameters: map[string]interface{}{
					"path_glob":  pathGlob,
					"has_header": boolValue(source, "has_header", true),
					"delimiter":  stringValue(source, "delimiter", ","),
					"checkpoint": "{{context.load_source_checkpoint.checkpoint}}",
				},
			}
			storeBatch.Parameters["items"] = "{{context.poll_csv_drop.items}}"
			storeBatch.Parameters["source_uri"] = pathGlob
			storeBatch.Parameters["source_type"] = "file"
			saveCheckpoint.Parameters["checkpoint"] = "{{context.poll_csv_drop.checkpoint}}"
			return []models.PipelineStep{loadCheckpoint, fetch, storeBatch, saveCheckpoint}, nil
		},
	}
}

func normalizedSourceConfig(source map[string]interface{}) map[string]interface{} {
	if source == nil {
		return map[string]interface{}{}
	}
	return source
}

func connectorBaseSteps(kind, storageID string) (models.PipelineStep, models.PipelineStep, models.PipelineStep) {
	loadCheckpoint := models.PipelineStep{
		Name:   "load_source_checkpoint",
		Plugin: "default",
		Action: "load_checkpoint",
		Parameters: map[string]interface{}{
			"step_name": "connector_state",
			"default":   map[string]interface{}{},
		},
	}
	storeBatch := models.PipelineStep{
		Name:   "store_source_records",
		Plugin: "default",
		Action: "store_cir_batch",
		Parameters: map[string]interface{}{
			"storage_id": storageID,
			"source_uri": "connector://" + kind,
		},
	}
	saveCheckpoint := models.PipelineStep{
		Name:   "save_source_checkpoint",
		Plugin: "default",
		Action: "save_checkpoint",
		Parameters: map[string]interface{}{
			"step_name":  "connector_state",
			"version":    "{{context.load_source_checkpoint.version}}",
			"checkpoint": nil,
		},
	}
	return loadCheckpoint, storeBatch, saveCheckpoint
}

func requiredString(values map[string]interface{}, key string) string {
	return strings.TrimSpace(stringValue(values, key, ""))
}

func stringValue(values map[string]interface{}, key, fallback string) string {
	if values == nil {
		return fallback
	}
	if raw, ok := values[key]; ok {
		switch typed := raw.(type) {
		case string:
			if strings.TrimSpace(typed) != "" {
				return typed
			}
		default:
			return strings.TrimSpace(fmt.Sprintf("%v", typed))
		}
	}
	return fallback
}

func intValue(values map[string]interface{}, key string, fallback int) int {
	if values == nil {
		return fallback
	}
	switch typed := values[key].(type) {
	case int:
		if typed > 0 {
			return typed
		}
	case float64:
		if typed > 0 {
			return int(typed)
		}
	case string:
		if strings.TrimSpace(typed) == "" {
			return fallback
		}
		var parsed int
		if _, err := fmt.Sscanf(typed, "%d", &parsed); err == nil && parsed > 0 {
			return parsed
		}
	}
	return fallback
}

func boolValue(values map[string]interface{}, key string, fallback bool) bool {
	if values == nil {
		return fallback
	}
	if raw, ok := values[key]; ok {
		switch typed := raw.(type) {
		case bool:
			return typed
		case string:
			switch strings.ToLower(strings.TrimSpace(typed)) {
			case "true", "1", "yes", "on":
				return true
			case "false", "0", "no", "off":
				return false
			}
		}
	}
	return fallback
}