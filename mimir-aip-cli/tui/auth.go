package tui

import (
	"encoding/json"
	"fmt"
	"github.com/yourorg/mimir-aip-cli/api"
)

// Auth TUI screen

type AuthInfo struct {
	User string `json:"user"`
	Role string `json:"role"`
}

type AuthModel struct {
	Info    []AuthInfo
	Loading bool
	Error   error
}

func NewAuthModel() *AuthModel {
	return &AuthModel{Loading: true}
}

func (m *AuthModel) FetchAuth(client *api.Client) {
	m.Loading = true
	m.Error = nil
	data, err := client.AuthMe()
	if err != nil {
		m.Error = err
		m.Loading = false
		return
	}
	var info []AuthInfo
	if err := json.Unmarshal(data, &info); err != nil {
		m.Error = err
		m.Loading = false
		return
	}
	m.Info = info
	m.Loading = false
}

func (m AuthModel) View() string {
	if m.Loading {
		return "Loading auth info..."
	}
	if m.Error != nil {
		return fmt.Sprintf("Error loading auth: %v\nPress b to go back.", m.Error)
	}
	s := "Auth Info:\n"
	for _, i := range m.Info {
		s += fmt.Sprintf("- User: %s, Role: %s\n", i.User, i.Role)
	}
	s += "\nPress b to go back."
	return s
}
