package tui

import (
	"encoding/json"
	"fmt"
	"github.com/yourorg/mimir-aip-cli/api"
)

// Performance TUI screen

type PerformanceMetric struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type PerformanceModel struct {
	Metrics []PerformanceMetric
	Loading bool
	Error   error
}

func NewPerformanceModel() *PerformanceModel {
	return &PerformanceModel{Loading: true}
}

func (m *PerformanceModel) FetchPerformance(client *api.Client) {
	m.Loading = true
	m.Error = nil
	data, err := client.GetPerformanceMetrics()
	if err != nil {
		m.Error = err
		m.Loading = false
		return
	}
	var metrics []PerformanceMetric
	if err := json.Unmarshal(data, &metrics); err != nil {
		m.Error = err
		m.Loading = false
		return
	}
	m.Metrics = metrics
	m.Loading = false
}

func (m PerformanceModel) View() string {
	if m.Loading {
		return "Loading performance metrics..."
	}
	if m.Error != nil {
		return fmt.Sprintf("Error loading performance: %v\nPress b to go back.", m.Error)
	}
	s := "Performance Metrics:\n"
	for _, metric := range m.Metrics {
		s += fmt.Sprintf("- %s: %s\n", metric.Name, metric.Value)
	}
	s += "\nPress b to go back."
	return s
}
