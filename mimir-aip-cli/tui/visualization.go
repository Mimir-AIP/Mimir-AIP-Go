package tui

import (
	"encoding/json"
	"fmt"
	"github.com/yourorg/mimir-aip-cli/api"
)

// Visualization TUI screen

type VisualizationData struct {
	Type string `json:"type"`
	Data string `json:"data"`
}

type VisualizationModel struct {
	Data    []VisualizationData
	Loading bool
	Error   error
}

func NewVisualizationModel() *VisualizationModel {
	return &VisualizationModel{Loading: true}
}

func (m *VisualizationModel) FetchVisualization(client *api.Client) {
	m.Loading = true
	m.Error = nil
	data, err := client.VisualizeStatus()
	if err != nil {
		m.Error = err
		m.Loading = false
		return
	}
	var viz []VisualizationData
	if err := json.Unmarshal(data, &viz); err != nil {
		m.Error = err
		m.Loading = false
		return
	}
	m.Data = viz
	m.Loading = false
}

func (m VisualizationModel) View() string {
	if m.Loading {
		return "Loading visualization..."
	}
	if m.Error != nil {
		return fmt.Sprintf("Error loading visualization: %v\nPress b to go back.", m.Error)
	}
	s := "Visualization:\n"
	for _, d := range m.Data {
		s += fmt.Sprintf("- %s: %s\n", d.Type, d.Data)
	}
	s += "\nPress b to go back."
	return s
}
