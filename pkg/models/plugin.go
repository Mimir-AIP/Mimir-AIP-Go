package models

import "time"

// PluginStatus represents the status of a plugin
type PluginStatus string

const (
	PluginStatusActive   PluginStatus = "active"
	PluginStatusDisabled PluginStatus = "disabled"
	PluginStatusError    PluginStatus = "error"
)

// Plugin represents a custom plugin
type Plugin struct {
	ID               string           `json:"id"`
	Name             string           `json:"name"`
	Version          string           `json:"version"`
	Description      string           `json:"description,omitempty"`
	Author           string           `json:"author,omitempty"`
	RepositoryURL    string           `json:"repository_url"`
	GitCommitHash    string           `json:"git_commit_hash,omitempty"`
	PluginDefinition PluginDefinition `json:"plugin_definition"`
	Status           PluginStatus     `json:"status"`
	CreatedAt        time.Time        `json:"created_at"`
	UpdatedAt        time.Time        `json:"updated_at"`
	LastLoadedAt     *time.Time       `json:"last_loaded_at,omitempty"`
	Actions          []PluginAction   `json:"actions"`
}

// PluginDefinition represents the parsed plugin.yaml content
type PluginDefinition struct {
	Name        string         `json:"name" yaml:"name"`
	Version     string         `json:"version" yaml:"version"`
	Description string         `json:"description,omitempty" yaml:"description,omitempty"`
	Author      string         `json:"author,omitempty" yaml:"author,omitempty"`
	Repository  string         `json:"repository,omitempty" yaml:"repository,omitempty"`
	Actions     []ActionSchema `json:"actions" yaml:"actions"`
}

// ActionSchema defines the schema for a plugin action
type ActionSchema struct {
	Name        string            `json:"name" yaml:"name"`
	Description string            `json:"description,omitempty" yaml:"description,omitempty"`
	Parameters  []ParameterSchema `json:"parameters,omitempty" yaml:"parameters,omitempty"`
	Returns     []ReturnSchema    `json:"returns,omitempty" yaml:"returns,omitempty"`
}

// ParameterSchema defines a parameter for an action
type ParameterSchema struct {
	Name        string `json:"name" yaml:"name"`
	Type        string `json:"type" yaml:"type"`
	Required    bool   `json:"required" yaml:"required"`
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
}

// ReturnSchema defines a return value for an action
type ReturnSchema struct {
	Name        string `json:"name" yaml:"name"`
	Type        string `json:"type" yaml:"type"`
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
}

// PluginAction represents an action provided by a plugin
type PluginAction struct {
	ID          string            `json:"id"`
	PluginID    string            `json:"plugin_id"`
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	Parameters  []ParameterSchema `json:"parameters,omitempty"`
	Returns     []ReturnSchema    `json:"returns,omitempty"`
}

// PluginInstallRequest represents a request to install a plugin
type PluginInstallRequest struct {
	RepositoryURL string `json:"repository_url"`
	GitRef        string `json:"git_ref,omitempty"` // branch, tag, or commit hash (defaults to main/master)
}

// PluginUpdateRequest represents a request to update a plugin
type PluginUpdateRequest struct {
	GitRef *string       `json:"git_ref,omitempty"`
	Status *PluginStatus `json:"status,omitempty"`
}
