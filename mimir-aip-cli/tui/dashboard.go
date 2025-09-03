package tui

import "github.com/yourorg/mimir-aip-cli/api"

// Dashboard title style is now in styles.go

// DashboardModel holds dashboard state
// Extend with health, metrics, recent jobs, etc.
type DashboardModel struct {
	Health string
}

func NewDashboardModel() DashboardModel {
	return DashboardModel{Health: "Unknown"}
}

func (m *DashboardModel) FetchDashboard(client *api.Client) {
	// Fetch health
	err := client.HealthCheck()
	if err != nil {
		m.Health = "Unhealthy"
	} else {
		m.Health = "Healthy"
	}
	// TODO: Fetch metrics and recent jobs
}

func (m DashboardModel) View() string {
	s := TitleStyle.Render("Dashboard") + "\n\n"
	s += "Health: " + m.Health + "\n"
	s += "Metrics: (to be implemented)\n"
	s += "Recent Jobs: (to be implemented)\n"
	s += "\nPress b to go back."
	return s
}
