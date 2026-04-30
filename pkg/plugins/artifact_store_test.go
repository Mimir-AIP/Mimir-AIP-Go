package plugins

import (
	"os"
	"strings"
	"testing"

	"github.com/mimir-aip/mimir-aip-go/pkg/models"
)

func TestArtifactStorePutStoresByDigest(t *testing.T) {
	store, err := NewArtifactStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewArtifactStore() error = %v", err)
	}
	source := t.TempDir() + "/plugin.so"
	if err := os.WriteFile(source, []byte("compiled plugin"), 0644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	path, digest, size, err := store.Put(models.PluginKindPipeline, "bad/name", source)
	if err != nil {
		t.Fatalf("Put() error = %v", err)
	}
	if !strings.HasPrefix(digest, "sha256:") {
		t.Fatalf("digest = %q, want sha256 prefix", digest)
	}
	if size != int64(len("compiled plugin")) {
		t.Fatalf("size = %d, want %d", size, len("compiled plugin"))
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read artifact: %v", err)
	}
	if string(data) != "compiled plugin" {
		t.Fatalf("artifact content = %q", string(data))
	}
	if strings.Contains(path, "bad/name") {
		t.Fatalf("artifact path was not sanitized: %s", path)
	}
}
