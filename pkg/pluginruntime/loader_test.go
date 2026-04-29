package pluginruntime

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func newTestLoader(t *testing.T) *Loader[any] {
	t.Helper()
	loader, err := NewLoader(BuildSpec[any]{
		LogPrefix:      "test loader",
		AppDir:         t.TempDir(),
		CacheDir:       t.TempDir(),
		TempDir:        t.TempDir(),
		HostPackageDir: "plugins",
		SymbolName:     "Plugin",
		Resolver: func(symbol any) (any, bool) {
			return symbol, true
		},
	})
	if err != nil {
		t.Fatalf("NewLoader() error = %v", err)
	}
	return loader
}

func TestCacheMetadataRequiresMatchingRuntimeFingerprint(t *testing.T) {
	loader := newTestLoader(t)
	artifact := loader.artifactPath("plugin/name", "abc123")
	if err := os.WriteFile(artifact, []byte("compiled"), 0644); err != nil {
		t.Fatalf("write artifact: %v", err)
	}
	if err := loader.writeMeta("plugin/name", "abc123", artifact); err != nil {
		t.Fatalf("writeMeta() error = %v", err)
	}

	if !loader.isCached("plugin/name", "abc123") {
		t.Fatal("expected matching cache metadata to be usable")
	}
	if loader.isCached("plugin/name", "def456") {
		t.Fatal("expected different commit hash to invalidate cache")
	}

	meta, err := loader.readMeta("plugin/name")
	if err != nil {
		t.Fatalf("readMeta() error = %v", err)
	}
	meta.GoVersion = "go0.invalid"
	data := []byte(`{"commit_hash":"abc123","artifact_path":"` + filepath.ToSlash(artifact) + `","go_version":"` + meta.GoVersion + `","goos":"` + runtime.GOOS + `","goarch":"` + runtime.GOARCH + `","host_package_dir":"plugins","symbol_name":"Plugin"}`)
	if err := os.WriteFile(loader.MetaPath("plugin/name"), data, 0644); err != nil {
		t.Fatalf("overwrite meta: %v", err)
	}
	if loader.isCached("plugin/name", "abc123") {
		t.Fatal("expected Go version mismatch to invalidate cache")
	}
}

func TestSoPathFallsBackForLegacyMetadata(t *testing.T) {
	loader := newTestLoader(t)
	legacyPath := filepath.Join(loader.spec.CacheDir, "legacy.so")
	if err := os.WriteFile(legacyPath, []byte("compiled"), 0644); err != nil {
		t.Fatalf("write legacy artifact: %v", err)
	}
	if err := os.WriteFile(loader.MetaPath("legacy"), []byte("abc123"), 0644); err != nil {
		t.Fatalf("write legacy metadata: %v", err)
	}

	if got := loader.SoPath("legacy"); got != legacyPath {
		t.Fatalf("SoPath() = %q, want %q", got, legacyPath)
	}
}
