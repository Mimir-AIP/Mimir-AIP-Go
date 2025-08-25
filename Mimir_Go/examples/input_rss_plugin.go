// RSS Feed Input Plugin Example
// This example shows how to create an Input plugin for Mimir AIP

package main

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/Mimir-AIP/Mimir-AIP-Go/pipelines"
)

// RSSItem represents an RSS feed item
type RSSItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	PubDate     string `xml:"pubDate"`
	GUID        string `xml:"guid"`
}

// RSSFeed represents an RSS feed
type RSSFeed struct {
	Title       string    `xml:"channel>title"`
	Description string    `xml:"channel>description"`
	Items       []RSSItem `xml:"channel>item"`
}

// RSSPlugin implements an RSS feed input plugin
type RSSPlugin struct {
	name    string
	version string
	client  *http.Client
}

// NewRSSPlugin creates a new RSS plugin instance
func NewRSSPlugin() *RSSPlugin {
	return &RSSPlugin{
		name:    "RSSPlugin",
		version: "1.0.0",
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// ExecuteStep fetches RSS feed data
func (p *RSSPlugin) ExecuteStep(ctx context.Context, stepConfig pipelines.StepConfig, globalContext pipelines.PluginContext) (pipelines.PluginContext, error) {
	fmt.Printf("Executing %s step: %s\n", p.name, stepConfig.Name)

	config := stepConfig.Config

	// Extract configuration
	url, ok := config["url"].(string)
	if !ok || url == "" {
		return nil, fmt.Errorf("url is required in config")
	}

	limit := 10 // default
	if l, ok := config["limit"].(float64); ok {
		limit = int(l)
	}

	// Validate configuration
	if err := p.ValidateConfig(config); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	// Fetch RSS feed
	feed, err := p.fetchRSSFeed(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch RSS feed: %w", err)
	}

	// Process items
	items := feed.Items
	if len(items) > limit {
		items = items[:limit]
	}

	// Format result
	result := map[string]interface{}{
		"feed_title":       feed.Title,
		"feed_description": feed.Description,
		"item_count":       len(items),
		"items":            p.formatItems(items),
		"fetched_at":       time.Now().Format(time.RFC3339),
		"source_url":       url,
	}

	return pipelines.PluginContext{
		stepConfig.Output: result,
	}, nil
}

// GetPluginType returns the plugin type
func (p *RSSPlugin) GetPluginType() string {
	return "Input"
}

// GetPluginName returns the plugin name
func (p *RSSPlugin) GetPluginName() string {
	return "rss"
}

// ValidateConfig validates the plugin configuration
func (p *RSSPlugin) ValidateConfig(config map[string]interface{}) error {
	if config["url"] == nil {
		return fmt.Errorf("url is required")
	}

	if url, ok := config["url"].(string); !ok || url == "" {
		return fmt.Errorf("url must be a non-empty string")
	}

	return nil
}

// fetchRSSFeed fetches and parses an RSS feed
func (p *RSSPlugin) fetchRSSFeed(ctx context.Context, url string) (*RSSFeed, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "Mimir-AIP-RSS-Plugin/1.0")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch feed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP error: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var feed RSSFeed
	if err := xml.Unmarshal(body, &feed); err != nil {
		return nil, fmt.Errorf("failed to parse RSS feed: %w", err)
	}

	return &feed, nil
}

// formatItems formats RSS items for output
func (p *RSSPlugin) formatItems(items []RSSItem) []map[string]interface{} {
	formatted := make([]map[string]interface{}, len(items))

	for i, item := range items {
		formatted[i] = map[string]interface{}{
			"title":       item.Title,
			"link":        item.Link,
			"description": item.Description,
			"published":   item.PubDate,
			"guid":        item.GUID,
		}
	}

	return formatted
}

// Example usage in pipeline YAML:
//
// pipelines:
//   - name: "RSS Feed Pipeline"
//     steps:
//       - name: "Fetch RSS Feed"
//         plugin: "Input.rss"
//         config:
//           url: "https://example.com/feed.xml"
//           limit: 5
//         output: "rss_data"
//       - name: "Process Feed"
//         plugin: "Data_Processing.transform"
//         config:
//           operation: "extract_titles"
//         output: "processed_feed"
