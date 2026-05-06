package tui

import "github.com/charmbracelet/lipgloss"

var (
	styleTitle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	styleSelected = lipgloss.NewStyle().Background(lipgloss.Color("236")).Bold(true)
	styleDisabled = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	styleEnabled  = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	styleHelp     = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	styleError    = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
	styleStatus   = lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
)
