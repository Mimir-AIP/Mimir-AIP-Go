package plugins

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/mimir-aip/mimir-aip-go/pkg/models"
)

type ArtifactStore struct {
	rootDir string
}

func NewArtifactStore(rootDir string) (*ArtifactStore, error) {
	if strings.TrimSpace(rootDir) == "" {
		return nil, fmt.Errorf("artifact store root is required")
	}
	if err := os.MkdirAll(rootDir, 0755); err != nil {
		return nil, fmt.Errorf("create artifact store: %w", err)
	}
	return &ArtifactStore{rootDir: rootDir}, nil
}

func (s *ArtifactStore) Put(kind models.PluginKind, name string, sourcePath string) (string, string, int64, error) {
	digest, size, err := hashFile(sourcePath)
	if err != nil {
		return "", "", 0, err
	}
	digestName := strings.TrimPrefix(digest, "sha256:")
	dir := filepath.Join(s.rootDir, safePathSegment(string(kind)), safePathSegment(name), digestName)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", "", 0, fmt.Errorf("create artifact directory: %w", err)
	}
	dest := filepath.Join(dir, "plugin.so")
	tmp := dest + ".tmp"
	if err := copyFile(tmp, sourcePath); err != nil {
		_ = os.Remove(tmp)
		return "", "", 0, err
	}
	if err := os.Rename(tmp, dest); err != nil {
		_ = os.Remove(tmp)
		return "", "", 0, fmt.Errorf("publish artifact: %w", err)
	}
	return dest, digest, size, nil
}

func (s *ArtifactStore) Open(path string) (*os.File, error) {
	return os.Open(path)
}

func (s *ArtifactStore) Delete(path string) error {
	if path == "" {
		return nil
	}
	return os.Remove(path)
}

func hashFile(path string) (string, int64, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", 0, fmt.Errorf("open artifact for hashing: %w", err)
	}
	defer file.Close()
	hash := sha256.New()
	size, err := io.Copy(hash, file)
	if err != nil {
		return "", 0, fmt.Errorf("hash artifact: %w", err)
	}
	return "sha256:" + hex.EncodeToString(hash.Sum(nil)), size, nil
}

func copyFile(dest, source string) error {
	in, err := os.Open(source)
	if err != nil {
		return fmt.Errorf("open source artifact: %w", err)
	}
	defer in.Close()
	out, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("create destination artifact: %w", err)
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return fmt.Errorf("copy artifact: %w", err)
	}
	return out.Sync()
}

func safePathSegment(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "unknown"
	}
	var b strings.Builder
	for _, r := range trimmed {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' || r == '.' {
			b.WriteRune(r)
		} else {
			b.WriteRune('_')
		}
	}
	return b.String()
}
