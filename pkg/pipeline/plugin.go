package pipeline

import (
	"bytes"
	"encoding/json"
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

// Plugin defines the interface for pipeline step executors
type Plugin interface {
	Execute(action string, params map[string]interface{}, ctx *models.PipelineContext) (map[string]interface{}, error)
}

// DefaultPlugin implements built-in pipeline actions
type DefaultPlugin struct {
	httpClient *http.Client
	storageSvc CIRStorer // nil when storage integration is not configured
}

// NewDefaultPlugin creates a new default plugin instance without storage integration.
func NewDefaultPlugin() *DefaultPlugin {
	return &DefaultPlugin{
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// NewDefaultPluginWithStorage creates a default plugin that can persist CIR data
// via the provided CIRStorer (typically *storage.Service).
func NewDefaultPluginWithStorage(svc CIRStorer) *DefaultPlugin {
	return &DefaultPlugin{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		storageSvc: svc,
	}
}

// Execute executes a default plugin action
func (p *DefaultPlugin) Execute(action string, params map[string]interface{}, ctx *models.PipelineContext) (map[string]interface{}, error) {
	switch action {
	case "http_request":
		return p.httpRequest(params, ctx)
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

// ── Template resolution ───────────────────────────────────────────────────────

// ResolveTemplates resolves {{context.step_name.key}} template variables in a string.
func (p *DefaultPlugin) ResolveTemplates(input string, ctx *models.PipelineContext) string {
	pattern := regexp.MustCompile(`\{\{([^}]+)\}\}`)

	result := pattern.ReplaceAllStringFunc(input, func(match string) string {
		expr := strings.TrimSpace(match[2 : len(match)-2])
		parts := strings.Split(expr, ".")

		if len(parts) < 2 {
			return match
		}

		if parts[0] != "context" {
			return p.evaluateTemplate(expr, ctx)
		}

		if len(parts) < 3 {
			return match
		}

		stepName := parts[1]
		key := parts[2]

		value, exists := ctx.GetStepData(stepName, key)
		if !exists {
			return match
		}

		// Handle nested access (e.g., {{context.step.key.nested}})
		if len(parts) > 3 {
			for i := 3; i < len(parts); i++ {
				if m, ok := value.(map[string]interface{}); ok {
					value, exists = m[parts[i]]
					if !exists {
						return match
					}
				} else {
					return match
				}
			}
		}

		return fmt.Sprintf("%v", value)
	})

	return result
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
