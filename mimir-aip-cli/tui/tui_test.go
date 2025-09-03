package tui

import (
	"github.com/yourorg/mimir-aip-cli/api"
	"testing"
)

func TestJobsModel(t *testing.T) {
	model := NewJobsModel()
	if model.Loading != true {
		t.Error("Expected loading to be true")
	}
	// Mock client for testing
	client := api.NewClient("http://localhost:8080")
	model.FetchJobs(client)
	// Note: This will fail without a real server, but tests the structure
}

func TestPipelinesModel(t *testing.T) {
	model := NewPipelinesModel()
	if model.Loading != true {
		t.Error("Expected loading to be true")
	}
}
