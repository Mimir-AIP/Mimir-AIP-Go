package pipeline

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/mimir-aip/mimir-aip-go/pkg/models"
)

func TestPollHTTPJSON_CheckpointPreventsReplay(t *testing.T) {
	ctx := models.NewPipelineContext(DefaultContextMaxSize)
	plugin := NewDefaultPlugin()

	var mu sync.RWMutex
	payload := `{"items":[{"id":"1","name":"A"},{"id":"2","name":"B"}]}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.RLock()
		defer mu.RUnlock()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(payload))
	}))
	defer server.Close()

	params := map[string]interface{}{
		"url":        server.URL,
		"items_path": "items",
	}

	first, err := plugin.Execute("poll_http_json", params, ctx)
	if err != nil {
		t.Fatalf("first poll failed: %v", err)
	}
	if asInt(t, first["new_count"]) != 2 {
		t.Fatalf("expected first poll new_count=2, got %v", first["new_count"])
	}

	checkpoint, ok := first["checkpoint"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected checkpoint object, got %T", first["checkpoint"])
	}

	second, err := plugin.Execute("poll_http_json", map[string]interface{}{
		"url":        server.URL,
		"items_path": "items",
		"checkpoint": checkpoint,
	}, ctx)
	if err != nil {
		t.Fatalf("second poll failed: %v", err)
	}
	if asInt(t, second["new_count"]) != 0 {
		t.Fatalf("expected second poll new_count=0, got %v", second["new_count"])
	}

	mu.Lock()
	payload = `{"items":[{"id":"1","name":"A"},{"id":"2","name":"B"},{"id":"3","name":"C"}]}`
	mu.Unlock()

	third, err := plugin.Execute("poll_http_json", map[string]interface{}{
		"url":        server.URL,
		"items_path": "items",
		"checkpoint": second["checkpoint"],
	}, ctx)
	if err != nil {
		t.Fatalf("third poll failed: %v", err)
	}
	if asInt(t, third["new_count"]) != 1 {
		t.Fatalf("expected third poll new_count=1, got %v", third["new_count"])
	}

	items, ok := third["items"].([]interface{})
	if !ok || len(items) != 1 {
		t.Fatalf("expected one new item, got %#v", third["items"])
	}
	itemMap, ok := items[0].(map[string]interface{})
	if !ok || fmt.Sprintf("%v", itemMap["id"]) != "3" {
		t.Fatalf("expected new item id=3, got %#v", items[0])
	}
}

func TestPollRSS_CheckpointPreventsReplay(t *testing.T) {
	ctx := models.NewPipelineContext(DefaultContextMaxSize)
	plugin := NewDefaultPlugin()

	var mu sync.RWMutex
	rssBody := `<?xml version="1.0"?><rss version="2.0"><channel><item><title>A</title><guid>1</guid><link>https://x/a</link><pubDate>Mon, 01 Jan 2024 00:00:00 GMT</pubDate></item><item><title>B</title><guid>2</guid><link>https://x/b</link><pubDate>Mon, 02 Jan 2024 00:00:00 GMT</pubDate></item></channel></rss>`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.RLock()
		defer mu.RUnlock()
		w.Header().Set("Content-Type", "application/rss+xml")
		_, _ = w.Write([]byte(rssBody))
	}))
	defer server.Close()

	first, err := plugin.Execute("poll_rss", map[string]interface{}{"url": server.URL}, ctx)
	if err != nil {
		t.Fatalf("first rss poll failed: %v", err)
	}
	if asInt(t, first["new_count"]) != 2 {
		t.Fatalf("expected first rss poll new_count=2, got %v", first["new_count"])
	}

	second, err := plugin.Execute("poll_rss", map[string]interface{}{
		"url":        server.URL,
		"checkpoint": first["checkpoint"],
	}, ctx)
	if err != nil {
		t.Fatalf("second rss poll failed: %v", err)
	}
	if asInt(t, second["new_count"]) != 0 {
		t.Fatalf("expected second rss poll new_count=0, got %v", second["new_count"])
	}

	mu.Lock()
	rssBody = `<?xml version="1.0"?><rss version="2.0"><channel><item><title>A</title><guid>1</guid><link>https://x/a</link><pubDate>Mon, 01 Jan 2024 00:00:00 GMT</pubDate></item><item><title>B</title><guid>2</guid><link>https://x/b</link><pubDate>Mon, 02 Jan 2024 00:00:00 GMT</pubDate></item><item><title>C</title><guid>3</guid><link>https://x/c</link><pubDate>Mon, 03 Jan 2024 00:00:00 GMT</pubDate></item></channel></rss>`
	mu.Unlock()

	third, err := plugin.Execute("poll_rss", map[string]interface{}{
		"url":        server.URL,
		"checkpoint": second["checkpoint"],
	}, ctx)
	if err != nil {
		t.Fatalf("third rss poll failed: %v", err)
	}
	if asInt(t, third["new_count"]) != 1 {
		t.Fatalf("expected third rss poll new_count=1, got %v", third["new_count"])
	}
}

func TestIngestCSV_CheckpointPreventsReplay(t *testing.T) {
	ctx := models.NewPipelineContext(DefaultContextMaxSize)
	plugin := NewDefaultPlugin()

	first, err := plugin.Execute("ingest_csv", map[string]interface{}{
		"csv_data": "id,name\n1,A\n2,B\n",
	}, ctx)
	if err != nil {
		t.Fatalf("first csv ingest failed: %v", err)
	}
	if asInt(t, first["new_count"]) != 2 {
		t.Fatalf("expected first csv ingest new_count=2, got %v", first["new_count"])
	}

	second, err := plugin.Execute("ingest_csv", map[string]interface{}{
		"csv_data":   "id,name\n1,A\n2,B\n",
		"checkpoint": first["checkpoint"],
	}, ctx)
	if err != nil {
		t.Fatalf("second csv ingest failed: %v", err)
	}
	if asInt(t, second["new_count"]) != 0 {
		t.Fatalf("expected second csv ingest new_count=0, got %v", second["new_count"])
	}

	third, err := plugin.Execute("ingest_csv", map[string]interface{}{
		"csv_data":   "id,name\n1,A\n2,B\n3,C\n",
		"checkpoint": second["checkpoint"],
	}, ctx)
	if err != nil {
		t.Fatalf("third csv ingest failed: %v", err)
	}
	if asInt(t, third["new_count"]) != 1 {
		t.Fatalf("expected third csv ingest new_count=1, got %v", third["new_count"])
	}
}

func asInt(t *testing.T, v interface{}) int {
	t.Helper()
	switch n := v.(type) {
	case int:
		return n
	case float64:
		return int(n)
	default:
		t.Fatalf("expected numeric value, got %T (%v)", v, v)
		return 0
	}
}
