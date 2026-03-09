package pipeline

import (
	"database/sql"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync"
	"testing"

	_ "modernc.org/sqlite"

	"github.com/mimir-aip/mimir-aip-go/pkg/metadatastore"
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

func TestIngestCSVURL_CheckpointPreventsReplay(t *testing.T) {
	ctx := models.NewPipelineContext(DefaultContextMaxSize)
	plugin := NewDefaultPlugin()

	var mu sync.RWMutex
	csvBody := "id,name\n1,A\n2,B\n"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.RLock()
		defer mu.RUnlock()
		w.Header().Set("Content-Type", "text/csv")
		_, _ = w.Write([]byte(csvBody))
	}))
	defer server.Close()

	first, err := plugin.Execute("ingest_csv_url", map[string]interface{}{"url": server.URL}, ctx)
	if err != nil {
		t.Fatalf("first csv url ingest failed: %v", err)
	}
	if asInt(t, first["new_count"]) != 2 {
		t.Fatalf("expected first csv url ingest new_count=2, got %v", first["new_count"])
	}

	second, err := plugin.Execute("ingest_csv_url", map[string]interface{}{
		"url":        server.URL,
		"checkpoint": first["checkpoint"],
	}, ctx)
	if err != nil {
		t.Fatalf("second csv url ingest failed: %v", err)
	}
	if asInt(t, second["new_count"]) != 0 {
		t.Fatalf("expected second csv url ingest new_count=0, got %v", second["new_count"])
	}

	mu.Lock()
	csvBody = "id,name\n1,A\n2,B\n3,C\n"
	mu.Unlock()

	third, err := plugin.Execute("ingest_csv_url", map[string]interface{}{
		"url":        server.URL,
		"checkpoint": second["checkpoint"],
	}, ctx)
	if err != nil {
		t.Fatalf("third csv url ingest failed: %v", err)
	}
	if asInt(t, third["new_count"]) != 1 {
		t.Fatalf("expected third csv url ingest new_count=1, got %v", third["new_count"])
	}
}

func TestLoadAndSaveCheckpoint_PersistsAcrossCalls(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := metadatastore.NewSQLiteStore(filepath.Join(tmpDir, "checkpoints.db"))
	if err != nil {
		t.Fatalf("failed to create sqlite store: %v", err)
	}
	defer store.Close()

	plugin := NewDefaultPluginWithDeps(nil, store)
	ctx := models.NewPipelineContext(DefaultContextMaxSize)
	ctx.SetStepData("_runtime", "project_id", "project-1")
	ctx.SetStepData("_runtime", "pipeline_id", "pipe-1")
	ctx.SetStepData("_runtime", "current_step", "source")

	saved, err := plugin.Execute("save_checkpoint", map[string]interface{}{
		"checkpoint": map[string]interface{}{"last_cursor": "cursor-1"},
	}, ctx)
	if err != nil {
		t.Fatalf("save_checkpoint failed: %v", err)
	}
	if asInt(t, saved["version"]) != 1 {
		t.Fatalf("expected saved version=1, got %v", saved["version"])
	}

	loaded, err := plugin.Execute("load_checkpoint", map[string]interface{}{}, ctx)
	if err != nil {
		t.Fatalf("load_checkpoint failed: %v", err)
	}
	if exists, _ := loaded["exists"].(bool); !exists {
		t.Fatalf("expected checkpoint to exist")
	}
	checkpointMap, ok := loaded["checkpoint"].(map[string]interface{})
	if !ok || fmt.Sprintf("%v", checkpointMap["last_cursor"]) != "cursor-1" {
		t.Fatalf("unexpected checkpoint payload: %#v", loaded["checkpoint"])
	}

	ctx.SetStepData("loaded", "checkpoint", loaded["checkpoint"])
	ctx.SetStepData("loaded", "version", loaded["version"])
	updated, err := plugin.Execute("save_checkpoint", map[string]interface{}{
		"checkpoint": "{{context.loaded.checkpoint}}",
		"version":    "{{context.loaded.version}}",
	}, ctx)
	if err != nil {
		t.Fatalf("second save_checkpoint failed: %v", err)
	}
	if asInt(t, updated["version"]) != 2 {
		t.Fatalf("expected saved version=2, got %v", updated["version"])
	}
}

func TestResolveArray_FromStructuredTemplate(t *testing.T) {
	ctx := models.NewPipelineContext(DefaultContextMaxSize)
	ctx.SetStepData("fetch", "items", []interface{}{map[string]interface{}{"id": 1}})

	plugin := NewDefaultPlugin()
	items, err := plugin.resolveArray("{{context.fetch.items}}", ctx)
	if err != nil {
		t.Fatalf("resolveArray failed: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected one item, got %d", len(items))
	}
}

func TestQuerySQL_ReturnsRowsAndCursor(t *testing.T) {
	tmpDir := t.TempDir()
	dsn := "file:" + filepath.Join(tmpDir, "query.db")
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatalf("failed to open sqlite db: %v", err)
	}
	defer db.Close()

	if _, err := db.Exec(`CREATE TABLE repairs (id INTEGER PRIMARY KEY, updated_at TEXT NOT NULL, amount INTEGER NOT NULL)`); err != nil {
		t.Fatalf("failed to create table: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO repairs (updated_at, amount) VALUES ('2024-01-01T00:00:00Z', 100), ('2024-01-02T00:00:00Z', 150)`); err != nil {
		t.Fatalf("failed to insert rows: %v", err)
	}

	plugin := NewDefaultPlugin()
	result, err := plugin.Execute("query_sql", map[string]interface{}{
		"driver":        "sqlite",
		"dsn":           dsn,
		"query":         "SELECT id, updated_at, amount FROM repairs ORDER BY updated_at",
		"cursor_column": "updated_at",
	}, models.NewPipelineContext(DefaultContextMaxSize))
	if err != nil {
		t.Fatalf("query_sql failed: %v", err)
	}
	if asInt(t, result["row_count"]) != 2 {
		t.Fatalf("expected row_count=2, got %v", result["row_count"])
	}
	if fmt.Sprintf("%v", result["next_cursor"]) != "2024-01-02T00:00:00Z" {
		t.Fatalf("unexpected next_cursor: %v", result["next_cursor"])
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
