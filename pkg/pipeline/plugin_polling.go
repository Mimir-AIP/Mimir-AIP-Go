package pipeline

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"github.com/mimir-aip/mimir-aip-go/pkg/models"
	"io"
	"net/http"
	"strings"
	"time"
)

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
