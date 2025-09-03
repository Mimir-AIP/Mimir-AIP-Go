package tui

import (
	"encoding/json"
	"fmt"
	"github.com/yourorg/mimir-aip-cli/api"
)

// Plugins TUI screen

type Plugin struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

type PluginsModel struct {
	Plugins []Plugin
	Loading bool
	Error   error
}

func NewPluginsModel() *PluginsModel {
	return &PluginsModel{Loading: true}
}

func (m *PluginsModel) FetchPlugins(client *api.Client) {
	m.Loading = true
	m.Error = nil
	data, err := client.ListPlugins()
	if err != nil {
		m.Error = err
		m.Loading = false
		return
	}
	var plugins []Plugin
	if err := json.Unmarshal(data, &plugins); err != nil {
		m.Error = err
		m.Loading = false
		return
	}
	m.Plugins = plugins
	m.Loading = false
}

func (m PluginsModel) View() string {
	if m.Loading {
		return "Loading plugins..."
	}
	if m.Error != nil {
		return fmt.Sprintf("Error loading plugins: %v\nPress b to go back.", m.Error)
	}
	s := "Plugins:\n"
	for _, p := range m.Plugins {
		s += fmt.Sprintf("- %s (%s)\n", p.Name, p.Type)
	}
	s += "\nPress b to go back."
	return s
}
