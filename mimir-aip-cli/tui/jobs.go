package tui

import (
	"encoding/json"
	"fmt"
	"github.com/yourorg/mimir-aip-cli/api"
)

// Jobs TUI screen

type Job struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status"`
}

type JobsModel struct {
	Jobs    []Job
	Loading bool
	Error   error
}

func NewJobsModel() *JobsModel {
	return &JobsModel{Loading: true}
}

func (m *JobsModel) FetchJobs(client *api.Client) {
	m.Loading = true
	m.Error = nil
	data, err := client.ListJobs()
	if err != nil {
		m.Error = err
		m.Loading = false
		return
	}
	var jobs []Job
	if err := json.Unmarshal(data, &jobs); err != nil {
		m.Error = err
		m.Loading = false
		return
	}
	m.Jobs = jobs
	m.Loading = false
}

func (m JobsModel) View() string {
	if m.Loading {
		return "Loading jobs..."
	}
	if m.Error != nil {
		return fmt.Sprintf("Error loading jobs: %v\nPress b to go back.", m.Error)
	}
	s := "Jobs:\n"
	for _, job := range m.Jobs {
		s += fmt.Sprintf("- %s (%s): %s\n", job.ID, job.Name, job.Status)
	}
	s += "\nPress b to go back."
	return s
}
