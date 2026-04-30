package models

import "time"

type PluginKind string

const (
	PluginKindPipeline    PluginKind = "pipeline"
	PluginKindMLProvider  PluginKind = "ml_provider"
	PluginKindStorage     PluginKind = "storage"
	PluginKindLLMProvider PluginKind = "llm_provider"
)

type PluginArtifactStatus string

const (
	PluginArtifactStatusBuilding PluginArtifactStatus = "building"
	PluginArtifactStatusActive   PluginArtifactStatus = "active"
	PluginArtifactStatusError    PluginArtifactStatus = "error"
)

type PluginArtifact struct {
	ID               string               `json:"id"`
	PluginKind       PluginKind           `json:"plugin_kind"`
	PluginName       string               `json:"plugin_name"`
	SourceRepository string               `json:"source_repository"`
	SourceRef        string               `json:"source_ref"`
	SourceCommit     string               `json:"source_commit"`
	Digest           string               `json:"digest"`
	LocalPath        string               `json:"-"`
	SizeBytes        int64                `json:"size_bytes"`
	GoVersion        string               `json:"go_version"`
	GOOS             string               `json:"goos"`
	GOARCH           string               `json:"goarch"`
	HostVersion      string               `json:"host_version"`
	SymbolName       string               `json:"symbol_name"`
	Status           PluginArtifactStatus `json:"status"`
	ErrorMessage     string               `json:"error_message,omitempty"`
	CreatedAt        time.Time            `json:"created_at"`
	UpdatedAt        time.Time            `json:"updated_at"`
}
