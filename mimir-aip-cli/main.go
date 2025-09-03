package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/yourorg/mimir-aip-cli/api"
	"github.com/yourorg/mimir-aip-cli/tui"
)

var (
	titleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("5"))
	menuStyle  = lipgloss.NewStyle().Padding(1, 2)
)

var menuChoices = []string{
	"Dashboard",
	"Pipelines",
	"Jobs",
	"Plugins",
	"Config",
	"Performance",
	"Visualization",
	"Auth",
	"Quit",
}

type screen int

const (
	mainMenu screen = iota
	dashboardScreen
	pipelinesScreen
	jobsScreen
	pluginsScreen
	configScreen
	performanceScreen
	visualizationScreen
	authScreen
)

type model struct {
	cursor        int
	selected      int
	screen        screen
	client        *api.Client
	dashboard     tui.DashboardModel
	pipelines     *tui.PipelinesModel
	jobs          *tui.JobsModel
	plugins       *tui.PluginsModel
	config        *tui.ConfigModel
	performance   *tui.PerformanceModel
	visualization *tui.VisualizationModel
	auth          *tui.AuthModel
}

func initialModel() model {
	client := api.NewClient("http://localhost:8080") // TODO: make configurable
	return model{
		cursor:        0,
		selected:      -1,
		screen:        mainMenu,
		client:        client,
		dashboard:     tui.NewDashboardModel(),
		pipelines:     tui.NewPipelinesModel(),
		jobs:          tui.NewJobsModel(),
		plugins:       tui.NewPluginsModel(),
		config:        tui.NewConfigModel(),
		performance:   tui.NewPerformanceModel(),
		visualization: tui.NewVisualizationModel(),
		auth:          tui.NewAuthModel(),
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch m.screen {
		case mainMenu:
			switch msg.String() {
			case "ctrl+c", "q":
				return m, tea.Quit
			case "up", "k":
				if m.cursor > 0 {
					m.cursor--
				}
			case "down", "j":
				if m.cursor < len(menuChoices)-1 {
					m.cursor++
				}
			case "enter":
				m.selected = m.cursor
				switch menuChoices[m.cursor] {
				case "Quit":
					return m, tea.Quit
				case "Dashboard":
					m.screen = dashboardScreen
					m.dashboard.FetchDashboard(m.client)
				case "Pipelines":
					m.screen = pipelinesScreen
					m.pipelines.FetchPipelines(m.client)
				case "Jobs":
					m.screen = jobsScreen
					m.jobs.FetchJobs(m.client)
				case "Plugins":
					m.screen = pluginsScreen
					m.plugins.FetchPlugins(m.client)
				case "Config":
					m.screen = configScreen
					m.config.FetchConfig(m.client)
				case "Performance":
					m.screen = performanceScreen
					m.performance.FetchPerformance(m.client)
				case "Visualization":
					m.screen = visualizationScreen
					m.visualization.FetchVisualization(m.client)
				case "Auth":
					m.screen = authScreen
					m.auth.FetchAuth(m.client)
				}
				if menuChoices[m.cursor] == "Dashboard" {
					m.screen = dashboardScreen
					m.dashboard.FetchDashboard(m.client)
				} else if menuChoices[m.cursor] == "Pipelines" {
					m.screen = pipelinesScreen
					m.pipelines.FetchPipelines(m.client)
				} else if menuChoices[m.cursor] == "Jobs" {
					m.screen = jobsScreen
					m.jobs.FetchJobs(m.client)
				} else if menuChoices[m.cursor] == "Plugins" {
					m.screen = pluginsScreen
					m.plugins.FetchPlugins(m.client)
				} else if menuChoices[m.cursor] == "Config" {
					m.screen = configScreen
				} else if menuChoices[m.cursor] == "Performance" {
					m.screen = performanceScreen
					m.performance.FetchPerformance(m.client)
				} else if menuChoices[m.cursor] == "Visualization" {
					m.screen = visualizationScreen
					m.visualization.FetchVisualization(m.client)
				} else if menuChoices[m.cursor] == "Auth" {
					m.screen = authScreen
					m.auth.FetchAuth(m.client)
				}
			}
		case dashboardScreen, pipelinesScreen, jobsScreen, pluginsScreen, configScreen, performanceScreen, visualizationScreen, authScreen:
			if msg.String() == "b" {
				m.screen = mainMenu
			}
		}
	}
	return m, nil
}

func (m model) View() string {
	switch m.screen {
	case mainMenu:
		s := titleStyle.Render("Mimir-AIP CLI/TUI") + "\n\n"
		for i, choice := range menuChoices {
			cursor := "  "
			if m.cursor == i {
				cursor = "> "
			}
			s += menuStyle.Render(fmt.Sprintf("%s%s", cursor, choice)) + "\n"
		}
		s += "\nUse ↑/↓ to navigate, Enter to select, q to quit."
		return s
	case dashboardScreen:
		return m.dashboard.View()
	case pipelinesScreen:
		return m.pipelines.View()
	case jobsScreen:
		return m.jobs.View()
	case pluginsScreen:
		return m.plugins.View()
	case configScreen:
		return m.config.View()
	case performanceScreen:
		return m.performance.View()
	case visualizationScreen:
		return m.visualization.View()
	case authScreen:
		return m.auth.View()
	}
	return ""
}

func main() {
	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}
