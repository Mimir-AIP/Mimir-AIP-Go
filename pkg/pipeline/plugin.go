package pipeline

import (
	"fmt"
	"github.com/mimir-aip/mimir-aip-go/pkg/models"
	"net/http"
	"time"
)

// CIRStorer is the subset of storage.Service required by the pipeline plugin
// to persist CIR records. Defined as an interface to avoid a hard import cycle.
type CIRStorer interface {
	Store(storageID string, cir *models.CIR) (*models.StorageResult, error)
}

// PipelineCheckpointStore is the subset of metadata persistence required by built-in
// checkpoint actions. Implemented by the metadata store in-process and an HTTP client
// inside workers.
type PipelineCheckpointStore interface {
	GetPipelineCheckpoint(projectID, pipelineID, stepName, scope string) (*models.PipelineCheckpoint, error)
	SavePipelineCheckpoint(checkpoint *models.PipelineCheckpoint) error
}

// Plugin defines the interface for pipeline step executors
type Plugin interface {
	Execute(action string, params map[string]interface{}, ctx *models.PipelineContext) (map[string]interface{}, error)
}

// DefaultPlugin implements built-in pipeline actions
type DefaultPlugin struct {
	httpClient      *http.Client
	storageSvc      CIRStorer               // nil when storage integration is not configured
	checkpointStore PipelineCheckpointStore // nil when checkpoint persistence is unavailable
}

// NewDefaultPlugin creates a new default plugin instance without storage integration.
func NewDefaultPlugin() *DefaultPlugin {
	return NewDefaultPluginWithDeps(nil, nil)
}

// NewDefaultPluginWithStorage creates a default plugin that can persist CIR data
// via the provided CIRStorer (typically *storage.Service).
func NewDefaultPluginWithStorage(svc CIRStorer) *DefaultPlugin {
	return NewDefaultPluginWithDeps(svc, nil)
}

// NewDefaultPluginWithDeps creates a default plugin with optional persistence dependencies.
func NewDefaultPluginWithDeps(storageSvc CIRStorer, checkpointStore PipelineCheckpointStore) *DefaultPlugin {
	return &DefaultPlugin{
		httpClient:      &http.Client{Timeout: 30 * time.Second},
		storageSvc:      storageSvc,
		checkpointStore: checkpointStore,
	}
}

// Execute executes a default plugin action
func (p *DefaultPlugin) Execute(action string, params map[string]interface{}, ctx *models.PipelineContext) (map[string]interface{}, error) {
	switch action {
	case "http_request":
		return p.httpRequest(params, ctx)
	case "poll_http_json":
		return p.pollHTTPJSON(params, ctx)
	case "poll_rss":
		return p.pollRSS(params, ctx)
	case "poll_sql_incremental":
		return p.pollSQLIncremental(params, ctx)
	case "poll_csv_drop":
		return p.pollCSVDrop(params, ctx)
	case "ingest_csv":
		return p.ingestCSV(params, ctx)
	case "ingest_csv_url":
		return p.ingestCSVURL(params, ctx)
	case "query_sql":
		return p.querySQL(params, ctx)
	case "load_checkpoint":
		return p.loadCheckpoint(params, ctx)
	case "save_checkpoint":
		return p.saveCheckpoint(params, ctx)
	case "parse_json":
		return p.parseJSON(params, ctx)
	case "if_else":
		return p.ifElse(params, ctx)
	case "set_context":
		return p.setContext(params, ctx)
	case "get_context":
		return p.getContext(params, ctx)
	case "goto":
		return p.gotoAction(params, ctx)
	case "store_cir":
		return p.storeCIR(params, ctx)
	case "store_cir_batch":
		return p.storeCIRBatch(params, ctx)
	case "send_email":
		return p.sendEmail(params, ctx)
	case "send_webhook":
		return p.sendWebhook(params, ctx)
	default:
		return nil, fmt.Errorf("unknown action: %s", action)
	}
}
