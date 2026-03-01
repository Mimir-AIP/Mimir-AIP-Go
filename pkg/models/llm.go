package models

import "time"

// ExternalLLMProvider records metadata about a dynamically installed LLM provider.
// The compiled .so is cached on disk; this record tracks its provenance and status.
type ExternalLLMProvider struct {
	Name          string    `json:"name"`
	RepositoryURL string    `json:"repository_url"`
	GitCommitHash string    `json:"git_commit_hash"`
	Status        string    `json:"status"` // "active" | "error"
	ErrorMessage  string    `json:"error_message,omitempty"`
	InstalledAt   time.Time `json:"installed_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// ExternalLLMProviderInstallRequest is the body for POST /api/llm/providers.
type ExternalLLMProviderInstallRequest struct {
	RepositoryURL string `json:"repository_url"`
	GitRef        string `json:"git_ref,omitempty"` // branch / tag / SHA; defaults to "main"
}
