package tui

import "github.com/charmbracelet/lipgloss"

// Styles for TUI components

var (
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("6")).
			MarginBottom(1)

	MenuStyle = lipgloss.NewStyle().
			Padding(1, 2)

	ErrorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("9")).
			Bold(true)

	LoadingStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("11")).
			Italic(true)
)
