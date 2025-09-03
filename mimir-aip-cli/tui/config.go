package tui

import (
	"encoding/json"
	"fmt"
	"github.com/yourorg/mimir-aip-cli/api"
)

// Config TUI screen

type Config struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type ConfigModel struct {
	Config  []Config
	Loading bool
	Error   error
}

func NewConfigModel() *ConfigModel {
	return &ConfigModel{Loading: true}
}

func (m *ConfigModel) FetchConfig(client *api.Client) {
	m.Loading = true
	m.Error = nil
	data, err := client.GetConfig()
	if err != nil {
		m.Error = err
		m.Loading = false
		return
	}
	var config []Config
	if err := json.Unmarshal(data, &config); err != nil {
		m.Error = err
		m.Loading = false
		return
	}
	m.Config = config
	m.Loading = false
}

func (m ConfigModel) View() string {
	if m.Loading {
		return "Loading config..."
	}
	if m.Error != nil {
		return fmt.Sprintf("Error loading config: %v\nPress b to go back.", m.Error)
	}
	s := "Config:\n"
	for _, c := range m.Config {
		s += fmt.Sprintf("- %s: %s\n", c.Key, c.Value)
	}
	s += "\nPress b to go back."
	return s
}
