package pipeline

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/smtp"
	"os"
	"regexp"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/mimir-aip/mimir-aip-go/pkg/models"
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

// ── HTTP ──────────────────────────────────────────────────────────────────────

// httpRequest makes an HTTP request
func (p *DefaultPlugin) httpRequest(params map[string]interface{}, ctx *models.PipelineContext) (map[string]interface{}, error) {
	url, ok := params["url"].(string)
	if !ok {
		return nil, fmt.Errorf("url parameter is required")
	}

	method, ok := params["method"].(string)
	if !ok {
		method = "GET"
	}

	url = p.ResolveTemplates(url, ctx)

	var body io.Reader
	if bodyData, ok := params["body"]; ok {
		if bodyStr, ok := bodyData.(string); ok {
			bodyStr = p.ResolveTemplates(bodyStr, ctx)
			body = bytes.NewBufferString(bodyStr)
		}
	}

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if headers, ok := params["headers"].(map[string]interface{}); ok {
		for key, value := range headers {
			if strValue, ok := value.(string); ok {
				req.Header.Set(key, p.ResolveTemplates(strValue, ctx))
			}
		}
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	return map[string]interface{}{
		"response": map[string]interface{}{
			"status_code": resp.StatusCode,
			"body":        string(respBody),
			"headers":     resp.Header,
		},
	}, nil
}

// ── JSON ──────────────────────────────────────────────────────────────────────

// parseJSON parses JSON data
func (p *DefaultPlugin) parseJSON(params map[string]interface{}, ctx *models.PipelineContext) (map[string]interface{}, error) {
	data, ok := params["data"].(string)
	if !ok {
		return nil, fmt.Errorf("data parameter is required")
	}

	data = p.ResolveTemplates(data, ctx)

	var parsed interface{}
	if err := json.Unmarshal([]byte(data), &parsed); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	return map[string]interface{}{
		"parsed": parsed,
	}, nil
}

// ── Control flow ──────────────────────────────────────────────────────────────

func (p *DefaultPlugin) ifElse(params map[string]interface{}, ctx *models.PipelineContext) (map[string]interface{}, error) {
	condition, ok := params["condition"]
	if !ok {
		return nil, fmt.Errorf("condition parameter is required")
	}

	conditionStr := p.ResolveTemplates(fmt.Sprintf("%v", condition), ctx)
	isTrue := conditionStr != "" && conditionStr != "0" && conditionStr != "false" && conditionStr != "null"

	var result string
	if isTrue {
		if ifTrue, ok := params["if_true"].(string); ok {
			result = ifTrue
		}
	} else {
		if ifFalse, ok := params["if_false"].(string); ok {
			result = ifFalse
		}
	}

	return map[string]interface{}{
		"result":    result,
		"condition": isTrue,
	}, nil
}

func (p *DefaultPlugin) setContext(params map[string]interface{}, ctx *models.PipelineContext) (map[string]interface{}, error) {
	key, ok := params["key"].(string)
	if !ok {
		return nil, fmt.Errorf("key parameter is required")
	}

	value, ok := params["value"]
	if !ok {
		return nil, fmt.Errorf("value parameter is required")
	}

	stepName, ok := params["step"].(string)
	if !ok {
		stepName = "_global"
	}

	if strValue, ok := value.(string); ok {
		value = p.ResolveTemplates(strValue, ctx)
	}

	ctx.SetStepData(stepName, key, value)
	return map[string]interface{}{"success": true}, nil
}

func (p *DefaultPlugin) getContext(params map[string]interface{}, ctx *models.PipelineContext) (map[string]interface{}, error) {
	key, ok := params["key"].(string)
	if !ok {
		return nil, fmt.Errorf("key parameter is required")
	}

	stepName, ok := params["step"].(string)
	if !ok {
		stepName = "_global"
	}

	value, exists := ctx.GetStepData(stepName, key)
	return map[string]interface{}{
		"exists": exists,
		"value":  value,
	}, nil
}

func (p *DefaultPlugin) gotoAction(params map[string]interface{}, ctx *models.PipelineContext) (map[string]interface{}, error) {
	target, ok := params["target"].(string)
	if !ok {
		return nil, fmt.Errorf("target parameter is required")
	}
	return map[string]interface{}{
		"goto":   target,
		"action": "goto",
	}, nil
}

// ── Storage ───────────────────────────────────────────────────────────────────

// storeCIR stores a single CIR record into Mimir storage.
//
// Parameters:
//   - storage_id  (string, required): ID of the Mimir storage config to write to.
//   - data        (map|string, required): The record data. A JSON string is decoded automatically.
//   - source_uri  (string, optional): Source URI for provenance. Default: "pipeline://ingestion".
//   - source_type (string, optional): "api", "file", "database", or "stream". Default: "api".
//   - format      (string, optional): "json", "csv", etc. Default: "json".
func (p *DefaultPlugin) storeCIR(params map[string]interface{}, ctx *models.PipelineContext) (map[string]interface{}, error) {
	if p.storageSvc == nil {
		return nil, fmt.Errorf("store_cir: storage service is not available in this pipeline context; ensure the orchestrator has been configured to inject the storage service into pipelines")
	}

	storageID, ok := params["storage_id"].(string)
	if !ok || storageID == "" {
		return nil, fmt.Errorf("store_cir: storage_id parameter is required")
	}
	storageID = p.ResolveTemplates(storageID, ctx)

	sourceURI, _ := params["source_uri"].(string)
	sourceURI = p.ResolveTemplates(sourceURI, ctx)
	if sourceURI == "" {
		sourceURI = "pipeline://ingestion"
	}

	sourceTypeStr, _ := params["source_type"].(string)
	if sourceTypeStr == "" {
		sourceTypeStr = "api"
	}

	formatStr, _ := params["format"].(string)
	if formatStr == "" {
		formatStr = "json"
	}

	rawData, hasData := params["data"]
	if !hasData {
		return nil, fmt.Errorf("store_cir: data parameter is required")
	}

	data, err := p.resolveData(rawData, ctx)
	if err != nil {
		return nil, fmt.Errorf("store_cir: %w", err)
	}

	cir := models.NewCIR(
		models.SourceType(sourceTypeStr),
		sourceURI,
		models.DataFormat(formatStr),
		data,
	)

	result, err := p.storageSvc.Store(storageID, cir)
	if err != nil {
		return nil, fmt.Errorf("store_cir: failed to store CIR: %w", err)
	}

	return map[string]interface{}{
		"stored":         true,
		"affected_items": result.AffectedItems,
	}, nil
}

// storeCIRBatch stores an array of records as individual CIR entries.
//
// Parameters:
//   - storage_id  (string, required): ID of the Mimir storage config.
//   - items       ([]interface{}|string, required): Array of records, or a template resolving to one.
//   - source_uri  (string, optional): Source URI for provenance.
//   - source_type (string, optional): Default "api".
//   - format      (string, optional): Default "json".
func (p *DefaultPlugin) storeCIRBatch(params map[string]interface{}, ctx *models.PipelineContext) (map[string]interface{}, error) {
	if p.storageSvc == nil {
		return nil, fmt.Errorf("store_cir_batch: storage service is not available")
	}

	storageID, ok := params["storage_id"].(string)
	if !ok || storageID == "" {
		return nil, fmt.Errorf("store_cir_batch: storage_id parameter is required")
	}
	storageID = p.ResolveTemplates(storageID, ctx)

	sourceURI, _ := params["source_uri"].(string)
	sourceURI = p.ResolveTemplates(sourceURI, ctx)
	if sourceURI == "" {
		sourceURI = "pipeline://ingestion"
	}

	sourceTypeStr, _ := params["source_type"].(string)
	if sourceTypeStr == "" {
		sourceTypeStr = "api"
	}

	formatStr, _ := params["format"].(string)
	if formatStr == "" {
		formatStr = "json"
	}

	// Resolve items array
	rawItems, hasItems := params["items"]
	if !hasItems {
		return nil, fmt.Errorf("store_cir_batch: items parameter is required")
	}

	items, err := p.resolveArray(rawItems, ctx)
	if err != nil {
		return nil, fmt.Errorf("store_cir_batch: %w", err)
	}

	stored := 0
	for i, item := range items {
		cir := models.NewCIR(
			models.SourceType(sourceTypeStr),
			sourceURI,
			models.DataFormat(formatStr),
			item,
		)
		if _, err := p.storageSvc.Store(storageID, cir); err != nil {
			return nil, fmt.Errorf("store_cir_batch: failed to store item %d: %w", i, err)
		}
		stored++
	}

	return map[string]interface{}{
		"stored": stored,
		"total":  len(items),
	}, nil
}

// ── Notifications ─────────────────────────────────────────────────────────────

// sendEmail sends an email via SMTP.
//
// Parameters:
//   - to          (string, required): Recipient address.
//   - subject     (string, optional): Email subject.
//   - body        (string, optional): Plain-text body.
//   - from        (string, optional): Sender address; defaults to username.
//   - smtp_host   (string, optional): SMTP server hostname. Fallback: $SMTP_HOST.
//   - smtp_port   (string, optional): SMTP port. Default: "587".
//   - username    (string, optional): SMTP username. Fallback: $SMTP_USERNAME.
//   - password    (string, optional): SMTP password. Fallback: $SMTP_PASSWORD.
func (p *DefaultPlugin) sendEmail(params map[string]interface{}, ctx *models.PipelineContext) (map[string]interface{}, error) {
	resolve := func(key, fallbackEnv string) string {
		v, _ := params[key].(string)
		v = p.ResolveTemplates(v, ctx)
		if v == "" && fallbackEnv != "" {
			v = os.Getenv(fallbackEnv)
		}
		return v
	}

	to := resolve("to", "")
	if to == "" {
		return nil, fmt.Errorf("send_email: to parameter is required")
	}

	subject := resolve("subject", "")
	body := resolve("body", "")
	smtpHost := resolve("smtp_host", "SMTP_HOST")
	username := resolve("username", "SMTP_USERNAME")
	password := resolve("password", "SMTP_PASSWORD")
	from := resolve("from", "")
	if from == "" {
		from = username
	}

	portStr := resolve("smtp_port", "SMTP_PORT")
	smtpPort := 587
	if portStr != "" {
		if p, err := strconv.Atoi(portStr); err == nil {
			smtpPort = p
		}
	}

	if smtpHost == "" {
		return nil, fmt.Errorf("send_email: smtp_host is required (param or $SMTP_HOST)")
	}

	addr := fmt.Sprintf("%s:%d", smtpHost, smtpPort)
	auth := smtp.PlainAuth("", username, password, smtpHost)
	msg := fmt.Sprintf(
		"From: %s\r\nTo: %s\r\nSubject: %s\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s",
		from, to, subject, body,
	)

	if err := smtp.SendMail(addr, auth, from, []string{to}, []byte(msg)); err != nil {
		return nil, fmt.Errorf("send_email: %w", err)
	}

	return map[string]interface{}{
		"sent": true,
		"to":   to,
	}, nil
}

// sendWebhook posts a JSON payload to an HTTP endpoint.
// This is useful for Slack, Teams, PagerDuty, and any other webhook-based system.
//
// Parameters:
//   - url       (string, required): Webhook endpoint URL.
//   - payload   (map|string, optional): JSON body. Supports templates. Default: {}.
//   - headers   (map, optional): Additional HTTP headers.
func (p *DefaultPlugin) sendWebhook(params map[string]interface{}, ctx *models.PipelineContext) (map[string]interface{}, error) {
	url, ok := params["url"].(string)
	if !ok || url == "" {
		return nil, fmt.Errorf("send_webhook: url parameter is required")
	}
	url = p.ResolveTemplates(url, ctx)

	var payloadBytes []byte
	rawPayload, hasPayload := params["payload"]
	if hasPayload {
		switch v := rawPayload.(type) {
		case string:
			resolved := p.ResolveTemplates(v, ctx)
			payloadBytes = []byte(resolved)
		default:
			var err error
			payloadBytes, err = json.Marshal(v)
			if err != nil {
				return nil, fmt.Errorf("send_webhook: failed to marshal payload: %w", err)
			}
		}
	} else {
		payloadBytes = []byte("{}")
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return nil, fmt.Errorf("send_webhook: failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	if headers, ok := params["headers"].(map[string]interface{}); ok {
		for k, v := range headers {
			if sv, ok := v.(string); ok {
				req.Header.Set(k, p.ResolveTemplates(sv, ctx))
			}
		}
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send_webhook: request failed: %w", err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)

	return map[string]interface{}{
		"sent":        true,
		"status_code": resp.StatusCode,
		"response":    string(respBody),
	}, nil
}

// ── Connector ingestion actions ──────────────────────────────────────────────

type rssFeed struct {
	Channel struct {
		Items []rssItem `xml:"item"`
	} `xml:"channel"`
}

type rssItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	GUID        string `xml:"guid"`
	PubDate     string `xml:"pubDate"`
	Description string `xml:"description"`
}

type connectorCheckpoint struct {
	Seen         []string
	ETag         string
	LastModified string
	LastCursor   interface{}
}

func (p *DefaultPlugin) pollHTTPJSON(params map[string]interface{}, ctx *models.PipelineContext) (map[string]interface{}, error) {
	url, ok := params["url"].(string)
	if !ok || strings.TrimSpace(url) == "" {
		return nil, fmt.Errorf("poll_http_json: url parameter is required")
	}
	url = p.ResolveTemplates(url, ctx)

	method, _ := params["method"].(string)
	if method == "" {
		method = "GET"
	}

	var body io.Reader
	if bodyData, ok := params["body"]; ok {
		if bodyStr, ok := bodyData.(string); ok {
			body = bytes.NewBufferString(p.ResolveTemplates(bodyStr, ctx))
		}
	}

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, fmt.Errorf("poll_http_json: failed to create request: %w", err)
	}

	if headers, ok := params["headers"].(map[string]interface{}); ok {
		for key, value := range headers {
			if strValue, ok := value.(string); ok {
				req.Header.Set(key, p.ResolveTemplates(strValue, ctx))
			}
		}
	}

	checkpoint, err := p.parseConnectorCheckpoint(params["checkpoint"], ctx)
	if err != nil {
		return nil, fmt.Errorf("poll_http_json: invalid checkpoint: %w", err)
	}
	if checkpoint.ETag != "" {
		req.Header.Set("If-None-Match", checkpoint.ETag)
	}
	if checkpoint.LastModified != "" {
		req.Header.Set("If-Modified-Since", checkpoint.LastModified)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("poll_http_json: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotModified {
		checkpoint.ETag = resp.Header.Get("ETag")
		checkpoint.LastModified = resp.Header.Get("Last-Modified")
		return map[string]interface{}{
			"items":       []interface{}{},
			"new_count":   0,
			"total_count": 0,
			"checkpoint":  checkpoint.toMap(),
		}, nil
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("poll_http_json: unexpected status %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("poll_http_json: failed to read response: %w", err)
	}

	var payload interface{}
	if err := json.Unmarshal(bodyBytes, &payload); err != nil {
		return nil, fmt.Errorf("poll_http_json: invalid JSON response: %w", err)
	}

	itemsPath, _ := params["items_path"].(string)
	items, err := extractItemsAtPath(payload, itemsPath)
	if err != nil {
		return nil, fmt.Errorf("poll_http_json: %w", err)
	}

	maxSeen := parseMaxCheckpointItems(params["max_checkpoint_items"])
	newItems, nextCheckpoint := dedupeByCheckpoint(items, checkpoint, maxSeen)
	nextCheckpoint.ETag = resp.Header.Get("ETag")
	nextCheckpoint.LastModified = resp.Header.Get("Last-Modified")

	return map[string]interface{}{
		"items":       newItems,
		"new_count":   len(newItems),
		"total_count": len(items),
		"checkpoint":  nextCheckpoint.toMap(),
	}, nil
}

func (p *DefaultPlugin) pollRSS(params map[string]interface{}, ctx *models.PipelineContext) (map[string]interface{}, error) {
	url, ok := params["url"].(string)
	if !ok || strings.TrimSpace(url) == "" {
		return nil, fmt.Errorf("poll_rss: url parameter is required")
	}
	url = p.ResolveTemplates(url, ctx)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("poll_rss: failed to create request: %w", err)
	}

	checkpoint, err := p.parseConnectorCheckpoint(params["checkpoint"], ctx)
	if err != nil {
		return nil, fmt.Errorf("poll_rss: invalid checkpoint: %w", err)
	}
	if checkpoint.ETag != "" {
		req.Header.Set("If-None-Match", checkpoint.ETag)
	}
	if checkpoint.LastModified != "" {
		req.Header.Set("If-Modified-Since", checkpoint.LastModified)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("poll_rss: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotModified {
		checkpoint.ETag = resp.Header.Get("ETag")
		checkpoint.LastModified = resp.Header.Get("Last-Modified")
		return map[string]interface{}{
			"items":       []interface{}{},
			"new_count":   0,
			"total_count": 0,
			"checkpoint":  checkpoint.toMap(),
		}, nil
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("poll_rss: unexpected status %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("poll_rss: failed to read response: %w", err)
	}

	var feed rssFeed
	if err := xml.Unmarshal(bodyBytes, &feed); err != nil {
		return nil, fmt.Errorf("poll_rss: invalid RSS XML: %w", err)
	}

	items := make([]interface{}, 0, len(feed.Channel.Items))
	for _, item := range feed.Channel.Items {
		items = append(items, map[string]interface{}{
			"title":        strings.TrimSpace(item.Title),
			"link":         strings.TrimSpace(item.Link),
			"guid":         strings.TrimSpace(item.GUID),
			"published_at": strings.TrimSpace(item.PubDate),
			"description":  strings.TrimSpace(item.Description),
		})
	}

	maxSeen := parseMaxCheckpointItems(params["max_checkpoint_items"])
	newItems, nextCheckpoint := dedupeByCheckpoint(items, checkpoint, maxSeen)
	nextCheckpoint.ETag = resp.Header.Get("ETag")
	nextCheckpoint.LastModified = resp.Header.Get("Last-Modified")

	return map[string]interface{}{
		"items":       newItems,
		"new_count":   len(newItems),
		"total_count": len(items),
		"checkpoint":  nextCheckpoint.toMap(),
	}, nil
}

func (p *DefaultPlugin) ingestCSV(params map[string]interface{}, ctx *models.PipelineContext) (map[string]interface{}, error) {
	rawCSV, ok := params["csv_data"].(string)
	if !ok || strings.TrimSpace(rawCSV) == "" {
		return nil, fmt.Errorf("ingest_csv: csv_data parameter is required")
	}
	csvData := p.ResolveTemplates(rawCSV, ctx)

	checkpoint, err := p.parseConnectorCheckpoint(params["checkpoint"], ctx)
	if err != nil {
		return nil, fmt.Errorf("ingest_csv: invalid checkpoint: %w", err)
	}

	result, err := p.ingestCSVWithCheckpoint(csvData, params, checkpoint)
	if err != nil {
		return nil, fmt.Errorf("ingest_csv: %w", err)
	}
	return result, nil
}

func parseMaxCheckpointItems(raw interface{}) int {
	const defaultMax = 200
	switch v := raw.(type) {
	case float64:
		if v > 0 {
			return int(v)
		}
	case int:
		if v > 0 {
			return v
		}
	}
	return defaultMax
}

func dedupeByCheckpoint(items []interface{}, checkpoint connectorCheckpoint, maxSeen int) ([]interface{}, connectorCheckpoint) {
	if maxSeen <= 0 {
		maxSeen = 200
	}

	seenSet := make(map[string]struct{}, len(checkpoint.Seen))
	orderedSeen := make([]string, 0, len(checkpoint.Seen))
	for _, hash := range checkpoint.Seen {
		if hash == "" {
			continue
		}
		if _, exists := seenSet[hash]; exists {
			continue
		}
		seenSet[hash] = struct{}{}
		orderedSeen = append(orderedSeen, hash)
	}

	newItems := make([]interface{}, 0, len(items))
	for _, item := range items {
		hash := hashPayload(item)
		if _, seen := seenSet[hash]; seen {
			continue
		}
		seenSet[hash] = struct{}{}
		orderedSeen = append(orderedSeen, hash)
		newItems = append(newItems, item)
	}

	if len(orderedSeen) > maxSeen {
		orderedSeen = orderedSeen[len(orderedSeen)-maxSeen:]
	}

	checkpoint.Seen = orderedSeen
	return newItems, checkpoint
}

func extractItemsAtPath(payload interface{}, path string) ([]interface{}, error) {
	if strings.TrimSpace(path) == "" {
		switch typed := payload.(type) {
		case []interface{}:
			return typed, nil
		case map[string]interface{}:
			if rawItems, ok := typed["items"]; ok {
				if arr, ok := rawItems.([]interface{}); ok {
					return arr, nil
				}
			}
			return []interface{}{typed}, nil
		default:
			return nil, fmt.Errorf("JSON payload must be an object or array")
		}
	}

	current := payload
	for _, part := range strings.Split(path, ".") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		obj, ok := current.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("items_path %q does not resolve to an object", path)
		}
		next, exists := obj[part]
		if !exists {
			return nil, fmt.Errorf("items_path %q not found", path)
		}
		current = next
	}

	arr, ok := current.([]interface{})
	if !ok {
		return nil, fmt.Errorf("items_path %q does not resolve to an array", path)
	}
	return arr, nil
}

func hashPayload(value interface{}) string {
	bytes, err := json.Marshal(value)
	if err != nil {
		bytes = []byte(fmt.Sprintf("%v", value))
	}
	sum := sha256.Sum256(bytes)
	return hex.EncodeToString(sum[:])
}

func (p *DefaultPlugin) parseConnectorCheckpoint(raw interface{}, ctx *models.PipelineContext) (connectorCheckpoint, error) {
	checkpoint := connectorCheckpoint{Seen: []string{}}
	if raw == nil {
		return checkpoint, nil
	}

	var asMap map[string]interface{}
	switch typed := raw.(type) {
	case map[string]interface{}:
		asMap = typed
	case string:
		resolved := p.ResolveTemplates(typed, ctx)
		if strings.TrimSpace(resolved) == "" {
			return checkpoint, nil
		}
		if err := json.Unmarshal([]byte(resolved), &asMap); err != nil {
			return checkpoint, fmt.Errorf("checkpoint must be object or JSON object string: %w", err)
		}
	default:
		return checkpoint, fmt.Errorf("checkpoint must be object or JSON object string")
	}

	if etag, ok := asMap["etag"].(string); ok {
		checkpoint.ETag = etag
	}
	if lm, ok := asMap["last_modified"].(string); ok {
		checkpoint.LastModified = lm
	}
	if lastCursor, exists := asMap["last_cursor"]; exists {
		checkpoint.LastCursor = lastCursor
	}

	rawSeen, hasSeen := asMap["seen_hashes"]
	if !hasSeen {
		rawSeen = asMap["seen"]
	}
	checkpoint.Seen = decodeStringSlice(rawSeen)
	return checkpoint, nil
}

func decodeStringSlice(raw interface{}) []string {
	if raw == nil {
		return []string{}
	}
	result := make([]string, 0)
	switch typed := raw.(type) {
	case []string:
		for _, s := range typed {
			s = strings.TrimSpace(s)
			if s != "" {
				result = append(result, s)
			}
		}
	case []interface{}:
		for _, v := range typed {
			s := strings.TrimSpace(fmt.Sprintf("%v", v))
			if s != "" {
				result = append(result, s)
			}
		}
	}
	return result
}

func (c connectorCheckpoint) toMap() map[string]interface{} {
	result := map[string]interface{}{
		"seen_hashes":    c.Seen,
		"etag":           c.ETag,
		"last_modified":  c.LastModified,
		"last_polled_at": time.Now().UTC().Format(time.RFC3339),
	}
	if c.LastCursor != nil {
		result["last_cursor"] = c.LastCursor
	}
	return result
}

// ── Template resolution ───────────────────────────────────────────────────────

// ResolveTemplates resolves {{context.step_name.key}} template variables in a string.
func (p *DefaultPlugin) ResolveTemplates(input string, ctx *models.PipelineContext) string {
	exactPattern := regexp.MustCompile(`^\s*\{\{([^}]+)\}\}\s*$`)
	if matches := exactPattern.FindStringSubmatch(input); len(matches) == 2 {
		expr := strings.TrimSpace(matches[1])
		if value, ok := p.resolveTemplateValue(expr, ctx); ok {
			switch typed := value.(type) {
			case string:
				return typed
			default:
				if bytes, err := json.Marshal(typed); err == nil {
					return string(bytes)
				}
				return fmt.Sprintf("%v", typed)
			}
		}
	}

	pattern := regexp.MustCompile(`\{\{([^}]+)\}\}`)
	return pattern.ReplaceAllStringFunc(input, func(match string) string {
		expr := strings.TrimSpace(match[2 : len(match)-2])
		if value, ok := p.resolveTemplateValue(expr, ctx); ok {
			return fmt.Sprintf("%v", value)
		}
		return match
	})
}

func (p *DefaultPlugin) resolveTemplateValue(expr string, ctx *models.PipelineContext) (interface{}, bool) {
	parts := strings.Split(expr, ".")
	if len(parts) < 2 {
		return nil, false
	}
	if parts[0] != "context" {
		return p.evaluateTemplate(expr, ctx), true
	}
	if len(parts) < 3 {
		return nil, false
	}

	stepName := parts[1]
	key := parts[2]
	value, exists := ctx.GetStepData(stepName, key)
	if !exists {
		return nil, false
	}

	if len(parts) > 3 {
		for i := 3; i < len(parts); i++ {
			if m, ok := value.(map[string]interface{}); ok {
				value, exists = m[parts[i]]
				if !exists {
					return nil, false
				}
				continue
			}
			return nil, false
		}
	}

	return value, true
}

func (p *DefaultPlugin) evaluateTemplate(expr string, ctx *models.PipelineContext) string {
	tmpl, err := template.New("expr").Parse("{{" + expr + "}}")
	if err != nil {
		return ""
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, ctx.Steps); err != nil {
		return ""
	}
	return buf.String()
}

// ── Helpers ───────────────────────────────────────────────────────────────────

// resolveData coerces a raw parameter value into an interface{} suitable for a CIR's Data field.
// JSON strings are decoded; everything else is passed through.
func (p *DefaultPlugin) resolveData(raw interface{}, ctx *models.PipelineContext) (interface{}, error) {
	switch v := raw.(type) {
	case string:
		resolved := p.ResolveTemplates(v, ctx)
		var out interface{}
		if err := json.Unmarshal([]byte(resolved), &out); err == nil {
			return out, nil
		}
		return resolved, nil
	default:
		return v, nil
	}
}

// resolveArray coerces a raw parameter value into []interface{}.
func (p *DefaultPlugin) resolveArray(raw interface{}, ctx *models.PipelineContext) ([]interface{}, error) {
	switch v := raw.(type) {
	case []interface{}:
		return v, nil
	case string:
		resolved := p.ResolveTemplates(v, ctx)
		var items []interface{}
		if err := json.Unmarshal([]byte(resolved), &items); err != nil {
			return nil, fmt.Errorf("items must resolve to a JSON array: %w", err)
		}
		return items, nil
	default:
		// Marshal → unmarshal to normalise (handles []map[string]interface{} etc.)
		b, err := json.Marshal(v)
		if err != nil {
			return nil, fmt.Errorf("items parameter must be an array")
		}
		var items []interface{}
		if err := json.Unmarshal(b, &items); err != nil {
			return nil, fmt.Errorf("items parameter must be an array")
		}
		return items, nil
	}
}
