package tui

import (
	"encoding/json"
	"fmt"
	"github.com/yourorg/mimir-aip-cli/api"
)

// Pipelines TUI screen

type Pipeline struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type PipelinesModel struct {
	Pipelines []Pipeline
	Loading   bool
	Error     error
}

func NewPipelinesModel() *PipelinesModel {
	return &PipelinesModel{Loading: true}
}

func (m *PipelinesModel) FetchPipelines(client *api.Client) {
	m.Loading = true
	m.Error = nil
	data, err := client.ListPipelines()
	if err != nil {
		m.Error = err
		m.Loading = false
		return
	}
	var pipelines []Pipeline
	if err := json.Unmarshal(data, &pipelines); err != nil {
		m.Error = err
		m.Loading = false
		return
	}
	m.Pipelines = pipelines
	m.Loading = false
}

func (m PipelinesModel) View() string {
	if m.Loading {
		return "Loading pipelines..."
	}
	if m.Error != nil {
		return fmt.Sprintf("Error loading pipelines: %v\nPress b to go back.", m.Error)
	}
	s := "Pipelines:\n"
	for _, p := range m.Pipelines {
		s += fmt.Sprintf("- %s: %s\n", p.ID, p.Name)
	}
	s += "\nPress b to go back."
	return s
}
