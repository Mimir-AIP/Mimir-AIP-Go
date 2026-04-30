package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/mimir-aip/mimir-aip-go/pkg/models"
	"github.com/mimir-aip/mimir-aip-go/pkg/plugins"
)

type PluginArtifactHandler struct {
	service *plugins.Service
}

func NewPluginArtifactHandler(service *plugins.Service) *PluginArtifactHandler {
	return &PluginArtifactHandler{service: service}
}

func (h *PluginArtifactHandler) HandleArtifacts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	kind := models.PluginKind(r.URL.Query().Get("kind"))
	name := r.URL.Query().Get("name")
	if kind == "" || name == "" {
		http.Error(w, "kind and name are required", http.StatusBadRequest)
		return
	}
	artifact, err := h.service.GetActiveArtifact(kind, name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(artifact) //nolint:errcheck
}

func (h *PluginArtifactHandler) HandleArtifact(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	artifactID := strings.TrimPrefix(r.URL.Path, "/api/plugin-artifacts/")
	artifactID = strings.Trim(artifactID, "/")
	if strings.HasSuffix(artifactID, "/download") {
		artifactID = strings.TrimSuffix(artifactID, "/download")
	}
	artifactID = strings.Trim(artifactID, "/")
	if artifactID == "" {
		http.Error(w, "artifact id is required", http.StatusBadRequest)
		return
	}
	if strings.HasSuffix(r.URL.Path, "/download") {
		h.handleDownload(w, r, artifactID)
		return
	}
	artifact, err := h.service.GetArtifact(artifactID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(artifact) //nolint:errcheck
}

func (h *PluginArtifactHandler) handleDownload(w http.ResponseWriter, r *http.Request, artifactID string) {
	artifact, file, err := h.service.OpenArtifact(artifactID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	defer file.Close()
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("X-Mimir-Artifact-Digest", artifact.Digest)
	w.Header().Set("X-Mimir-Source-Commit", artifact.SourceCommit)
	w.Header().Set("X-Mimir-Go-Version", artifact.GoVersion)
	w.Header().Set("X-Mimir-Host-Version", artifact.HostVersion)
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", artifact.PluginName+".so"))
	if artifact.SizeBytes > 0 {
		w.Header().Set("Content-Length", fmt.Sprintf("%d", artifact.SizeBytes))
	}
	http.ServeContent(w, r, artifact.PluginName+".so", artifact.UpdatedAt, file)
}
